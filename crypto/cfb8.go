package crypto

import (
	"crypto/cipher"
)

// CFB8 is the 8-bit CFB (Cipher Feedback) mode used by Minecraft.
type CFB8 struct {
	block cipher.Block
	iv    []byte
	tmp   []byte
	de    bool
}

// NewCFB8Encrypt creates a new CFB8 encryption stream.
func NewCFB8Encrypt(block cipher.Block, iv []byte) *CFB8 {
	s := &CFB8{
		block: block,
		iv:    make([]byte, len(iv)),
		tmp:   make([]byte, block.BlockSize()),
		de:    false,
	}
	copy(s.iv, iv)
	return s
}

// NewCFB8Decrypt creates a new CFB8 decryption stream.
func NewCFB8Decrypt(block cipher.Block, iv []byte) *CFB8 {
	s := &CFB8{
		block: block,
		iv:    make([]byte, len(iv)),
		tmp:   make([]byte, block.BlockSize()),
		de:    true,
	}
	copy(s.iv, iv)
	return s
}

// XORKeyStream XORs the given src and writes to dst.
func (s *CFB8) XORKeyStream(dst, src []byte) {
	if len(src) == 0 {
		return
	}
	if len(dst) < len(src) {
		panic("crypto/cfb8: dst smaller than src")
	}

	blockSize := s.block.BlockSize()

	if len(src) >= blockSize*2 {
		s.bulkXOR(dst, src, blockSize)
		return
	}

	s.byteByByteXOR(dst, src, blockSize)
}

func (s *CFB8) bulkXOR(dst, src []byte, blockSize int) {
	blocks := len(src) / blockSize
	remainder := len(src) % blockSize

	for i := 0; i < blocks; i++ {
		start := i * blockSize
		s.block.Encrypt(s.tmp, s.iv)
		for j := 0; j < blockSize; j++ {
			dst[start+j] = src[start+j] ^ s.tmp[j]
		}
		if s.de {
			copy(s.iv, src[start:start+blockSize])
		} else {
			copy(s.iv, dst[start:start+blockSize])
		}
	}

	if remainder > 0 {
		start := blocks * blockSize
		s.block.Encrypt(s.tmp, s.iv)
		for j := 0; j < remainder; j++ {
			dst[start+j] = src[start+j] ^ s.tmp[j]
			if s.de {
				s.iv[j] = src[start+j]
			} else {
				s.iv[j] = dst[start+j]
			}
		}
	}
}

func (s *CFB8) byteByByteXOR(dst, src []byte, blockSize int) {
	overflow := make([]byte, blockSize*2)
	copy(overflow, s.iv)

	for i := range src {
		if i%blockSize == 0 {
			s.block.Encrypt(s.tmp, overflow[:blockSize])
		}
		dst[i] = src[i] ^ s.tmp[i%blockSize]
		if s.de {
			overflow[i%blockSize+blockSize] = src[i]
		} else {
			overflow[i%blockSize+blockSize] = dst[i]
		}
		if i%blockSize == blockSize-1 {
			copy(overflow, overflow[blockSize:])
		}
	}

	copy(s.iv, overflow)
}
