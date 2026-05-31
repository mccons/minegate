package transport

import (
	"context"
	"net"
	"time"

	kcp "github.com/xtaci/kcp-go"
)

// KCPTransport is a KCP (UDP-based, reliable) transport.
// It offers better performance than TCP in high-loss environments.
type KCPTransport struct {
	DialTimeout time.Duration
	Block       kcp.BlockCrypt
	DataShard   int
	ParityShard int
}

// NewKCPTransport creates a new KCPTransport.
func NewKCPTransport() *KCPTransport {
	return &KCPTransport{
		DialTimeout: 30 * time.Second,
		DataShard:   10,
		ParityShard: 3,
	}
}

// Dial opens a KCP connection.
func (k *KCPTransport) Dial(ctx context.Context, addr string) (net.Conn, error) {
	var sess *kcp.UDPSession
	var err error

	if k.Block != nil {
		sess, err = kcp.DialWithOptions(addr, k.Block, k.DataShard, k.ParityShard)
	} else {
		sess, err = kcp.DialWithOptions(addr, nil, k.DataShard, k.ParityShard)
	}
	if err != nil {
		return nil, err
	}

	sess.SetStreamMode(true)
	sess.SetWriteDelay(false)
	sess.SetNoDelay(1, 10, 2, 1)
	sess.SetWindowSize(1024, 1024)
	sess.SetMtu(1400)
	sess.SetACKNoDelay(true)

	return sess, nil
}

// Listen starts a KCP listener.
func (k *KCPTransport) Listen(ctx context.Context, addr string) (net.Listener, error) {
	var lis net.Listener
	var err error

	if k.Block != nil {
		lis, err = kcp.ListenWithOptions(addr, k.Block, k.DataShard, k.ParityShard)
	} else {
		lis, err = kcp.ListenWithOptions(addr, nil, k.DataShard, k.ParityShard)
	}
	if err != nil {
		return nil, err
	}

	return &kcpListener{Listener: lis}, nil
}

type kcpListener struct {
	net.Listener
}

func (kl *kcpListener) Accept() (net.Conn, error) {
	conn, err := kl.Listener.Accept()
	if err != nil {
		return nil, err
	}

	sess := conn.(*kcp.UDPSession)
	sess.SetStreamMode(true)
	sess.SetWriteDelay(false)
	sess.SetNoDelay(1, 10, 2, 1)
	sess.SetWindowSize(1024, 1024)
	sess.SetMtu(1400)
	sess.SetACKNoDelay(true)

	return conn, nil
}

// KCPDialSession opens a KCP session optimized for low latency.
func KCPDialSession(addr string, block kcp.BlockCrypt) (*kcp.UDPSession, error) {
	sess, err := kcp.DialWithOptions(addr, block, 10, 3)
	if err != nil {
		return nil, err
	}

	sess.SetStreamMode(true)
	sess.SetWriteDelay(false)
	sess.SetNoDelay(1, 10, 2, 1)
	sess.SetWindowSize(512, 512)
	sess.SetMtu(1400)
	sess.SetACKNoDelay(true)

	return sess, nil
}


