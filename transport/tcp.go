package transport

import (
	"context"
	"net"
	"time"
)

// TCPTransport is the standard TCP transport.
type TCPTransport struct {
	DialTimeout time.Duration
}

// Dial opens a TCP connection.
func (t *TCPTransport) Dial(ctx context.Context, addr string) (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout: t.DialTimeout,
	}
	if t.DialTimeout == 0 {
		dialer.Timeout = 30 * time.Second
	}
	return dialer.DialContext(ctx, "tcp", addr)
}

// Listen starts a TCP listener.
func (t *TCPTransport) Listen(ctx context.Context, addr string) (net.Listener, error) {
	lc := &net.ListenConfig{}
	return lc.Listen(ctx, "tcp", addr)
}
