// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// ResourceScopeResponse represents a resource scope in API responses.
type ResourceScopeResponse struct {
	ID        string `json:"id"`
	EntityID  string `json:"entity_id"`
	Type      string `json:"type"`
	Key       string `json:"key"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type createResourceScopeRequest struct {
	Type string `json:"type"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

type updateResourceScopeRequest struct {
	Name *string `json:"name,omitempty"`
}

type assignResourceScopeRequest struct {
	RoleID          string `json:"role_id"`
	ResourceScopeID string `json:"resource_scope_id"`
	Effect          string `json:"effect"` // allow / deny
}

// ResourceScopeHandler handles resource scope CRUD endpoints.
type ResourceScopeHandler struct {
	service resourceScopeService
}

type resourceScopeService interface {
	ListResourceScopes(ctx context.Context, entityID string, scopeType pgtype.Text, limit, offset int32) ([]ResourceScopeResponse, error)
	CountResourceScopes(ctx context.Context, entityID string, scopeType pgtype.Text) (int64, error)
	GetResourceScopeByID(ctx context.Context, entityID, id string) (ResourceScopeResponse, error)
	CreateResourceScope(ctx context.Context, entityID string, scopeType, key, name string) (ResourceScopeResponse, error)
	UpdateResourceScope(ctx context.Context, entityID, id string, name pgtype.Text) (ResourceScopeResponse, error)
	DeleteResourceScope(ctx context.Context, entityID, id string) error
	AssignResourceScopeToRole(ctx context.Context, entityID, roleID, scopeID string, effect string) error
	RemoveResourceScopeFromRole(ctx context.Context, entityID, roleID, scopeID string) error
}

func NewResourceScopeHandler(service resourceScopeService) ResourceScopeHandler {
	return ResourceScopeHandler{service: service}
}

func (h ResourceScopeHandler) RegisterRoutes(r chi.Router) {
	r.Get("/admin/v1/resource-scopes", h.listResourceScopes)
	r.Get("/api/admin/v1/resource-scopes", h.listResourceScopes)
	r.Post("/admin/v1/resource-scopes", h.createResourceScope)
	r.Post("/api/admin/v1/resource-scopes", h.createResourceScope)
	r.Get("/admin/v1/resource-scopes/{id}", h.getResourceScope)
	r.Get("/api/admin/v1/resource-scopes/{id}", h.getResourceScope)
	r.Put("/admin/v1/resource-scopes/{id}", h.updateResourceScope)
	r.Put("/api/admin/v1/resource-scopes/{id}", h.updateResourceScope)
	r.Delete("/admin/v1/resource-scopes/{id}", h.deleteResourceScope)
	r.Delete("/api/admin/v1/resource-scopes/{id}", h.deleteResourceScope)

	// Role ↔ resource scope assignment
	r.Post("/admin/v1/roles/scopes", h.assignScopeToRole)
	r.Post("/api/admin/v1/roles/scopes", h.assignScopeToRole)
	r.Delete("/admin/v1/roles/{roleID}/scopes/{scopeID}", h.removeScopeFromRole)
	r.Delete("/api/admin/v1/roles/{roleID}/scopes/{scopeID}", h.removeScopeFromRole)
}

func (h ResourceScopeHandler) listResourceScopes(w http.ResponseWriter, r *http.Request) {
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

	var scopeType pgtype.Text
	if v := r.URL.Query().Get("type"); v != "" {
		scopeType = pgtype.Text{String: v, Valid: true}
	}

	scopes, err := h.service.ListResourceScopes(r.Context(), entityID, scopeType, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "resource_scope_list_failed", err.Error())
		return
	}
	total, err := h.service.CountResourceScopes(r.Context(), entityID, scopeType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "resource_scope_count_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, PagedResponse{
		Items:  scopes,
		Total:  total,
		Limit:  int(limit),
		Offset: int(offset),
	})
}

func (h ResourceScopeHandler) getResourceScope(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, http.StatusBadRequest, "invalid_resource_scope_id", err.Error())
		return
	}
	scope, err := h.service.GetResourceScopeByID(r.Context(), entityID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "resource_scope_not_found", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, scope)
}

func (h ResourceScopeHandler) createResourceScope(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	var req createResourceScopeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request_body", err.Error())
		return
	}
	if req.Type == "" || req.Key == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "missing_fields", "type, key, and name are required")
		return
	}
	scope, err := h.service.CreateResourceScope(r.Context(), entityID, req.Type, req.Key, req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "resource_scope_create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, scope)
}

func (h ResourceScopeHandler) updateResourceScope(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, http.StatusBadRequest, "invalid_resource_scope_id", err.Error())
		return
	}
	var req updateResourceScopeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request_body", err.Error())
		return
	}
	var name pgtype.Text
	if req.Name != nil {
		name = pgtype.Text{String: *req.Name, Valid: true}
	}
	scope, err := h.service.UpdateResourceScope(r.Context(), entityID, id, name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "resource_scope_update_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, scope)
}

func (h ResourceScopeHandler) deleteResourceScope(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, http.StatusBadRequest, "invalid_resource_scope_id", err.Error())
		return
	}
	if err := h.service.DeleteResourceScope(r.Context(), entityID, id); err != nil {
		writeError(w, http.StatusInternalServerError, "resource_scope_delete_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (h ResourceScopeHandler) assignScopeToRole(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	var req assignResourceScopeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request_body", err.Error())
		return
	}
	roleID, err := ulidValue(req.RoleID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_role_id", err.Error())
		return
	}
	scopeID, err := ulidValue(req.ResourceScopeID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_resource_scope_id", err.Error())
		return
	}
	if req.Effect == "" {
		req.Effect = "allow"
	}
	if err := h.service.AssignResourceScopeToRole(r.Context(), entityID, roleID, scopeID, req.Effect); err != nil {
		writeError(w, http.StatusInternalServerError, "assign_scope_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "assigned"})
}

func (h ResourceScopeHandler) removeScopeFromRole(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	roleID, err := ulidValue(chi.URLParam(r, "roleID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_role_id", err.Error())
		return
	}
	scopeID, err := ulidValue(chi.URLParam(r, "scopeID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_resource_scope_id", err.Error())
		return
	}
	if err := h.service.RemoveResourceScopeFromRole(r.Context(), entityID, roleID, scopeID); err != nil {
		writeError(w, http.StatusInternalServerError, "remove_scope_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}
