// SPDX-License-Identifier: MIT

package sso

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/smices/open-idb/internal/db/generated"
)

type DatabaseQuerier struct {
	queries *generated.Queries
}

func NewDatabaseQuerier(queries *generated.Queries) *DatabaseQuerier {
	return &DatabaseQuerier{queries: queries}
}

func (q *DatabaseQuerier) LookupToken(ctx context.Context, entityID string, tokenHash string) (SSOTokenLookup, error) {
	token, err := q.queries.GetSSOTokenByHash(ctx, generated.GetSSOTokenByHashParams{
		EntityID:  entityID,
		TokenHash: tokenHash,
	})
	if err != nil {
		return SSOTokenLookup{}, err
	}
	return tokenLookupFromRow(token), nil
}

func (q *DatabaseQuerier) LookupTokenGlobally(ctx context.Context, tokenHash string) (SSOTokenLookup, error) {
	token, err := q.queries.GetSSOTokenByHashGlobally(ctx, tokenHash)
	if err != nil {
		return SSOTokenLookup{}, err
	}
	return tokenLookupFromRow(token), nil
}

func tokenLookupFromRow(token generated.OauthToken) SSOTokenLookup {
	var revokedAt, expiresAt *time.Time
	if token.RevokedAt.Valid {
		revokedAt = &token.RevokedAt.Time
	}
	if token.ExpiresAt.Valid {
		expiresAt = &token.ExpiresAt.Time
	}

	return SSOTokenLookup{
		ID:        token.ID,
		EntityID:  token.EntityID,
		UserID:    token.UserID,
		ClientID:  token.ClientID,
		TokenType: token.TokenType,
		Scopes:    token.Scopes,
		RevokedAt: revokedAt,
		ExpiresAt: expiresAt,
	}
}

func (q *DatabaseQuerier) MarkTokenRevoked(ctx context.Context, entityID string, tokenHash string) error {
	return q.queries.RevokeSSOTokenByHash(ctx, generated.RevokeSSOTokenByHashParams{
		EntityID:  entityID,
		TokenHash: tokenHash,
	})
}

func (q *DatabaseQuerier) MarkTokenRevokedGlobally(ctx context.Context, tokenHash string) (string, error) {
	return q.queries.RevokeSSOTokenByHashGlobally(ctx, tokenHash)
}

func (q *DatabaseQuerier) FetchUserInfo(ctx context.Context, entityID string, userID string) (UserInfoClaims, error) {
	user, err := q.queries.GetUserInfoForClaims(ctx, generated.GetUserInfoForClaimsParams{
		EntityID: entityID,
		ID:       userID,
	})
	if err != nil {
		return UserInfoClaims{}, err
	}

	return UserInfoClaims{
		ID:          user.ID,
		EntityID:    user.EntityID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Email:       stringFromText(user.Email),
		Phone:       stringFromText(user.Phone),
		AvatarURL:   stringFromText(user.AvatarUrl),
		Locale:      stringFromText(user.Locale),
	}, nil
}

func stringFromText(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}
