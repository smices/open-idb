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
	"github.com/smices/open-idb/internal/id"
	"github.com/smices/open-idb/internal/secureconfig"
)

// LoginProvider represents a configured third-party login provider.
type LoginProvider struct {
	Provider             string `json:"provider"`
	DisplayName          string `json:"display_name"`
	OAuthURL             string `json:"oauth_url,omitempty"`
	AppID                string `json:"app_id,omitempty"`
	WorkplaceExchangeURL string `json:"workplace_exchange_url,omitempty"`
}

type FeishuProviderConfig struct {
	AppID             string `json:"app_id"`
	AppSecret         string `json:"app_secret"`
	VerificationToken string `json:"verification_token,omitempty"`
	EncryptKey        string `json:"encrypt_key,omitempty"`
}

// providerQueries is the database interface for login provider discovery.
// *generated.Queries satisfies this interface.
type providerQueries interface {
	ListLoginProviders(ctx context.Context, entityID string) ([]generated.ListLoginProvidersRow, error)
	GetIdentitySourceConfigByID(ctx context.Context, arg generated.GetIdentitySourceConfigByIDParams) (generated.GetIdentitySourceConfigByIDRow, error)
	GetOIDCClientByClientID(ctx context.Context, arg generated.GetOIDCClientByClientIDParams) (generated.OidcClient, error)
	GetEntityBySlug(ctx context.Context, slug string) (generated.BusinessEntity, error)
}

// LoginProviderService returns configured third-party login providers for a entity.
type LoginProviderService struct {
	queries           providerQueries
	feishuAppID       string
	feishuAppSecret   string
	feishuRedirectURI string
	configCodec       *secureconfig.Codec
}

// NewLoginProviderService creates a LoginProviderService.
func NewLoginProviderService(queries *generated.Queries, feishuAppID string, feishuAppSecret string, redirectURI string) *LoginProviderService {
	codec, _ := secureconfig.New("")
	return &LoginProviderService{
		queries:           queries,
		feishuAppID:       feishuAppID,
		feishuAppSecret:   feishuAppSecret,
		feishuRedirectURI: redirectURI,
		configCodec:       codec,
	}
}

func (s *LoginProviderService) SetConfigEncryptionKey(encodedKey string) error {
	codec, err := secureconfig.New(strings.TrimSpace(encodedKey))
	if err != nil {
		return err
	}
	s.configCodec = codec
	return nil
}

// ListProviders returns configured third-party login providers for a entity.
// For Feishu providers, it constructs the OAuth authorize URL.
func (s *LoginProviderService) ListProviders(ctx context.Context, entityID string) ([]LoginProvider, error) {
	return s.ListProvidersForClient(ctx, entityID, "")
}

