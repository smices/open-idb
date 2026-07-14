// SPDX-License-Identifier: MIT

package sso

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"
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

// ParseRSAPrivateKeyPEM loads an RSA signing key from a PEM value supplied by
// the deployment secret manager. PKCS#1 and PKCS#8 encodings are accepted so
// existing operational tooling can be used without conversion.
func ParseRSAPrivateKeyPEM(raw string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(raw)))
	if block == nil {
		return nil, fmt.Errorf("oidc private key must be PEM encoded")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		if err := key.Validate(); err != nil {
			return nil, fmt.Errorf("validate oidc private key: %w", err)
		}
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse oidc private key: %w", err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("oidc private key must be RSA")
	}
	if err := key.Validate(); err != nil {
		return nil, fmt.Errorf("validate oidc private key: %w", err)
	}
	return key, nil
}

// EncodeRSAPrivateKeyPEM is intentionally small and primarily supports
// operator tooling and round-trip tests. Applications should provide the
// resulting PEM through a secret manager, never through source control.
func EncodeRSAPrivateKeyPEM(key *rsa.PrivateKey) (string, error) {
	if key == nil {
		return "", fmt.Errorf("rsa private key is required")
	}
	if err := key.Validate(); err != nil {
		return "", fmt.Errorf("validate rsa private key: %w", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})), nil
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
