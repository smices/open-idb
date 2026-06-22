-- SPDX-License-Identifier: MIT

-- === SSO Token Introspection & Revocation ===
-- These queries support the /oauth2/userinfo and /oauth2/revoke endpoints.
-- Named with SSO prefix to avoid collisions with internal.sql queries.

-- name: GetSSOTokenByHash :one
SELECT id, entity_id, user_id, client_id, token_type, token_hash, scopes, revoked_at, expires_at, created_at
FROM oauth_tokens
WHERE entity_id = $1 AND token_hash = $2;

-- name: RevokeSSOTokenByHash :exec
UPDATE oauth_tokens
SET revoked_at = now()
WHERE entity_id = $1 AND token_hash = $2;

-- name: GetUserInfoForClaims :one
SELECT id, entity_id, username, display_name, email, phone, avatar_url, locale
FROM users
WHERE entity_id = $1 AND id = $2;
