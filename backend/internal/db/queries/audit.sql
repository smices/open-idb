-- SPDX-License-Identifier: MIT

-- name: CreateAuditLog :one
INSERT INTO audit_logs (
    entity_id,
    actor_user_id,
    actor_type,
    action,
    resource_type,
    resource_id,
    before_state,
    after_state,
    ip,
    user_agent,
    trace_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
RETURNING id, entity_id, actor_user_id, actor_type, action, resource_type, resource_id, before_state, after_state, ip, user_agent, trace_id, created_at;

-- name: ListAuditLogs :many
SELECT id, entity_id, actor_user_id, actor_type, action, resource_type, resource_id, before_state, after_state, ip, user_agent, trace_id, created_at
FROM audit_logs
WHERE entity_id = $1
  AND (sqlc.narg('action')::text IS NULL OR action = sqlc.narg('action')::text)
  AND (sqlc.narg('resource_type')::text IS NULL OR resource_type = sqlc.narg('resource_type')::text)
  AND (sqlc.narg('actor_type')::text IS NULL OR actor_type = sqlc.narg('actor_type')::text)
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountAuditLogs :one
SELECT count(*)::bigint
FROM audit_logs
WHERE entity_id = $1
  AND (sqlc.narg('action')::text IS NULL OR action = sqlc.narg('action')::text)
  AND (sqlc.narg('resource_type')::text IS NULL OR resource_type = sqlc.narg('resource_type')::text)
  AND (sqlc.narg('actor_type')::text IS NULL OR actor_type = sqlc.narg('actor_type')::text);
