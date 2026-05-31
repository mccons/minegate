package packet

import (
	"encoding/binary"
	"io"
	"math"

	"github.com/user/minegate/internal"
)

// String is a length-prefixed UTF-8 string in the Minecraft protocol.
type String string

func (s String) WriteTo(w io.Writer) (int64, error) {
	data := []byte(s)
	length := VarInt(len(data))
	n, err := length.WriteTo(w)
	if err != nil {
		return n, err
	}
	wn, err := w.Write(data)
	return n + int64(wn), err
}

func (s *String) ReadFrom(r io.ByteReader) error {
	var length VarInt
	if err := length.ReadFrom(r); err != nil {
		return err
	}
	if length < 0 || length > math.MaxInt16 {
		return internal.ErrPacketTooLarge
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r.(io.Reader), buf); err != nil {
		return err
	}
	*s = String(buf)
	return nil
}

// Byte is a signed 8-bit integer in the Minecraft protocol.
type Byte int8

func (b Byte) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write([]byte{byte(b)})
	return int64(n), err
}

func (b *Byte) ReadFrom(r io.ByteReader) error {
	val, err := r.ReadByte()
	if err != nil {
		return err
	}
	*b = Byte(val)
	return nil
}

// UnsignedByte is an unsigned 8-bit integer in the Minecraft protocol.
type UnsignedByte uint8

func (ub UnsignedByte) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write([]byte{byte(ub)})
	return int64(n), err
}

func (ub *UnsignedByte) ReadFrom(r io.ByteReader) error {
	val, err := r.ReadByte()
	if err != nil {
		return err
	}
	*ub = UnsignedByte(val)
	return nil
}

// Short is a signed 16-bit integer in the Minecraft protocol (big-endian).
type Short int16

func (s Short) WriteTo(w io.Writer) (int64, error) {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, uint16(s))
	n, err := w.Write(buf)
	return int64(n), err
}

func (s *Short) ReadFrom(r io.ByteReader) error {
	buf := make([]byte, 2)
	if _, err := io.ReadFull(r.(io.Reader), buf); err != nil {
		return err
	}
	*s = Short(binary.BigEndian.Uint16(buf))
	return nil
}

// UnsignedShort is an unsigned 16-bit integer in the Minecraft protocol.
type UnsignedShort uint16

func (us UnsignedShort) WriteTo(w io.Writer) (int64, error) {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, uint16(us))
	n, err := w.Write(buf)
	return int64(n), err
}

func (us *UnsignedShort) ReadFrom(r io.ByteReader) error {
	buf := make([]byte, 2)
	if _, err := io.ReadFull(r.(io.Reader), buf); err != nil {
		return err
	}
	*us = UnsignedShort(binary.BigEndian.Uint16(buf))
	return nil
}

// Int is a signed 32-bit integer in the Minecraft protocol (big-endian).
type Int int32

func (i Int) WriteTo(w io.Writer) (int64, error) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(i))
	n, err := w.Write(buf)
	return int64(n), err
}

func (i *Int) ReadFrom(r io.ByteReader) error {
	buf := make([]byte, 4)
	if _, err := io.ReadFull(r.(io.Reader), buf); err != nil {
		return err
	}
	*i = Int(binary.BigEndian.Uint32(buf))
	return nil
}

// Long is a signed 64-bit integer in the Minecraft protocol (big-endian).
type Long int64

func (l Long) WriteTo(w io.Writer) (int64, error) {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(l))
	n, err := w.Write(buf)
	return int64(n), err
}

func (l *Long) ReadFrom(r io.ByteReader) error {
	buf := make([]byte, 8)
	if _, err := io.ReadFull(r.(io.Reader), buf); err != nil {
		return err
	}
	*l = Long(binary.BigEndian.Uint64(buf))
	return nil
}

// Float is a 32-bit floating point number in the Minecraft protocol.
type Float float32

func (f Float) WriteTo(w io.Writer) (int64, error) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, math.Float32bits(float32(f)))
	n, err := w.Write(buf)
	return int64(n), err
}

func (f *Float) ReadFrom(r io.ByteReader) error {
	buf := make([]byte, 4)
	if _, err := io.ReadFull(r.(io.Reader), buf); err != nil {
		return err
	}
	*f = Float(math.Float32frombits(binary.BigEndian.Uint32(buf)))
	return nil
}

// Double is a 64-bit floating point number in the Minecraft protocol.
type Double float64

func (d Double) WriteTo(w io.Writer) (int64, error) {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, math.Float64bits(float64(d)))
	n, err := w.Write(buf)
	return int64(n), err
}

func (d *Double) ReadFrom(r io.ByteReader) error {
	buf := make([]byte, 8)
	if _, err := io.ReadFull(r.(io.Reader), buf); err != nil {
		return err
	}
	*d = Double(math.Float64frombits(binary.BigEndian.Uint64(buf)))
	return nil
}

// Boolean is a boolean value in the Minecraft protocol (encoded as a byte).
type Boolean bool

func (b Boolean) WriteTo(w io.Writer) (int64, error) {
	var val byte
	if b {
		val = 1
	}
	n, err := w.Write([]byte{val})
	return int64(n), err
}

