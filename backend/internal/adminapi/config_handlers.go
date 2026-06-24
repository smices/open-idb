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
	GetFeishuConfig(ctx context.Context, session auth.Session) (FeishuIdentitySourceConfig, error)
	UpsertFeishuConfig(ctx context.Context, session auth.Session, input UpsertFeishuIdentitySourceConfigInput) (FeishuIdentitySourceConfig, error)
}

type FeishuIdentitySourceConfig struct {
	ID              string          `json:"id,omitempty"`
	Provider        string          `json:"provider"`
	DisplayName     string          `json:"display_name"`
	Status          string          `json:"status"`
	OAuthConfigured bool            `json:"oauth_configured"`
	SyncEnabled     bool            `json:"sync_enabled"`
	RedirectURI     string          `json:"redirect_uri"`
	Config          json.RawMessage `json:"config"`
}

type UpsertFeishuIdentitySourceConfigInput struct {
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
	r.Get("/sapi/identity-sources/feishu/config", h.getFeishuConfig)
	r.Put("/sapi/identity-sources/feishu/config", h.upsertFeishuConfig)
}

func (h ConfigHandler) getFeishuConfig(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	provider, err := h.service.GetFeishuConfig(r.Context(), session)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "identity_source_config_fetch_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, provider)
}

func (h ConfigHandler) upsertFeishuConfig(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	var input UpsertFeishuIdentitySourceConfigInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_identity_source_config_request", "invalid json body")
		return
	}
	input.Provider = "feishu"
	provider, err := h.service.UpsertFeishuConfig(r.Context(), session, input)
	if err != nil {
		writeError(w, http.StatusBadRequest, "identity_source_config_save_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, provider)
}
