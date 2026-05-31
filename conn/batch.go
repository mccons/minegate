package conn

import (
	"io"
	"sync"
	"time"

	"github.com/pozii/minegate/internal"
	"github.com/pozii/minegate/packet"
)

// BatchWriter performs packet batching.
// It reduces syscall count by sending multiple packets in a single TCP write.
type BatchWriter struct {
	dst       io.Writer
	buf       []byte
	buffered  int
	threshold int
	flushInt  time.Duration
	mu        sync.Mutex
	closed    bool

	flushTimer *time.Timer
	flushCh    chan struct{}
}

// NewBatchWriter creates a new BatchWriter.
// flushInterval: max time to wait after the last packet (0 = flush immediately)
// bufferSize: internal buffer size
func NewBatchWriter(dst io.Writer, flushInterval time.Duration, bufferSize int) *BatchWriter {
	bw := &BatchWriter{
		dst:       dst,
		buf:       internal.GetBuffer(bufferSize),
		flushInt:  flushInterval,
		flushCh:   make(chan struct{}, 1),
		threshold: bufferSize / 2,
	}

	if flushInterval > 0 {
		bw.flushTimer = time.NewTimer(flushInterval)
		bw.flushTimer.Stop()
		go bw.flushLoop()
	}

	return bw
}

// Write appends data to the buffer.
func (bw *BatchWriter) Write(data []byte) (int, error) {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	if bw.closed {
		return 0, internal.ErrConnectionClosed
	}

	if len(data) > cap(bw.buf)-bw.buffered {
		if err := bw.flushLocked(); err != nil {
			return 0, err
		}
	}

	if len(data) > cap(bw.buf) {
		return bw.dst.Write(data)
	}

	n := copy(bw.buf[bw.buffered:], data)
	bw.buffered += n

	if bw.buffered >= bw.threshold {
		if err := bw.flushLocked(); err != nil {
			return n, err
		}
	} else if bw.flushInt > 0 {
		bw.flushTimer.Reset(bw.flushInt)
	}

	return len(data), nil
}

// Flush writes all buffered data.
func (bw *BatchWriter) Flush() error {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	return bw.flushLocked()
}

func (bw *BatchWriter) flushLocked() error {
	if bw.buffered == 0 {
		return nil
	}
	_, err := bw.dst.Write(bw.buf[:bw.buffered])
	bw.buffered = 0
	if bw.flushTimer != nil {
		bw.flushTimer.Stop()
	}
	return err
}

func (bw *BatchWriter) flushLoop() {
	for range bw.flushTimer.C {
		bw.mu.Lock()
		if !bw.closed && bw.buffered > 0 {
			bw.flushLocked()
		}
		bw.mu.Unlock()
	}
}

// Close closes the BatchWriter.
func (bw *BatchWriter) Close() error {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	bw.closed = true
	if bw.flushTimer != nil {
		bw.flushTimer.Stop()
	}
	err := bw.flushLocked()
	internal.PutBuffer(bw.buf)
	return err
}

// NewBatchedPacketWriter returns a PacketWriter with batching support.
func NewBatchedPacketWriter(dst io.Writer, threshold int, flushInterval time.Duration) *packet.PacketWriter {
	bw := NewBatchWriter(dst, flushInterval, 65536)
	return packet.NewPacketWriter(bw, threshold)
}

var _ = packet.Packet{}