func (s *LoginProviderService) ListProvidersForClient(ctx context.Context, entityID string, clientID string) ([]LoginProvider, error) {
	entityULID, err := resolveEntityRef(ctx, s.queries, entityID)
	if err != nil {
		return nil, fmt.Errorf("invalid entity_id: %w", err)
	}

	rows, err := s.queries.ListLoginProviders(ctx, entityULID)
	if err != nil {
		return nil, err
	}

	providers := make([]LoginProvider, 0, len(rows))
	feishuConfig, err := s.parseStoredFeishuProviderConfig(safeJSON(s.feishuAppIDConfig(rows, entityID)))
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
			workplaceAppID, err := s.feishuWorkplaceAppIDFromClient(ctx, entityULID, clientID)
			if err != nil {
				return nil, err
			}
			if strings.TrimSpace(clientID) == "" && strings.TrimSpace(workplaceAppID) == "" {
				workplaceAppID = rowAppID
			}
			if strings.TrimSpace(rowAppID) != "" {
				p.OAuthURL = s.buildFeishuOAuthURL(entityID, rowAppID)
			}
			if strings.TrimSpace(workplaceAppID) != "" {
				p.AppID = strings.TrimSpace(workplaceAppID)
				p.WorkplaceExchangeURL = "/api/auth/feishu/exchange"
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
		rowCfg, err := s.parseStoredFeishuProviderConfig(row.Config)
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

// ResolveFeishuSourceConfig reads one exact identity source. Sync and webhook
// callers must not silently use another Feishu source from the same entity.
func (s *LoginProviderService) ResolveFeishuSourceConfig(ctx context.Context, entityID string, sourceID string) (FeishuProviderConfig, error) {
	entityULID, err := resolveEntityRef(ctx, s.queries, entityID)
	if err != nil {
		return FeishuProviderConfig{}, fmt.Errorf("invalid entity_id: %w", err)
	}
	sourceID = strings.TrimSpace(sourceID)
	if err := id.ValidateULID(sourceID); err != nil {
		return FeishuProviderConfig{}, fmt.Errorf("invalid source_id: %w", err)
	}
	row, err := s.queries.GetIdentitySourceConfigByID(ctx, generated.GetIdentitySourceConfigByIDParams{
		EntityID: entityULID,
		ID:       sourceID,
	})
	if err != nil {
		return FeishuProviderConfig{}, err
	}
	if row.Type != "feishu" {
		return FeishuProviderConfig{}, fmt.Errorf("identity source is not a feishu source")
	}
	if row.Status != "active" {
		return FeishuProviderConfig{}, fmt.Errorf("feishu identity source is not active")
	}
	stored, err := s.parseStoredFeishuProviderConfig(row.Config)
	if err != nil {
		return FeishuProviderConfig{}, err
	}
	cfg := FeishuProviderConfig{
		AppID:             strings.TrimSpace(stored.AppID),
		AppSecret:         strings.TrimSpace(stored.AppSecret),
		VerificationToken: strings.TrimSpace(stored.VerificationToken),
		EncryptKey:        strings.TrimSpace(stored.EncryptKey),
	}
	if cfg.AppID == "" {
		cfg.AppID = strings.TrimSpace(s.feishuAppID)
	}
	if cfg.AppSecret == "" {
		cfg.AppSecret = strings.TrimSpace(s.feishuAppSecret)
	}
	if cfg.AppID == "" || cfg.AppSecret == "" {
		return FeishuProviderConfig{}, fmt.Errorf("feishu app_id and app_secret are not configured")
	}
	return cfg, nil
}

func (s *LoginProviderService) ResolveFeishuWorkplaceConfig(ctx context.Context, entityID string, clientID string) (FeishuProviderConfig, error) {
	cfg, err := s.ResolveFeishuConfig(ctx, entityID)
	if err != nil {
		return FeishuProviderConfig{}, err
	}
	entityULID, err := resolveEntityRef(ctx, s.queries, entityID)
	if err != nil {
		return FeishuProviderConfig{}, fmt.Errorf("invalid entity_id: %w", err)
	}
	clientID = strings.TrimSpace(clientID)
	if clientID != "" {
		client, err := s.queries.GetOIDCClientByClientID(ctx, generated.GetOIDCClientByClientIDParams{
			EntityID: entityULID,
			ClientID: clientID,
		})
		if err != nil {
			return FeishuProviderConfig{}, err
		}
		if strings.TrimSpace(client.WorkplaceProvider) != "feishu" ||
			strings.TrimSpace(client.WorkplaceAppID) == "" ||
			strings.TrimSpace(client.WorkplaceAppSecret) == "" {
			return FeishuProviderConfig{}, fmt.Errorf("feishu workplace app_id and app_secret are not configured for client")
		}
		cfg.AppID = strings.TrimSpace(client.WorkplaceAppID)
		cfg.AppSecret = strings.TrimSpace(client.WorkplaceAppSecret)
	}
	if strings.TrimSpace(cfg.AppID) == "" || strings.TrimSpace(cfg.AppSecret) == "" {
		return FeishuProviderConfig{}, fmt.Errorf("feishu workplace app_id and app_secret are not configured")
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
	cfg, err := s.parseStoredFeishuProviderConfig(row.Config)
	if err != nil {
		return "", err
	}
	appID := strings.TrimSpace(cfg.AppID)
	if appID != "" {
		return appID, nil
	}
	return strings.TrimSpace(s.feishuAppID), nil
}

func (s *LoginProviderService) feishuWorkplaceAppIDFromClient(ctx context.Context, entityID string, clientID string) (string, error) {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return "", nil
	}
	client, err := s.queries.GetOIDCClientByClientID(ctx, generated.GetOIDCClientByClientIDParams{
		EntityID: entityID,
		ClientID: clientID,
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(client.WorkplaceProvider) != "feishu" {
		return "", nil
	}
	return strings.TrimSpace(client.WorkplaceAppID), nil
}

func (s *LoginProviderService) feishuAppIDConfig(rows []generated.ListLoginProvidersRow, entityID string) []byte {
	for _, row := range rows {
		if row.Provider == "feishu" {
			cfg, err := s.parseStoredFeishuProviderConfig(row.Config)
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

func (s *LoginProviderService) parseStoredFeishuProviderConfig(raw []byte) (FeishuProviderConfig, error) {
	plain := raw
	if s.configCodec != nil {
		var err error
		plain, _, err = s.configCodec.Open(raw)
		if err != nil {
			return FeishuProviderConfig{}, err
		}
	}
	return ParseFeishuProviderConfig(plain)
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
