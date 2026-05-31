package packet

import (
	"encoding/binary"
	"io"

	"github.com/pozii/minegate/internal"
)

// maxVarIntLen = 5 bytes, maxVarLongLen = 10 bytes
const (
	MaxVarIntLen  = 5
	MaxVarLongLen = 10
)

// VarInt is a variable-length integer in the Minecraft protocol.
type VarInt int32

// Len returns the byte length of the VarInt in wire format.
func (v VarInt) Len() int {
	return VarIntLen(int32(v))
}

// WriteTo writes the VarInt to w.
func (v VarInt) WriteTo(w io.Writer) (int64, error) {
	buf := make([]byte, MaxVarIntLen)
	n := PutVarInt(buf, int32(v))
	wn, err := w.Write(buf[:n])
	return int64(wn), err
}

// ReadFrom reads a VarInt from r.
func (v *VarInt) ReadFrom(r io.ByteReader) error {
	val, err := ReadVarInt(r)
	if err != nil {
		return err
	}
	*v = VarInt(val)
	return nil
}

// VarLong is a variable-length 64-bit integer in the Minecraft protocol.
type VarLong int64

func (v VarLong) Len() int {
	return VarLongLen(int64(v))
}

func (v VarLong) WriteTo(w io.Writer) (int64, error) {
	buf := make([]byte, MaxVarLongLen)
	n := PutVarLong(buf, int64(v))
	wn, err := w.Write(buf[:n])
	return int64(wn), err
}

func (v *VarLong) ReadFrom(r io.ByteReader) error {
	val, err := ReadVarLong(r)
	if err != nil {
		return err
	}
	*v = VarLong(val)
	return nil
}

// VarIntLen returns how many bytes an int32 occupies as a VarInt.
func VarIntLen(num int32) int {
	u := uint32(num)
	switch {
	case u < 0x80:
		return 1
	case u < 0x4000:
		return 2
	case u < 0x200000:
		return 3
	case u < 0x10000000:
		return 4
	default:
		return 5
	}
}

// VarLongLen returns how many bytes an int64 occupies as a VarLong.
func VarLongLen(num int64) int {
	u := uint64(num)
	switch {
	case u < 0x80:
		return 1
	case u < 0x4000:
		return 2
	case u < 0x200000:
		return 3
	case u < 0x10000000:
		return 4
	case u < 0x800000000:
		return 5
	case u < 0x40000000000:
		return 6
	case u < 0x2000000000000:
		return 7
	case u < 0x100000000000000:
		return 8
	case u < 0x8000000000000000:
		return 9
	default:
		return 10
	}
}

// PutVarInt encodes a VarInt into buf and returns the number of bytes written.
func PutVarInt(buf []byte, num int32) int {
	u := uint32(num)
	switch {
	case u < 0x80:
		buf[0] = byte(u)
		return 1
	case u < 0x4000:
		buf[0] = byte(u) | 0x80
		buf[1] = byte(u >> 7)
		return 2
	case u < 0x200000:
		buf[0] = byte(u) | 0x80
		buf[1] = byte(u>>7) | 0x80
		buf[2] = byte(u >> 14)
		return 3
	case u < 0x10000000:
		buf[0] = byte(u) | 0x80
		buf[1] = byte(u>>7) | 0x80
		buf[2] = byte(u>>14) | 0x80
		buf[3] = byte(u >> 21)
		return 4
	default:
		buf[0] = byte(u) | 0x80
		buf[1] = byte(u>>7) | 0x80
		buf[2] = byte(u>>14) | 0x80
		buf[3] = byte(u>>21) | 0x80
		buf[4] = byte(u >> 28)
		return 5
	}
}

