// SPDX-License-Identifier: MIT

package sso

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/smices/open-idb/internal/id"
)

func TestDiscoveryHandler(t *testing.T) {
	router := newHandlerTestRouter(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var doc DiscoveryDocument
	if err := json.NewDecoder(rec.Body).Decode(&doc); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if doc.Issuer != "https://idb.example.test" {
		t.Fatalf("Issuer = %q", doc.Issuer)
	}
}

func TestAPIDiscoveryHandlerUsesAPIEndpoints(t *testing.T) {
	router := newHandlerTestRouter(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/.well-known/openid-configuration", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var doc DiscoveryDocument
	if err := json.NewDecoder(rec.Body).Decode(&doc); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if doc.Issuer != "https://idb.example.test" {
		t.Fatalf("Issuer = %q", doc.Issuer)
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
}

func TestJWKSHandler(t *testing.T) {
	router := newHandlerTestRouter(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/.well-known/jwks.json", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var jwks JWKS
	if err := json.NewDecoder(rec.Body).Decode(&jwks); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(jwks.Keys) != 1 || jwks.Keys[0].KeyID != "dev-key-1" {
		t.Fatalf("JWKS = %#v", jwks)
	}
}

func TestAPIJWKSHandler(t *testing.T) {
	router := newHandlerTestRouter(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/.well-known/jwks.json", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var jwks JWKS
	if err := json.NewDecoder(rec.Body).Decode(&jwks); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(jwks.Keys) != 1 || jwks.Keys[0].KeyID != "dev-key-1" {
		t.Fatalf("JWKS = %#v", jwks)
	}
}

func TestAuthorizeRedirectsToLoginWhenUnauthenticated(t *testing.T) {
	router := newHandlerTestRouter(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oauth2/authorize?client_id=test&redirect_uri=http://example.com/callback", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "/login") {
		t.Fatalf("Location = %q, want redirect to /login", location)
	}
	if !strings.Contains(location, "return_to") {
		t.Fatalf("Location = %q, want return_to parameter", location)
	}
}

func TestAPIAuthorizeRedirectsToLoginWhenUnauthenticated(t *testing.T) {
	router := newHandlerTestRouter(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/oauth2/authorize?client_id=test&redirect_uri=http://example.com/callback", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "/login") {
		t.Fatalf("Location = %q, want redirect to /login", location)
	}
	if !strings.Contains(location, "return_to") {
		t.Fatalf("Location = %q, want return_to parameter", location)
	}
}

func TestShouldRestartAuthorizeLogin(t *testing.T) {
	if !shouldRestartAuthorizeLogin(fmt.Errorf("%w: status is disabled", ErrUserInactive)) {
		t.Fatal("inactive user should restart login")
	}
	if !shouldRestartAuthorizeLogin(ErrUserNotEligibleForApplicationSSO) {
		t.Fatal("ineligible SSO user should restart login")
	}
	if shouldRestartAuthorizeLogin(fmt.Errorf("invalid redirect uri")) {
		t.Fatal("invalid authorize request should not restart login")
	}
}

func TestUnsupportedEndpoints(t *testing.T) {
	router := newHandlerTestRouter(t)
	for _, tc := range []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{name: "userinfo without bearer token", method: http.MethodGet, path: "/oauth2/userinfo", wantStatus: http.StatusUnauthorized},
		{name: "revoke with no body returns 200", method: http.MethodPost, path: "/oauth2/revoke", wantStatus: http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, nil)

			router.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body = %q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestUserInfoWithValidToken(t *testing.T) {
	entityID := id.NewULID()
	userID := id.NewULID()
	rawToken := "test-access-token-xyz"
	tokenHash := HashToken(rawToken)

	querier := &mockSSOQuerier{
		token: SSOTokenLookup{
			ID:        id.NewULID(),
			EntityID:  entityID,
			UserID:    userID,
			ClientID:  "client-1",
			TokenType: "access",
			Scopes:    []string{"openid", "profile", "email"},
		},
		user: UserInfoClaims{
			ID:          userID,
			EntityID:    entityID,
			Username:    "alice",
			DisplayName: "Alice Smith",
			Email:       "alice@example.com",
			Phone:       "+15555550100",
			AvatarURL:   "https://example.com/alice.png",
			Locale:      "en-US",
		},
	}
	router := newHandlerTestRouterWithQuerier(t, querier)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oauth2/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)
	req.Header.Set("X-IDB-Entity-ID", entityID)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body = %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	if querier.lastLookupHash != tokenHash {
		t.Fatalf("lookup hash = %q, want %q", querier.lastLookupHash, tokenHash)
	}

	var claims map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&claims); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if claims["sub"] != userID {
		t.Fatalf("sub = %v, want %q", claims["sub"], userID)
	}
	if claims["name"] != "Alice Smith" {
		t.Fatalf("name = %v, want %q", claims["name"], "Alice Smith")
	}
	if claims["email"] != "alice@example.com" {
		t.Fatalf("email = %v, want %q", claims["email"], "alice@example.com")
	}
	if claims["preferred_username"] != "alice" {
		t.Fatalf("preferred_username = %v, want %q", claims["preferred_username"], "alice")
	}
	// phone is not in scopes (no "phone" scope), so should be absent.
	if _, ok := claims["phone_number"]; ok {
		t.Fatalf("phone_number should be absent when phone scope not granted, got %v", claims["phone_number"])
	}
}

func TestUserInfoRejectsRevokedToken(t *testing.T) {
	entityID := id.NewULID()
	now := time.Now()
	querier := &mockSSOQuerier{
		token: SSOTokenLookup{
			ID:        id.NewULID(),
			EntityID:  entityID,
			UserID:    id.NewULID(),
			TokenType: "access",
			Scopes:    []string{"openid"},
			RevokedAt: &now,
		},
	}
	router := newHandlerTestRouterWithQuerier(t, querier)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oauth2/userinfo", nil)
	req.Header.Set("Authorization", "Bearer revoked-token")
	req.Header.Set("X-IDB-Entity-ID", entityID)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestUserInfoRejectsNonAccessToken(t *testing.T) {
	entityID := id.NewULID()
	querier := &mockSSOQuerier{
		token: SSOTokenLookup{
			ID:        id.NewULID(),
			EntityID:  entityID,
			UserID:    id.NewULID(),
			TokenType: "id",
			Scopes:    []string{"openid"},
		},
	}
	router := newHandlerTestRouterWithQuerier(t, querier)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oauth2/userinfo", nil)
	req.Header.Set("Authorization", "Bearer id-token-value")
	req.Header.Set("X-IDB-Entity-ID", entityID)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRevokeCallsQuerier(t *testing.T) {
	entityID := id.NewULID()
	rawToken := "token-to-revoke"
	tokenHash := HashToken(rawToken)

	querier := &mockSSOQuerier{}
	router := newHandlerTestRouterWithQuerier(t, querier)

	body := url.Values{"token": {rawToken}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth2/revoke", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-IDB-Entity-ID", entityID)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if querier.lastRevokeHash != tokenHash {
		t.Fatalf("revoke hash = %q, want %q", querier.lastRevokeHash, tokenHash)
	}
}

func TestRevokeReturns200ForUnknownToken(t *testing.T) {
	entityID := id.NewULID()
	querier := &mockSSOQuerier{revokeErr: fmt.Errorf("no rows")}
	router := newHandlerTestRouterWithQuerier(t, querier)

	body := url.Values{"token": {"unknown-token"}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth2/revoke", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-IDB-Entity-ID", entityID)

	router.ServeHTTP(rec, req)

	// RFC 7009: always 200, even if token not found.
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestBuildUserInfoResponseScopes(t *testing.T) {
	user := UserInfoClaims{
		ID:          id.NewULID(),
		EntityID:    id.NewULID(),
		Username:    "alice",
		DisplayName: "Alice Smith",
		Email:       "alice@example.com",
		Phone:       "+15555550100",
		AvatarURL:   "https://example.com/alice.png",
		Locale:      "en-US",
	}

	t.Run("openid only returns sub and entity_id", func(t *testing.T) {
		claims := buildUserInfoResponse(user, []string{"openid"})
		if _, ok := claims["name"]; ok {
			t.Fatal("name should not be present without profile scope")
		}
		if _, ok := claims["email"]; ok {
			t.Fatal("email should not be present without email scope")
		}
		if claims["sub"] != user.ID {
			t.Fatalf("sub = %v, want %q", claims["sub"], user.ID)
		}
	})

	t.Run("profile scope includes profile claims", func(t *testing.T) {
		claims := buildUserInfoResponse(user, []string{"openid", "profile"})
		if claims["name"] != "Alice Smith" {
			t.Fatalf("name = %v", claims["name"])
		}
		if claims["locale"] != "en-US" {
			t.Fatalf("locale = %v", claims["locale"])
		}
		if _, ok := claims["email"]; ok {
			t.Fatal("email should not be present without email scope")
		}
	})

	t.Run("email scope includes email", func(t *testing.T) {
		claims := buildUserInfoResponse(user, []string{"openid", "email"})
		if claims["email"] != "alice@example.com" {
			t.Fatalf("email = %v", claims["email"])
		}
		if _, ok := claims["name"]; ok {
			t.Fatal("name should not be present without profile scope")
		}
	})

	t.Run("phone scope includes phone_number", func(t *testing.T) {
		claims := buildUserInfoResponse(user, []string{"openid", "phone"})
		if claims["phone_number"] != "+15555550100" {
			t.Fatalf("phone_number = %v", claims["phone_number"])
		}
	})
}

// mockSSOQuerier is a test double for the ssoQuerier interface.
type mockSSOQuerier struct {
	token          SSOTokenLookup
	tokenErr       error
	user           UserInfoClaims
	userErr        error
	revokeErr      error
	lastLookupHash string
	lastRevokeHash string
}

func (m *mockSSOQuerier) LookupToken(_ context.Context, _ string, tokenHash string) (SSOTokenLookup, error) {
	m.lastLookupHash = tokenHash
	return m.token, m.tokenErr
}

func (m *mockSSOQuerier) MarkTokenRevoked(_ context.Context, _ string, tokenHash string) error {
	m.lastRevokeHash = tokenHash
	return m.revokeErr
}

func (m *mockSSOQuerier) FetchUserInfo(_ context.Context, _ string, _ string) (UserInfoClaims, error) {
	return m.user, m.userErr
}

func newHandlerTestRouter(t *testing.T) http.Handler {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	service, err := NewService(ServiceConfig{
		Issuer:     "https://idb.example.test",
		KeyID:      "dev-key-1",
		PrivateKey: key,
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	handler := NewHandler(service)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux := chi.NewRouter()
		handler.RegisterRoutes(mux)
		mux.ServeHTTP(w, r)
	})
}

func newHandlerTestRouterWithQuerier(t *testing.T, querier ssoQuerier) http.Handler {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	service, err := NewService(ServiceConfig{
		Issuer:     "https://idb.example.test",
		KeyID:      "dev-key-1",
		PrivateKey: key,
		Querier:    querier,
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	handler := NewHandler(service)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux := chi.NewRouter()
		handler.RegisterRoutes(mux)
		mux.ServeHTTP(w, r)
	})
}
