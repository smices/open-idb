package secureconfig

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

var (
	encodedPrefix = []byte("idb:v1:")
	aad           = []byte("idbridge-config-v1")
)

// Codec encrypts configuration values while continuing to read the legacy
// plaintext JSON format used by existing installations.
type Codec struct {
	aead cipher.AEAD
}

// New creates a codec from a base64-encoded 32-byte AES-256 key. An empty key
// keeps legacy plaintext writes enabled so an upgrade does not make existing
// deployments fail before the key is configured.
func New(encodedKey string) (*Codec, error) {
	if encodedKey == "" {
		return &Codec{}, nil
	}
	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil {
		return nil, fmt.Errorf("decode config encryption key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("config encryption key must decode to 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create config cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create config AEAD: %w", err)
	}
	return &Codec{aead: aead}, nil
}

func (c *Codec) Enabled() bool {
	return c != nil && c.aead != nil
}

// Seal returns the versioned encrypted representation. Without a configured
// key it returns a copy of the legacy plaintext value for upgrade compatibility.
func (c *Codec) Seal(plain []byte) ([]byte, error) {
	if !c.Enabled() {
		return append([]byte(nil), plain...), nil
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate config nonce: %w", err)
	}
	sealed := c.aead.Seal(nonce, nonce, plain, aad)
	encoded := make([]byte, len(encodedPrefix)+base64.StdEncoding.EncodedLen(len(sealed)))
	copy(encoded, encodedPrefix)
	base64.StdEncoding.Encode(encoded[len(encodedPrefix):], sealed)
	return encoded, nil
}

// Open returns plaintext, whether the input used the encrypted format, and an
// error. Values without the version prefix are treated as legacy plaintext.
func (c *Codec) Open(value []byte) ([]byte, bool, error) {
	if len(value) < len(encodedPrefix) || string(value[:len(encodedPrefix)]) != string(encodedPrefix) {
		return append([]byte(nil), value...), false, nil
	}
	if !c.Enabled() {
		return nil, true, errors.New("encrypted config requires IDB_CONFIG_ENCRYPTION_KEY")
	}
	sealed, err := base64.StdEncoding.DecodeString(string(value[len(encodedPrefix):]))
	if err != nil {
		return nil, true, fmt.Errorf("decode encrypted config: %w", err)
	}
	if len(sealed) < c.aead.NonceSize() {
		return nil, true, errors.New("encrypted config is truncated")
	}
	nonce := sealed[:c.aead.NonceSize()]
	ciphertext := sealed[c.aead.NonceSize():]
	plain, err := c.aead.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, true, fmt.Errorf("decrypt config: %w", err)
	}
	return plain, true, nil
}
