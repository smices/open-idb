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
SELECT
    a.id,
    a.entity_id,
    a.actor_user_id,
    a.actor_type,
    a.action,
    a.resource_type,
    a.resource_id,
    a.before_state,
    a.after_state,
    a.ip,
    a.user_agent,
    a.trace_id,
    a.created_at,
    COALESCE(NULLIF(admin_actor.display_name, ''), NULLIF(admin_actor.username, ''), NULLIF(actor.display_name, ''), NULLIF(actor.username, ''), '') AS actor_display_name,
    COALESCE(
        NULLIF(resource_user.display_name, ''),
        NULLIF(resource_user.username, ''),
        NULLIF(directory_user.name, ''),
        NULLIF(application.name, ''),
        NULLIF(oidc_client.client_id, ''),
        NULLIF(admin_resource.display_name, ''),
        NULLIF(admin_resource.username, ''),
        NULLIF(role.name, ''),
        NULLIF(permission.name, ''),
        NULLIF(identity_source.name, ''),
        NULLIF(sync_job.trace_id, ''),
        NULLIF(a.after_state ->> 'display_name', ''),
        NULLIF(a.after_state ->> 'username', ''),
        NULLIF(a.after_state ->> 'name', ''),
        NULLIF(a.after_state ->> 'client_id', ''),
        NULLIF(a.before_state ->> 'display_name', ''),
        NULLIF(a.before_state ->> 'username', ''),
        NULLIF(a.before_state ->> 'name', ''),
        NULLIF(a.before_state ->> 'client_id', ''),
        ''
    ) AS resource_display_name,
    CASE
        WHEN a.action LIKE '%.failed' OR lower(COALESCE(a.after_state ->> 'status', '')) = 'failed' THEN 'failed'
        ELSE 'succeeded'
    END AS outcome
FROM audit_logs a
LEFT JOIN users actor
    ON actor.entity_id = a.entity_id AND actor.id = a.actor_user_id
LEFT JOIN admin_users admin_actor
    ON a.actor_type = 'admin' AND admin_actor.id = a.actor_user_id
LEFT JOIN users resource_user
    ON a.resource_type = 'user' AND resource_user.entity_id = a.entity_id AND resource_user.id = a.resource_id
LEFT JOIN directory_users directory_user
    ON a.resource_type = 'directory_user' AND directory_user.entity_id = a.entity_id AND directory_user.id = a.resource_id
LEFT JOIN applications application
    ON a.resource_type = 'application' AND application.entity_id = a.entity_id AND application.id = a.resource_id
LEFT JOIN oidc_clients oidc_client
    ON a.resource_type = 'oidc_client' AND oidc_client.entity_id = a.entity_id AND oidc_client.id = a.resource_id
LEFT JOIN admin_users admin_resource
    ON a.resource_type = 'admin_user' AND admin_resource.id = a.resource_id
LEFT JOIN roles role
    ON a.resource_type = 'role' AND role.entity_id = a.entity_id AND role.id = a.resource_id
LEFT JOIN permissions permission
    ON a.resource_type = 'permission' AND permission.entity_id = a.entity_id AND permission.id = a.resource_id
LEFT JOIN identity_sources identity_source
    ON a.resource_type = 'identity_source' AND identity_source.entity_id = a.entity_id AND identity_source.id = a.resource_id
LEFT JOIN sync_jobs sync_job
    ON a.resource_type = 'sync_job' AND sync_job.entity_id = a.entity_id AND sync_job.id = a.resource_id
WHERE a.entity_id = $1
  AND (sqlc.narg('action')::text IS NULL OR a.action = sqlc.narg('action')::text)
  AND (sqlc.narg('resource_type')::text IS NULL OR a.resource_type = sqlc.narg('resource_type')::text)
  AND (sqlc.narg('actor_type')::text IS NULL OR a.actor_type = sqlc.narg('actor_type')::text)
ORDER BY a.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountAuditLogs :one
SELECT count(*)::bigint
FROM audit_logs
WHERE entity_id = $1
  AND (sqlc.narg('action')::text IS NULL OR action = sqlc.narg('action')::text)
  AND (sqlc.narg('resource_type')::text IS NULL OR resource_type = sqlc.narg('resource_type')::text)
  AND (sqlc.narg('actor_type')::text IS NULL OR actor_type = sqlc.narg('actor_type')::text);

-- name: DeleteAuditLog :execrows
DELETE FROM audit_logs
WHERE entity_id = sqlc.arg('entity_id')
  AND id = sqlc.arg('id');

-- name: ClearAuditLogs :execrows
DELETE FROM audit_logs
WHERE entity_id = sqlc.arg('entity_id');
