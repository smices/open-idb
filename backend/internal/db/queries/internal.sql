-- SPDX-License-Identifier: MIT

-- === Token Introspection ===

-- name: GetOAuthTokenByHash :one
SELECT id, entity_id, user_id, client_id, token_type, token_hash, scopes, revoked_at, expires_at
FROM oauth_tokens
WHERE entity_id = $1 AND token_hash = $2;

-- name: RevokeOAuthToken :exec
UPDATE oauth_tokens
SET revoked_at = now()
WHERE entity_id = $1 AND token_hash = $2;

-- name: RevokeOAuthTokensByUser :exec
UPDATE oauth_tokens
SET revoked_at = now()
WHERE entity_id = $1 AND user_id = $2 AND revoked_at IS NULL;

-- name: RevokeOAuthTokensByClient :exec
UPDATE oauth_tokens
SET revoked_at = now()
WHERE entity_id = $1 AND client_id = $2 AND revoked_at IS NULL;

-- === User Access Check ===

-- name: GetUserApplicationAccess :many
SELECT DISTINCT a.id, a.entity_id, a.name, a.type, a.status
FROM applications a
JOIN application_assignments aa ON aa.entity_id = a.entity_id AND aa.application_id = a.id
WHERE aa.entity_id = $1
  AND aa.effect = 'allow'
  AND (
    (aa.subject_type = 'user' AND aa.subject_id = $2)
    OR
    (aa.subject_type = 'role' AND aa.subject_id IN (
        SELECT ur.role_id FROM user_roles ur WHERE ur.entity_id = $1 AND ur.user_id = $2
    ))
    OR
    (aa.subject_type = 'group' AND aa.subject_id IN (
        SELECT gm.group_id FROM group_members gm WHERE gm.entity_id = $1 AND gm.user_id = $2
    ))
  )
  AND NOT EXISTS (
    SELECT 1 FROM application_assignments deny
    WHERE deny.entity_id = $1
      AND deny.application_id = a.id
      AND deny.effect = 'deny'
      AND (
        (deny.subject_type = 'user' AND deny.subject_id = $2)
        OR
        (deny.subject_type = 'role' AND deny.subject_id IN (
            SELECT ur.role_id FROM user_roles ur WHERE ur.entity_id = $1 AND ur.user_id = $2
        ))
      )
  );

-- name: GetUserPermissions :many
SELECT DISTINCT p.id, p.entity_id, p.code, p.name, p.type
FROM permissions p
JOIN role_permissions rp ON rp.entity_id = p.entity_id AND rp.permission_id = p.id
JOIN user_roles ur ON ur.entity_id = rp.entity_id AND ur.role_id = rp.role_id
WHERE ur.entity_id = $1 AND ur.user_id = $2;

-- name: GetUserResourceScopes :many
SELECT DISTINCT rs.id, rs.entity_id, rs.type, rs.key, rs.name, rrs.effect
FROM resource_scopes rs
JOIN role_resource_scopes rrs ON rrs.entity_id = rs.entity_id AND rrs.resource_scope_id = rs.id
JOIN user_roles ur ON ur.entity_id = rrs.entity_id AND ur.role_id = rrs.role_id
WHERE ur.entity_id = $1 AND ur.user_id = $2;

-- name: GetUserLifecycleStatus :one
SELECT lifecycle_status
FROM users
WHERE entity_id = $1 AND id = $2;
