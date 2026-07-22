// SPDX-License-Identifier: MIT

package audit

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/smices/open-idb/internal/auth"
)

// AuditService is the subset of audit.Service methods used by the handler.
type AuditService interface {
	List(ctx context.Context, entityID string, opts ListOptions) (ListResult, error)
}

// Handler serves audit log HTTP endpoints.
type Handler struct {
	service AuditService
}

// NewHandler creates an audit Handler backed by the given service.
func NewHandler(service AuditService) Handler {
	return Handler{service: service}
}

// RegisterRoutes mounts audit log routes on the given router.
func (h Handler) RegisterRoutes(r chi.Router) {
	r.Get("/sapi/audit-logs", h.listAuditLogs)
}

func (h Handler) listAuditLogs(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	opts := ListOptions{
		Action:       r.URL.Query().Get("action"),
		ResourceType: r.URL.Query().Get("resource_type"),
		ActorType:    r.URL.Query().Get("actor_type"),
		Limit:        parseIntQuery(r, "limit", 50),
		Offset:       parseIntQuery(r, "offset", 0),
	}

	result, err := h.service.List(r.Context(), session.EntityID, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "audit_list_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":  result.Items,
		"total":  result.Total,
		"limit":  opts.Limit,
		"offset": opts.Offset,
	})
}

// ---------------------------------------------------------------------------
// Package-local HTTP helpers (copied from adminapi to keep audit isolated)
// ---------------------------------------------------------------------------

func readSession(w http.ResponseWriter, r *http.Request) (auth.Session, bool) {
	cookie, err := r.Cookie("idb_admin_session")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "admin_session_required", "idb_admin_session cookie is required")
		return auth.Session{}, false
	}
	adminSession, err := auth.ResolveAdminSession(r.Context(), cookie.Value)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_admin_session", "idb_admin_session cookie is invalid")
		return auth.Session{}, false
	}
	return auth.Session{
		ID:          adminSession.ID,
		UserID:      adminSession.AdminID,
		EntityID:    adminSession.EntityID,
		Username:    adminSession.Username,
		DisplayName: adminSession.DisplayName,
		ExpiresAt:   adminSession.ExpiresAt,
	}, true
}

func parseIntQuery(r *http.Request, key string, fallback int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 {
		return fallback
	}
	return v
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, map[string]string{
		"error":             code,
		"error_description": message,
	})
}
