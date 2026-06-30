// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/smices/open-idb/internal/auth"
	"github.com/smices/open-idb/internal/db/generated"
)

type ConfigDBService struct {
	queries     *generated.Queries
	redirectURI string
}

func NewConfigService(queries *generated.Queries, redirectURI string) (*ConfigDBService, error) {
	if queries == nil {
		return nil, fmt.Errorf("queries are required")
	}
	return &ConfigDBService{queries: queries, redirectURI: strings.TrimSpace(redirectURI)}, nil
}

func (s *ConfigDBService) GetFeishuConfig(ctx context.Context, session auth.Session) (FeishuIdentitySourceConfig, error) {
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		return FeishuIdentitySourceConfig{}, err
	}
	row, err := s.queries.GetFeishuIdentitySourceConfig(ctx, entityID)
	if err == nil {
		return s.feishuIdentitySourceConfigFromGetRow(row), nil
	}
	if err != pgx.ErrNoRows {
		return FeishuIdentitySourceConfig{}, err
	}
	return FeishuIdentitySourceConfig{
		Provider:    "feishu",
		DisplayName: "Feishu",
		Status:      "disabled",
		RedirectURI: s.redirectURI,
		Config:      json.RawMessage("{}"),
	}, nil
}

func (s *ConfigDBService) UpsertFeishuConfig(ctx context.Context, session auth.Session, input UpsertFeishuIdentitySourceConfigInput) (FeishuIdentitySourceConfig, error) {
	input.Provider = "feishu"
	if err := validateFeishuIdentitySourceInput(input); err != nil {
		return FeishuIdentitySourceConfig{}, err
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		return FeishuIdentitySourceConfig{}, err
	}
	row, err := s.queries.UpdateFeishuIdentitySourceConfig(ctx, generated.UpdateFeishuIdentitySourceConfigParams{
		EntityID:        entityID,
		Name:            input.DisplayName,
		Status:          input.Status,
		SyncEnabled:     input.SyncEnabled,
		ConfigEncrypted: normalizeFeishuIdentitySourceConfig(input.Config),
	})
	if err != nil {
		return FeishuIdentitySourceConfig{}, err
	}
	return s.feishuIdentitySourceConfigFromUpdateRow(row), nil
}

func (s *ConfigDBService) feishuIdentitySourceConfigFromGetRow(row generated.GetFeishuIdentitySourceConfigRow) FeishuIdentitySourceConfig {
	return FeishuIdentitySourceConfig{
		ID:              ulidString(row.ID),
		Provider:        row.Provider,
		DisplayName:     row.DisplayName,
		Status:          row.Status,
		OAuthConfigured: row.OauthConfigured,
		SyncEnabled:     row.SyncEnabled,
		RedirectURI:     s.redirectURI,
		Config:          row.Config,
	}
}

func (s *ConfigDBService) feishuIdentitySourceConfigFromUpdateRow(row generated.UpdateFeishuIdentitySourceConfigRow) FeishuIdentitySourceConfig {
	return FeishuIdentitySourceConfig{
		ID:              ulidString(row.ID),
		Provider:        row.Provider,
		DisplayName:     row.DisplayName,
		Status:          row.Status,
		OAuthConfigured: row.OauthConfigured,
		SyncEnabled:     row.SyncEnabled,
		RedirectURI:     s.redirectURI,
		Config:          row.Config,
	}
}

func validateFeishuIdentitySourceInput(input UpsertFeishuIdentitySourceConfigInput) error {
	if input.Provider != "feishu" {
		return fmt.Errorf("unsupported identity source provider")
	}
	if input.DisplayName == "" {
		return fmt.Errorf("display_name is required")
	}

	if err := validateFeishuIdentitySourceConfig(input.OAuthConfigured, input.Config); err != nil {
		return err
	}

	return validateStatus(input.Status)
}

func validateFeishuIdentitySourceConfig(oauthConfigured bool, rawConfig json.RawMessage) error {
	if !oauthConfigured {
		return nil
	}

	config, err := auth.ParseFeishuProviderConfig(rawConfig)
	if err != nil {
		return fmt.Errorf("invalid config json: %w", err)
	}

	if strings.TrimSpace(config.AppID) == "" {
		return fmt.Errorf("app_id is required when oauth_configured is true")
	}
	if strings.TrimSpace(config.AppSecret) == "" {
		return fmt.Errorf("app_secret is required when oauth_configured is true")
	}
	if strings.TrimSpace(config.WorkplaceAppID) != "" && strings.TrimSpace(config.WorkplaceAppSecret) == "" {
		return fmt.Errorf("workplace_app_secret is required when workplace_app_id is configured")
	}
	if strings.TrimSpace(config.WorkplaceAppSecret) != "" && strings.TrimSpace(config.WorkplaceAppID) == "" {
		return fmt.Errorf("workplace_app_id is required when workplace_app_secret is configured")
	}
	return nil
}

func normalizeFeishuIdentitySourceConfig(rawConfig json.RawMessage) json.RawMessage {
	if len(strings.TrimSpace(string(rawConfig))) == 0 {
		return json.RawMessage("{}")
	}
	return rawConfig
}

func validateStatus(status string) error {
	if status != "active" && status != "disabled" {
		return fmt.Errorf("status must be active or disabled")
	}
	return nil
}
