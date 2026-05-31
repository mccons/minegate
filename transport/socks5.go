package transport

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/user/minegate/internal"
)

// SOCKS5Transport is a SOCKS5 proxy client.
// It routes Minecraft traffic through a SOCKS5 proxy.
type SOCKS5Transport struct {
	ProxyAddr   string
	Username    string
	Password    string
	DialTimeout time.Duration
}

// NewSOCKS5Transport creates a new SOCKS5Transport.
func NewSOCKS5Transport(proxyAddr string) *SOCKS5Transport {
	return &SOCKS5Transport{
		ProxyAddr:   proxyAddr,
		DialTimeout: 30 * time.Second,
	}
}

// SOCKS5 constants
const (
	socks5Version        = 5
	socks5AuthNone       = 0
	socks5AuthPassword   = 2
	socks5CmdConnect     = 1
	socks5AddrTypeIPv4   = 1
	socks5AddrTypeDomain = 3
	socks5AddrTypeIPv6   = 4
)

// Dial opens a connection to the target address via SOCKS5 proxy.
func (s *SOCKS5Transport) Dial(ctx context.Context, targetAddr string) (net.Conn, error) {
	proxyConn, err := (&net.Dialer{Timeout: s.DialTimeout}).DialContext(ctx, "tcp", s.ProxyAddr)
	if err != nil {
		return nil, fmt.Errorf("socks5: proxy connection failed: %w", err)
	}

	if err := s.handshake(proxyConn); err != nil {
		proxyConn.Close()
		return nil, err
	}

	if err := s.request(proxyConn, targetAddr); err != nil {
		proxyConn.Close()
		return nil, err
	}

	return proxyConn, nil
}

// Listen is not supported for SOCKS5 transport (client only).
func (s *SOCKS5Transport) Listen(ctx context.Context, addr string) (net.Listener, error) {
	return nil, errors.New("socks5: listen not supported")
}

func (s *SOCKS5Transport) handshake(conn net.Conn) error {
	// Negotiate authentication methods
	var methods []byte
	if s.Username != "" {
		methods = []byte{socks5Version, 2, socks5AuthNone, socks5AuthPassword}
	} else {
		methods = []byte{socks5Version, 1, socks5AuthNone}
	}

	if _, err := conn.Write(methods); err != nil {
		return fmt.Errorf("socks5: handshake write failed: %w", err)
	}

	resp := make([]byte, 2)
	if _, err := conn.Read(resp); err != nil {
		return fmt.Errorf("socks5: handshake read failed: %w", err)
	}

	if resp[0] != socks5Version {
		return fmt.Errorf("socks5: unsupported version: %d", resp[0])
	}

	if resp[1] == socks5AuthPassword {
		return s.passwordAuth(conn)
	}

	if resp[1] != socks5AuthNone {
		return fmt.Errorf("socks5: auth method %d not supported", resp[1])
	}

	return nil
}

func (s *SOCKS5Transport) passwordAuth(conn net.Conn) error {
	// RFC 1929 username/password auth
	auth := []byte{1, byte(len(s.Username))}
	auth = append(auth, []byte(s.Username)...)
	auth = append(auth, byte(len(s.Password)))
	auth = append(auth, []byte(s.Password)...)

	if _, err := conn.Write(auth); err != nil {
		return fmt.Errorf("socks5: auth write failed: %w", err)
	}

	resp := make([]byte, 2)
	if _, err := conn.Read(resp); err != nil {
		return fmt.Errorf("socks5: auth read failed: %w", err)
	}

	if resp[1] != 0 {
		return fmt.Errorf("socks5: auth rejected: code %d", resp[1])
	}

	return nil
}

func (s *SOCKS5Transport) request(conn net.Conn, targetAddr string) error {
	host, portStr, err := net.SplitHostPort(targetAddr)
	if err != nil {
		return fmt.Errorf("socks5: invalid target address: %w", err)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("socks5: invalid port: %w", err)
	}

	// Build request: VER | CMD | RSV | ATYP | DST.ADDR | DST.PORT
	req := []byte{socks5Version, socks5CmdConnect, 0}

	ip := net.ParseIP(host)
	if ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			req = append(req, socks5AddrTypeIPv4)
			req = append(req, ip4...)
		} else if ip16 := ip.To16(); ip16 != nil {
			req = append(req, socks5AddrTypeIPv6)
			req = append(req, ip16...)
		}
	} else {
		req = append(req, socks5AddrTypeDomain)
		req = append(req, byte(len(host)))
		req = append(req, []byte(host)...)
	}

	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, uint16(port))
	req = append(req, portBytes...)

	if _, err := conn.Write(req); err != nil {
		return fmt.Errorf("socks5: request write failed: %w", err)
	}

	resp := make([]byte, 4)
	if _, err := conn.Read(resp); err != nil {
		return fmt.Errorf("socks5: request read failed: %w", err)
	}

	if resp[1] != 0 {
		return fmt.Errorf("socks5: request rejected: code %d", resp[1])
	}

	// Read BND.ADDR (variable length)
	switch resp[3] {
	case socks5AddrTypeIPv4:
		resp = make([]byte, 6)
	case socks5AddrTypeIPv6:
		resp = make([]byte, 18)
	case socks5AddrTypeDomain:
		if _, err := conn.Read(resp[:1]); err != nil {
			return fmt.Errorf("socks5: read domain length failed: %w", err)
		}
		resp = make([]byte, resp[0]+2)
	default:
		return fmt.Errorf("socks5: unknown address type: %d", resp[3])
	}

	if _, err := conn.Read(resp); err != nil {
		return fmt.Errorf("socks5: read bind address failed: %w", err)
	}

	return nil
}

// Ensure internal is used
var _ = internal.ErrConnectionClosed
