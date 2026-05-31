package main

import (
	"fmt"
	"log"

	"github.com/user/minegate/proxy"
	"github.com/user/minegate/transport"
	"github.com/user/minegate/tunnel"
)

func main() {
	tcpTr := &transport.TCPTransport{}
	ln := tunnel.NewListener(tcpTr)
	if err := ln.Listen(":25577"); err != nil {
		log.Fatal(err)
	}
	fmt.Println("minegate proxy listening on :25577")

	dialer := tunnel.NewDialer(tcpTr)
	p := proxy.NewProxy(ln, dialer)

	log.Fatal(p.Start())
}
