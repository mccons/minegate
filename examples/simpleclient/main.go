package main

import (
	"fmt"
	"log"

	"github.com/pozii/minegate/packet"
	"github.com/pozii/minegate/tunnel"
	"github.com/pozii/minegate/transport"
)

func main() {
	dialer := tunnel.NewDialer(&transport.TCPTransport{})
	conn, err := dialer.DialMinecraft("localhost:25565")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	fmt.Println("Connected to", conn.RemoteAddr())

	// Send handshake
	handshake := buildHandshake(765, "localhost", 25565, 2)
	if err := conn.Writer.WritePacket(handshake); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Handshake sent")
}

func buildHandshake(protocolVersion int32, host string, port uint16, nextState int32) packet.Packet {
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
