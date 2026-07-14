// SPDX-License-Identifier: MIT

package sso

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/smices/open-idb/internal/db/generated"
)

type ClientStore interface {
	GetOIDCClientByClientID(ctx context.Context, arg generated.GetOIDCClientByClientIDParams) (generated.OidcClient, error)
}

type UserAuthorizationStore interface {
	GetUserLifecycleStatus(ctx context.Context, arg generated.GetUserLifecycleStatusParams) (string, error)
	GetUserClaimsForToken(ctx context.Context, arg generated.GetUserClaimsForTokenParams) (generated.GetUserClaimsForTokenRow, error)
	HasApplicationAccess(ctx context.Context, arg generated.HasApplicationAccessParams) (pgtype.Bool, error)
	GetUserRolesForToken(ctx context.Context, arg generated.GetUserRolesForTokenParams) ([]string, error)
	GetPermissionsVersion(ctx context.Context, entityID string) (int64, error)
	GetResourceScopesVersion(ctx context.Context, entityID string) (int64, error)
}

type AuthorizationCodeStore interface {
	CreateAuthorizationCode(ctx context.Context, arg generated.CreateAuthorizationCodeParams) (generated.OauthAuthorizationCode, error)
	GetAuthorizationCode(ctx context.Context, arg generated.GetAuthorizationCodeParams) (generated.OauthAuthorizationCode, error)
	GetAuthorizationCodeByHash(ctx context.Context, codeHash string) (generated.OauthAuthorizationCode, error)
	MarkAuthorizationCodeUsed(ctx context.Context, arg generated.MarkAuthorizationCodeUsedParams) (generated.OauthAuthorizationCode, error)
	FinalizeAuthorizationCodeExchange(ctx context.Context, arg generated.FinalizeAuthorizationCodeExchangeParams) (generated.FinalizeAuthorizationCodeExchangeRow, error)
}

type OAuthTokenStore interface {
	CreateOAuthToken(ctx context.Context, arg generated.CreateOAuthTokenParams) (generated.OauthToken, error)
}

// Store is the authoritative persistence boundary for the OAuth/OIDC code
// flow. PostgreSQL is the only production implementation today; this interface
// keeps service logic from depending on a concrete sqlc query container.
type Store interface {
	ClientStore
	UserAuthorizationStore
	AuthorizationCodeStore
	OAuthTokenStore
}

// TokenLookupStore handles token introspection, revocation, and userinfo reads.
// It is separate from Store because those endpoints operate on bearer tokens,
// not on the authorization-code exchange workflow.
type TokenLookupStore interface {
	LookupToken(ctx context.Context, entityID string, tokenHash string) (SSOTokenLookup, error)
	MarkTokenRevoked(ctx context.Context, entityID string, tokenHash string) error
	FetchUserInfo(ctx context.Context, entityID string, userID string) (UserInfoClaims, error)
}

// GlobalTokenLookupStore is an optional capability used when standard OAuth
// clients do not send the IdBridge-specific entity header. Implementations
// must reject token hashes that exist in more than one entity.
type GlobalTokenLookupStore interface {
	LookupTokenGlobally(ctx context.Context, tokenHash string) (SSOTokenLookup, error)
	MarkTokenRevokedGlobally(ctx context.Context, tokenHash string) (string, error)
}
