// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/smices/open-idb/internal/auth"
)

type ConfigService interface {
	ListIMProviderConfigs(ctx context.Context, session auth.Session) ([]IMProviderConfig, error)
	UpsertIMProviderConfig(ctx context.Context, session auth.Session, input UpsertIMProviderConfigInput) (IMProviderConfig, error)
	ListMCPConnectors(ctx context.Context, session auth.Session) ([]MCPConnector, error)
	CreateMCPConnector(ctx context.Context, session auth.Session, input CreateMCPConnectorInput) (MCPConnector, error)
}

type IMProviderConfig struct {
	ID              string          `json:"id,omitempty"`
	Provider        string          `json:"provider"`
	DisplayName     string          `json:"display_name"`
	Status          string          `json:"status"`
	OAuthConfigured bool            `json:"oauth_configured"`
	SyncEnabled     bool            `json:"sync_enabled"`
	Config          json.RawMessage `json:"config"`
}

type UpsertIMProviderConfigInput struct {
	Provider        string
	DisplayName     string          `json:"display_name"`
	Status          string          `json:"status"`
	OAuthConfigured bool            `json:"oauth_configured"`
	SyncEnabled     bool            `json:"sync_enabled"`
	Config          json.RawMessage `json:"config"`
}

type MCPConnector struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	EndpointURL string `json:"endpoint_url"`
	AuthType    string `json:"auth_type"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

type CreateMCPConnectorInput struct {
	Name        string `json:"name"`
	EndpointURL string `json:"endpoint_url"`
	AuthType    string `json:"auth_type"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

type ConfigHandler struct {
	service ConfigService
}

func NewConfigHandler(service ConfigService) ConfigHandler {
	return ConfigHandler{service: service}
}

func (h ConfigHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/admin/v1/integrations/im", h.listIMProviderConfigs)
	r.Put("/api/admin/v1/integrations/im/{provider}", h.upsertIMProviderConfig)
	r.Get("/api/admin/v1/mcp/connectors", h.listMCPConnectors)
	r.Post("/api/admin/v1/mcp/connectors", h.createMCPConnector)
}

func (h ConfigHandler) listIMProviderConfigs(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	providers, err := h.service.ListIMProviderConfigs(r.Context(), session)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "im_provider_list_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, providers)
}

func (h ConfigHandler) upsertIMProviderConfig(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	var input UpsertIMProviderConfigInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_im_provider_request", "invalid json body")
		return
	}
	input.Provider = chi.URLParam(r, "provider")
	provider, err := h.service.UpsertIMProviderConfig(r.Context(), session, input)
	if err != nil {
		writeError(w, http.StatusBadRequest, "im_provider_upsert_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, provider)
}

func (h ConfigHandler) listMCPConnectors(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	connectors, err := h.service.ListMCPConnectors(r.Context(), session)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "mcp_connector_list_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, connectors)
}

func (h ConfigHandler) createMCPConnector(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	var input CreateMCPConnectorInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_mcp_connector_request", "invalid json body")
		return
	}
	connector, err := h.service.CreateMCPConnector(r.Context(), session, input)
	if err != nil {
		writeError(w, http.StatusBadRequest, "mcp_connector_create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, connector)
}
