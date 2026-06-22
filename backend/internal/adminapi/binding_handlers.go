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

// BindingResponse represents an account binding in API responses.
type BindingResponse struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	SourceID        string    `json:"source_id"`
	SourceType      string    `json:"source_type"`
	SourceName      string    `json:"source_name"`
	DirectoryUserID string    `json:"directory_user_id"`
	ProviderUID     string    `json:"provider_uid"`
	ProviderUnionID string    `json:"provider_union_id,omitempty"`
	IsPrimary       bool      `json:"is_primary"`
	BoundAt         time.Time `json:"bound_at"`
}

// bindingService defines the data-access contract for account binding
// operations. *AdminService satisfies this interface.
type bindingService interface {
	ListUserBindings(ctx context.Context, entityID, userID string) ([]BindingResponse, error)
	GetBindingByID(ctx context.Context, entityID, id string) (BindingResponse, error)
	CreateUserBinding(ctx context.Context, entityID, userID string, sourceID, directoryUserID string, providerUID string, providerUnionID pgtype.Text, isPrimary bool) (BindingResponse, error)
	DeleteUserBinding(ctx context.Context, entityID, userID, id string) error
}

// BindingHandler handles account binding management endpoints.
type BindingHandler struct {
	service bindingService
}

func NewBindingHandler(service bindingService) BindingHandler {
	return BindingHandler{service: service}
}

func (h BindingHandler) RegisterRoutes(r chi.Router) {
	r.Get("/admin/v1/users/{id}/bindings", h.listBindings)
	r.Get("/api/admin/v1/users/{id}/bindings", h.listBindings)
	r.Post("/admin/v1/users/{id}/bindings", h.createBinding)
	r.Post("/api/admin/v1/users/{id}/bindings", h.createBinding)
	r.Delete("/admin/v1/users/{id}/bindings/{binding_id}", h.deleteBinding)
	r.Delete("/api/admin/v1/users/{id}/bindings/{binding_id}", h.deleteBinding)
}

func (h BindingHandler) listBindings(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	userID, err := ulidValue(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_user_id", err.Error())
		return
	}
	bindings, err := h.service.ListUserBindings(r.Context(), entityID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "binding_list_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, bindings)
}

func (h BindingHandler) createBinding(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	userID, err := ulidValue(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_user_id", err.Error())
		return
	}
	var body struct {
		SourceID        string `json:"source_id"`
		DirectoryUserID string `json:"directory_user_id"`
		ProviderUID     string `json:"provider_uid"`
		ProviderUnionID string `json:"provider_union_id"`
		IsPrimary       bool   `json:"is_primary"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}
	if body.SourceID == "" || body.DirectoryUserID == "" || body.ProviderUID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "source_id, directory_user_id, and provider_uid are required")
		return
	}
	sourceID, err := ulidValue(body.SourceID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_source_id", err.Error())
		return
	}
	directoryUserID, err := ulidValue(body.DirectoryUserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_directory_user_id", err.Error())
		return
	}
	binding, err := h.service.CreateUserBinding(r.Context(), entityID, userID,
		sourceID, directoryUserID,
		body.ProviderUID,
		optionalText(body.ProviderUnionID),
		body.IsPrimary,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "binding_create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, binding)
}

func (h BindingHandler) deleteBinding(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	userID, err := ulidValue(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_user_id", err.Error())
		return
	}
	bindingID, err := ulidValue(chi.URLParam(r, "binding_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_binding_id", err.Error())
		return
	}
	if err := h.service.DeleteUserBinding(r.Context(), entityID, userID, bindingID); err != nil {
		writeError(w, http.StatusInternalServerError, "binding_delete_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// bindingFromRow converts a sqlc-generated binding row (from the JOIN
// queries in binding.sql) into an API response struct.
func bindingFromRow(id, entityID, userID, sourceID, directoryUserID string, providerUID string, providerUnionID pgtype.Text, isPrimary bool, boundAt pgtype.Timestamptz, sourceType, sourceName string) BindingResponse {
	return BindingResponse{
		ID:              ulidString(id),
		UserID:          ulidString(userID),
		SourceID:        ulidString(sourceID),
		SourceType:      sourceType,
		SourceName:      sourceName,
		DirectoryUserID: ulidString(directoryUserID),
		ProviderUID:     providerUID,
		ProviderUnionID: textString(providerUnionID),
		IsPrimary:       isPrimary,
		BoundAt:         boundAt.Time,
	}
}
