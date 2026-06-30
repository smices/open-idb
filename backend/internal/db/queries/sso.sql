-- SPDX-License-Identifier: MIT

-- name: CreateApplication :one
INSERT INTO applications (entity_id, name, type, status)
VALUES ($1, $2, $3, 'active')
RETURNING id, entity_id, name, type, status, created_at, updated_at;

-- name: CreateOIDCClient :one
INSERT INTO oidc_clients (
    entity_id,
    application_id,
    client_id,
    client_secret_hash,
    redirect_uris,
    allowed_scopes,
    grant_types,
    response_types,
    pkce_required,
    workplace_provider,
    workplace_app_id,
    workplace_app_secret,
    status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 'active'
)
RETURNING id, entity_id, application_id, client_id, client_secret_hash, redirect_uris, allowed_scopes, grant_types, response_types, pkce_required, workplace_provider, workplace_app_id, workplace_app_secret, status, created_at, updated_at;

-- name: GetOIDCClientByClientID :one
SELECT id, entity_id, application_id, client_id, client_secret_hash, redirect_uris, allowed_scopes, grant_types, response_types, pkce_required, workplace_provider, workplace_app_id, workplace_app_secret, status, created_at, updated_at
FROM oidc_clients
WHERE entity_id = $1 AND client_id = $2;

-- name: CreateAuthorizationCode :one
INSERT INTO oauth_authorization_codes (
    entity_id,
    client_id,
    user_id,
    code_hash,
    redirect_uri,
    scopes,
    code_challenge,
    code_challenge_method,
    nonce,
    expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, 'S256', $8, $9
)
RETURNING id, entity_id, client_id, user_id, code_hash, redirect_uri, scopes, code_challenge, code_challenge_method, nonce, used_at, expires_at, created_at;

-- name: GetAuthorizationCode :one
SELECT id, entity_id, client_id, user_id, code_hash, redirect_uri, scopes, code_challenge, code_challenge_method, nonce, used_at, expires_at, created_at
FROM oauth_authorization_codes
WHERE entity_id = $1 AND code_hash = $2;

-- name: MarkAuthorizationCodeUsed :one
UPDATE oauth_authorization_codes
SET used_at = now()
WHERE entity_id = $1
  AND code_hash = $2
  AND used_at IS NULL
  AND expires_at > now()
RETURNING id, entity_id, client_id, user_id, code_hash, redirect_uri, scopes, code_challenge, code_challenge_method, nonce, used_at, expires_at, created_at;

-- name: CreateOAuthToken :one
INSERT INTO oauth_tokens (entity_id, user_id, client_id, token_type, token_hash, scopes, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, entity_id, user_id, client_id, token_type, token_hash, scopes, revoked_at, expires_at, created_at;

-- name: DeleteExpiredAuthorizationCodes :execrows
DELETE FROM oauth_authorization_codes
WHERE expires_at < now();

-- name: DeleteExpiredOAuthTokens :execrows
DELETE FROM oauth_tokens
WHERE expires_at < now();
