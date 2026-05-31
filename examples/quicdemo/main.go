package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"time"

	"github.com/user/minegate/transport"
	"github.com/user/minegate/tunnel"
)

func main() {
	go quicServer(":9000")
	time.Sleep(500 * time.Millisecond)
	quicClient("localhost:9000")
}

func generateTLSConfig() *tls.Config {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
	}
	certDER, _ := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	cert, _ := tls.X509KeyPair(certPEM, keyPEM)
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"minegate"},
	}
}

func quicServer(addr string) {
	tlsConfig := generateTLSConfig()
	quicTr := transport.NewQUICTransport(tlsConfig)
	ln := tunnel.NewListener(quicTr)
	if err := ln.Listen(addr); err != nil {
		log.Fatal(err)
	}
	fmt.Println("QUIC server listening on", addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go func(c net.Conn) {
			defer c.Close()
			io.Copy(c, c)
		}(conn)
	}
}

func quicClient(addr string) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"minegate"},
	}
	quicTr := transport.NewQUICTransport(tlsConfig)
	dialer := tunnel.NewDialer(quicTr)
	conn, err := dialer.Dial(addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	msg := []byte("Hello minegate over QUIC!")
	if _, err := conn.Write(msg); err != nil {
		log.Fatal(err)
	}

	buf := make([]byte, len(msg))
	if _, err := conn.Read(buf); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Echo received:", string(buf))
}
