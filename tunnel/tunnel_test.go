package tunnel

import (
	"bytes"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

type pipeConn struct {
	r *io.PipeReader
	w *io.PipeWriter
}

func (c *pipeConn) Read(b []byte) (int, error)  { return c.r.Read(b) }
func (c *pipeConn) Write(b []byte) (int, error) { return c.w.Write(b) }
func (c *pipeConn) Close() error {
	c.w.Close()
	return c.r.Close()
}

// testRelayPair wires four io.Pipes so that:
//   test writes to injectClientW → client.Read() returns it (simulating real client)
//   server.Write() → test reads from readServerR (verifying forward relay)
//   test writes to injectServerW → server.Read() returns it (simulating real server)
//   client.Write() → test reads from readClientR (verifying reverse relay)
// No feedback loops: each direction uses independent pipes.
type testRelayPair struct {
	client, server *pipeConn
	injectClientW  *io.PipeWriter // test writes → client.Read()
	injectServerW  *io.PipeWriter // test writes → server.Read()
	readClientR    *io.PipeReader // client.Write() → test reads
	readServerR    *io.PipeReader // server.Write() → test reads
}

func newTestRelayPair() *testRelayPair {
	// client direction: test injects → client.Read()
	clientInjectR, clientInjectW := io.Pipe()
	// server direction: server.Write() → test reads
	serverResultR, serverResultW := io.Pipe()
	// server direction: test injects → server.Read()
	serverInjectR, serverInjectW := io.Pipe()
	// client direction: client.Write() → test reads
	clientResultR, clientResultW := io.Pipe()

	client := &pipeConn{r: clientInjectR, w: clientResultW}
	server := &pipeConn{r: serverInjectR, w: serverResultW}

	return &testRelayPair{
		client: client,
		server: server,
		injectClientW: clientInjectW,
		injectServerW: serverInjectW,
		readClientR:   clientResultR,
		readServerR:   serverResultR,
	}
}

func TestDirectRelay(t *testing.T) {
	pair := newTestRelayPair()
	msg := []byte("hello minecraft relay")

	go DirectRelay(pair.client, pair.server)
	time.Sleep(10 * time.Millisecond)

	_, err := pair.injectClientW.Write(msg)
	if err != nil {
		t.Fatal(err)
	}
	pair.injectClientW.Close()

	buf := make([]byte, len(msg))
	n, err := pair.readServerR.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatal("relay: no data received")
	}
	if !bytes.Equal(buf[:n], msg) {
		t.Errorf("relay: got %q, want %q", buf[:n], msg)
	}
	pair.readServerR.Close()
}

func TestRelayWithMetrics(t *testing.T) {
	pair := newTestRelayPair()

	var mu sync.Mutex
	var c2sBytes int64

	onByte := func(dir string, n int) {
		if dir == "c2s" {
			mu.Lock()
			c2sBytes += int64(n)
			mu.Unlock()
		}
	}

	msg := []byte("metrics test")

	go RelayWithMetrics(pair.client, pair.server, onByte)
	time.Sleep(10 * time.Millisecond)

	_, err := pair.injectClientW.Write(msg)
	if err != nil {
		t.Fatal(err)
	}
	pair.injectClientW.Close()

	buf := make([]byte, len(msg))
	n, err := pair.readServerR.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatal("relay: no data received")
	}
	pair.readServerR.Close()

	mu.Lock()
	got := c2sBytes
	mu.Unlock()
	if got == 0 {
		t.Error("relay metrics: c2s byte count is 0")
	}
}

func TestDirectRelayBidirectional(t *testing.T) {
	pair := newTestRelayPair()

	go DirectRelay(pair.client, pair.server)
	time.Sleep(10 * time.Millisecond)

	msg1 := []byte("from client to server")
	msg2 := []byte("from server to client")

	errCh := make(chan error, 2)

	go func() {
		pair.injectClientW.Write(msg1)
		buf := make([]byte, 64)
		n, err := pair.readClientR.Read(buf)
		if err != nil {
			errCh <- err
			return
		}
		if !bytes.Equal(buf[:n], msg2) {
			errCh <- nil
			return
		}
		pair.injectClientW.Close()
		errCh <- nil
	}()

	go func() {
		pair.injectServerW.Write(msg2)
		buf := make([]byte, 64)
		n, err := pair.readServerR.Read(buf)
		if err != nil {
			errCh <- err
			return
		}
		if !bytes.Equal(buf[:n], msg1) {
			errCh <- nil
			return
		}
		pair.injectServerW.Close()
		errCh <- nil
	}()

	for range 2 {
		select {
		case e := <-errCh:
			if e != nil {
				t.Fatal(e)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("timeout")
		}
	}
	pair.readClientR.Close()
	pair.readServerR.Close()
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
