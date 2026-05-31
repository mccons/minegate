package main

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/pozii/minegate/conn"
	"github.com/pozii/minegate/packet"
)

func main() {
	// Normal write vs batched write performance comparison
	server, client := net.Pipe()

	// Normal writing
	start := time.Now()
	for i := 0; i < 10000; i++ {
		pw := packet.NewPacketWriter(client, -1)
		pw.WritePacket(packet.Packet{ID: 0x00, Data: make([]byte, 100)})
	}
	normalDur := time.Since(start)

	// Batched writing
	_ = server
	_ = client
	server2, client2 := net.Pipe()
	defer server2.Close()
	defer client2.Close()

	go io.Copy(io.Discard, server2)

	bw := conn.NewBatchWriter(client2, time.Millisecond, 65536)
	pw2 := packet.NewPacketWriter(bw, -1)

	start = time.Now()
	for i := 0; i < 10000; i++ {
		pw2.WritePacket(packet.Packet{ID: 0x00, Data: make([]byte, 100)})
	}
	bw.Flush()
	batchedDur := time.Since(start)

	fmt.Printf("Normal write (10000 packets): %v\n", normalDur)
	fmt.Printf("Batched write (10000 packets): %v\n", batchedDur)
	fmt.Printf("Speedup: %.2fx\n", float64(normalDur)/float64(batchedDur))
}
