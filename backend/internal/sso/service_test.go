// SPDX-License-Identifier: MIT

package sso

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/ory/fosite"
)

func TestNewServiceRequiresCoreDependencies(t *testing.T) {
	if _, err := NewService(ServiceConfig{}); err == nil {
		t.Fatal("NewService() error = nil, want error")
	}
}

func TestServiceDiscoveryDocument(t *testing.T) {
	service := newTestService(t)

	doc := service.DiscoveryDocument()

	if doc.Issuer != "https://idb.example.test" {
		t.Fatalf("Issuer = %q, want %q", doc.Issuer, "https://idb.example.test")
	}
	if doc.AuthorizationEndpoint != "https://idb.example.test/oauth2/authorize" {
		t.Fatalf("AuthorizationEndpoint = %q", doc.AuthorizationEndpoint)
	}
}

func TestServiceJWKS(t *testing.T) {
	service := newTestService(t)

	jwks := service.JWKS()

	if len(jwks.Keys) != 1 {
		t.Fatalf("len(Keys) = %d, want 1", len(jwks.Keys))
	}
	if jwks.Keys[0].KeyID != "dev-key-1" {
		t.Fatalf("kid = %q, want %q", jwks.Keys[0].KeyID, "dev-key-1")
	}
}

func TestValidateAuthorizeRequest(t *testing.T) {
	service := newTestService(t)

	decision, err := service.ValidateAuthorizeRequest(context.Background(), AuthorizeInput{
		EntityID:            "entity-1",
		ClientID:            "client-1",
		RedirectURI:         "https://app.example.test/callback",
		ResponseType:        "code",
		Scopes:              []string{"openid", "email"},
		State:               "state-1",
		Nonce:               "nonce-1",
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
	})
	if err != nil {
		t.Fatalf("ValidateAuthorizeRequest() error = %v", err)
	}
	if decision.ClientID != "client-1" {
		t.Fatalf("ClientID = %q, want %q", decision.ClientID, "client-1")
	}
	if decision.Nonce != "nonce-1" {
		t.Fatalf("Nonce = %q, want %q", decision.Nonce, "nonce-1")
	}
}

func TestValidateAuthorizeRequestRejectsUnsupportedGrantShape(t *testing.T) {
	service := newTestService(t)

	_, err := service.ValidateAuthorizeRequest(context.Background(), AuthorizeInput{
		EntityID:            "entity-1",
		ClientID:            "client-1",
		RedirectURI:         "https://app.example.test/callback",
		ResponseType:        "token",
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
	})
	if err == nil {
		t.Fatal("ValidateAuthorizeRequest() error = nil, want error")
	}
}

func TestValidateAuthorizeRequestDelegatesToFosite(t *testing.T) {
	errBoom := errors.New("boom")
	service := newTestService(t)
	service.fosite = fakeFositeProvider{authorizeErr: errBoom}

	_, err := service.ValidateAuthorizeRequest(context.Background(), AuthorizeInput{
		EntityID:            "entity-1",
		ClientID:            "client-1",
		RedirectURI:         "https://app.example.test/callback",
		ResponseType:        "code",
		Scopes:              []string{"openid"},
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
	})
	if !errors.Is(err, errBoom) {
		t.Fatalf("ValidateAuthorizeRequest() error = %v, want %v", err, errBoom)
	}
}

func TestHasExternalIdentitySource(t *testing.T) {
	for _, tc := range []struct {
		name    string
		sources []string
		want    bool
	}{
		{name: "empty", sources: nil, want: false},
		{name: "local only", sources: []string{"local"}, want: false},
		{name: "feishu", sources: []string{"feishu"}, want: true},
		{name: "local and feishu", sources: []string{"local", "feishu"}, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasExternalIdentitySource(tc.sources); got != tc.want {
				t.Fatalf("hasExternalIdentitySource(%v) = %v, want %v", tc.sources, got, tc.want)
			}
		})
	}
}

func TestParseIdentitySourcesAcceptsDecodedJSONArray(t *testing.T) {
	got := parseIdentitySources([]interface{}{"feishu", "ldap"})
	if len(got) != 2 || got[0] != "feishu" || got[1] != "ldap" {
		t.Fatalf("parseIdentitySources decoded array = %#v", got)
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	service, err := NewService(ServiceConfig{
		Issuer:         "https://idb.example.test/",
		KeyID:          "dev-key-1",
		PrivateKey:     key,
		AuthCodeTTL:    5 * time.Minute,
		AccessTokenTTL: 15 * time.Minute,
		IDTokenTTL:     15 * time.Minute,
		Now:            func() time.Time { return time.Unix(1_800_000_000, 0).UTC() },
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

type fakeFositeProvider struct {
	authorizeErr error
}

func (f fakeFositeProvider) NewAuthorizeRequest(context.Context, *http.Request) (fosite.AuthorizeRequester, error) {
	return nil, f.authorizeErr
}

func (f fakeFositeProvider) NewAccessRequest(context.Context, *http.Request, fosite.Session) (fosite.AccessRequester, error) {
	return nil, nil
}

func (f fakeFositeProvider) WriteAuthorizeError(context.Context, http.ResponseWriter, fosite.AuthorizeRequester, error) {
}

func (f fakeFositeProvider) WriteAccessError(context.Context, http.ResponseWriter, fosite.AccessRequester, error) {
}
