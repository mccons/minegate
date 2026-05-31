package proxy

import (
	"github.com/pozii/minegate/internal"
	"github.com/pozii/minegate/packet"
)

// Handshake packet fields:
// Protocol Version (VarInt)
// Server Address (String)
// Server Port (UnsignedShort)
// Next State (VarInt): 1=Status, 2=Login

// ParseHandshake parses a handshake packet.
// Returns host, port, nextState, and error.
func ParseHandshake(pkt packet.Packet) (host string, port uint16, nextState int32, err error) {
	if len(pkt.Data) < 2 {
		return "", 0, 0, internal.ErrPacketTooShort
	}

	data := pkt.Data

	// Protocol Version (VarInt)
	protoVer, remaining, err := packet.ReadVarIntFromBytes(data)
	if err != nil {
		return "", 0, 0, err
	}
	_ = protoVer

	// Server Address (String)
	strLen, remaining, err := packet.ReadVarIntFromBytes(remaining)
	if err != nil {
		return "", 0, 0, err
	}
	if int(strLen) > len(remaining) || strLen < 0 {
		return "", 0, 0, internal.ErrPacketTooShort
	}
	host = string(remaining[:strLen])
	remaining = remaining[strLen:]

	// Server Port (UnsignedShort)
	if len(remaining) < 2 {
		return "", 0, 0, internal.ErrPacketTooShort
	}
	port = uint16(remaining[0])<<8 | uint16(remaining[1])
	remaining = remaining[2:]

	// Next State (VarInt)
	var ns packet.VarInt
	ns, _, err = packet.ReadVarIntFromBytes(remaining)
	if err != nil {
		return "", 0, 0, err
	}

	return host, port, int32(ns), nil
}

// BuildHandshake creates a handshake packet.
func BuildHandshake(protocolVersion int32, host string, port uint16, nextState int32) packet.Packet {
	data := make([]byte, 0, len(host)+10)

	buf := make([]byte, packet.MaxVarIntLen)
	n := packet.PutVarInt(buf, protocolVersion)
	data = append(data, buf[:n]...)

	n = packet.PutVarInt(buf, int32(len(host)))
	data = append(data, buf[:n]...)
	data = append(data, []byte(host)...)

	data = append(data, byte(port>>8), byte(port))

	n = packet.PutVarInt(buf, nextState)
	data = append(data, buf[:n]...)

	return packet.Packet{ID: 0x00, Data: data}
}

// ModifyHandshakeHost changes the host address in a handshake packet.
func ModifyHandshakeHost(pkt packet.Packet, newHost string) (packet.Packet, error) {
	_, _, _, err := ParseHandshake(pkt)
	if err != nil {
		return packet.Packet{}, err
	}

	data := pkt.Data
	_, remaining, err := packet.ReadVarIntFromBytes(data)
	if err != nil {
		return packet.Packet{}, err
	}
	protoLen := len(data) - len(remaining)

	strLen, remaining2, err := packet.ReadVarIntFromBytes(remaining)
	if err != nil {
		return packet.Packet{}, err
	}
	_ = strLen

	// Skip old host
	oldHostLen := int(strLen)
	afterHost := remaining2[oldHostLen:]

	// protocolVersion + string length + newHost + afterHost
	newData := make([]byte, 0, protoLen+packet.MaxVarIntLen+len(newHost)+len(afterHost))
	newData = append(newData, data[:protoLen]...)

	buf := make([]byte, packet.MaxVarIntLen)
	n := packet.PutVarInt(buf, int32(len(newHost)))
	newData = append(newData, buf[:n]...)
	newData = append(newData, []byte(newHost)...)
	newData = append(newData, afterHost...)

	return packet.Packet{ID: pkt.ID, Data: newData}, nil
}

// HandshakeState contains the next state values for handshake.
const (
	StateStatus = 1
	StateLogin  = 2
	StateTransfer = 3 // 1.20.5+
)
