// SPDX-License-Identifier: MIT

package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/smices/open-idb/internal/auth"
	"github.com/smices/open-idb/internal/i18n"
)

type Option func(chi.Router)

func NewRouter(options ...Option) http.Handler {
	r := chi.NewRouter()

	// i18n middleware for locale extraction from Accept-Language header
	catalog := i18n.NewCatalog()
	r.Use(i18n.Middleware(catalog))
	r.Use(RequireAuthenticatedRequest)

	r.Get("/", FrontendRouteHandler)
	r.Get("/login", FrontendRouteHandler)
	r.Get("/t/{entity}/admin/login", FrontendRouteHandler)
	r.Get("/auth/continue", FrontendRouteHandler)
	r.Get("/dashboard", FrontendRouteHandler)
	r.Get("/healthz", HealthHandler)
	r.Get("/readyz", HealthHandler)
	for _, option := range options {
		option(r)
	}
	return r
}

func RequireAuthenticatedRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicRoute(r.Method, r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		cookie, err := r.Cookie("idb_session")
		if err != nil || strings.TrimSpace(cookie.Value) == "" {
			writeJSONError(w, http.StatusUnauthorized, "session_required", "idb_session cookie is required")
			return
		}
		if _, err := auth.DecodeSession(cookie.Value); err != nil {
			writeJSONError(w, http.StatusUnauthorized, "invalid_session", "idb_session cookie is invalid")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isPublicRoute(method string, path string) bool {
	switch path {
	case "/healthz", "/readyz":
		return method == http.MethodGet
	case "/api/admin/v1/auth/context", "/api/admin/v1/auth/providers":
		return method == http.MethodGet
	case "/api/login/account", "/auth/feishu/exchange", "/api/auth/feishu/exchange", "/login/legacy", "/api/admin/v1/login/legacy", "/admin/v1/login/legacy":
		return method == http.MethodPost
	case "/auth/feishu/login", "/auth/feishu/callback", "/api/auth/feishu/login", "/api/auth/feishu/callback":
		return method == http.MethodGet
	case "/.well-known/openid-configuration", "/.well-known/jwks.json", "/oauth2/authorize", "/oauth2/userinfo":
		return method == http.MethodGet
	case "/oauth2/token", "/oauth2/revoke":
		return method == http.MethodPost
	case "/api/.well-known/openid-configuration", "/api/.well-known/jwks.json", "/api/oauth2/authorize", "/api/oauth2/userinfo":
		return method == http.MethodGet
	case "/api/oauth2/token", "/api/oauth2/revoke":
		return method == http.MethodPost
	case "/api/webhooks/feishu":
		return method == http.MethodPost
	}
	return method == http.MethodPost && strings.HasPrefix(path, "/api/webhooks/feishu/")
}

func FrontendRouteHandler(w http.ResponseWriter, r *http.Request) {
	writeJSONError(w, http.StatusNotFound, "not_found", "The requested resource was not found.")
}

func writeJSONError(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":             code,
		"error_description": message,
	})
}
