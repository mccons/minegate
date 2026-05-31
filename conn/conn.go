package conn

import (
	"crypto/cipher"
	"io"
	"sync"
	"sync/atomic"

	"github.com/user/minegate/compress"
	"github.com/user/minegate/crypto"
	"github.com/user/minegate/internal"
	"github.com/user/minegate/packet"
)

// MineConn manages a Minecraft connection.
// It combines packet reading/writing, encryption, and compression.
type MineConn struct {
	conn      io.ReadWriteCloser
	reader    *packet.PacketReader
	writer    *packet.PacketWriter
	threshold int32

	encrypt cipher.Stream
	decrypt cipher.Stream

	readBuf  []byte
	writeBuf []byte

	closed  atomic.Bool
	closeMu sync.Mutex

	flow *FlowController
	prio *PriorityQueue
}

// NewMineConn creates a new MineConn.
func NewMineConn(conn io.ReadWriteCloser) *MineConn {
	mc := &MineConn{
		conn:   conn,
		reader: packet.NewPacketReader(conn, -1),
		writer: packet.NewPacketWriter(conn, -1),
		readBuf:   internal.GetBuffer(internal.DefaultBufferSize),
		writeBuf:  internal.GetBuffer(internal.DefaultBufferSize),
		flow: NewFlowController(1000, internal.MaxPacketSize*100),
		prio: NewPriorityQueue(3),
	}
	mc.prio.cap[Urgent] = 64
	mc.prio.cap[Normal] = 256
	mc.prio.cap[Low] = 1024
	return mc
}

// SetThreshold sets the compression threshold.
func (mc *MineConn) SetThreshold(t int) {
	atomic.StoreInt32(&mc.threshold, int32(t))
	mc.reader = packet.NewPacketReader(mc.conn, t)
	mc.writer = packet.NewPacketWriter(mc.conn, t)
}

// Threshold returns the current threshold value.
func (mc *MineConn) Threshold() int {
	return int(atomic.LoadInt32(&mc.threshold))
}

// SetCipher sets the encryption streams.
func (mc *MineConn) SetCipher(encrypt, decrypt cipher.Stream) {
	mc.encrypt = encrypt
	mc.decrypt = decrypt
}

// ReadPacket reads a packet.
func (mc *MineConn) ReadPacket() (packet.Packet, error) {
	if mc.IsClosed() {
		return packet.Packet{}, internal.ErrConnectionClosed
	}

	p, err := mc.reader.ReadPacket()
	if err != nil {
		mc.Close()
		return packet.Packet{}, err
	}

	if mc.decrypt != nil {
		dec := make([]byte, len(p.Data))
		mc.decrypt.XORKeyStream(dec, p.Data)
		p.Data = dec
	}

	return p, nil
}

// ReadRawPacket reads a raw packet (zero-copy).
func (mc *MineConn) ReadRawPacket() (packet.RawPacket, error) {
	if mc.IsClosed() {
		return packet.RawPacket{}, internal.ErrConnectionClosed
	}

	rp, err := mc.reader.ReadRawPacket()
	if err != nil {
		mc.Close()
		return packet.RawPacket{}, err
	}

	if mc.decrypt != nil {
		mc.decrypt.XORKeyStream(rp.Buf, rp.Buf)
	}

	return rp, nil
}

// WritePacket writes a packet.
func (mc *MineConn) WritePacket(p packet.Packet) error {
	if mc.IsClosed() {
		return internal.ErrConnectionClosed
	}

	if mc.encrypt != nil {
		enc := make([]byte, len(p.Data))
		mc.encrypt.XORKeyStream(enc, p.Data)
		p.Data = enc
	}

	if err := mc.flow.Acquire(len(p.Data)); err != nil {
		return err
	}
	defer mc.flow.Release(len(p.Data))

	return mc.writer.WritePacket(p)
}

// WritePacketPriority writes a packet with a priority level.
func (mc *MineConn) WritePacketPriority(p packet.Packet, priority Priority) error {
	if mc.IsClosed() {
		return internal.ErrConnectionClosed
	}

	if mc.encrypt != nil {
		enc := make([]byte, len(p.Data))
		mc.encrypt.XORKeyStream(enc, p.Data)
		p.Data = enc
	}

	item := &PriorityItem{
		Data:     p,
		Priority: priority,
		Size:     len(p.Data),
	}

	return mc.prio.Push(item)
}

// IsClosed returns whether the connection is closed.
func (mc *MineConn) IsClosed() bool {
	return mc.closed.Load()
}

// Close closes the connection.
func (mc *MineConn) Close() error {
	mc.closeMu.Lock()
	defer mc.closeMu.Unlock()

	if mc.closed.Load() {
		return nil
	}
	mc.closed.Store(true)

	internal.PutBuffer(mc.readBuf)
	internal.PutBuffer(mc.writeBuf)

	return mc.conn.Close()
}

// LocalAddr returns the local address.
func (mc *MineConn) LocalAddr() string {
	if addr, ok := mc.conn.(interface{ LocalAddr() string }); ok {
		return addr.LocalAddr()
	}
	return ""
}

// RemoteAddr returns the remote address.
func (mc *MineConn) RemoteAddr() string {
	if addr, ok := mc.conn.(interface{ RemoteAddr() string }); ok {
		return addr.RemoteAddr()
	}
	return ""
}

var _ = compress.Compress
var _ = crypto.NewCFB8Encrypt
