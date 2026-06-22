// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// IdentitySourceResponse represents an identity source in API responses.
type IdentitySourceResponse struct {
	ID          string    `json:"id"`
	EntityID    string    `json:"entity_id"`
	Type        string    `json:"type"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	SyncEnabled bool      `json:"sync_enabled"`
	CreatedAt   time.Time `json:"created_at"`
}

// identitySourceService defines the data-access contract for identity
// source CRUD operations. *AdminService satisfies this interface once
// the corresponding methods are added.
type identitySourceService interface {
	ListIdentitySources(ctx context.Context, entityID string, limit, offset int32) ([]IdentitySourceResponse, error)
	CountIdentitySources(ctx context.Context, entityID string) (int64, error)
	GetIdentitySourceByID(ctx context.Context, entityID, id string) (IdentitySourceResponse, error)
	CreateIdentitySource(ctx context.Context, entityID string, sourceType, name string, syncEnabled bool) (IdentitySourceResponse, error)
	UpdateIdentitySource(ctx context.Context, entityID, id string, name, status pgtype.Text, syncEnabled pgtype.Bool) (IdentitySourceResponse, error)
	DeleteIdentitySource(ctx context.Context, entityID, id string) error
}

// IdentitySourceHandler handles identity source CRUD endpoints.
type IdentitySourceHandler struct {
	service identitySourceService
}

func NewIdentitySourceHandler(service identitySourceService) IdentitySourceHandler {
	return IdentitySourceHandler{service: service}
}

func (h IdentitySourceHandler) RegisterRoutes(r chi.Router) {
	r.Get("/admin/v1/identity-sources", h.listIdentitySources)
	r.Get("/api/admin/v1/identity-sources", h.listIdentitySources)
	r.Get("/admin/v1/identity-sources/{id}", h.getIdentitySource)
	r.Get("/api/admin/v1/identity-sources/{id}", h.getIdentitySource)
	r.Post("/admin/v1/identity-sources", h.createIdentitySource)
	r.Post("/api/admin/v1/identity-sources", h.createIdentitySource)
	r.Put("/admin/v1/identity-sources/{id}", h.updateIdentitySource)
	r.Put("/api/admin/v1/identity-sources/{id}", h.updateIdentitySource)
	r.Delete("/admin/v1/identity-sources/{id}", h.deleteIdentitySource)
	r.Delete("/api/admin/v1/identity-sources/{id}", h.deleteIdentitySource)
}

func (h IdentitySourceHandler) listIdentitySources(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	limit, offset := parsePagination(r)

	sources, err := h.service.ListIdentitySources(r.Context(), entityID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "identity_source_list_failed", err.Error())
		return
	}
	total, err := h.service.CountIdentitySources(r.Context(), entityID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "identity_source_count_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, PagedResponse{
		Items:  sources,
		Total:  total,
		Limit:  int(limit),
		Offset: int(offset),
	})
}

func (h IdentitySourceHandler) getIdentitySource(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	id, err := ulidValue(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_identity_source_id", err.Error())
		return
	}
	source, err := h.service.GetIdentitySourceByID(r.Context(), entityID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "identity_source_not_found", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, source)
}

func (h IdentitySourceHandler) createIdentitySource(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	var body struct {
		Type        string `json:"type"`
		Name        string `json:"name"`
		SyncEnabled bool   `json:"sync_enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}
	if body.Name == "" || body.Type == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "name and type are required")
		return
	}
	source, err := h.service.CreateIdentitySource(r.Context(), entityID, body.Type, body.Name, body.SyncEnabled)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "identity_source_create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, source)
}

func (h IdentitySourceHandler) updateIdentitySource(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	id, err := ulidValue(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_identity_source_id", err.Error())
		return
	}
	var body struct {
		Name        string `json:"name"`
		Status      string `json:"status"`
		SyncEnabled *bool  `json:"sync_enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}
	var syncEnabled pgtype.Bool
	if body.SyncEnabled != nil {
		syncEnabled = pgtype.Bool{Valid: true, Bool: *body.SyncEnabled}
	}
	source, err := h.service.UpdateIdentitySource(r.Context(), entityID, id,
		optionalText(body.Name),
		optionalText(body.Status),
		syncEnabled,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "identity_source_update_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, source)
}

func (h IdentitySourceHandler) deleteIdentitySource(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	id, err := ulidValue(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_identity_source_id", err.Error())
		return
	}
	if err := h.service.DeleteIdentitySource(r.Context(), entityID, id); err != nil {
		writeError(w, http.StatusInternalServerError, "identity_source_delete_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// identitySourceFromRow converts a sqlc-generated identity source row
// (ListIdentitySourcesRow, GetIdentitySourceByIDRow, or
// UpdateIdentitySourceRow) into an API response struct.
func identitySourceFromRow(id, entityID string, sourceType, name, status string, syncEnabled bool, createdAt pgtype.Timestamptz) IdentitySourceResponse {
	return IdentitySourceResponse{
		ID:          ulidString(id),
		EntityID:    ulidString(entityID),
		Type:        sourceType,
		Name:        name,
		Status:      status,
		SyncEnabled: syncEnabled,
		CreatedAt:   createdAt.Time,
	}
}
