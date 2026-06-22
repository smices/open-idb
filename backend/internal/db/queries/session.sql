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

-- name: MarkExpiredSessions :execrows
UPDATE sessions
SET status = 'expired'
WHERE status = 'active' AND expires_at < now();

-- name: GetSessionByID :one
SELECT id, entity_id, user_id, device_id, ip, user_agent, login_method, status, created_at, expires_at
FROM sessions
WHERE entity_id = $1 AND id = $2;

-- name: GetActiveSessionIdentity :one
SELECT
    s.id,
    s.entity_id,
    s.user_id,
    u.username,
    u.display_name,
    COALESCE(lc.must_change_password, false)::boolean AS must_change_password,
    COALESCE(lc.weak_password, false)::boolean AS weak_password,
    s.expires_at
FROM sessions s
JOIN users u ON u.entity_id = s.entity_id AND u.id = s.user_id
LEFT JOIN local_credentials lc ON lc.entity_id = s.entity_id AND lc.user_id = s.user_id
WHERE s.id = $1
  AND s.status = 'active'
  AND s.expires_at > now()
  AND u.lifecycle_status = 'active';

-- name: ListSessionsByUser :many
SELECT id, entity_id, user_id, device_id, ip, user_agent, login_method, status, created_at, expires_at
FROM sessions
WHERE entity_id = $1 AND user_id = $2
ORDER BY created_at DESC
LIMIT $3;
