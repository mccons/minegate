package internal

import "io"

func DiscardVarInt(r io.ByteReader) {
	for i := 0; i < 10; i++ {
		b, err := r.ReadByte()
		if err != nil || b&0x80 == 0 {
			return
		}
	}
}
