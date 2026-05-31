package tunnel

import (
	"context"
	"net"
	"time"

	"github.com/user/minegate/packet"
	"github.com/user/minegate/transport"
)

// Listener is a listener that accepts Minecraft connections.
type Listener struct {
	transport transport.Transport
	ln        net.Listener
}

// NewListener creates a new Listener.
func NewListener(tr transport.Transport) *Listener {
	return &Listener{transport: tr}
}

// Listen starts listening on the given address.
func (l *Listener) Listen(addr string) error {
	ctx := context.Background()
	ln, err := l.transport.Listen(ctx, addr)
	if err != nil {
		return err
	}
	l.ln = ln
	return nil
}

// Accept accepts a connection.
func (l *Listener) Accept() (*MineConnWrapper, error) {
	if l.ln == nil {
		return nil, net.ErrClosed
	}

	conn, err := l.ln.Accept()
	if err != nil {
		return nil, err
	}

	return &MineConnWrapper{
		Reader: packet.NewPacketReader(conn, -1),
		Writer: packet.NewPacketWriter(conn, -1),
		Conn:   conn,
	}, nil
}

// Close closes the listener.
func (l *Listener) Close() error {
	if l.ln != nil {
		return l.ln.Close()
	}
	return nil
}

// Addr returns the listener address.
func (l *Listener) Addr() net.Addr {
	if l.ln != nil {
		return l.ln.Addr()
	}
	return nil
}

// SetTransport changes the transport in use.
func (l *Listener) SetTransport(tr transport.Transport) {
	l.transport = tr
}

// WaitTimeout is the maximum time to wait for a connection.
var WaitTimeout = 30 * time.Second
