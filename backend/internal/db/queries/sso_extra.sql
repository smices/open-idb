-- SPDX-License-Identifier: MIT

-- === SSO Token Introspection & Revocation ===
-- These queries support the /oauth2/userinfo and /oauth2/revoke endpoints.
-- Named with SSO prefix to avoid collisions with internal.sql queries.

-- name: GetSSOTokenByHash :one
SELECT id, entity_id, user_id, client_id, token_type, token_hash, scopes, revoked_at, expires_at, created_at
FROM oauth_tokens
WHERE entity_id = $1 AND token_hash = $2;

-- name: GetSSOTokenByHashGlobally :one
SELECT token.id, token.entity_id, token.user_id, token.client_id, token.token_type, token.token_hash, token.scopes, token.revoked_at, token.expires_at, token.created_at
FROM oauth_tokens token
WHERE token.token_hash = $1
  AND NOT EXISTS (
      SELECT 1
      FROM oauth_tokens duplicate
      WHERE duplicate.token_hash = token.token_hash
        AND duplicate.entity_id <> token.entity_id
  );

-- name: RevokeSSOTokenByHash :exec
UPDATE oauth_tokens
SET revoked_at = now()
WHERE entity_id = $1 AND token_hash = $2;

-- name: RevokeSSOTokenByHashGlobally :one
UPDATE oauth_tokens token
SET revoked_at = now()
WHERE token.token_hash = $1
  AND NOT EXISTS (
      SELECT 1
      FROM oauth_tokens duplicate
      WHERE duplicate.token_hash = token.token_hash
        AND duplicate.entity_id <> token.entity_id
  )
RETURNING token.entity_id;

-- name: GetUserInfoForClaims :one
SELECT id, entity_id, username, display_name, email, phone, avatar_url, locale
FROM users
WHERE entity_id = $1 AND id = $2;
