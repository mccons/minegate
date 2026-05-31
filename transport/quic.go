package transport

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/quic-go/quic-go"
)

// QUICTransport is a QUIC transport.
type QUICTransport struct {
	TLSConfig   *tls.Config
	DialTimeout time.Duration
}

func NewQUICTransport(tlsConfig *tls.Config) *QUICTransport {
	return &QUICTransport{
		TLSConfig:   tlsConfig,
		DialTimeout: 30 * time.Second,
	}
}

// QUICConn wraps *quic.Stream as a net.Conn.
type QUICConn struct {
	stream *quic.Stream
	laddr  net.Addr
	raddr  net.Addr
}

func (qc *QUICConn) Read(b []byte) (int, error) {
	return qc.stream.Read(b)
}

func (qc *QUICConn) Write(b []byte) (int, error) {
	return qc.stream.Write(b)
}

func (qc *QUICConn) Close() error {
	return qc.stream.Close()
}

func (qc *QUICConn) LocalAddr() net.Addr  { return qc.laddr }
func (qc *QUICConn) RemoteAddr() net.Addr { return qc.raddr }

func (qc *QUICConn) SetDeadline(t time.Time) error {
	return qc.stream.SetDeadline(t)
}

func (qc *QUICConn) SetReadDeadline(t time.Time) error {
	return qc.stream.SetReadDeadline(t)
}

func (qc *QUICConn) SetWriteDeadline(t time.Time) error {
	return qc.stream.SetWriteDeadline(t)
}

// Dial opens a QUIC connection.
func (q *QUICTransport) Dial(ctx context.Context, addr string) (net.Conn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	udpConn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, err
	}

	tlsConf := q.TLSConfig
	if tlsConf == nil {
		tlsConf = &tls.Config{InsecureSkipVerify: true, NextProtos: []string{"minegate"}}
	}

	conn, err := quic.Dial(ctx, udpConn, udpAddr, tlsConf, nil)
	if err != nil {
		udpConn.Close()
		return nil, err
	}

	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		conn.CloseWithError(0, "stream error")
		udpConn.Close()
		return nil, err
	}

	return &QUICConn{
		stream: stream,
		laddr:  udpConn.LocalAddr(),
		raddr:  udpAddr,
	}, nil
}

// Listen starts a QUIC listener.
func (q *QUICTransport) Listen(ctx context.Context, addr string) (net.Listener, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	tlsConf := q.TLSConfig
	if tlsConf == nil {
		tlsConf = &tls.Config{InsecureSkipVerify: true, NextProtos: []string{"minegate"}}
	}

	ln, err := quic.Listen(udpConn, tlsConf, nil)
	if err != nil {
		udpConn.Close()
		return nil, err
	}

	return &quicListener{listener: ln, laddr: udpAddr}, nil
}

type quicListener struct {
	listener *quic.Listener
	laddr    net.Addr
}

func (ql *quicListener) Accept() (net.Conn, error) {
	conn, err := ql.listener.Accept(context.Background())
	if err != nil {
		return nil, err
	}

	stream, err := conn.AcceptStream(context.Background())
	if err != nil {
		conn.CloseWithError(0, "stream error")
		return nil, err
	}

	return &QUICConn{
		stream: stream,
		laddr:  ql.laddr,
		raddr:  conn.RemoteAddr(),
	}, nil
}

func (ql *quicListener) Close() error {
	return ql.listener.Close()
}

func (ql *quicListener) Addr() net.Addr {
	return ql.listener.Addr()
}
