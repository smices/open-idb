// SPDX-License-Identifier: MIT

package sso

import (
	"reflect"
	"testing"
)

func TestBuildDiscoveryDocument(t *testing.T) {
	doc := BuildDiscoveryDocument("https://idb.example.test")

	if doc.Issuer != "https://idb.example.test" {
		t.Fatalf("Issuer = %q, want %q", doc.Issuer, "https://idb.example.test")
	}
	if doc.AuthorizationEndpoint != "https://idb.example.test/oauth2/authorize" {
		t.Fatalf("AuthorizationEndpoint = %q", doc.AuthorizationEndpoint)
	}
	if doc.TokenEndpoint != "https://idb.example.test/oauth2/token" {
		t.Fatalf("TokenEndpoint = %q", doc.TokenEndpoint)
	}
	if doc.UserinfoEndpoint != "https://idb.example.test/oauth2/userinfo" {
		t.Fatalf("UserinfoEndpoint = %q", doc.UserinfoEndpoint)
	}
	if doc.JWKSURI != "https://idb.example.test/.well-known/jwks.json" {
		t.Fatalf("JWKSURI = %q", doc.JWKSURI)
	}
	if !reflect.DeepEqual(doc.ResponseTypesSupported, []string{"code"}) {
		t.Fatalf("ResponseTypesSupported = %#v", doc.ResponseTypesSupported)
	}
	if !reflect.DeepEqual(doc.SubjectTypesSupported, []string{"public"}) {
		t.Fatalf("SubjectTypesSupported = %#v", doc.SubjectTypesSupported)
	}
	if !reflect.DeepEqual(doc.IDTokenSigningAlgValuesSupported, []string{"RS256"}) {
		t.Fatalf("IDTokenSigningAlgValuesSupported = %#v", doc.IDTokenSigningAlgValuesSupported)
	}
	if !reflect.DeepEqual(doc.ScopesSupported, []string{"openid", "profile", "email", "directory:read"}) {
		t.Fatalf("ScopesSupported = %#v", doc.ScopesSupported)
	}
	if !reflect.DeepEqual(doc.TokenEndpointAuthMethodsSupported, []string{"client_secret_basic", "client_secret_post", "none"}) {
		t.Fatalf("TokenEndpointAuthMethodsSupported = %#v", doc.TokenEndpointAuthMethodsSupported)
	}
	if !reflect.DeepEqual(doc.CodeChallengeMethodsSupported, []string{"S256"}) {
		t.Fatalf("CodeChallengeMethodsSupported = %#v", doc.CodeChallengeMethodsSupported)
	}
}

func TestBuildDiscoveryDocumentTrimsIssuerSlash(t *testing.T) {
	doc := BuildDiscoveryDocument("https://idb.example.test/")

	if doc.Issuer != "https://idb.example.test" {
		t.Fatalf("Issuer = %q, want %q", doc.Issuer, "https://idb.example.test")
	}
	if doc.AuthorizationEndpoint != "https://idb.example.test/oauth2/authorize" {
		t.Fatalf("AuthorizationEndpoint = %q", doc.AuthorizationEndpoint)
	}
}

func TestBuildDiscoveryDocumentWithEndpointPrefix(t *testing.T) {
	doc := BuildDiscoveryDocumentWithEndpointPrefix("https://idb.example.test/", "/api/")

	if doc.Issuer != "https://idb.example.test" {
		t.Fatalf("Issuer = %q, want %q", doc.Issuer, "https://idb.example.test")
	}
	if doc.AuthorizationEndpoint != "https://idb.example.test/api/oauth2/authorize" {
		t.Fatalf("AuthorizationEndpoint = %q", doc.AuthorizationEndpoint)
	}
	if doc.TokenEndpoint != "https://idb.example.test/api/oauth2/token" {
		t.Fatalf("TokenEndpoint = %q", doc.TokenEndpoint)
	}
	if doc.UserinfoEndpoint != "https://idb.example.test/api/oauth2/userinfo" {
		t.Fatalf("UserinfoEndpoint = %q", doc.UserinfoEndpoint)
	}
	if doc.JWKSURI != "https://idb.example.test/api/.well-known/jwks.json" {
		t.Fatalf("JWKSURI = %q", doc.JWKSURI)
	}
}
