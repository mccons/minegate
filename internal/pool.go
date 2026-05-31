package internal

import "sync"

const (
	DefaultBufferSize  = 1024
	MaxPacketSize      = 0x200000 // 2MB
	MaxCompressedSize  = 0x200000
	BufferPoolMax      = 1 << 20 // 1MB
)

var (
	bytePool = sync.Pool{New: func() any { b := make([]byte, DefaultBufferSize); return &b }}
	bufPool  = make(chan []byte, 64)
)

func GetBuffer(size int) []byte {
	if size > BufferPoolMax {
		return make([]byte, size)
	}
	select {
	case b := <-bufPool:
		if cap(b) >= size {
			return b[:size]
		}
	default:
	}
	return make([]byte, size)
}

func PutBuffer(b []byte) {
	if cap(b) > BufferPoolMax || cap(b) == 0 {
		return
	}
	select {
	case bufPool <- b[:cap(b)]:
	default:
	}
}

type PooledBuffer struct {
	Buf []byte
}

func NewPooledBuffer(size int) *PooledBuffer {
	return &PooledBuffer{Buf: GetBuffer(size)}
}

func (pb *PooledBuffer) Release() {
	if pb.Buf != nil {
		PutBuffer(pb.Buf)
		pb.Buf = nil
	}
}
