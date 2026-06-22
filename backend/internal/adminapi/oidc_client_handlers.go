// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/smices/open-idb/internal/db/generated"
)

// OIDCClientResponse represents an OIDC client in API responses.
type OIDCClientResponse struct {
	ID            string   `json:"id"`
	EntityID      string   `json:"entity_id"`
	ApplicationID string   `json:"application_id"`
	ClientID      string   `json:"client_id"`
	RedirectURIs  []string `json:"redirect_uris"`
	AllowedScopes []string `json:"allowed_scopes"`
	GrantTypes    []string `json:"grant_types"`
	ResponseTypes []string `json:"response_types"`
	PKCERequired  bool     `json:"pkce_required"`
	Status        string   `json:"status"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

// oidcClientService defines the data-access contract for OIDC client operations.
type oidcClientService interface {
	ListOIDCClients(ctx context.Context, entityID string, limit, offset int32) ([]OIDCClientResponse, error)
	CountOIDCClients(ctx context.Context, entityID string) (int64, error)
	GetOIDCClientByID(ctx context.Context, entityID, id string) (OIDCClientResponse, error)
	CreateOIDCClient(ctx context.Context, params generated.CreateOIDCClientParams) (OIDCClientResponse, string, error)
	UpdateOIDCClient(ctx context.Context, params generated.UpdateOIDCClientParams) (OIDCClientResponse, error)
	DeleteOIDCClient(ctx context.Context, entityID, id string) error
	RotateOIDCClientSecret(ctx context.Context, entityID, id string) (OIDCClientResponse, string, error)
}

// OIDCClientHandler handles OIDC client CRUD endpoints.
type OIDCClientHandler struct {
	service oidcClientService
}

func NewOIDCClientHandler(service oidcClientService) OIDCClientHandler {
	return OIDCClientHandler{service: service}
}

func (h OIDCClientHandler) RegisterRoutes(r chi.Router) {
	r.Get("/admin/v1/oidc-clients", h.listOIDCClients)
	r.Get("/api/admin/v1/oidc-clients", h.listOIDCClients)
	r.Get("/admin/v1/oidc-clients/{id}", h.getOIDCClient)
	r.Get("/api/admin/v1/oidc-clients/{id}", h.getOIDCClient)
	r.Post("/admin/v1/oidc-clients", h.createOIDCClient)
	r.Post("/api/admin/v1/oidc-clients", h.createOIDCClient)
	r.Put("/admin/v1/oidc-clients/{id}", h.updateOIDCClient)
	r.Put("/api/admin/v1/oidc-clients/{id}", h.updateOIDCClient)
	r.Delete("/admin/v1/oidc-clients/{id}", h.deleteOIDCClient)
	r.Delete("/api/admin/v1/oidc-clients/{id}", h.deleteOIDCClient)
	r.Post("/admin/v1/oidc-clients/{id}/rotate-secret", h.rotateSecret)
	r.Post("/api/admin/v1/oidc-clients/{id}/rotate-secret", h.rotateSecret)
}

func (h OIDCClientHandler) listOIDCClients(w http.ResponseWriter, r *http.Request) {
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
	clients, err := h.service.ListOIDCClients(r.Context(), entityID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "oidc_client_list_failed", err.Error())
		return
	}
	total, err := h.service.CountOIDCClients(r.Context(), entityID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "oidc_client_count_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":   clients,
		"clients": clients,
		"total":   total,
		"limit":   int(limit),
		"offset":  int(offset),
	})
}

func (h OIDCClientHandler) getOIDCClient(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, http.StatusBadRequest, "invalid_oidc_client_id", err.Error())
		return
	}
	client, err := h.service.GetOIDCClientByID(r.Context(), entityID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "oidc_client_not_found", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, client)
}

func (h OIDCClientHandler) createOIDCClient(w http.ResponseWriter, r *http.Request) {
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
		ApplicationID string   `json:"application_id"`
		ClientID      string   `json:"client_id"`
		RedirectURIs  []string `json:"redirect_uris"`
		AllowedScopes []string `json:"allowed_scopes"`
		GrantTypes    []string `json:"grant_types"`
		ResponseTypes []string `json:"response_types"`
		PKCERequired  bool     `json:"pkce_required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}
	if body.ApplicationID == "" || body.ClientID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "application_id and client_id are required")
		return
	}
	appID, err := ulidValue(body.ApplicationID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_application_id", err.Error())
		return
	}

	client, secret, err := h.service.CreateOIDCClient(r.Context(), generated.CreateOIDCClientParams{
		EntityID:      entityID,
		ApplicationID: appID,
		ClientID:      body.ClientID,
		RedirectUris:  body.RedirectURIs,
		AllowedScopes: body.AllowedScopes,
		GrantTypes:    body.GrantTypes,
		ResponseTypes: body.ResponseTypes,
		PkceRequired:  body.PKCERequired,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "oidc_client_create_failed", err.Error())
		return
	}

	// Return the secret only once during creation
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"client":        client,
		"client_secret": secret,
	})
}

func (h OIDCClientHandler) updateOIDCClient(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, http.StatusBadRequest, "invalid_oidc_client_id", err.Error())
		return
	}
	var body struct {
		RedirectURIs  []string `json:"redirect_uris"`
		AllowedScopes []string `json:"allowed_scopes"`
		GrantTypes    []string `json:"grant_types"`
		ResponseTypes []string `json:"response_types"`
		PKCERequired  *bool    `json:"pkce_required"`
		Status        string   `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}

	var pkceRequired pgtype.Bool
	if body.PKCERequired != nil {
		pkceRequired = pgtype.Bool{Valid: true, Bool: *body.PKCERequired}
	}

	client, err := h.service.UpdateOIDCClient(r.Context(), generated.UpdateOIDCClientParams{
		EntityID:      entityID,
		ID:            id,
		RedirectUris:  body.RedirectURIs,
		AllowedScopes: body.AllowedScopes,
		GrantTypes:    body.GrantTypes,
		ResponseTypes: body.ResponseTypes,
		PkceRequired:  pkceRequired,
		Status:        optionalText(body.Status),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "oidc_client_update_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, client)
}

func (h OIDCClientHandler) deleteOIDCClient(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, http.StatusBadRequest, "invalid_oidc_client_id", err.Error())
		return
	}
	if err := h.service.DeleteOIDCClient(r.Context(), entityID, id); err != nil {
		writeError(w, http.StatusInternalServerError, "oidc_client_delete_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h OIDCClientHandler) rotateSecret(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, http.StatusBadRequest, "invalid_oidc_client_id", err.Error())
		return
	}
	client, secret, err := h.service.RotateOIDCClientSecret(r.Context(), entityID, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "oidc_client_rotate_secret_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"client":        client,
		"client_secret": secret,
	})
}

// oidcClientFromRow converts a generated OIDC client row to a response.
func oidcClientFromRow(row generated.ListOIDCClientsRow) OIDCClientResponse {
	return OIDCClientResponse{
		ID:            ulidString(row.ID),
		EntityID:      ulidString(row.EntityID),
		ApplicationID: ulidString(row.ApplicationID),
		ClientID:      row.ClientID,
		RedirectURIs:  row.RedirectUris,
		AllowedScopes: row.AllowedScopes,
		GrantTypes:    row.GrantTypes,
		ResponseTypes: row.ResponseTypes,
		PKCERequired:  row.PkceRequired,
		Status:        row.Status,
		CreatedAt:     row.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:     row.UpdatedAt.Time.Format(time.RFC3339),
	}
}
