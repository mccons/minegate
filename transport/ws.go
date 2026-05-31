package transport

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// WSTransport is a WebSocket transport.
type WSTransport struct {
	DialTimeout time.Duration
	TLSConfig   *tls.Config
	Path        string
}

func NewWSTransport() *WSTransport {
	return &WSTransport{
		DialTimeout: 30 * time.Second,
		Path:        "/",
	}
}

// wsConn wraps websocket.Conn as a net.Conn.
type wsConn struct {
	*websocket.Conn
	raddr net.Addr
	laddr net.Addr
}

func (wc *wsConn) Read(b []byte) (int, error) {
	_, msg, err := wc.ReadMessage()
	if err != nil {
		return 0, err
	}
	return copy(b, msg), nil
}

func (wc *wsConn) Write(b []byte) (int, error) {
	if err := wc.WriteMessage(websocket.BinaryMessage, b); err != nil {
		return 0, err
	}
	return len(b), nil
}

func (wc *wsConn) LocalAddr() net.Addr  { return wc.laddr }
func (wc *wsConn) RemoteAddr() net.Addr { return wc.raddr }

func (wc *wsConn) SetDeadline(t time.Time) error {
	if err := wc.SetReadDeadline(t); err != nil {
		return err
	}
	return wc.SetWriteDeadline(t)
}

// Dial opens a WebSocket connection.
func (w *WSTransport) Dial(ctx context.Context, addr string) (net.Conn, error) {
	dialer := &websocket.Dialer{
		HandshakeTimeout: w.DialTimeout,
		TLSClientConfig:  w.TLSConfig,
	}

	scheme := "ws"
	if w.TLSConfig != nil {
		scheme = "wss"
	}

	c, _, err := dialer.DialContext(ctx, scheme+"://"+addr+w.Path, nil)
	if err != nil {
		return nil, err
	}

	lc := c.LocalAddr()
	rc := c.RemoteAddr()
	if lc == nil {
		lc = &addrWrapper{addr: "ws-local"}
	}
	if rc == nil {
		rc = &addrWrapper{addr: addr}
	}

	return &wsConn{Conn: c, laddr: lc, raddr: rc}, nil
}

// Listen starts a WebSocket listener.
func (w *WSTransport) Listen(ctx context.Context, addr string) (net.Listener, error) {
	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}

	wsl := &wsListener{
		tcpLn:   ln,
		path:    w.Path,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		connCh: make(chan net.Conn, 64),
		closeCh: make(chan struct{}),
	}

	mux := http.NewServeMux()
	mux.HandleFunc(w.Path, wsl.serveWS)
	wsl.server = &http.Server{
		Handler: mux,
	}

	go func() {
		if w.TLSConfig != nil {
			wsl.server.ServeTLS(ln, "", "")
		} else {
			wsl.server.Serve(ln)
		}
	}()

	return wsl, nil
}

type addrWrapper struct {
	addr string
}

func (a *addrWrapper) Network() string { return "ws" }
func (a *addrWrapper) String() string  { return a.addr }

type wsListener struct {
	tcpLn    net.Listener
	server   *http.Server
	path     string
	upgrader websocket.Upgrader
	connCh   chan net.Conn
	closeCh  chan struct{}
}

func (wl *wsListener) serveWS(w http.ResponseWriter, r *http.Request) {
	c, err := wl.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	select {
	case wl.connCh <- &wsConn{Conn: c, laddr: c.LocalAddr(), raddr: c.RemoteAddr()}:
	case <-wl.closeCh:
		c.Close()
	}
}

func (wl *wsListener) Accept() (net.Conn, error) {
	select {
	case conn := <-wl.connCh:
		return conn, nil
	case <-wl.closeCh:
		return nil, net.ErrClosed
	}
}

func (wl *wsListener) Close() error {
	close(wl.closeCh)
	return wl.server.Close()
}

func (wl *wsListener) Addr() net.Addr {
	return wl.tcpLn.Addr()
}

var _ = websocket.ErrBadHandshake