func (b *Boolean) ReadFrom(r io.ByteReader) error {
	val, err := r.ReadByte()
	if err != nil {
		return err
	}
	*b = val != 0
	return nil
}

// Position is a block position in the Minecraft protocol (encoded as an unsigned long).
type Position struct {
	X, Y, Z int
}

func (p Position) WriteTo(w io.Writer) (int64, error) {
	val := ((uint64(p.X) & 0x3FFFFFF) << 38) |
		((uint64(p.Z) & 0x3FFFFFF) << 12) |
		(uint64(p.Y) & 0xFFF)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, val)
	n, err := w.Write(buf)
	return int64(n), err
}

func (p *Position) ReadFrom(r io.ByteReader) error {
	buf := make([]byte, 8)
	if _, err := io.ReadFull(r.(io.Reader), buf); err != nil {
		return err
	}
	val := binary.BigEndian.Uint64(buf)
	p.X = int(val >> 38)
	p.Y = int(val & 0xFFF)
	p.Z = int(val >> 12 & 0x3FFFFFF)
	if p.X >= 0x2000000 {
		p.X -= 0x4000000
	}
	if p.Z >= 0x2000000 {
		p.Z -= 0x4000000
	}
	if p.Y >= 0x800 {
		p.Y -= 0x1000
	}
	return nil
}

// Angle is a 1-byte angle in the Minecraft protocol (360 degrees = 256 units).
type Angle byte

func (a Angle) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write([]byte{byte(a)})
	return int64(n), err
}

func (a *Angle) ReadFrom(r io.ByteReader) error {
	val, err := r.ReadByte()
	if err != nil {
		return err
	}
	*a = Angle(val)
	return nil
}

// UUID is a 16-byte UUID in the Minecraft protocol.
type UUID [16]byte

func (u UUID) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(u[:])
	return int64(n), err
}

func (u *UUID) ReadFrom(r io.ByteReader) error {
	if _, err := io.ReadFull(r.(io.Reader), u[:]); err != nil {
		return err
	}
	return nil
}

// ByteArray is a raw byte slice (without length prefix).
type ByteArray []byte

func (ba ByteArray) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(ba)
	return int64(n), err
}

func (ba *ByteArray) ReadFrom(r io.ByteReader, length int) error {
	*ba = make([]byte, length)
	if _, err := io.ReadFull(r.(io.Reader), *ba); err != nil {
		return err
	}
	return nil
}

// NBT represents Named Binary Tag data.
type NBT struct {
	Data []byte
}

func (nbt NBT) WriteTo(w io.Writer) (int64, error) {
	wn, err := w.Write(nbt.Data)
	return int64(wn), err
}

func (nbt *NBT) ReadFrom(r io.ByteReader) error {
	// NBT always starts with 0x0A (compound tag) and ends with 0x00
	// For now, reading raw bytes
	var buf []byte
	for {
		b, err := r.ReadByte()
		if err != nil {
			return err
		}
		buf = append(buf, b)
		if b == 0x00 && len(buf) > 1 && buf[len(buf)-2] == 0x00 {
			break
		}
	}
	nbt.Data = buf
	return nil
}

// Slot is an inventory slot data structure in the Minecraft protocol.
type Slot struct {
	Present bool
	ItemID  VarInt
	Count   Byte
	NBT     NBT
}

func (s Slot) WriteTo(w io.Writer) (int64, error) {
	var n int64
	if !s.Present {
		wn, err := Boolean(false).WriteTo(w)
		return wn, err
	}
	wn, err := Boolean(true).WriteTo(w)
	n += wn
	if err != nil {
		return n, err
	}
	wn, err = s.ItemID.WriteTo(w)
	n += wn
	if err != nil {
		return n, err
	}
	wn, err = s.Count.WriteTo(w)
	n += wn
	if err != nil {
		return n, err
	}
	wn, err = s.NBT.WriteTo(w)
	n += wn
	return n, err
}

func (s *Slot) ReadFrom(r io.ByteReader) error {
	var present Boolean
	if err := present.ReadFrom(r); err != nil {
		return err
	}
	s.Present = bool(present)
	if !s.Present {
		return nil
	}
	if err := s.ItemID.ReadFrom(r); err != nil {
		return err
	}
	if err := s.Count.ReadFrom(r); err != nil {
		return err
	}
	if err := s.NBT.ReadFrom(r); err != nil {
		return err
	}
	return nil
}

// BitSet is a variable-length bit set in the Minecraft protocol.
type BitSet []uint64

func (bs BitSet) WriteTo(w io.Writer) (int64, error) {
	length := VarInt(len(bs))
	n, err := length.WriteTo(w)
	if err != nil {
		return n, err
	}
	for _, v := range bs {
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, v)
		wn, err := w.Write(buf)
		n += int64(wn)
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (bs *BitSet) ReadFrom(r io.ByteReader) error {
	var length VarInt
	if err := length.ReadFrom(r); err != nil {
		return err
	}
	*bs = make(BitSet, length)
	for i := 0; i < int(length); i++ {
		buf := make([]byte, 8)
		if _, err := io.ReadFull(r.(io.Reader), buf); err != nil {
			return err
		}
		(*bs)[i] = binary.BigEndian.Uint64(buf)
	}
	return nil
}


