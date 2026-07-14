// SPDX-License-Identifier: MIT

package sso

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestBuildIDTokenClaims(t *testing.T) {
	now := time.Unix(1_800_000_000, 0).UTC()
	subject := testTokenSubject()

	claims := BuildIDTokenClaims("https://idb.example.test", subject, now, 15*time.Minute)

	assertClaim(t, claims, "iss", "https://idb.example.test")
	assertClaim(t, claims, "sub", subject.UserID)
	assertClaim(t, claims, "aud", subject.ClientID)
	assertClaim(t, claims, "entity_id", subject.EntityID)
	assertClaim(t, claims, "sid", subject.SessionID)
	assertClaim(t, claims, "nonce", "nonce-1")
	assertClaim(t, claims, "locale", "zh-CN")
	assertClaim(t, claims, "name", "Ada Lovelace")
	assertClaim(t, claims, "email", "ada@example.test")
	assertClaim(t, claims, "phone_number", "+8613800000000")
	assertClaim(t, claims, "picture", "https://example.test/avatar.png")
	assertClaim(t, claims, "preferred_username", "ada")

	if got := claims["identity_sources"].([]string); len(got) != 2 || got[0] != "feishu" || got[1] != "dingtalk" {
		t.Fatalf("identity_sources = %#v", claims["identity_sources"])
	}
	if got := claims["iat"].(*jwt.NumericDate).Time; !got.Equal(now) {
		t.Fatalf("iat = %s, want %s", got, now)
	}
	if got := claims["exp"].(*jwt.NumericDate).Time; !got.Equal(now.Add(15 * time.Minute)) {
		t.Fatalf("exp = %s, want %s", got, now.Add(15*time.Minute))
	}
}

func TestBuildAccessTokenClaims(t *testing.T) {
	now := time.Unix(1_800_000_000, 0).UTC()
	subject := testTokenSubject()

	claims := BuildAccessTokenClaims("https://idb.example.test", subject, []string{"openid", "email"}, now, 15*time.Minute)

	assertClaim(t, claims, "iss", "https://idb.example.test")
	assertClaim(t, claims, "sub", subject.UserID)
	assertClaim(t, claims, "aud", subject.ClientID)
	assertClaim(t, claims, "entity_id", subject.EntityID)
	assertClaim(t, claims, "sid", subject.SessionID)
	assertClaim(t, claims, "scope", "openid email")
	assertClaim(t, claims, "permissions_version", int64(1))
	assertClaim(t, claims, "resource_scopes_version", int64(1))
	if _, ok := claims["identity_sources"]; ok {
		t.Fatal("access token includes identity_sources, want omitted")
	}
	if _, ok := claims["resource_scopes"]; ok {
		t.Fatal("access token includes full resource scopes, want omitted")
	}
	if got := claims["roles"].([]string); len(got) != 2 || got[0] != "admin" || got[1] != "operator" {
		t.Fatalf("roles = %#v", claims["roles"])
	}
}

func TestSignRS256(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	signed, err := SignRS256(jwt.MapClaims{"sub": "user-1"}, "dev-key-1", key)
	if err != nil {
		t.Fatalf("SignRS256() error = %v", err)
	}

	token, err := jwt.Parse(signed, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != "RS256" {
			t.Fatalf("alg = %q, want RS256", token.Method.Alg())
		}
		if kid := token.Header["kid"]; kid != "dev-key-1" {
			t.Fatalf("kid = %#v, want %q", kid, "dev-key-1")
		}
		return &key.PublicKey, nil
	})
	if err != nil {
		t.Fatalf("parse signed token: %v", err)
	}
	if !token.Valid {
		t.Fatal("token.Valid = false, want true")
	}
}

func TestSignRS256RejectsInvalidInput(t *testing.T) {
	if _, err := SignRS256(jwt.MapClaims{}, "", &rsa.PrivateKey{}); err == nil {
		t.Fatal("SignRS256() error = nil, want error for empty key id")
	}
	if _, err := SignRS256(jwt.MapClaims{}, "kid", nil); err == nil {
		t.Fatal("SignRS256() error = nil, want error for nil key")
	}
}

func testTokenSubject() TokenSubject {
	return TokenSubject{
		EntityID:          "entity-1",
		UserID:            "user-1",
		ClientID:          "client-1",
		SessionID:         "session-1",
		Nonce:             "nonce-1",
		Name:              "Ada Lovelace",
		Email:             "ada@example.test",
		PhoneNumber:       "+8613800000000",
		Picture:           "https://example.test/avatar.png",
		PreferredUsername: "ada",
		Locale:            "zh-CN",
		IdentitySources:   []string{"feishu", "dingtalk"},
		Roles:             []string{"admin", "operator"},
	}
}

func assertClaim(t *testing.T, claims jwt.MapClaims, key string, want interface{}) {
	t.Helper()
	if got := claims[key]; got != want {
		t.Fatalf("%s = %#v, want %#v", key, got, want)
	}
}
