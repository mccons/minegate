package transport

import (
	"context"
	"net"
)

// Type represents a transport protocol type.
type Type int

const (
	TCP     Type = iota
	TLS
	WebSocket
	KCP
	QUIC
	SOCKS5
)

func (t Type) String() string {
	switch t {
	case TCP:
		return "tcp"
	case TLS:
		return "tls"
	case WebSocket:
		return "ws"
	case KCP:
		return "kcp"
	case QUIC:
		return "quic"
	case SOCKS5:
		return "socks5"
	default:
		return "unknown"
	}
}

// Transport is the transport protocol interface.
// Each transport type (TCP, TLS, WS, KCP, QUIC, SOCKS5) implements this interface.
type Transport interface {
	// Dial opens a connection to the given address.
	Dial(ctx context.Context, addr string) (net.Conn, error)

	// Listen starts listening on the given address.
	Listen(ctx context.Context, addr string) (net.Listener, error)
}

// TransportFactory is a factory that returns Transport by type.
type TransportFactory struct {
	transports map[Type]Transport
}

// NewTransportFactory creates a new TransportFactory.
func NewTransportFactory() *TransportFactory {
	return &TransportFactory{
		transports: make(map[Type]Transport),
	}
}

// Register registers a transport type.
func (tf *TransportFactory) Register(t Type, tr Transport) {
	tf.transports[t] = tr
}

// Get returns the registered transport.
func (tf *TransportFactory) Get(t Type) (Transport, bool) {
	tr, ok := tf.transports[t]
	return tr, ok
}

// MustGet returns the registered transport (panics if missing).
func (tf *TransportFactory) MustGet(t Type) Transport {
	tr, ok := tf.transports[t]
	if !ok {
		panic("transport: " + t.String() + " not registered")
	}
	return tr
}

// DefaultFactory is the default transport factory.
var DefaultFactory = NewTransportFactory()

func init() {
	DefaultFactory.Register(TCP, &TCPTransport{})
	// TLS, WS, KCP, QUIC, SOCKS5 are registered optionally
}
