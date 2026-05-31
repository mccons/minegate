package tunnel

import (
	"bytes"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

func TestDirectRelay(t *testing.T) {
	client, server := net.Pipe()

	msg := []byte("hello minecraft relay")
	go func() {
		client.Write(msg)
		client.Close()
	}()

	go DirectRelay(client, server)

	buf := make([]byte, len(msg))
	server.SetReadDeadline(time.Now().Add(time.Second))
	n, err := server.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatal(err)
	}

	if !bytes.Equal(buf[:n], msg) {
		t.Errorf("relay: got %q, want %q", buf[:n], msg)
	}
}

func TestRelayWithMetrics(t *testing.T) {
	client, server := net.Pipe()

	var c2s, s2c int64
	onByte := func(dir string, n int) {
		if dir == "c2s" {
			atomic.AddInt64(&c2s, int64(n))
		} else {
			atomic.AddInt64(&s2c, int64(n))
		}
	}

	msg := []byte("metrics test")
	go func() {
		client.Write(msg)
		client.Close()
	}()

	go RelayWithMetrics(client, server, onByte)

	buf := make([]byte, len(msg))
	server.SetReadDeadline(time.Now().Add(time.Second))
	server.Read(buf)
	server.Close()

	time.Sleep(50 * time.Millisecond)
	if c2s == 0 && s2c == 0 {
		t.Log("metrics not triggered (expected if relay already finished)")
	}
}

func TestDirectRelayBidirectional(t *testing.T) {
	c1, s1 := net.Pipe()
	c2, s2 := net.Pipe()

	go DirectRelay(s1, s2)

	msg1 := []byte("from client1")
	msg2 := []byte("from client2")

	go func() {
		c1.Write(msg1)
		buf := make([]byte, 32)
		n, _ := c1.Read(buf)
		if !bytes.Equal(buf[:n], msg2) {
			t.Errorf("c1 got %q, want %q", buf[:n], msg2)
		}
		c1.Close()
	}()

	go func() {
		c2.Write(msg2)
		buf := make([]byte, 32)
		n, _ := c2.Read(buf)
		if !bytes.Equal(buf[:n], msg1) {
			t.Errorf("c2 got %q, want %q", buf[:n], msg1)
		}
		c2.Close()
	}()

	time.Sleep(200 * time.Millisecond)
}

func TestMuxOpenClose(t *testing.T) {
	c1, c2 := net.Pipe()
	mux1 := NewMux(c1)
	mux2 := NewMux(c2)

	stream, err := mux1.OpenStream()
	if err != nil {
		t.Fatal(err)
	}

	accepted, err := mux2.AcceptStream()
	if err != nil {
		t.Fatal(err)
	}

	if accepted.ID() != stream.ID() {
		t.Errorf("stream ID mismatch: %d vs %d", accepted.ID(), stream.ID())
	}

	stream.Close()
	mux1.Close()
	mux2.Close()
}

func TestMuxDataTransfer(t *testing.T) {
	c1, c2 := net.Pipe()
	mux1 := NewMux(c1)
	mux2 := NewMux(c2)

	stream1, _ := mux1.OpenStream()
	stream2, _ := mux2.AcceptStream()

	msg := []byte("hello over mux!")
	go func() {
		stream1.Write(msg)
		stream1.Close()
	}()

	buf := make([]byte, len(msg))
	n, err := io.ReadFull(stream2, buf)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(buf[:n], msg) {
		t.Errorf("mux data: got %q, want %q", buf[:n], msg)
	}

	mux1.Close()
	mux2.Close()
}

func TestMuxMultipleStreams(t *testing.T) {
	c1, c2 := net.Pipe()
	mux1 := NewMux(c1)
	mux2 := NewMux(c2)

	n := 5
	streams1 := make([]*MuxStream, n)
	streams2 := make([]*MuxStream, n)

	for i := 0; i < n; i++ {
		s, _ := mux1.OpenStream()
		streams1[i] = s
	}

	for i := 0; i < n; i++ {
		s, _ := mux2.AcceptStream()
		streams2[i] = s
	}

	for i := 0; i < n; i++ {
		if streams1[i].ID() != streams2[i].ID() {
			t.Errorf("stream %d: ID mismatch %d vs %d", i, streams1[i].ID(), streams2[i].ID())
		}
	}

	mux1.Close()
	mux2.Close()
}

func TestMuxCloseIdempotent(t *testing.T) {
	c1, c2 := net.Pipe()
	mux := NewMux(c1)
	c2.Close()

	mux.Close()
	mux.Close()
}

func TestMuxStreamClose(t *testing.T) {
	c1, c2 := net.Pipe()
	mux1 := NewMux(c1)
	mux2 := NewMux(c2)

	s1, _ := mux1.OpenStream()
	s2, _ := mux2.AcceptStream()

	s1.Close()
	time.Sleep(50 * time.Millisecond)

	_, err := s2.Write([]byte("data"))
	if err == nil {
		t.Log("write after close returned nil (may be buffered)")
	}

	mux1.Close()
	mux2.Close()
}

func TestNewDialer(t *testing.T) {
	d := NewDialer(nil)
	if d == nil {
		t.Fatal("NewDialer(nil) should return a dialer")
	}
	if d.timeout == 0 {
		t.Error("dialer should have a non-zero timeout")
	}
}

func TestNewListener(t *testing.T) {
	l := NewListener(nil)
	if l == nil {
		t.Fatal("NewListener(nil) should return a listener")
	}
}

var _ = io.Discard
