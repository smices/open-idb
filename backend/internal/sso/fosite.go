// SPDX-License-Identifier: MIT

package sso

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/ory/fosite"
	"github.com/ory/fosite/compose"
	"github.com/smices/open-idb/internal/db/generated"
)

type FositeProvider interface {
	NewAuthorizeRequest(ctx context.Context, r *http.Request) (fosite.AuthorizeRequester, error)
	NewAccessRequest(ctx context.Context, r *http.Request, session fosite.Session) (fosite.AccessRequester, error)
	WriteAuthorizeError(ctx context.Context, w http.ResponseWriter, requester fosite.AuthorizeRequester, err error)
	WriteAccessError(ctx context.Context, w http.ResponseWriter, requester fosite.AccessRequester, err error)
}

type fositeProvider struct {
	provider fosite.OAuth2Provider
}

func NewFositeProvider(store fosite.Storage, cfg *fosite.Config) FositeProvider {
	provider := compose.Compose(
		cfg,
		store,
		compose.NewOAuth2HMACStrategy(cfg),
		compose.OAuth2AuthorizeExplicitFactory,
		compose.OAuth2PKCEFactory,
	)
	return fositeProvider{provider: provider}
}

func (p fositeProvider) NewAuthorizeRequest(ctx context.Context, r *http.Request) (fosite.AuthorizeRequester, error) {
	return p.provider.NewAuthorizeRequest(ctx, r)
}

func (p fositeProvider) NewAccessRequest(ctx context.Context, r *http.Request, session fosite.Session) (fosite.AccessRequester, error) {
	return p.provider.NewAccessRequest(ctx, r, session)
}

func (p fositeProvider) WriteAuthorizeError(ctx context.Context, w http.ResponseWriter, requester fosite.AuthorizeRequester, err error) {
	p.provider.WriteAuthorizeError(ctx, w, requester, err)
}

func (p fositeProvider) WriteAccessError(ctx context.Context, w http.ResponseWriter, requester fosite.AccessRequester, err error) {
	p.provider.WriteAccessError(ctx, w, requester, err)
}

type entityIDContextKey struct{}

func ContextWithEntityID(ctx context.Context, entityID string) context.Context {
	return context.WithValue(ctx, entityIDContextKey{}, entityID)
}

type fositeStorage struct {
	queries *generated.Queries
}

func NewFositeStorage(queries *generated.Queries) fosite.Storage {
	return &fositeStorage{queries: queries}
}

func (s *fositeStorage) GetClient(ctx context.Context, id string) (fosite.Client, error) {
	entityID, err := entityULIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	client, err := s.queries.GetOIDCClientByClientID(ctx, generated.GetOIDCClientByClientIDParams{
		EntityID: entityID,
		ClientID: id,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fosite.ErrNotFound
		}
		return nil, err
	}
	if client.Status != "active" {
		return nil, fosite.ErrNotFound
	}

	return &fosite.DefaultOpenIDConnectClient{
		DefaultClient: &fosite.DefaultClient{
			ID:            client.ClientID,
			Secret:        []byte(client.ClientSecretHash.String),
			RedirectURIs:  client.RedirectUris,
			GrantTypes:    client.GrantTypes,
			ResponseTypes: client.ResponseTypes,
			Scopes:        client.AllowedScopes,
			Public:        !client.ClientSecretHash.Valid,
		},
		TokenEndpointAuthMethod: tokenEndpointAuthMethod(client.ClientSecretHash.Valid),
	}, nil
}

func (s *fositeStorage) ClientAssertionJWTValid(context.Context, string) error {
	return unsupportedFositeStorage("client assertion jwt")
}

func (s *fositeStorage) SetClientAssertionJWT(context.Context, string, time.Time) error {
	return unsupportedFositeStorage("client assertion jwt")
}

func entityULIDFromContext(ctx context.Context) (string, error) {
	value, ok := ctx.Value(entityIDContextKey{}).(string)
	if !ok || value == "" {
		return "", fmt.Errorf("entity id missing from fosite context")
	}
	var entityID string
	entityID = value
	return entityID, nil
}

func tokenEndpointAuthMethod(hasSecret bool) string {
	if hasSecret {
		return "client_secret_basic"
	}
	return "none"
}

func unsupportedFositeStorage(feature string) error {
	return fmt.Errorf("%s is not supported in this OIDC milestone", feature)
}

func (s *fositeStorage) CreateAuthorizeCodeSession(context.Context, string, fosite.Requester) error {
	return unsupportedFositeStorage("fosite-managed authorization code creation")
}

func (s *fositeStorage) GetAuthorizeCodeSession(context.Context, string, fosite.Session) (fosite.Requester, error) {
	return nil, unsupportedFositeStorage("fosite-managed authorization code lookup")
}

func (s *fositeStorage) InvalidateAuthorizeCodeSession(context.Context, string) error {
	return unsupportedFositeStorage("fosite-managed authorization code invalidation")
}

func (s *fositeStorage) CreateAccessTokenSession(context.Context, string, fosite.Requester) error {
	return unsupportedFositeStorage("fosite-managed access token creation")
}

func (s *fositeStorage) GetAccessTokenSession(context.Context, string, fosite.Session) (fosite.Requester, error) {
	return nil, unsupportedFositeStorage("fosite-managed access token lookup")
}

func (s *fositeStorage) DeleteAccessTokenSession(context.Context, string) error {
	return unsupportedFositeStorage("fosite-managed access token deletion")
}

func (s *fositeStorage) CreateRefreshTokenSession(context.Context, string, string, fosite.Requester) error {
	return unsupportedFositeStorage("refresh tokens")
}

func (s *fositeStorage) GetRefreshTokenSession(context.Context, string, fosite.Session) (fosite.Requester, error) {
	return nil, unsupportedFositeStorage("refresh tokens")
}

func (s *fositeStorage) DeleteRefreshTokenSession(context.Context, string) error {
	return unsupportedFositeStorage("refresh tokens")
}

func (s *fositeStorage) RotateRefreshToken(context.Context, string, string) error {
	return unsupportedFositeStorage("refresh tokens")
}

func (s *fositeStorage) RevokeRefreshToken(context.Context, string) error {
	return unsupportedFositeStorage("refresh tokens")
}

func (s *fositeStorage) RevokeAccessToken(context.Context, string) error {
	return unsupportedFositeStorage("access token revocation")
}

func (s *fositeStorage) GetPKCERequestSession(context.Context, string, fosite.Session) (fosite.Requester, error) {
	return nil, unsupportedFositeStorage("fosite-managed pkce lookup")
}

func (s *fositeStorage) CreatePKCERequestSession(context.Context, string, fosite.Requester) error {
	return unsupportedFositeStorage("fosite-managed pkce creation")
}

func (s *fositeStorage) DeletePKCERequestSession(context.Context, string) error {
	return unsupportedFositeStorage("fosite-managed pkce deletion")
}
