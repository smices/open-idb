// SPDX-License-Identifier: MIT

package sso

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
)

type JWK struct {
	KeyID     string `json:"kid"`
	KeyType   string `json:"kty"`
	Use       string `json:"use"`
	Algorithm string `json:"alg"`
	Modulus   string `json:"n"`
	Exponent  string `json:"e"`
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

func GenerateRSAKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 2048)
}

func PublicJWK(keyID string, key *rsa.PrivateKey) (JWK, error) {
	if keyID == "" {
		return JWK{}, fmt.Errorf("key id is required")
	}
	if key == nil {
		return JWK{}, fmt.Errorf("rsa private key is required")
	}
	publicKey := key.PublicKey
	exponentBytes := intToBytes(publicKey.E)

	return JWK{
		KeyID:     keyID,
		KeyType:   "RSA",
		Use:       "sig",
		Algorithm: SigningAlgorithmRS256,
		Modulus:   base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes()),
		Exponent:  base64.RawURLEncoding.EncodeToString(exponentBytes),
	}, nil
}

func intToBytes(value int) []byte {
	if value == 0 {
		return []byte{0}
	}

	var out []byte
	for value > 0 {
		out = append([]byte{byte(value)}, out...)
		value >>= 8
	}
	return out
}
