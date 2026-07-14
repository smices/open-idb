// SPDX-License-Identifier: MIT

package sso

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"testing"
)

func TestGenerateRSAKey(t *testing.T) {
	key, err := GenerateRSAKey()
	if err != nil {
		t.Fatalf("GenerateRSAKey() error = %v", err)
	}
	if key.N.BitLen() != 2048 {
		t.Fatalf("key bits = %d, want %d", key.N.BitLen(), 2048)
	}
}

func TestParseRSAPrivateKeyPEM_RoundTripsGeneratedKey(t *testing.T) {
	original, err := GenerateRSAKey()
	if err != nil {
		t.Fatalf("GenerateRSAKey() error = %v", err)
	}

	encoded, err := EncodeRSAPrivateKeyPEM(original)
	if err != nil {
		t.Fatalf("EncodeRSAPrivateKeyPEM() error = %v", err)
	}
	parsed, err := ParseRSAPrivateKeyPEM(encoded)
	if err != nil {
		t.Fatalf("ParseRSAPrivateKeyPEM() error = %v", err)
	}
	if parsed.N.Cmp(original.N) != 0 || parsed.D.Cmp(original.D) != 0 {
		t.Fatal("parsed private key does not match original key")
	}
}

func TestParseRSAPrivateKeyPEMRejectsInvalidInput(t *testing.T) {
	if _, err := ParseRSAPrivateKeyPEM("not a private key"); err == nil {
		t.Fatal("ParseRSAPrivateKeyPEM() error = nil, want error")
	}
}

func TestPublicJWK(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	jwk, err := PublicJWK("dev-key-1", key)
	if err != nil {
		t.Fatalf("PublicJWK() error = %v", err)
	}

	if jwk.KeyID != "dev-key-1" {
		t.Fatalf("kid = %q, want %q", jwk.KeyID, "dev-key-1")
	}
	if jwk.KeyType != "RSA" {
		t.Fatalf("kty = %q, want %q", jwk.KeyType, "RSA")
	}
	if jwk.Use != "sig" {
		t.Fatalf("use = %q, want %q", jwk.Use, "sig")
	}
	if jwk.Algorithm != "RS256" {
		t.Fatalf("alg = %q, want %q", jwk.Algorithm, "RS256")
	}
	if jwk.Modulus == "" {
		t.Fatal("modulus is empty")
	}
	if jwk.Exponent != base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}) {
		t.Fatalf("exponent = %q, want AQAB", jwk.Exponent)
	}
}

func TestPublicJWKRejectsInvalidInput(t *testing.T) {
	if _, err := PublicJWK("", &rsa.PrivateKey{}); err == nil {
		t.Fatal("PublicJWK() error = nil, want error for empty key id")
	}
	if _, err := PublicJWK("kid", nil); err == nil {
		t.Fatal("PublicJWK() error = nil, want error for nil key")
	}
}
