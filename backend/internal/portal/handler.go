// SPDX-License-Identifier: MIT

// Package portal provides user-facing Portal APIs. It intentionally does not
// reuse admin handlers or models, so its response contract remains a safe
// application catalogue rather than an administration document.
package portal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/smices/open-idb/internal/auth"
	"github.com/smices/open-idb/internal/db/generated"
)

type applicationStore interface {
	ListPortalApplications(context.Context, string) ([]generated.ListPortalApplicationsRow, error)
}

// Application is the deliberately limited public catalogue representation.
// It contains no configuration, credential, role, permission, or access data.
type Application struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	LogoURL     string `json:"logo_url,omitempty"`
	EntryURL    string `json:"entry_url,omitempty"`
}

type applicationService interface {
	ListApplications(context.Context, string) ([]Application, error)
}

type service struct {
	store applicationStore
}

func (s service) ListApplications(ctx context.Context, entityID string) ([]Application, error) {
	rows, err := s.store.ListPortalApplications(ctx, entityID)
	if err != nil {
		return nil, err
	}
	applications := make([]Application, 0, len(rows))
	for _, row := range rows {
		applications = append(applications, Application{
			ID:          row.ID,
			Name:        row.Name,
			Type:        row.Type,
			Description: catalogueText(row.Description),
			LogoURL:     catalogueText(row.LogoUrl),
			EntryURL:    portalEntryURL(catalogueText(row.EntryUrl)),
		})
	}
	return applications, nil
}

// portalEntryURL only exposes absolute web URLs. Application configuration is
// admin-controlled data, but it must not become an executable URL in a user
// Portal response when malformed or using a non-web scheme.
func portalEntryURL(value string) string {
	value = strings.TrimSpace(value)
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return ""
	}
	return value
}

func catalogueText(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		return ""
	}
}

// Handler serves authenticated user Portal routes.
type Handler struct {
	applications applicationService
}

// NewHandler constructs a Portal handler backed by the generated query store.
func NewHandler(store applicationStore) Handler {
	return newHandler(service{store: store})
}

func newHandler(applications applicationService) Handler {
	return Handler{applications: applications}
}

func (h Handler) RegisterRoutes(r chi.Router) {
	r.Get("/api/portal/applications", h.listApplications)
}

func (h Handler) listApplications(w http.ResponseWriter, r *http.Request) {
	session, ok := readUserSession(w, r)
	if !ok {
		return
	}
	applications, err := h.applications.ListApplications(r.Context(), session.EntityID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "portal_application_list_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, struct {
		Applications []Application `json:"applications"`
	}{Applications: applications})
}

func readUserSession(w http.ResponseWriter, r *http.Request) (auth.Session, bool) {
	cookie, err := r.Cookie("idb_session")
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		writeError(w, http.StatusUnauthorized, "session_required", "idb_session cookie is required")
		return auth.Session{}, false
	}
	session, err := auth.ResolveSession(r.Context(), cookie.Value)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_session", "idb_session cookie is invalid")
		return auth.Session{}, false
	}
	return session, true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{
		"error":             code,
		"error_description": message,
	})
}
