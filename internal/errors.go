package internal

import "errors"

var (
	ErrPacketTooLarge    = errors.New("packet too large")
	ErrInvalidPacketID   = errors.New("invalid packet ID")
	ErrConnectionClosed  = errors.New("connection closed")
	ErrBufferOverflow    = errors.New("buffer overflow")
	ErrCompressionFailed = errors.New("compression failed")
	ErrEncryptionFailed  = errors.New("encryption failed")
	ErrHandshakeFailed   = errors.New("handshake failed")
	ErrUnsupportedState  = errors.New("unsupported protocol state")
	ErrPacketTooShort    = errors.New("packet too short")
	ErrMalformedVarInt   = errors.New("malformed VarInt")
	ErrMalformedVarLong  = errors.New("malformed VarLong")
	ErrQueueFull         = errors.New("queue is full")
	ErrQueueEmpty        = errors.New("queue is empty")
	ErrTimeout           = errors.New("operation timed out")
	ErrInvalidTransport  = errors.New("invalid transport type")
	ErrProxyRejected     = errors.New("proxy rejected connection")
)
