-- SPDX-License-Identifier: MIT

-- name: ListIdentitySources :many
SELECT id, entity_id, type, name, status, sync_enabled, created_at
FROM identity_sources
WHERE entity_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountIdentitySources :one
SELECT count(*)::bigint
FROM identity_sources
WHERE entity_id = $1;

-- name: GetIdentitySourceByID :one
SELECT id, entity_id, type, name, status, sync_enabled, created_at
FROM identity_sources
WHERE entity_id = $1 AND id = $2;

-- name: UpdateIdentitySource :one
UPDATE identity_sources
SET name = COALESCE(sqlc.narg('name'), name),
    status = COALESCE(sqlc.narg('status'), status),
    sync_enabled = COALESCE(sqlc.narg('sync_enabled'), sync_enabled)
WHERE entity_id = $1 AND id = $2
RETURNING id, entity_id, type, name, status, sync_enabled, created_at;

-- name: DeleteIdentitySource :exec
DELETE FROM identity_sources
WHERE entity_id = $1 AND id = $2;
