package tunnel

import (
	"io"
	"log"
	"sync"
	"sync/atomic"

	"github.com/pozii/minegate/internal"
	"github.com/pozii/minegate/packet"
)

// Relay performs zero-copy bidirectional packet forwarding between two connections.
type Relay struct {
	client packetReaderWriter
	server packetReaderWriter
	closed atomic.Bool
	wg     sync.WaitGroup
	metrics RelayMetrics
}

// RelayMetrics holds relay metrics.
type RelayMetrics struct {
	ClientToServer atomic.Int64
	ServerToClient atomic.Int64
	Errors         atomic.Int64
}

type packetReaderWriter interface {
	ReadRawPacket() (packet.RawPacket, error)
	WritePacket(packet.Packet) error
	Close() error
}

// NewRelay creates a new Relay.
func NewRelay(client, server packetReaderWriter) *Relay {
	return &Relay{
		client: client,
		server: server,
	}
}

// Start starts the bidirectional relay.
func (r *Relay) Start() {
	r.wg.Add(2)
	go r.relay(r.client, r.server, &r.metrics.ClientToServer)
	go r.relay(r.server, r.client, &r.metrics.ServerToClient)
}

// Wait waits for the relay to finish.
func (r *Relay) Wait() {
	r.wg.Wait()
}

func (r *Relay) relay(src, dst packetReaderWriter, counter *atomic.Int64) {
	defer r.wg.Done()
	defer r.closeBoth()

	for {
		if r.closed.Load() {
			return
		}

		rp, err := src.ReadRawPacket()
		if err != nil {
			if err != io.EOF && !r.closed.Load() {
				r.metrics.Errors.Add(1)
			}
			return
		}

		p := packet.Packet{
			ID:   packet.VarInt(rp.Buf[0]), // approximate
			Data: rp.Buf[1:],
		}

		// Zero-copy: use RawPacket's buffer directly
		if err := dst.WritePacket(p); err != nil {
			if !r.closed.Load() {
				r.metrics.Errors.Add(1)
			}
			rp.Release()
			return
		}

		counter.Add(1)
		rp.Release()
	}
}

func (r *Relay) closeBoth() {
	if !r.closed.CompareAndSwap(false, true) {
		return
	}
	r.client.Close()
	r.server.Close()
}

// DirectRelay performs direct byte copying between two net.Conn.
// It does not parse packets, making it the fastest relay method.
func DirectRelay(client, server io.ReadWriteCloser) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(server, client)
		server.Close()
		client.Close()
	}()

	go func() {
		defer wg.Done()
		io.Copy(client, server)
		client.Close()
		server.Close()
	}()

	wg.Wait()
}

// RelayWithMetrics is a relay with metrics collection.
func RelayWithMetrics(client, server io.ReadWriteCloser, onByte func(direction string, n int)) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		buf := make([]byte, 32768)
		for {
			n, err := client.Read(buf)
			if n > 0 {
				if _, werr := server.Write(buf[:n]); werr != nil {
					break
				}
				onByte("c2s", n)
			}
			if err != nil {
				break
			}
		}
		server.Close()
		client.Close()
	}()

	go func() {
		defer wg.Done()
		buf := make([]byte, 32768)
		for {
			n, err := server.Read(buf)
			if n > 0 {
				if _, werr := client.Write(buf[:n]); werr != nil {
					break
				}
				onByte("s2c", n)
			}
			if err != nil {
				break
			}
		}
		client.Close()
		server.Close()
	}()

	wg.Wait()
}

// Ensure log is used
var _ = log.Printf
var _ = internal.ErrConnectionClosed
