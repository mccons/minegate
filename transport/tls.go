package transport

import (
	"context"
	"crypto/tls"
	"net"
	"time"
)

// TLSTransport is a TLS wrapper transport.
// It wraps the underlying transport with TLS.
type TLSTransport struct {
	Config    *tls.Config
	DialTimeout time.Duration
	Next      Transport // wrapped transport (usually TCP)
}

// NewTLSTransport creates a new TLSTransport.
func NewTLSTransport(config *tls.Config, next Transport) *TLSTransport {
	if next == nil {
		next = &TCPTransport{DialTimeout: 30 * time.Second}
	}
	return &TLSTransport{Config: config, Next: next, DialTimeout: 30 * time.Second}
}

// Dial opens a TLS-wrapped connection.
func (t *TLSTransport) Dial(ctx context.Context, addr string) (net.Conn, error) {
	conn, err := t.Next.Dial(ctx, addr)
	if err != nil {
		return nil, err
	}

	tlsConn := tls.Client(conn, t.Config)
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		conn.Close()
		return nil, err
	}

	return tlsConn, nil
}

// Listen starts a TLS-wrapped listener.
func (t *TLSTransport) Listen(ctx context.Context, addr string) (net.Listener, error) {
	ln, err := t.Next.Listen(ctx, addr)
	if err != nil {
		return nil, err
	}

	return tls.NewListener(ln, t.Config), nil
}
