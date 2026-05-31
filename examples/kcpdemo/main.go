package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/pozii/minegate/tunnel"
)

func main() {
	// KCP echo server
	go kcpServer(":9000")

	time.Sleep(500 * time.Millisecond)

	// KCP echo client
	kcpClient("localhost:9000")
}

func kcpServer(addr string) {
	ln := tunnel.NewListener(nil)
	if err := ln.Listen(addr); err != nil {
		log.Fatal(err)
	}
	fmt.Println("KCP server listening on", addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go func(c net.Conn) {
			defer c.Close()
			io.Copy(c, c) // echo
		}(conn)
	}
}

func kcpClient(addr string) {
	dialer := tunnel.NewDialer(nil)
	conn, err := dialer.Dial(addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	msg := []byte("Hello minegate over KCP!")
	if _, err := conn.Write(msg); err != nil {
		log.Fatal(err)
	}

	buf := make([]byte, len(msg))
	if _, err := conn.Read(buf); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Echo received:", string(buf))
}
