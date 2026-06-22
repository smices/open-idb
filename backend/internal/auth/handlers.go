// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	auditmodel "github.com/smices/open-idb/internal/audit/model"
	"github.com/smices/open-idb/internal/ephemeral"
	"github.com/smices/open-idb/internal/id"
)

// AuditEventWriter records audit events. A nil value disables audit logging.
// *audit.Service satisfies this interface.
type AuditEventWriter interface {
	Write(ctx context.Context, event auditmodel.Event) error
}

type LoginService interface {
	AuthenticateLocal(ctx context.Context, username string, password string) (LoginResult, error)
	AuthenticateLocalWithEntity(ctx context.Context, entityID string, username string, password string) (LoginResult, error)
	CreateLoginSession(ctx context.Context, result LoginResult, meta SessionMetadata) (Session, error)
}

type Handler struct {
	service    LoginService
	audit      AuditEventWriter
	sessionTTL time.Duration
	ephemeral  ephemeral.Store
}

// NewHandler creates an auth Handler. An optional AuditEventWriter may be
// provided to enable audit logging of login events. Existing callers that
// do not pass a writer are unaffected (audit logging is silently disabled).
func NewHandler(service LoginService, writers ...AuditEventWriter) Handler {
	h := Handler{service: service, sessionTTL: 24 * time.Hour}
	if len(writers) > 0 {
		h.audit = writers[0]
	}
	return h
}

func (h *Handler) SetSessionTTL(ttl time.Duration) {
	if ttl > 0 {
		h.sessionTTL = ttl
	}
}

func (h *Handler) SetEphemeralStore(store ephemeral.Store) {
	h.ephemeral = store
}

func (h Handler) RegisterRoutes(r chi.Router) {
	r.Get("/api/admin/v1/auth/context", h.loginContext)
	r.Post("/api/login/account", h.loginAccount)
}

func (h Handler) loginAccount(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_login_request", "invalid form body")
		return
	}
	traceID := id.NewULID()
	account := r.PostForm.Get("account")
	password := r.PostForm.Get("password")
	entityID := r.PostForm.Get("entity_id")
	limitKey := ephemeral.Key("rate:login", entityID, account, r.RemoteAddr)
	limit, limitErr := ephemeral.CheckLimit(r.Context(), h.ephemeral, limitKey, 10, 15*time.Minute)
	if limitErr != nil {
		writeError(w, http.StatusServiceUnavailable, "rate_limit_unavailable", "login rate limit is unavailable")
		return
	}
	if !limit.Allowed {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "too many login attempts")
		return
	}
	var result LoginResult
	var err error
	if entityID != "" {
		result, err = h.service.AuthenticateLocalWithEntity(r.Context(), entityID, account, password)
	} else {
		result, err = h.service.AuthenticateLocal(r.Context(), account, password)
	}
	if err != nil {
		h.writeAudit(r, auditmodel.Event{
			Action:    auditmodel.ActionLoginFailed,
			ActorType: "user",
			IP:        r.RemoteAddr,
			UserAgent: r.UserAgent(),
			TraceID:   traceID,
			After:     map[string]string{"login_method": "account", "username": account, "reason": err.Error()},
		})
		if acceptsHTML(r) {
			http.Redirect(w, r, "/?login_error=invalid_credentials", http.StatusSeeOther)
			return
		}
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid account or password")
		return
	}
	session, err := h.service.CreateLoginSession(r.Context(), result, SessionMetadata{
		LoginMethod: "password",
		IP:          r.RemoteAddr,
		UserAgent:   r.UserAgent(),
		TTL:         h.sessionTTL,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "session_create_failed", "could not create login session")
		return
	}
	h.writeAudit(r, auditmodel.Event{
		EntityID:     result.EntityID,
		ActorUserID:  result.UserID,
		ActorType:    "user",
		Action:       auditmodel.ActionLoginSuccess,
		ResourceType: "user",
		ResourceID:   result.UserID,
		IP:           r.RemoteAddr,
		UserAgent:    r.UserAgent(),
		TraceID:      traceID,
		After:        map[string]string{"login_method": "account", "username": result.Username},
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "idb_session",
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  session.ExpiresAt,
		MaxAge:   int(time.Until(session.ExpiresAt).Seconds()),
	})
	http.Redirect(w, r, safeReturnTo(r.PostForm.Get("return_to")), http.StatusFound)
}

// writeAudit records an audit event if an audit writer is configured.
// Errors writing the audit log are silently ignored so they do not
// disrupt the primary request flow.
func (h Handler) writeAudit(r *http.Request, event auditmodel.Event) {
	if h.audit == nil {
		return
	}
	_ = h.audit.Write(r.Context(), event)
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":             code,
		"error_description": message,
	})
}

func acceptsHTML(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "text/html")
}

func safeReturnTo(value string) string {
	if value == "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return "/dashboard"
	}
	u, err := url.Parse(value)
	if err != nil || u.IsAbs() || u.Host != "" {
		return "/dashboard"
	}
	return u.RequestURI()
}
