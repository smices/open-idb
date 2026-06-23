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

type ConfigHandler struct {
	service ConfigService
}

func NewConfigHandler(service ConfigService) ConfigHandler {
	return ConfigHandler{service: service}
}

func (h ConfigHandler) RegisterRoutes(r chi.Router) {
	r.Get("/sapi/integrations/im", h.listIMProviderConfigs)
	r.Put("/sapi/integrations/im/{provider}", h.upsertIMProviderConfig)
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
