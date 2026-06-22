// SPDX-License-Identifier: MIT

package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/smices/open-idb/internal/auth"
)

func TestHealthRoutesReturnOK(t *testing.T) {
	for _, path := range []string{"/healthz", "/readyz"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()

			NewRouter().ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if contentType := rec.Header().Get("Content-Type"); contentType != "application/json" {
				t.Fatalf("Content-Type = %q, want %q", contentType, "application/json")
			}

			var response HealthResponse
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response.Status != "ok" {
				t.Fatalf("status body = %q, want %q", response.Status, "ok")
			}
		})
	}
}

func TestFrontendRoutesDoNotServeBackendUI(t *testing.T) {
	for _, path := range []string{"/", "/login", "/t/configured_entity/admin/login", "/auth/continue", "/dashboard"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			addSessionCookie(t, req)
			rec := httptest.NewRecorder()

			NewRouter().ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
			}
			assertJSONError(t, rec, "not_found")
		})
	}
}

func TestAnonymousFrontendRoutesRequireSession(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()

	NewRouter().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	assertJSONError(t, rec, "session_required")
}

func TestAnonymousAdminAPIsRequireSession(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/dashboard/summary", nil)
	rec := httptest.NewRecorder()

	NewRouter(func(r chi.Router) {
		r.Get("/api/admin/v1/dashboard/summary", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
	}).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	assertJSONError(t, rec, "session_required")
}

func TestAuthenticatedAdminAPIsReachHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/dashboard/summary", nil)
	addSessionCookie(t, req)
	rec := httptest.NewRecorder()

	NewRouter(func(r chi.Router) {
		r.Get("/api/admin/v1/dashboard/summary", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
	}).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestPublicAuthAndProtocolRoutesRemainReachable(t *testing.T) {
	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/admin/v1/auth/context"},
		{http.MethodGet, "/api/admin/v1/auth/providers"},
		{http.MethodPost, "/api/login/account"},
		{http.MethodGet, "/auth/feishu/login"},
		{http.MethodGet, "/auth/feishu/callback"},
		{http.MethodPost, "/auth/feishu/exchange"},
		{http.MethodGet, "/api/auth/feishu/login"},
		{http.MethodGet, "/api/auth/feishu/callback"},
		{http.MethodPost, "/api/auth/feishu/exchange"},
		{http.MethodPost, "/login/legacy"},
		{http.MethodPost, "/api/admin/v1/login/legacy"},
		{http.MethodPost, "/admin/v1/login/legacy"},
		{http.MethodGet, "/.well-known/openid-configuration"},
		{http.MethodGet, "/.well-known/jwks.json"},
		{http.MethodGet, "/oauth2/authorize"},
		{http.MethodPost, "/oauth2/token"},
		{http.MethodGet, "/oauth2/userinfo"},
		{http.MethodPost, "/oauth2/revoke"},
		{http.MethodGet, "/api/.well-known/openid-configuration"},
		{http.MethodGet, "/api/.well-known/jwks.json"},
		{http.MethodGet, "/api/oauth2/authorize"},
		{http.MethodPost, "/api/oauth2/token"},
		{http.MethodGet, "/api/oauth2/userinfo"},
		{http.MethodPost, "/api/oauth2/revoke"},
		{http.MethodPost, "/api/webhooks/feishu"},
		{http.MethodPost, "/api/webhooks/feishu/entity/source"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()

			NewRouter(func(r chi.Router) {
				r.HandleFunc(tc.path, func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				})
			}).ServeHTTP(rec, req)

			if rec.Code != http.StatusNoContent {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
			}
		})
	}
}

func addSessionCookie(t *testing.T, req *http.Request) {
	t.Helper()
	session, err := auth.EncodeSession(auth.Session{
		UserID:      "01KT1698M30H8R3BQ4F0DBY0BF",
		EntityID:    "01KT1698M2BRVAAX31FM80J9JV",
		Username:    "admin",
		DisplayName: "Administrator",
	})
	if err != nil {
		t.Fatalf("EncodeSession: %v", err)
	}
	req.AddCookie(&http.Cookie{Name: "idb_session", Value: session})
}

func assertJSONError(t *testing.T, rec *httptest.ResponseRecorder, code string) {
	t.Helper()
	if contentType := rec.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("Content-Type = %q, want %q", contentType, "application/json")
	}
	var response map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["error"] != code {
		t.Fatalf("error = %q, want %q", response["error"], code)
	}
}
