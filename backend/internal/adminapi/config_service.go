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
	"github.com/smices/open-idb/internal/secureconfig"
)

type ConfigDBService struct {
	queries     *generated.Queries
	redirectURI string
	configCodec *secureconfig.Codec
}

func NewConfigService(queries *generated.Queries, redirectURI string, encryptionKey ...string) (*ConfigDBService, error) {
	if queries == nil {
		return nil, fmt.Errorf("queries are required")
	}
	key := ""
	if len(encryptionKey) > 0 {
		key = strings.TrimSpace(encryptionKey[0])
	}
	codec, err := secureconfig.New(key)
	if err != nil {
		return nil, err
	}
	return &ConfigDBService{queries: queries, redirectURI: strings.TrimSpace(redirectURI), configCodec: codec}, nil
}

func (s *ConfigDBService) GetFeishuConfig(ctx context.Context, session auth.Session) (FeishuIdentitySourceConfig, error) {
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		return FeishuIdentitySourceConfig{}, err
	}
	row, err := s.queries.GetFeishuIdentitySourceConfig(ctx, entityID)
	if err == nil {
		return s.feishuIdentitySourceConfigFromGetRow(row)
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
	storedConfig, err := s.sealStoredConfig(normalizeFeishuIdentitySourceConfig(input.Config))
	if err != nil {
		return FeishuIdentitySourceConfig{}, err
	}
	row, err := s.queries.UpdateFeishuIdentitySourceConfig(ctx, generated.UpdateFeishuIdentitySourceConfigParams{
		EntityID:        entityID,
		Name:            input.DisplayName,
		Status:          input.Status,
		SyncEnabled:     input.SyncEnabled,
		ConfigEncrypted: storedConfig,
	})
	if err != nil {
		return FeishuIdentitySourceConfig{}, err
	}
	return s.feishuIdentitySourceConfigFromUpdateRow(row)
}

func (s *ConfigDBService) feishuIdentitySourceConfigFromGetRow(row generated.GetFeishuIdentitySourceConfigRow) (FeishuIdentitySourceConfig, error) {
	config, err := s.openStoredConfig(row.Config)
	if err != nil {
		return FeishuIdentitySourceConfig{}, err
	}
	return FeishuIdentitySourceConfig{
		ID:              ulidString(row.ID),
		Provider:        row.Provider,
		DisplayName:     row.DisplayName,
		Status:          row.Status,
		OAuthConfigured: row.OauthConfigured,
		SyncEnabled:     row.SyncEnabled,
		RedirectURI:     s.redirectURI,
		Config:          config,
	}, nil
}

func (s *ConfigDBService) feishuIdentitySourceConfigFromUpdateRow(row generated.UpdateFeishuIdentitySourceConfigRow) (FeishuIdentitySourceConfig, error) {
	config, err := s.openStoredConfig(row.Config)
	if err != nil {
		return FeishuIdentitySourceConfig{}, err
	}
	return FeishuIdentitySourceConfig{
		ID:              ulidString(row.ID),
		Provider:        row.Provider,
		DisplayName:     row.DisplayName,
		Status:          row.Status,
		OAuthConfigured: row.OauthConfigured,
		SyncEnabled:     row.SyncEnabled,
		RedirectURI:     s.redirectURI,
		Config:          config,
	}, nil
}

func (s *ConfigDBService) sealStoredConfig(config []byte) ([]byte, error) {
	if s.configCodec == nil {
		return append([]byte(nil), config...), nil
	}
	return s.configCodec.Seal(config)
}

func (s *ConfigDBService) openStoredConfig(config []byte) ([]byte, error) {
	if s.configCodec == nil {
		return append([]byte(nil), config...), nil
	}
	plain, _, err := s.configCodec.Open(config)
	return plain, err
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
	if len(strings.TrimSpace(string(rawConfig))) == 0 {
		if oauthConfigured {
			return fmt.Errorf("app_id is required when oauth_configured is true")
		}
		return nil
	}

	config, err := auth.ParseFeishuProviderConfig(rawConfig)
	if err != nil {
		return fmt.Errorf("invalid config json: %w", err)
	}

	if oauthConfigured {
		if strings.TrimSpace(config.AppID) == "" {
			return fmt.Errorf("app_id is required when oauth_configured is true")
		}
		if strings.TrimSpace(config.AppSecret) == "" {
			return fmt.Errorf("app_secret is required when oauth_configured is true")
		}
	}
	if err := validateOptionalFeishuSecret("verification_token", config.VerificationToken); err != nil {
		return err
	}
	if err := validateOptionalFeishuSecret("encrypt_key", config.EncryptKey); err != nil {
		return err
	}
	return nil
}

func validateOptionalFeishuSecret(name, value string) error {
	if value == "" {
		return nil
	}
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s cannot contain only whitespace", name)
	}
	if len(value) > 512 {
		return fmt.Errorf("%s must be at most 512 bytes", name)
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
