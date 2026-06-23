// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/smices/open-idb/internal/auth"
	"github.com/smices/open-idb/internal/db/generated"
)

type ConfigDBService struct {
	queries *generated.Queries
}

func NewConfigService(queries *generated.Queries) (*ConfigDBService, error) {
	if queries == nil {
		return nil, fmt.Errorf("queries are required")
	}
	return &ConfigDBService{queries: queries}, nil
}

func (s *ConfigDBService) ListIMProviderConfigs(ctx context.Context, session auth.Session) ([]IMProviderConfig, error) {
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListIMProviderConfigs(ctx, entityID)
	if err != nil {
		return nil, err
	}
	providers := make([]IMProviderConfig, 0, len(rows))
	for _, row := range rows {
		providers = append(providers, imProviderConfigFromRow(row))
	}
	return providers, nil
}

func (s *ConfigDBService) UpsertIMProviderConfig(ctx context.Context, session auth.Session, input UpsertIMProviderConfigInput) (IMProviderConfig, error) {
	if err := validateIMProviderInput(input); err != nil {
		return IMProviderConfig{}, err
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		return IMProviderConfig{}, err
	}
	row, err := s.queries.UpsertIMProviderConfig(ctx, generated.UpsertIMProviderConfigParams{
		EntityID:        entityID,
		Provider:        input.Provider,
		DisplayName:     input.DisplayName,
		Status:          input.Status,
		OauthConfigured: input.OAuthConfigured,
		BotConfigured:   false,
		SyncEnabled:     input.SyncEnabled,
		Config:          normalizeIMProviderConfig(input.Config),
	})
	if err != nil {
		return IMProviderConfig{}, err
	}
	return imProviderConfigFromRow(row), nil
}

func imProviderConfigFromRow(row generated.ImProviderConfig) IMProviderConfig {
	return IMProviderConfig{
		ID:              ulidString(row.ID),
		Provider:        row.Provider,
		DisplayName:     row.DisplayName,
		Status:          row.Status,
		OAuthConfigured: row.OauthConfigured,
		SyncEnabled:     row.SyncEnabled,
		Config:          row.Config,
	}
}

func validateIMProviderInput(input UpsertIMProviderConfigInput) error {
	if input.Provider != "feishu" && input.Provider != "dingtalk" && input.Provider != "wecom" {
		return fmt.Errorf("unsupported im provider")
	}
	if input.DisplayName == "" {
		return fmt.Errorf("display_name is required")
	}

	if err := validateIMProviderConfig(input.Provider, input.OAuthConfigured, input.Config); err != nil {
		return err
	}

	return validateStatus(input.Status)
}

func validateIMProviderConfig(provider string, oauthConfigured bool, rawConfig json.RawMessage) error {
	if !oauthConfigured {
		return nil
	}

	config, err := auth.ParseFeishuProviderConfig(rawConfig)
	if err != nil {
		return fmt.Errorf("invalid config json: %w", err)
	}

	if provider == "feishu" {
		if oauthConfigured {
			if strings.TrimSpace(config.AppID) == "" {
				return fmt.Errorf("app_id is required when oauth_configured is true")
			}
			if strings.TrimSpace(config.AppSecret) == "" {
				return fmt.Errorf("app_secret is required when oauth_configured is true")
			}
		}
		return nil
	}

	if rawConfig != nil {
		// provider type is not feishu; keep config as optional for compatibility, but verify JSON is valid when provided.
		if !json.Valid(rawConfig) {
			return fmt.Errorf("invalid config json")
		}
	}
	return nil
}

func normalizeIMProviderConfig(rawConfig json.RawMessage) json.RawMessage {
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
