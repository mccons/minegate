package proxy

import (
	"encoding/binary"
	"net"

	"github.com/user/minegate/internal"
	"github.com/user/minegate/packet"
)

// ForwardingMode represents a proxy forwarding mode.
type ForwardingMode int

const (
	ForwardNone      ForwardingMode = iota // No forwarding
	ForwardLegacy                           // BungeeCord legacy (via host:port)
	ForwardModern                           // BungeeCord modern (UUID+IP+profile)
	ForwardVelocity                         // Velocity modern forwarding
)

// ForwardingData holds the data to be appended during forwarding.
type ForwardingData struct {
	Mode     ForwardingMode
	UUID     [16]byte
	IP       net.IP
	Username string
	Profile  []byte // Base64 skin profile (BungeeCord modern)
	Secret   []byte // Velocity secret key
}

// AppendLegacyForwarding appends the client IP to the host for legacy BungeeCord forwarding.
func AppendLegacyForwarding(host string, clientIP net.IP) string {
	return host + "\x00" + clientIP.String() + "\x00"
}

// AppendModernForwarding appends UUID+IP+profile to the login success packet
// for modern BungeeCord forwarding.
func AppendModernForwarding(pkt packet.Packet, data ForwardingData) (packet.Packet, error) {
	// Login Success (0x03) packet: UUID + Username + ...
	// Modern forwarding appends extra data at the end of this packet
	// Leaving this passive for now
	return pkt, nil
}

// AppendVelocityForwarding appends encrypted data to a plugin message packet
// for Velocity forwarding.
func AppendVelocityForwarding(pkt packet.Packet, data ForwardingData) (packet.Packet, error) {
	return pkt, nil
}

// CreateVelocityForwardingPacket creates a Velocity forwarding packet.
func CreateVelocityForwardingPacket(data ForwardingData) (packet.Packet, error) {
	// Velocity Info Request Packet (0x00 | 0x01 Play)

	// Player Info: UUID as [16]byte + Username
	var payload []byte
	payload = append(payload, data.UUID[:]...)
	payload = append(payload, []byte(data.Username)...)
	payload = append(payload, 0x00) // null terminated

	// Modern forwarding: IP as String
	ipStr := data.IP.String()
	buf := make([]byte, packet.MaxVarIntLen)
	n := packet.PutVarInt(buf, int32(len(ipStr)))
	payload = append(payload, buf[:n]...)
	payload = append(payload, []byte(ipStr)...)

	// Port: int
	portBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(portBuf, 0) // default port
	payload = append(payload, portBuf...)

	return packet.Packet{ID: 0x00, Data: payload}, nil
}

var _ = internal.ErrHandshakeFailed
