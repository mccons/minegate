package transport

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

func TestTCPDialAndListen(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()

	msg := []byte("minegate transport test")

	errCh := make(chan error, 1)
	go func() {
		conn, aErr := listener.Accept()
		if aErr != nil {
			errCh <- aErr
			return
		}
		defer conn.Close()
		buf := make([]byte, 32)
		n, rErr := conn.Read(buf)
		if rErr != nil {
			errCh <- rErr
			return
		}
		if !bytes.Equal(buf[:n], msg) {
			errCh <- fmt.Errorf("got %q, want %q", buf[:n], msg)
			return
		}
		errCh <- nil
	}()

	tcp := &TCPTransport{}
	ctx := context.Background()
	conn, dErr := tcp.Dial(ctx, addr)
	if dErr != nil {
		t.Fatal(dErr)
	}

	_, wErr := conn.Write(msg)
	if wErr != nil {
		t.Fatal(wErr)
	}
	conn.Close()
	listener.Close()

	select {
	case e := <-errCh:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for server")
	}
}

func TestTCPListenAccept(t *testing.T) {
	tcp := &TCPTransport{}
	ctx := context.Background()
	ln, err := tcp.Listen(ctx, "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	addr := ln.Addr().String()
	if addr == "" {
		t.Fatal("empty listener address")
	}

	go func() {
		conn, dErr := net.DialTimeout("tcp", addr, time.Second)
		if dErr != nil {
			t.Error(dErr)
			return
		}
		conn.Close()
	}()

	conn, aErr := ln.Accept()
	if aErr != nil {
		t.Fatal(aErr)
	}
	conn.Close()
}

func TestTLSTransportNew(t *testing.T) {
	tr := NewTLSTransport(nil, nil)
	if tr == nil {
		t.Fatal("TLSTransport should not be nil")
	}
}

func TestWSTransportNew(t *testing.T) {
	ws := NewWSTransport()
	if ws == nil {
		t.Fatal("WSTransport should not be nil")
	}
}

func TestKCPTransportNew(t *testing.T) {
	kcp := NewKCPTransport()
	if kcp == nil {
		t.Fatal("KCPTransport should not be nil")
	}
}

func TestQUICTransportNew(t *testing.T) {
	q := NewQUICTransport(nil)
	if q == nil {
		t.Fatal("QUICTransport should not be nil")
	}
}

func TestSOCKS5TransportNew(t *testing.T) {
	socks := NewSOCKS5Transport("127.0.0.1:1080")
	if socks == nil {
		t.Fatal("SOCKS5Transport should not be nil")
	}
	if socks.ProxyAddr != "127.0.0.1:1080" {
		t.Errorf("ProxyAddr: got %q, want %q", socks.ProxyAddr, "127.0.0.1:1080")
	}
}

func TestSOCKS5ListenNotSupported(t *testing.T) {
	socks := NewSOCKS5Transport("127.0.0.1:1080")
	ctx := context.Background()
	_, err := socks.Listen(ctx, "127.0.0.1:0")
	if err == nil {
		t.Fatal("expected error for SOCKS5 listen")
	}
}

func TestTransportInterface(t *testing.T) {
	var transport Transport = &TCPTransport{}
	ctx := context.Background()
	_, err := transport.Dial(ctx, "127.0.0.1:1")
	if err == nil {
		t.Log("expected dial error, got nil (may connect in rare cases)")
	}
}

func TestTCPLargeData(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()

	data := make([]byte, 65535)
	for i := range data {
		data[i] = byte(i % 256)
	}

	errCh := make(chan error, 1)
	go func() {
		conn, aErr := listener.Accept()
		if aErr != nil {
			errCh <- aErr
			return
		}
		defer conn.Close()
		buf := make([]byte, len(data))
		_, rErr := io.ReadFull(conn, buf)
		if rErr != nil {
			errCh <- rErr
			return
		}
		if !bytes.Equal(buf, data) {
			errCh <- fmt.Errorf("data mismatch")
			return
		}
		conn.Write([]byte("ok"))
		errCh <- nil
	}()

	tcp := &TCPTransport{}
	ctx := context.Background()
	conn, dErr := tcp.Dial(ctx, addr)
	if dErr != nil {
		t.Fatal(dErr)
	}

	_, wErr := conn.Write(data)
	if wErr != nil {
		t.Fatal(wErr)
	}

	reply := make([]byte, 2)
	conn.SetReadDeadline(time.Now().Add(time.Second))
	io.ReadFull(conn, reply)
	conn.Close()
	listener.Close()

	if !bytes.Equal(reply, []byte("ok")) {
		t.Fatalf("expected 'ok', got %q", reply)
	}

	select {
	case e := <-errCh:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestTransportFactory(t *testing.T) {
	tf := NewTransportFactory()
	if tf == nil {
		t.Fatal("NewTransportFactory should not be nil")
	}

	tcp := &TCPTransport{}
	tf.Register(TCP, tcp)

	got, ok := tf.Get(TCP)
	if !ok {
		t.Fatal("TCP transport should be registered")
	}
	if got != tcp {
		t.Fatal("Get returned wrong transport")
	}

	must := tf.MustGet(TCP)
	if must != tcp {
		t.Fatal("MustGet returned wrong transport")
	}

	// Test panic on missing type
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("MustGet should panic for unregistered type")
		}
	}()
	tf.MustGet(KCP)
}

func TestDefaultFactory(t *testing.T) {
	if DefaultFactory == nil {
		t.Fatal("DefaultFactory should not be nil")
	}
	tcp, ok := DefaultFactory.Get(TCP)
	if !ok {
		t.Fatal("TCP should be registered in DefaultFactory")
	}
	if tcp == nil {
		t.Fatal("TCP transport should not be nil")
	}
	// Verify TLS is not registered by default (optional)
	_, ok = DefaultFactory.Get(TLS)
	if ok {
		t.Log("TLS is registered (may be registered optionally)")
	}
}

func TestTypeString(t *testing.T) {
	tests := []struct {
		typ Type
		str string
	}{
		{TCP, "tcp"},
		{TLS, "tls"},
		{WebSocket, "ws"},
		{KCP, "kcp"},
		{QUIC, "quic"},
		{SOCKS5, "socks5"},
		{Type(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.typ.String(); got != tt.str {
			t.Errorf("Type(%d).String() = %q, want %q", tt.typ, got, tt.str)
		}
	}
}

func TestSOCKS5HandshakeBytes(t *testing.T) {
	socks := NewSOCKS5Transport("127.0.0.1:1080")

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	errCh := make(chan error, 1)
	go func() {
		buf := make([]byte, 3)
		_, err := io.ReadFull(server, buf)
		if err != nil {
			errCh <- err
			return
		}
		// VER=5, NMETHODS=1, METHOD=0 (no auth)
		if buf[0] != 5 || buf[1] != 1 || buf[2] != 0 {
			errCh <- fmt.Errorf("bad handshake: %v", buf)
			return
		}
		server.Write([]byte{5, 0}) // VER=5, METHOD=0
		errCh <- nil
	}()

	err := socks.handshake(client)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case e := <-errCh:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestSOCKS5HandshakeAuthBytes(t *testing.T) {
	socks := NewSOCKS5Transport("127.0.0.1:1080")
	socks.Username = "user"
	socks.Password = "pass"

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	errCh := make(chan error, 1)
	go func() {
		buf := make([]byte, 4)
		_, err := io.ReadFull(server, buf)
		if err != nil {
			errCh <- err
			return
		}
		// VER=5, NMETHODS=2, METHOD[0]=0 (no auth), METHOD[1]=2 (password)
		if buf[0] != 5 || buf[1] != 2 || buf[2] != 0 || buf[3] != 2 {
			errCh <- fmt.Errorf("bad auth method list: %v", buf)
			return
		}
		server.Write([]byte{5, 2}) // VER=5, METHOD=2 (password)

		// Read password auth: VER(1) + ULEN(1) + UNAME + PLEN(1) + PASS
		authLen := 1 + 1 + len(socks.Username) + 1 + len(socks.Password)
		auth := make([]byte, authLen)
		_, err = io.ReadFull(server, auth)
		if err != nil {
			errCh <- err
			return
		}
		if auth[0] != 1 || int(auth[1]) != len(socks.Username) || string(auth[2:2+len(socks.Username)]) != socks.Username {
			errCh <- fmt.Errorf("bad auth: %v", auth)
			return
		}
		server.Write([]byte{1, 0}) // auth success
		errCh <- nil
	}()

	err := socks.handshake(client)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case e := <-errCh:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestSOCKS5RequestBytes(t *testing.T) {
	socks := NewSOCKS5Transport("127.0.0.1:1080")

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	errCh := make(chan error, 1)
	go func() {
		// Expect: VER=5, CMD=1, RSV=0, ATYP=3 (domain), len=14, "mc.example.com", port=25565
		req := make([]byte, 4+1+14+2)
		_, err := io.ReadFull(server, req)
		if err != nil {
			errCh <- err
			return
		}
		if req[0] != 5 || req[1] != 1 || req[3] != 3 {
			errCh <- fmt.Errorf("bad request header: %v", req[:4])
			return
		}
		domainLen := int(req[4])
		domain := string(req[5 : 5+domainLen])
		if domain != "mc.example.com" {
			errCh <- fmt.Errorf("bad domain: got %q, want %q", domain, "mc.example.com")
			return
		}
		port := uint16(req[5+domainLen])<<8 | uint16(req[6+domainLen])
		if port != 25565 {
			errCh <- fmt.Errorf("bad port: got %d, want %d", port, 25565)
			return
		}
		// Send response: VER=5, REP=0, RSV=0, ATYP=1, 4 bytes bind IP, 2 bytes port
		server.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0})
		errCh <- nil
	}()

	err := socks.request(client, "mc.example.com:25565")
	if err != nil {
		t.Fatal(err)
	}

	select {
	case e := <-errCh:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}
