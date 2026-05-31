package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"

	"github.com/pozii/minegate/internal"
)

// KeyExchange manages the encryption key exchange used during Minecraft login.
type KeyExchange struct {
	// ServerID is used for Mojang session authentication.
	ServerID string
	// PublicKey is the server's RSA public key.
	PublicKey []byte
	// VerifyToken is the verification token sent by the server.
	VerifyToken []byte
}

// GenerateKey generates a random AES-128 key on the client side.
func GenerateKey() ([]byte, error) {
	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

// EncryptKey encrypts the AES key with the server's RSA public key.
func EncryptKey(key, publicKey []byte) ([]byte, error) {
	pub, err := x509.ParsePKIXPublicKey(publicKey)
	if err != nil {
		pub, err = x509.ParsePKCS1PublicKey(publicKey)
		if err != nil {
			return nil, internal.ErrEncryptionFailed
		}
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, internal.ErrEncryptionFailed
	}

	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, rsaPub, key)
	if err != nil {
		return nil, internal.ErrEncryptionFailed
	}

	return encrypted, nil
}

// EncryptToken encrypts the verify token with the server's RSA public key.
func EncryptToken(token, publicKey []byte) ([]byte, error) {
	return EncryptKey(token, publicKey)
}

// CreateCipher creates a pair of CFB8 ciphers from the given AES key
// (one for encryption, one for decryption).
func CreateCipher(key []byte) (encrypt, decrypt cipher.Stream, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	encrypt = NewCFB8Encrypt(block, key)
	decrypt = NewCFB8Decrypt(block, key)
	return encrypt, decrypt, nil
}

// ComputeServerHash computes the server ID hash for Mojang session authentication.
func ComputeServerHash(serverID string, publicKey, secretKey []byte) string {
	h := sha1.New()
	h.Write([]byte(serverID))
	h.Write(secretKey)
	h.Write(publicKey)
	hash := h.Sum(nil)

	// Minecraft's special hash format (negative sign handling)
	var negative bool
	for i := 0; i < len(hash); i++ {
		if hash[i] != 0 {
			break
		}
		if i == len(hash)-1 {
			return ""
		}
	}

	if hash[0]&0x80 != 0 {
		negative = true
		for i := range hash {
			hash[i] = ^hash[i]
		}
		for i := len(hash) - 1; i >= 0; i-- {
			hash[i]++
			if hash[i] != 0 {
				break
			}
		}
	}

	hex := ""
	for _, b := range hash {
		hex += string("0123456789abcdef"[b>>4]) + string("0123456789abcdef"[b&0xf])
	}

	if negative {
		hex = "-" + hex
	}

	return hex
}

// VerifyHash compares the server's hash with the client's computed hash.
func VerifyHash(serverHash, computedHash string) bool {
	return serverHash == computedHash
}


