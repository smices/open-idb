-- SPDX-License-Identifier: MIT

-- === Sessions ===

-- name: CreateSession :one
INSERT INTO sessions (entity_id, user_id, device_id, ip, user_agent, login_method, status, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, 'active', $7)
RETURNING id, entity_id, user_id, device_id, ip, user_agent, login_method, status, created_at, expires_at;

-- name: RevokeSession :exec
UPDATE sessions
SET status = 'revoked'
WHERE entity_id = $1 AND id = $2 AND status = 'active';

-- name: GetSessionByID :one
SELECT id, entity_id, user_id, device_id, ip, user_agent, login_method, status, created_at, expires_at
FROM sessions
WHERE entity_id = $1 AND id = $2;

-- name: ListSessionsByUser :many
SELECT id, entity_id, user_id, device_id, ip, user_agent, login_method, status, created_at, expires_at
FROM sessions
WHERE entity_id = $1 AND user_id = $2
ORDER BY created_at DESC
LIMIT $3;
