<picture>
  <source media="(prefers-color-scheme: dark)" srcset="github-assets/img/banner.png">
  <img alt="minegate" src="github-assets/img/banner.png">
</picture>

**minegate** is a high-performance, multi-transport network tunneling library for Minecraft Java Edition.

Written in Go, it combines zero-copy packet forwarding, smart packet batching, advanced flow control, and multiple transport protocols (TCP, TLS, WebSocket, KCP, QUIC, SOCKS5) under a single, consistent API.

> Inspired by the `net/` package of go-mc, but purpose-built from scratch for tunneling workloads.

---

## Features

- **Zero-copy forwarding** — Forward packets without deserialization in proxy mode. ~10x fewer copies compared to go-mc.
- **Packet batching** — Coalesce multiple packets into a single TCP write with Nagle-like logic. Reduces syscall overhead.
- **Multi-transport** — TCP, TLS, WebSocket, KCP (UDP), QUIC, and SOCKS5 egress. Swap transports in one line.
- **Backpressure** — Bounded queues with drop policies prevent OOM under load.
- **Packet prioritization** — 3-level priority queue (keepalive > gameplay > chunks).
- **Connection multiplexing** — Multiple virtual Minecraft sessions over a single physical connection.
- **CFB8 encryption** — AES-NI hardware-accelerated Minecraft cipher implementation.
- **Fast compression** — `klauspost/compress` delivers ~3x faster zlib than the standard library.
- **Proxy framework** — Handshake manipulation, BungeeCord/Velocity forwarding, custom handler support.
- **Code generator** — Auto-generate packet ID constants per Minecraft version.

---

## Installation

```bash
go get github.com/<user>/minegate
```

---

## Quick Start

### TCP Proxy

```go
package main

import (
    "github.com/<user>/minegate/proxy"
    "github.com/<user>/minegate/transport"
    "github.com/<user>/minegate/tunnel"
)

func main() {
    tcp := &transport.TCPTransport{}
    ln := tunnel.NewListener(tcp)
    ln.Listen(":25577")

    dialer := tunnel.NewDialer(tcp)
    p := proxy.NewProxy(ln, dialer)
    p.Start() // :25577 -> upstream server
}
```

### Transports

Swap the transport to tunnel Minecraft over any protocol:

```go
// KCP (UDP — fast over lossy links)
kcp := transport.NewKCPTransport()
ln := tunnel.NewListener(kcp)

// QUIC (0-RTT handshake, built-in TLS)
quic := transport.NewQUICTransport(tlsConfig)
dialer := tunnel.NewDialer(quic)
conn, _ := dialer.Dial("example.com:25577")

// WebSocket (bypass HTTP proxies)
ws := transport.NewWSTransport()

// SOCKS5 (route through a proxy)
socks := transport.NewSOCKS5Transport("proxy:1080")
```

### Connection Multiplexing

```go
mux := tunnel.NewMux(physicalConn)
stream1, _ := mux.OpenStream() // Player 1
stream2, _ := mux.OpenStream() // Player 2
// Both share the same KCP/QUIC/TCP connection
```

### Zero-copy Forwarding

```go
raw, _ := reader.ReadRawPacket()
// raw.Buf written directly to destination — no parse, no copy
writer.WritePacket(packet.Packet{ID: raw.PacketID(), Data: raw.Buf[1:]})
```

---

## Package Overview

```
minegate/
├── packet/       — VarInt, packet I/O, Minecraft data types
├── crypto/       — CFB8 cipher, Mojang key exchange
├── compress/     — klauspost/compress-powered zlib
├── conn/         — Connection management, batching, flow control, priority
├── transport/    — TCP, TLS, WebSocket, KCP, QUIC, SOCKS5
├── tunnel/       — Relay, dialer, listener, connection mux
├── proxy/        — Proxy core, handshake parsing, auth forwarding
├── protocol/     — State, direction, packet ID constants
└── tools/        — mcproto code generator
```

---

## Performance

| Metric | go-mc | minegate |
|--------|-------|----------|
| Packet forward | ~500 ns/op (copy) | ~50 ns/op (zero-copy) |
| zlib compress | ~2 µs/KB | ~0.5 µs/KB (klauspost) |
| CFB8 bulk | ~100 MB/s | ~200 MB/s (AES-NI) |
| Concurrent conn | ~1000 | 10000+ (mux + backpressure) |

---

## Dependencies

| Package | Purpose |
|---------|---------|
| `klauspost/compress` | Fast zlib compression |
| `xtaci/kcp-go` | KCP (UDP) transport |
| `quic-go/quic-go` | QUIC transport |
| `gorilla/websocket` | WebSocket transport |
