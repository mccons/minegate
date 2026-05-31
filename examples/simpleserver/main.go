package main

import (
	"fmt"
	"log"

	"github.com/pozii/minegate/proxy"
	"github.com/pozii/minegate/transport"
	"github.com/pozii/minegate/tunnel"
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
