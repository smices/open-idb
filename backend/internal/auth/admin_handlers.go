// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/smices/open-idb/internal/ephemeral"
)

type AdminLoginResult struct {
	AdminID     string `json:"id"`
	EntityID    string `json:"entity_id,omitempty"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

type AdminCurrentUser struct {
	ID          string `json:"id"`
	EntityID    string `json:"entity_id,omitempty"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

type AdminAuthService interface {
	AuthenticateAdmin(ctx context.Context, username string, password string) (AdminLoginResult, error)
	CreateAdminSession(ctx context.Context, result AdminLoginResult, meta SessionMetadata) (AdminSession, error)
	CurrentAdmin(ctx context.Context, session AdminSession) (AdminCurrentUser, error)
	UpdateAdminProfile(ctx context.Context, session AdminSession, displayName string) (AdminCurrentUser, error)
	UpdateAdminPassword(ctx context.Context, session AdminSession, currentPassword string, newPassword string) error
}

type AdminHandler struct {
	service    AdminAuthService
	sessionTTL time.Duration
	ephemeral  ephemeral.Store
}

func NewAdminHandler(service AdminAuthService) AdminHandler {
	return AdminHandler{service: service, sessionTTL: 24 * time.Hour}
}

func (h *AdminHandler) SetSessionTTL(ttl time.Duration) {
	if ttl > 0 {
		h.sessionTTL = ttl
	}
}

func (h *AdminHandler) SetEphemeralStore(store ephemeral.Store) {
	h.ephemeral = store
}

func (h AdminHandler) RegisterRoutes(r chi.Router) {
	r.Post("/sapi/login/account", h.loginAccount)
	r.Get("/sapi/me", h.currentAdmin)
	r.Patch("/sapi/me", h.updateAdminProfile)
	r.Post("/sapi/me/password", h.updateAdminPassword)
}

func (h AdminHandler) loginAccount(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeError(w, http.StatusServiceUnavailable, "admin_auth_unavailable", "admin authentication is unavailable")
		return
	}
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_login_request", "invalid form body")
		return
	}
	account := r.PostForm.Get("account")
	password := r.PostForm.Get("password")
	limitKey := ephemeral.Key("rate:admin_login", account, r.RemoteAddr)
	limit, limitErr := ephemeral.CheckLimit(r.Context(), h.ephemeral, limitKey, 10, 15*time.Minute)
	if limitErr != nil {
		writeError(w, http.StatusServiceUnavailable, "rate_limit_unavailable", "login rate limit is unavailable")
		return
	}
	if !limit.Allowed {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "too many login attempts")
		return
	}
	result, err := h.service.AuthenticateAdmin(r.Context(), account, password)
	if err != nil {
		if acceptsHTML(r) {
			http.Redirect(w, r, "/admin/login?login_error=invalid_credentials", http.StatusSeeOther)
			return
		}
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid account or password")
		return
	}
	session, err := h.service.CreateAdminSession(r.Context(), result, SessionMetadata{
		LoginMethod: "password",
		IP:          r.RemoteAddr,
		UserAgent:   r.UserAgent(),
		TTL:         h.sessionTTL,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "session_create_failed", "could not create admin session")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "idb_admin_session",
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  session.ExpiresAt,
		MaxAge:   int(time.Until(session.ExpiresAt).Seconds()),
	})
	http.Redirect(w, r, safeReturnToWithDefault(r.PostForm.Get("return_to"), "/admin"), http.StatusFound)
}

func (h AdminHandler) currentAdmin(w http.ResponseWriter, r *http.Request) {
	session, ok := h.readAdminSession(w, r)
	if !ok {
		return
	}
	user, err := h.service.CurrentAdmin(r.Context(), session)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_admin_session", "idb_admin_session cookie is invalid")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(user)
}

func (h AdminHandler) updateAdminProfile(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeError(w, http.StatusServiceUnavailable, "admin_auth_unavailable", "admin authentication is unavailable")
		return
	}
	session, ok := h.readAdminSession(w, r)
	if !ok {
		return
	}
	var request struct {
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_profile_request", "invalid json body")
		return
	}
	user, err := h.service.UpdateAdminProfile(r.Context(), session, request.DisplayName)
	if err != nil {
		writeError(w, http.StatusBadRequest, "profile_update_failed", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(user)
}

func (h AdminHandler) updateAdminPassword(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeError(w, http.StatusServiceUnavailable, "admin_auth_unavailable", "admin authentication is unavailable")
		return
	}
	session, ok := h.readAdminSession(w, r)
	if !ok {
		return
	}
	var request struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_password_request", "invalid json body")
		return
	}
	if isWeakAdminPassword(request.NewPassword) {
		writeError(w, http.StatusBadRequest, "weak_password", "new password does not meet minimum strength requirements")
		return
	}
	if err := h.service.UpdateAdminPassword(r.Context(), session, request.CurrentPassword, request.NewPassword); err != nil {
		writeError(w, http.StatusUnauthorized, "password_update_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h AdminHandler) readAdminSession(w http.ResponseWriter, r *http.Request) (AdminSession, bool) {
	session, ok := AdminSessionFromContext(r.Context())
	if ok {
		return session, true
	}
	cookie, err := r.Cookie("idb_admin_session")
	if err != nil || cookie.Value == "" {
		writeError(w, http.StatusUnauthorized, "admin_session_required", "idb_admin_session cookie is required")
		return AdminSession{}, false
	}
	resolved, err := ResolveAdminSession(r.Context(), cookie.Value)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_admin_session", "idb_admin_session cookie is invalid")
		return AdminSession{}, false
	}
	return resolved, true
}

func isWeakAdminPassword(password string) bool {
	if len(password) < 12 {
		return true
	}
	hasLower := strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz")
	hasUpper := strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	hasDigit := strings.ContainsAny(password, "0123456789")
	hasSymbol := strings.ContainsAny(password, "!@#$%^&*()-_=+[]{};:,.<>/?")
	return !(hasLower && hasUpper && hasDigit && hasSymbol)
}

func safeReturnToWithDefault(value string, fallback string) string {
	resolved := safeReturnTo(value)
	if value == "" && fallback != "" {
		return fallback
	}
	return resolved
}
