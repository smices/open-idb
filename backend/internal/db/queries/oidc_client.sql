-- SPDX-License-Identifier: MIT

-- name: ListOIDCClients :many
SELECT id, entity_id, application_id, client_id, redirect_uris, allowed_scopes, grant_types, response_types, pkce_required, status, created_at, updated_at
FROM oidc_clients
WHERE entity_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountOIDCClients :one
SELECT count(*)::bigint
FROM oidc_clients
WHERE entity_id = $1;

-- name: GetOIDCClientByID :one
SELECT id, entity_id, application_id, client_id, client_secret_hash, redirect_uris, allowed_scopes, grant_types, response_types, pkce_required, status, created_at, updated_at
FROM oidc_clients
WHERE entity_id = $1 AND id = $2;

-- name: UpdateOIDCClient :one
UPDATE oidc_clients
SET redirect_uris = COALESCE(sqlc.narg('redirect_uris'), redirect_uris),
    allowed_scopes = COALESCE(sqlc.narg('allowed_scopes'), allowed_scopes),
    grant_types = COALESCE(sqlc.narg('grant_types'), grant_types),
    response_types = COALESCE(sqlc.narg('response_types'), response_types),
    pkce_required = COALESCE(sqlc.narg('pkce_required'), pkce_required),
    status = COALESCE(sqlc.narg('status'), status),
    updated_at = now()
WHERE entity_id = $1 AND id = $2
RETURNING id, entity_id, application_id, client_id, redirect_uris, allowed_scopes, grant_types, response_types, pkce_required, status, created_at, updated_at;

-- name: DeleteOIDCClient :exec
DELETE FROM oidc_clients
WHERE entity_id = $1 AND id = $2;

-- name: RotateOIDCClientSecret :one
UPDATE oidc_clients
SET client_secret_hash = $3, updated_at = now()
WHERE entity_id = $1 AND id = $2
RETURNING id, entity_id, application_id, client_id, redirect_uris, allowed_scopes, grant_types, response_types, pkce_required, status, created_at, updated_at;
