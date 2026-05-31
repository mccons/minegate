package tunnel

import (
	"context"
	"net"
	"time"

	"github.com/user/minegate/internal"
	"github.com/user/minegate/packet"
	"github.com/user/minegate/transport"
)

// Dialer is used to open connections to a Minecraft server.
// Inspired by go-mc's MCDialer but provides multi-transport support.
type Dialer struct {
	transport transport.Transport
	timeout   time.Duration
}

// NewDialer creates a new Dialer.
func NewDialer(tr transport.Transport) *Dialer {
	if tr == nil {
		tr = &transport.TCPTransport{DialTimeout: 30 * time.Second}
	}
	return &Dialer{
		transport: tr,
		timeout:   30 * time.Second,
	}
}

// Dial opens a connection to the given address.
func (d *Dialer) Dial(addr string) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.timeout)
	defer cancel()
	return d.transport.Dial(ctx, addr)
}

// DialContext opens a connection using the given context.
func (d *Dialer) DialContext(ctx context.Context, addr string) (net.Conn, error) {
	return d.transport.Dial(ctx, addr)
}

// DialMinecraft connects to a Minecraft server and returns a packet.Reader/Writer.
func (d *Dialer) DialMinecraft(addr string) (*MineConnWrapper, error) {
	conn, err := d.Dial(addr)
	if err != nil {
		return nil, err
	}

	return &MineConnWrapper{
		Reader: packet.NewPacketReader(conn, -1),
		Writer: packet.NewPacketWriter(conn, -1),
		Conn:   conn,
	}, nil
}

// SetTransport changes the transport in use.
func (d *Dialer) SetTransport(tr transport.Transport) {
	d.transport = tr
}

// Transport returns the current transport.
func (d *Dialer) Transport() transport.Transport {
	return d.transport
}

// MineConnWrapper is a combination of net.Conn + packet reader/writer.
type MineConnWrapper struct {
	Reader *packet.PacketReader
	Writer *packet.PacketWriter
	net.Conn
}

// ReadRawPacket reads a raw packet (zero-copy).
func (m *MineConnWrapper) ReadRawPacket() (packet.RawPacket, error) {
	return m.Reader.ReadRawPacket()
}

// Ensure internal used
var _ = internal.ErrTimeout