// PutVarLong encodes a VarLong into buf and returns the number of bytes written.
func PutVarLong(buf []byte, num int64) int {
	u := uint64(num)
	switch {
	case u < 0x80:
		buf[0] = byte(u)
		return 1
	case u < 0x4000:
		buf[0] = byte(u) | 0x80
		buf[1] = byte(u >> 7)
		return 2
	case u < 0x200000:
		buf[0] = byte(u) | 0x80
		buf[1] = byte(u>>7) | 0x80
		buf[2] = byte(u >> 14)
		return 3
	case u < 0x10000000:
		buf[0] = byte(u) | 0x80
		buf[1] = byte(u>>7) | 0x80
		buf[2] = byte(u>>14) | 0x80
		buf[3] = byte(u >> 21)
		return 4
	case u < 0x800000000:
		buf[0] = byte(u) | 0x80
		buf[1] = byte(u>>7) | 0x80
		buf[2] = byte(u>>14) | 0x80
		buf[3] = byte(u>>21) | 0x80
		buf[4] = byte(u >> 28)
		return 5
	case u < 0x40000000000:
		buf[0] = byte(u) | 0x80
		buf[1] = byte(u>>7) | 0x80
		buf[2] = byte(u>>14) | 0x80
		buf[3] = byte(u>>21) | 0x80
		buf[4] = byte(u>>28) | 0x80
		buf[5] = byte(u >> 35)
		return 6
	case u < 0x2000000000000:
		buf[0] = byte(u) | 0x80
		buf[1] = byte(u>>7) | 0x80
		buf[2] = byte(u>>14) | 0x80
		buf[3] = byte(u>>21) | 0x80
		buf[4] = byte(u>>28) | 0x80
		buf[5] = byte(u>>35) | 0x80
		buf[6] = byte(u >> 42)
		return 7
	case u < 0x100000000000000:
		buf[0] = byte(u) | 0x80
		buf[1] = byte(u>>7) | 0x80
		buf[2] = byte(u>>14) | 0x80
		buf[3] = byte(u>>21) | 0x80
		buf[4] = byte(u>>28) | 0x80
		buf[5] = byte(u>>35) | 0x80
		buf[6] = byte(u>>42) | 0x80
		buf[7] = byte(u >> 49)
		return 8
	case u < 0x8000000000000000:
		buf[0] = byte(u) | 0x80
		buf[1] = byte(u>>7) | 0x80
		buf[2] = byte(u>>14) | 0x80
		buf[3] = byte(u>>21) | 0x80
		buf[4] = byte(u>>28) | 0x80
		buf[5] = byte(u>>35) | 0x80
		buf[6] = byte(u>>42) | 0x80
		buf[7] = byte(u>>49) | 0x80
		buf[8] = byte(u >> 56)
		return 9
	default:
		buf[0] = byte(u) | 0x80
		buf[1] = byte(u>>7) | 0x80
		buf[2] = byte(u>>14) | 0x80
		buf[3] = byte(u>>21) | 0x80
		buf[4] = byte(u>>28) | 0x80
		buf[5] = byte(u>>35) | 0x80
		buf[6] = byte(u>>42) | 0x80
		buf[7] = byte(u>>49) | 0x80
		buf[8] = byte(u>>56) | 0x80
		buf[9] = byte(u >> 63)
		return 10
	}
}

// ReadVarInt reads a VarInt from a ByteReader.
func ReadVarInt(r io.ByteReader) (int32, error) {
	var (
		val   int32
		shift uint
	)

	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		val |= int32(b&0x7F) << shift
		if b&0x80 == 0 {
			return val, nil
		}
		shift += 7
		if shift >= 35 {
			internal.DiscardVarInt(r)
			return 0, internal.ErrMalformedVarInt
		}
	}
}

// ReadVarLong reads a VarLong from a ByteReader.
func ReadVarLong(r io.ByteReader) (int64, error) {
	var (
		val   int64
		shift uint
	)

	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		val |= int64(b&0x7F) << shift
		if b&0x80 == 0 {
			return val, nil
		}
		shift += 7
		if shift >= 70 {
			internal.DiscardVarInt(r)
			return 0, internal.ErrMalformedVarLong
		}
	}
}

// ReadVarIntN reads exactly n bytes from a reader and returns a VarInt.
func ReadVarIntN(r io.Reader, n int) (int32, error) {
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return 0, err
	}
	var val int32
	for i := 0; i < n; i++ {
		val |= int32(buf[i]&0x7F) << (7 * i)
		if buf[i]&0x80 == 0 {
			break
		}
	}
	return val, nil
}

// PeekVarIntSize estimates the VarInt size from the first byte.
// This gives the maximum size, not the actual length.
func PeekVarIntSize(b byte) int {
	if b&0x80 == 0 {
		return 1
	}
	return MaxVarIntLen
}

// UuidToInts converts a UUID string to two int64s.
func UuidToInts(uuid [16]byte) (int64, int64) {
	return int64(binary.BigEndian.Uint64(uuid[:8])), int64(binary.BigEndian.Uint64(uuid[8:]))
}

// IntsToUUID converts two int64s to a [16]byte UUID.
func IntsToUUID(hi, lo int64) [16]byte {
	var uuid [16]byte
	binary.BigEndian.PutUint64(uuid[:8], uint64(hi))
	binary.BigEndian.PutUint64(uuid[8:], uint64(lo))
	return uuid
}

// ReadVarIntFromBytes reads a VarInt from a byte slice and returns the remaining slice.
func ReadVarIntFromBytes(data []byte) (VarInt, []byte, error) {
	if len(data) == 0 {
		return 0, nil, internal.ErrPacketTooShort
	}
	var val VarInt
	var shift uint
	for i, b := range data {
		val |= VarInt(b&0x7F) << shift
		if b&0x80 == 0 {
			return val, data[i+1:], nil
		}
		shift += 7
		if shift >= 35 {
			return 0, nil, internal.ErrMalformedVarInt
		}
	}
	return 0, nil, internal.ErrPacketTooShort
}

// WriteVarIntToBytes writes a VarInt to a byte slice.
func WriteVarIntToBytes(buf []byte, v VarInt) int {
	return PutVarInt(buf, int32(v))
}


