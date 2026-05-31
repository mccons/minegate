package tunnel

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/user/minegate/internal"
)

// MuxType is the multiplexer frame type.
type MuxType byte

const (
	MuxData  MuxType = iota // Normal data
	MuxOpen                 // Open new stream
	MuxClose                // Close stream
	MuxPing                 // Keepalive ping
	MuxPong                 // Keepalive pong
	MuxError                // Error notification
)

// MuxFrame is the multiplexer frame structure.
type MuxFrame struct {
	Type   MuxType
	Stream uint32
	Data   []byte
}

// Mux manages multiple virtual streams over a single physical connection.
type Mux struct {
	conn    net.Conn
	nextID  atomic.Uint32
	streams sync.Map
	mu      sync.Mutex
	closed  atomic.Bool
	readCh  chan *MuxFrame
	writeCh chan *MuxFrame
	stopCh  chan struct{}
}

// NewMux creates a new Mux.
func NewMux(conn net.Conn) *Mux {
	m := &Mux{
		conn:    conn,
		readCh:  make(chan *MuxFrame, 256),
		writeCh: make(chan *MuxFrame, 256),
		stopCh:  make(chan struct{}),
	}

	go m.readLoop()
	go m.writeLoop()

	return m
}

// OpenStream opens a new virtual stream.
func (m *Mux) OpenStream() (*MuxStream, error) {
	if m.closed.Load() {
		return nil, internal.ErrConnectionClosed
	}

	id := m.nextID.Add(1)
	stream := &MuxStream{
		id:     id,
		mux:    m,
		readCh: make(chan []byte, 64),
		done:   make(chan struct{}),
	}

	m.streams.Store(id, stream)

	// Send MuxOpen frame
	m.writeCh <- &MuxFrame{Type: MuxOpen, Stream: id}

	return stream, nil
}

// AcceptStream accepts a new virtual stream.
func (m *Mux) AcceptStream() (*MuxStream, error) {
	for {
		frame := <-m.readCh
		if frame == nil {
			return nil, io.EOF
		}
		if frame.Type == MuxOpen {
			stream := &MuxStream{
				id:     frame.Stream,
				mux:    m,
				readCh: make(chan []byte, 64),
				done:   make(chan struct{}),
			}
			m.streams.Store(frame.Stream, stream)
			return stream, nil
		}
	}
}

// Close closes the multiplexer.
func (m *Mux) Close() error {
	if !m.closed.CompareAndSwap(false, true) {
		return nil
	}

	m.streams.Range(func(_, v interface{}) bool {
		if s, ok := v.(*MuxStream); ok {
			s.close()
		}
		return true
	})

	close(m.stopCh)
	return m.conn.Close()
}

func (m *Mux) readLoop() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("mux readLoop panic: %v", r)
		}
	}()

	for {
		if m.closed.Load() {
			return
		}

		frame, err := m.readFrame()
		if err != nil {
			m.Close()
			return
		}

		switch frame.Type {
		case MuxData:
			if v, ok := m.streams.Load(frame.Stream); ok {
				stream := v.(*MuxStream)
				select {
				case stream.readCh <- frame.Data:
				default:
					// Stream buffer full, drop
				}
			}
		case MuxOpen:
			select {
			case m.readCh <- frame:
			default:
			}
		case MuxClose:
			if v, ok := m.streams.LoadAndDelete(frame.Stream); ok {
				v.(*MuxStream).close()
			}
		case MuxPing:
			m.writeCh <- &MuxFrame{Type: MuxPong}
		default:
			// ignore
		}
	}
}

func (m *Mux) writeLoop() {
	for {
		select {
		case frame := <-m.writeCh:
			if m.closed.Load() {
				return
			}
			if err := m.writeFrame(frame); err != nil {
				m.Close()
				return
			}
		case <-m.stopCh:
			return
		}
	}
}

func (m *Mux) readFrame() (*MuxFrame, error) {
	header := make([]byte, 5) // type(1) + streamID(4)
	if _, err := io.ReadFull(m.conn, header); err != nil {
		return nil, err
	}

	frame := &MuxFrame{
		Type:   MuxType(header[0]),
		Stream: binary.BigEndian.Uint32(header[1:5]),
	}

	if frame.Type == MuxData {
		lenBuf := make([]byte, 4)
		if _, err := io.ReadFull(m.conn, lenBuf); err != nil {
			return nil, err
		}
		dataLen := binary.BigEndian.Uint32(lenBuf)
		frame.Data = make([]byte, dataLen)
		if _, err := io.ReadFull(m.conn, frame.Data); err != nil {
			return nil, err
		}
	}

	return frame, nil
}

func (m *Mux) writeFrame(frame *MuxFrame) error {
	header := make([]byte, 5)
	header[0] = byte(frame.Type)
	binary.BigEndian.PutUint32(header[1:5], frame.Stream)

	if frame.Type == MuxData {
		lenBuf := make([]byte, 4)
		binary.BigEndian.PutUint32(lenBuf, uint32(len(frame.Data)))
		data := append(header, lenBuf...)
		data = append(data, frame.Data...)
		_, err := m.conn.Write(data)
		return err
	}

	_, err := m.conn.Write(header)
	return err
}

// MuxStream is a virtual stream on top of the multiplexer.
type MuxStream struct {
	id     uint32
	mux    *Mux
	readCh chan []byte
	done   chan struct{}
	closed atomic.Bool
}

func (ms *MuxStream) Read(b []byte) (int, error) {
	data, ok := <-ms.readCh
	if !ok {
		return 0, io.EOF
	}
	return copy(b, data), nil
}

func (ms *MuxStream) Write(b []byte) (int, error) {
	if ms.closed.Load() {
		return 0, internal.ErrConnectionClosed
	}
	ms.mux.writeCh <- &MuxFrame{
		Type:   MuxData,
		Stream: ms.id,
		Data:   append([]byte{}, b...),
	}
	return len(b), nil
}

func (ms *MuxStream) Close() error {
	if !ms.closed.CompareAndSwap(false, true) {
		return nil
	}
	ms.mux.writeCh <- &MuxFrame{Type: MuxClose, Stream: ms.id}
	ms.mux.streams.Delete(ms.id)
	close(ms.readCh)
	return nil
}

func (ms *MuxStream) close() {
	if !ms.closed.CompareAndSwap(false, true) {
		return
	}
	close(ms.readCh)
}

func (ms *MuxStream) LocalAddr() net.Addr {
	return ms.mux.conn.LocalAddr()
}

func (ms *MuxStream) RemoteAddr() net.Addr {
	return ms.mux.conn.RemoteAddr()
}

func (ms *MuxStream) SetDeadline(t time.Time) error {
	return ms.mux.conn.SetDeadline(t)
}

func (ms *MuxStream) SetReadDeadline(t time.Time) error {
	return ms.mux.conn.SetReadDeadline(t)
}

func (ms *MuxStream) SetWriteDeadline(t time.Time) error {
	return ms.mux.conn.SetWriteDeadline(t)
}

// ID returns the stream identifier.
func (ms *MuxStream) ID() uint32 { return ms.id }
