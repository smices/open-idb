-- SPDX-License-Identifier: MIT

-- === Roles ===

-- name: CreateRole :one
INSERT INTO roles (entity_id, name, code, description)
VALUES ($1, $2, $3, $4)
RETURNING id, entity_id, name, code, description, created_at, updated_at;

-- name: ListRoles :many
SELECT id, entity_id, name, code, description, created_at, updated_at
FROM roles
WHERE entity_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountRoles :one
SELECT count(*)::bigint
FROM roles
WHERE entity_id = $1;

-- name: GetRoleByID :one
SELECT id, entity_id, name, code, description, created_at, updated_at
FROM roles
WHERE entity_id = $1 AND id = $2;

-- name: GetRoleByCode :one
SELECT id, entity_id, name, code, description, created_at, updated_at
FROM roles
WHERE entity_id = $1 AND code = $2;

-- name: UpdateRole :one
UPDATE roles
SET name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    updated_at = now()
WHERE entity_id = $1 AND id = $2
RETURNING id, entity_id, name, code, description, created_at, updated_at;

-- name: DeleteRole :exec
DELETE FROM roles
WHERE entity_id = $1 AND id = $2;

-- === Role Permissions ===

-- name: ListRolePermissions :many
SELECT p.id, p.entity_id, p.code, p.name, p.type, p.created_at, p.updated_at
FROM permissions p
JOIN role_permissions rp ON rp.entity_id = p.entity_id AND rp.permission_id = p.id
WHERE rp.entity_id = $1 AND rp.role_id = $2;

-- name: AddPermissionToRole :exec
INSERT INTO role_permissions (entity_id, role_id, permission_id)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;

-- name: RemovePermissionFromRole :exec
DELETE FROM role_permissions
WHERE entity_id = $1 AND role_id = $2 AND permission_id = $3;

-- === User Roles ===

-- name: AssignRoleToUser :exec
INSERT INTO user_roles (entity_id, user_id, role_id)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;

-- name: ListUserRoles :many
SELECT r.id, r.entity_id, r.name, r.code, r.description, r.created_at, r.updated_at
FROM roles r
JOIN user_roles ur ON ur.entity_id = r.entity_id AND ur.role_id = r.id
WHERE ur.entity_id = $1 AND ur.user_id = $2;

-- name: RemoveRoleFromUser :exec
DELETE FROM user_roles
WHERE entity_id = $1 AND user_id = $2 AND role_id = $3;

-- === Permissions ===

-- name: ListPermissions :many
SELECT id, entity_id, code, name, type, created_at, updated_at
FROM permissions
WHERE entity_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountPermissions :one
SELECT count(*)::bigint
FROM permissions
WHERE entity_id = $1;

-- name: GetPermissionByID :one
SELECT id, entity_id, code, name, type, created_at, updated_at
FROM permissions
WHERE entity_id = $1 AND id = $2;

-- name: GetPermissionByCode :one
SELECT id, entity_id, code, name, type, created_at, updated_at
FROM permissions
WHERE entity_id = $1 AND code = $2;

-- name: UpdatePermission :one
UPDATE permissions
SET name = COALESCE(sqlc.narg('name'), name),
    updated_at = now()
WHERE entity_id = $1 AND id = $2
RETURNING id, entity_id, code, name, type, created_at, updated_at;

-- name: DeletePermission :exec
DELETE FROM permissions
WHERE entity_id = $1 AND id = $2;

-- name: CreatePermission :one
INSERT INTO permissions (entity_id, code, name, type)
VALUES ($1, $2, $3, $4)
RETURNING id, entity_id, code, name, type, created_at, updated_at;

-- === Resource Scopes ===

-- name: ListResourceScopes :many
SELECT id, entity_id, type, key, name, created_at, updated_at
FROM resource_scopes
WHERE entity_id = $1
  AND (sqlc.narg('type')::text IS NULL OR type = sqlc.narg('type')::text)
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountResourceScopes :one
SELECT count(*)::bigint
FROM resource_scopes
WHERE entity_id = $1
  AND (sqlc.narg('type')::text IS NULL OR type = sqlc.narg('type')::text);

-- name: GetResourceScopeByID :one
SELECT id, entity_id, type, key, name, created_at, updated_at
FROM resource_scopes
WHERE entity_id = $1 AND id = $2;

-- name: CreateResourceScope :one
INSERT INTO resource_scopes (entity_id, type, key, name)
VALUES ($1, $2, $3, $4)
RETURNING id, entity_id, type, key, name, created_at, updated_at;

-- name: UpdateResourceScope :one
UPDATE resource_scopes
SET name = COALESCE(sqlc.narg('name'), name),
    updated_at = now()
WHERE entity_id = $1 AND id = $2
RETURNING id, entity_id, type, key, name, created_at, updated_at;

-- name: DeleteResourceScope :exec
DELETE FROM resource_scopes
WHERE entity_id = $1 AND id = $2;

-- === Role Resource Scopes ===

-- name: ListRoleResourceScopes :many
SELECT rs.id, rs.entity_id, rs.type, rs.key, rs.name, rrs.effect
FROM resource_scopes rs
JOIN role_resource_scopes rrs ON rrs.entity_id = rs.entity_id AND rrs.resource_scope_id = rs.id
WHERE rrs.entity_id = $1 AND rrs.role_id = $2;

-- name: AddResourceScopeToRole :exec
INSERT INTO role_resource_scopes (entity_id, role_id, resource_scope_id, effect)
VALUES ($1, $2, $3, $4)
ON CONFLICT DO NOTHING;

-- name: RemoveResourceScopeFromRole :exec
DELETE FROM role_resource_scopes
WHERE entity_id = $1 AND role_id = $2 AND resource_scope_id = $3;

-- === Application Assignments ===

-- name: CreateApplicationAssignment :one
INSERT INTO application_assignments (entity_id, application_id, subject_type, subject_id, effect)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, entity_id, application_id, subject_type, subject_id, effect, created_at;

-- name: HasApplicationAccess :one
SELECT
    EXISTS (
        SELECT 1
        FROM application_assignments aa
        WHERE aa.entity_id = $1
          AND aa.application_id = $2
          AND aa.effect = 'allow'
          AND (
            (aa.subject_type = 'user' AND aa.subject_id = $3)
            OR
            (aa.subject_type = 'role' AND aa.subject_id IN (
                SELECT ur.role_id
                FROM user_roles ur
                WHERE ur.entity_id = $1 AND ur.user_id = $3
            ))
          )
    )
    AND NOT EXISTS (
        SELECT 1
        FROM application_assignments aa
        WHERE aa.entity_id = $1
          AND aa.application_id = $2
          AND aa.effect = 'deny'
          AND (
            (aa.subject_type = 'user' AND aa.subject_id = $3)
            OR
            (aa.subject_type = 'role' AND aa.subject_id IN (
                SELECT ur.role_id
                FROM user_roles ur
                WHERE ur.entity_id = $1 AND ur.user_id = $3
            ))
          )
    ) AS has_access;

-- name: ListApplicationAssignments :many
SELECT id, entity_id, application_id, subject_type, subject_id, effect, created_at
FROM application_assignments
WHERE entity_id = $1 AND application_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountApplicationAssignments :one
SELECT count(*)::bigint
FROM application_assignments
WHERE entity_id = $1 AND application_id = $2;

-- name: DeleteApplicationAssignment :exec
DELETE FROM application_assignments
WHERE entity_id = $1 AND id = $2;
