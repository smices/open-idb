-- SPDX-License-Identifier: MIT

-- === Organizations ===

-- name: CreateOrganization :one
INSERT INTO organizations (entity_id, name, parent_id)
VALUES ($1, $2, $3)
RETURNING id, entity_id, name, parent_id, created_at, updated_at;

-- name: ListOrganizations :many
SELECT id, entity_id, name, parent_id, created_at, updated_at
FROM organizations
WHERE entity_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountOrganizations :one
SELECT count(*)::bigint
FROM organizations
WHERE entity_id = $1;

-- name: GetOrganizationByID :one
SELECT id, entity_id, name, parent_id, created_at, updated_at
FROM organizations
WHERE entity_id = $1 AND id = $2;

-- name: UpdateOrganization :one
UPDATE organizations
SET name = COALESCE(sqlc.narg('name'), name),
    parent_id = COALESCE(sqlc.narg('parent_id'), parent_id),
    updated_at = now()
WHERE entity_id = $1 AND id = $2
RETURNING id, entity_id, name, parent_id, created_at, updated_at;

-- name: DeleteOrganization :exec
DELETE FROM organizations
WHERE entity_id = $1 AND id = $2;

-- === Departments ===

-- name: CreateDepartment :one
INSERT INTO departments (entity_id, organization_id, name, parent_id, source_id, external_department_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, entity_id, organization_id, name, parent_id, source_id, external_department_id, created_at, updated_at;

-- name: ListDepartments :many
SELECT id, entity_id, organization_id, name, parent_id, source_id, external_department_id, created_at, updated_at
FROM departments
WHERE entity_id = $1
  AND (sqlc.narg('organization_id')::text IS NULL OR organization_id = sqlc.narg('organization_id')::text)
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountDepartments :one
SELECT count(*)::bigint
FROM departments
WHERE entity_id = $1
  AND (sqlc.narg('organization_id')::text IS NULL OR organization_id = sqlc.narg('organization_id')::text);

-- name: GetDepartmentByID :one
SELECT id, entity_id, organization_id, name, parent_id, source_id, external_department_id, created_at, updated_at
FROM departments
WHERE entity_id = $1 AND id = $2;

-- name: UpdateDepartment :one
UPDATE departments
SET name = COALESCE(sqlc.narg('name'), name),
    parent_id = COALESCE(sqlc.narg('parent_id'), parent_id),
    updated_at = now()
WHERE entity_id = $1 AND id = $2
RETURNING id, entity_id, organization_id, name, parent_id, source_id, external_department_id, created_at, updated_at;

-- name: DeleteDepartment :exec
DELETE FROM departments
WHERE entity_id = $1 AND id = $2;

-- name: GetFirstOrganization :one
SELECT id, entity_id, name, parent_id, created_at, updated_at
FROM organizations
WHERE entity_id = $1
ORDER BY created_at ASC
LIMIT 1;

-- name: UpsertDepartmentBySource :one
INSERT INTO departments (entity_id, organization_id, name, parent_id, source_id, external_department_id)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (entity_id, source_id, external_department_id) WHERE source_id IS NOT NULL AND external_department_id IS NOT NULL
DO UPDATE SET
    name = EXCLUDED.name,
    parent_id = EXCLUDED.parent_id,
    organization_id = EXCLUDED.organization_id,
    updated_at = now()
RETURNING id, entity_id, organization_id, name, parent_id, source_id, external_department_id, created_at, updated_at;

-- === Groups ===

-- name: CreateGroup :one
INSERT INTO groups (entity_id, name, type)
VALUES ($1, $2, $3)
RETURNING id, entity_id, name, type, created_at, updated_at;

-- name: ListGroups :many
SELECT id, entity_id, name, type, created_at, updated_at
FROM groups
WHERE entity_id = $1
  AND (sqlc.narg('type')::text IS NULL OR type = sqlc.narg('type')::text)
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountGroups :one
SELECT count(*)::bigint
FROM groups
WHERE entity_id = $1
  AND (sqlc.narg('type')::text IS NULL OR type = sqlc.narg('type')::text);

-- name: GetGroupByID :one
SELECT id, entity_id, name, type, created_at, updated_at
FROM groups
WHERE entity_id = $1 AND id = $2;

-- name: UpdateGroup :one
UPDATE groups
SET name = COALESCE(sqlc.narg('name'), name),
    updated_at = now()
WHERE entity_id = $1 AND id = $2
RETURNING id, entity_id, name, type, created_at, updated_at;

-- name: DeleteGroup :exec
DELETE FROM groups
WHERE entity_id = $1 AND id = $2;

-- === Group Members ===

-- name: AddGroupMember :exec
INSERT INTO group_members (entity_id, group_id, user_id)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;

-- name: RemoveGroupMember :exec
DELETE FROM group_members
WHERE entity_id = $1 AND group_id = $2 AND user_id = $3;

-- name: ListGroupMembers :many
SELECT u.id, u.entity_id, u.username, u.display_name, u.email, u.lifecycle_status
FROM users u
JOIN group_members gm ON gm.entity_id = u.entity_id AND gm.user_id = u.id
WHERE gm.entity_id = $1 AND gm.group_id = $2
ORDER BY gm.added_at DESC
LIMIT $3 OFFSET $4;

-- name: CountGroupMembers :one
SELECT count(*)::bigint
FROM group_members
WHERE entity_id = $1 AND group_id = $2;

-- name: ListUserGroups :many
SELECT g.id, g.entity_id, g.name, g.type, g.created_at, g.updated_at
FROM groups g
JOIN group_members gm ON gm.entity_id = g.entity_id AND gm.group_id = g.id
WHERE gm.entity_id = $1 AND gm.user_id = $2;
