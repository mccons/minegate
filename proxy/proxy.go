package proxy

import (
	"context"
	"log"
	"net"
	"sync"
	"sync/atomic"

	"github.com/pozii/minegate/internal"
	"github.com/pozii/minegate/tunnel"
)

// Proxy is the core of the Minecraft proxy server.
// It accepts client connections, reads the handshake, and forwards to the target server.
type Proxy struct {
	ln       *tunnel.Listener
	dialer   *tunnel.Dialer
	handler  func(HandlerContext)
	active   sync.WaitGroup
	closed   atomic.Bool
}

// HandlerContext is the context passed to the proxy handler.
type HandlerContext struct {
	Client  *tunnel.MineConnWrapper
	Host    string
	Port    uint16
	State   int32
	Cancel  context.CancelFunc
}

// NewProxy creates a new Proxy.
func NewProxy(ln *tunnel.Listener, dialer *tunnel.Dialer) *Proxy {
	return &Proxy{
		ln:     ln,
		dialer: dialer,
		handler: func(ctx HandlerContext) {
			defaultHandler(ctx)
		},
	}
}

// SetHandler sets a custom handler function.
func (p *Proxy) SetHandler(h func(HandlerContext)) {
	p.handler = h
}

// Start starts the proxy.
func (p *Proxy) Start() error {
	for {
		if p.closed.Load() {
			return nil
		}

		client, err := p.ln.Accept()
		if err != nil {
			if !p.closed.Load() {
				log.Printf("proxy: accept error: %v", err)
			}
			return err
		}

		p.active.Add(1)
		go p.handleClient(client)
	}
}

// Stop stops the proxy.
func (p *Proxy) Stop() {
	p.closed.Store(true)
	p.ln.Close()
	p.active.Wait()
}

func (p *Proxy) handleClient(client *tunnel.MineConnWrapper) {
	defer p.active.Done()
	defer client.Close()

	// Read handshake packet
	pkt, err := client.Reader.ReadPacket()
	if err != nil {
		return
	}

	host, port, state, err := ParseHandshake(pkt)
	if err != nil {
		return
	}

	_, cancel := context.WithCancel(context.Background())

	hc := HandlerContext{
		Client: client,
		Host:   host,
		Port:   port,
		State:  state,
		Cancel: cancel,
	}

	p.handler(hc)
}

func defaultHandler(hc HandlerContext) {
	hedef := hc.Host + ":" + itoa(int(hc.Port))
	log.Printf("proxy: %s -> %s (state=%d)", hc.Client.RemoteAddr(), hedef, hc.State)

	server, err := hc.Client.Conn.(interface {
		Dial(addr string) (net.Conn, error)
	}).Dial(hedef)
	if err != nil {
		log.Printf("proxy: dial error: %v", err)
		return
	}
	defer server.Close()

	tunnel.DirectRelay(hc.Client.Conn, server)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

var _ = internal.ErrProxyRejected
