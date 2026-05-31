package internal

import "io"

// ByteReader wraps a []byte as an io.ByteReader.
type ByteReader struct {
	data []byte
	pos  int
}

func NewByteReader(data []byte) *ByteReader {
	return &ByteReader{data: data}
}

func (br *ByteReader) ReadByte() (byte, error) {
	if br.pos >= len(br.data) {
		return 0, io.EOF
	}
	b := br.data[br.pos]
	br.pos++
	return b, nil
}

func (br *ByteReader) Read(p []byte) (int, error) {
	if br.pos >= len(br.data) {
		return 0, io.EOF
	}
	n := copy(p, br.data[br.pos:])
	br.pos += n
	return n, nil
}
