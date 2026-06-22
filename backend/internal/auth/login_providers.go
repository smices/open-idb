// SPDX-License-Identifier: MIT

package auth

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/smices/open-idb/internal/db/generated"
)

// LoginProvider represents a configured third-party login provider.
type LoginProvider struct {
	Provider    string `json:"provider"`
	DisplayName string `json:"display_name"`
	OAuthURL    string `json:"oauth_url,omitempty"`
}

type FeishuProviderConfig struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

// providerQueries is the database interface for login provider discovery.
// *generated.Queries satisfies this interface.
type providerQueries interface {
	ListLoginProviders(ctx context.Context, entityID string) ([]generated.ListLoginProvidersRow, error)
	GetEntityBySlug(ctx context.Context, slug string) (generated.BusinessEntity, error)
}

// LoginProviderService returns configured third-party login providers for a entity.
type LoginProviderService struct {
	queries           providerQueries
	feishuAppID       string
	feishuAppSecret   string
	feishuRedirectURI string
}

// NewLoginProviderService creates a LoginProviderService.
func NewLoginProviderService(queries *generated.Queries, feishuAppID string, feishuAppSecret string, redirectURI string) *LoginProviderService {
	return &LoginProviderService{
		queries:           queries,
		feishuAppID:       feishuAppID,
		feishuAppSecret:   feishuAppSecret,
		feishuRedirectURI: redirectURI,
	}
}

// ListProviders returns configured third-party login providers for a entity.
// For Feishu providers, it constructs the OAuth authorize URL.
func (s *LoginProviderService) ListProviders(ctx context.Context, entityID string) ([]LoginProvider, error) {
	entityULID, err := resolveEntityRef(ctx, s.queries, entityID)
	if err != nil {
		return nil, fmt.Errorf("invalid entity_id: %w", err)
	}

	rows, err := s.queries.ListLoginProviders(ctx, entityULID)
	if err != nil {
		return nil, err
	}

	providers := make([]LoginProvider, 0, len(rows))
	feishuConfig, err := ParseFeishuProviderConfig(safeJSON(s.feishuAppIDConfig(rows, entityID)))
	if err != nil {
		return nil, err
	}
	feishuAppID := strings.TrimSpace(feishuConfig.AppID)
	if feishuAppID == "" {
		feishuAppID = strings.TrimSpace(s.feishuAppID)
	}

	for _, row := range rows {
		p := LoginProvider{
			Provider:    row.Provider,
			DisplayName: row.DisplayName,
		}
		if row.Provider == "feishu" {
			rowAppID, _ := s.feishuAppIDFromRow(ctx, entityID, row)
			if strings.TrimSpace(rowAppID) != "" {
				p.OAuthURL = s.buildFeishuOAuthURL(entityID, rowAppID)
			}
		}
		providers = append(providers, p)
	}

	return providers, nil
}

func (s *LoginProviderService) ResolveFeishuConfig(ctx context.Context, entityID string) (FeishuProviderConfig, error) {
	entityULID, err := resolveEntityRef(ctx, s.queries, entityID)
	if err != nil {
		return FeishuProviderConfig{}, fmt.Errorf("invalid entity_id: %w", err)
	}
	rows, err := s.queries.ListLoginProviders(ctx, entityULID)
	if err != nil {
		return FeishuProviderConfig{}, err
	}

	cfg := FeishuProviderConfig{
		AppID:     strings.TrimSpace(s.feishuAppID),
		AppSecret: strings.TrimSpace(s.feishuAppSecret),
	}

	for _, row := range rows {
		if row.Provider != "feishu" {
			continue
		}
		rowCfg, err := ParseFeishuProviderConfig(row.Config)
		if err != nil {
			return FeishuProviderConfig{}, err
		}
		if strings.TrimSpace(rowCfg.AppID) != "" {
			cfg.AppID = strings.TrimSpace(rowCfg.AppID)
		}
		if strings.TrimSpace(rowCfg.AppSecret) != "" {
			cfg.AppSecret = strings.TrimSpace(rowCfg.AppSecret)
		}
	}

	if strings.TrimSpace(cfg.AppID) == "" || strings.TrimSpace(cfg.AppSecret) == "" {
		return FeishuProviderConfig{}, fmt.Errorf("feishu app_id and app_secret are not configured")
	}

	return cfg, nil
}

func mustULID(value string) string {
	entityULID, err := parseULID(value)
	if err != nil {
		panic(err)
	}
	return entityULID
}

func (s *LoginProviderService) feishuAppIDFromRow(ctx context.Context, entityID string, row generated.ListLoginProvidersRow) (string, error) {
	_ = ctx
	_ = entityID
	cfg, err := ParseFeishuProviderConfig(row.Config)
	if err != nil {
		return "", err
	}
	appID := strings.TrimSpace(cfg.AppID)
	if appID != "" {
		return appID, nil
	}
	return strings.TrimSpace(s.feishuAppID), nil
}

func (s *LoginProviderService) feishuAppIDConfig(rows []generated.ListLoginProvidersRow, entityID string) []byte {
	for _, row := range rows {
		if row.Provider == "feishu" {
			cfg, err := ParseFeishuProviderConfig(row.Config)
			if err == nil && strings.TrimSpace(cfg.AppID) != "" {
				return row.Config
			}
		}
	}
	if strings.TrimSpace(entityID) == "" {
		return nil
	}
	return nil
}

func (s *LoginProviderService) buildFeishuOAuthURL(entityID string, appID string) string {
	state := oauthState{EntityID: entityID}
	stateBytes, _ := json.Marshal(state)
	stateEncoded := base64.RawURLEncoding.EncodeToString(stateBytes)

	params := url.Values{}
	params.Set("app_id", appID)
	params.Set("redirect_uri", s.feishuRedirectURI)
	params.Set("state", stateEncoded)
	params.Set("response_type", "code")
	return "https://open.feishu.cn/open-apis/authen/v1/authorize?" + params.Encode()
}

// ParseFeishuProviderConfig parses JSON configuration for a Feishu integration.
func ParseFeishuProviderConfig(raw json.RawMessage) (FeishuProviderConfig, error) {
	var cfg FeishuProviderConfig
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(trimmed, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse feishu provider config: %w", err)
	}
	return cfg, nil
}

func safeJSON(raw []byte) []byte {
	if len(bytes.TrimSpace(raw)) == 0 {
		return []byte("{}")
	}
	return raw
}
