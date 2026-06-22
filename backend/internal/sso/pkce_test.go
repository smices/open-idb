// SPDX-License-Identifier: MIT

package sso

import (
	"regexp"
	"testing"
)

func TestVerifyS256CodeChallenge(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

	if !VerifyS256CodeChallenge(verifier, challenge) {
		t.Fatal("VerifyS256CodeChallenge() = false, want true")
	}
}

func TestVerifyS256CodeChallengeRejectsWrongVerifier(t *testing.T) {
	if VerifyS256CodeChallenge("wrong", "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM") {
		t.Fatal("VerifyS256CodeChallenge() = true, want false")
	}
}

func TestRandomURLSafeToken(t *testing.T) {
	token, err := RandomURLSafeToken(32)
	if err != nil {
		t.Fatalf("RandomURLSafeToken() error = %v", err)
	}
	if token == "" {
		t.Fatal("token is empty")
	}
	if !regexp.MustCompile(`^[A-Za-z0-9_-]+$`).MatchString(token) {
		t.Fatalf("token %q is not URL-safe base64 without padding", token)
	}
}

func TestRandomURLSafeTokenRejectsInvalidLength(t *testing.T) {
	if _, err := RandomURLSafeToken(0); err == nil {
		t.Fatal("RandomURLSafeToken() error = nil, want error")
	}
}

func TestHashToken(t *testing.T) {
	got := HashToken("code")
	want := "5694d08a2e53ffcae0c3103e5ad6f6076abd960eb1f8a56577040bc1028f702b"
	if got != want {
		t.Fatalf("HashToken() = %q, want %q", got, want)
	}
	if HashToken("code") != got {
		t.Fatal("HashToken() is not deterministic")
	}
}
