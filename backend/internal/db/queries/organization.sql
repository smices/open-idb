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

-- name: GetDepartmentBySourceExternalID :one
SELECT id, entity_id, organization_id, name, parent_id, source_id, external_department_id, created_at, updated_at
FROM departments
WHERE entity_id = sqlc.arg('entity_id')
  AND source_id = sqlc.arg('source_id')
  AND external_department_id = sqlc.arg('external_department_id');

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

-- name: ListRootOrganizations :many
SELECT id, entity_id, name, parent_id, created_at, updated_at
FROM organizations
WHERE entity_id = $1
  AND (parent_id IS NULL OR parent_id = '')
ORDER BY name ASC, created_at ASC
LIMIT $2 OFFSET $3;

-- name: ListChildOrganizations :many
SELECT id, entity_id, name, parent_id, created_at, updated_at
FROM organizations
WHERE entity_id = $1 AND parent_id = $2
ORDER BY name ASC, created_at ASC
LIMIT $3 OFFSET $4;

-- name: CountChildOrganizations :one
SELECT count(*)::bigint
FROM organizations
WHERE entity_id = $1 AND parent_id = $2;

-- name: ListRootDepartments :many
SELECT id, entity_id, organization_id, name, parent_id, source_id, external_department_id, created_at, updated_at
FROM departments
WHERE entity_id = $1
  AND (
    (sqlc.narg('organization_id')::text IS NULL AND (organization_id IS NULL OR organization_id = ''))
    OR organization_id = sqlc.narg('organization_id')::text
  )
  AND (parent_id IS NULL OR parent_id = '')
ORDER BY name ASC, created_at ASC
LIMIT $2 OFFSET $3;

-- name: ListChildDepartments :many
SELECT id, entity_id, organization_id, name, parent_id, source_id, external_department_id, created_at, updated_at
FROM departments
WHERE entity_id = $1 AND parent_id = $2
ORDER BY name ASC, created_at ASC
LIMIT $3 OFFSET $4;

-- name: CountChildDepartments :one
SELECT count(*)::bigint
FROM departments
WHERE entity_id = $1 AND parent_id = $2;

-- name: ListDirectoryUsersByDepartmentExternalID :many
SELECT id, entity_id, source_id, external_user_id, external_union_id, external_open_id, name, english_name, employee_no, job_title, email, phone, avatar_url, status, raw_profile, last_synced_at, created_at, updated_at
FROM directory_users
WHERE directory_users.entity_id = $1
  AND directory_users.source_id = $2
  AND directory_users.status <> 'deleted'
  AND EXISTS (
    SELECT 1
    FROM directory_departments target_department
    WHERE target_department.entity_id = directory_users.entity_id
      AND target_department.source_id = directory_users.source_id
      AND target_department.external_department_id = $3::text
      AND (
        directory_users.raw_profile->'department_ids' ?| ARRAY[target_department.external_department_id, COALESCE(target_department.raw_profile->>'department_id', ''), COALESCE(target_department.raw_profile->>'open_department_id', '')]
        OR directory_users.raw_profile->'departmentIds' ?| ARRAY[target_department.external_department_id, COALESCE(target_department.raw_profile->>'department_id', ''), COALESCE(target_department.raw_profile->>'open_department_id', '')]
        OR directory_users.raw_profile->'department_id_list' ?| ARRAY[target_department.external_department_id, COALESCE(target_department.raw_profile->>'department_id', ''), COALESCE(target_department.raw_profile->>'open_department_id', '')]
        OR directory_users.raw_profile->>'department_id' = ANY(ARRAY[target_department.external_department_id, COALESCE(target_department.raw_profile->>'department_id', ''), COALESCE(target_department.raw_profile->>'open_department_id', '')])
        OR directory_users.raw_profile->>'departmentId' = ANY(ARRAY[target_department.external_department_id, COALESCE(target_department.raw_profile->>'department_id', ''), COALESCE(target_department.raw_profile->>'open_department_id', '')])
      )
  )
ORDER BY directory_users.name ASC, directory_users.created_at ASC
LIMIT $4 OFFSET $5;

-- name: CountDirectoryUsersByDepartmentExternalID :one
SELECT count(*)::bigint
FROM directory_users
WHERE directory_users.entity_id = $1
  AND directory_users.source_id = $2
  AND directory_users.status <> 'deleted'
  AND EXISTS (
    SELECT 1
    FROM directory_departments target_department
    WHERE target_department.entity_id = directory_users.entity_id
      AND target_department.source_id = directory_users.source_id
      AND target_department.external_department_id = $3::text
      AND (
        directory_users.raw_profile->'department_ids' ?| ARRAY[target_department.external_department_id, COALESCE(target_department.raw_profile->>'department_id', ''), COALESCE(target_department.raw_profile->>'open_department_id', '')]
        OR directory_users.raw_profile->'departmentIds' ?| ARRAY[target_department.external_department_id, COALESCE(target_department.raw_profile->>'department_id', ''), COALESCE(target_department.raw_profile->>'open_department_id', '')]
        OR directory_users.raw_profile->'department_id_list' ?| ARRAY[target_department.external_department_id, COALESCE(target_department.raw_profile->>'department_id', ''), COALESCE(target_department.raw_profile->>'open_department_id', '')]
        OR directory_users.raw_profile->>'department_id' = ANY(ARRAY[target_department.external_department_id, COALESCE(target_department.raw_profile->>'department_id', ''), COALESCE(target_department.raw_profile->>'open_department_id', '')])
        OR directory_users.raw_profile->>'departmentId' = ANY(ARRAY[target_department.external_department_id, COALESCE(target_department.raw_profile->>'department_id', ''), COALESCE(target_department.raw_profile->>'open_department_id', '')])
      )
  );

-- name: ListRootDirectoryUsers :many
SELECT id, entity_id, source_id, external_user_id, external_union_id, external_open_id, name, english_name, employee_no, job_title, email, phone, avatar_url, status, raw_profile, last_synced_at, created_at, updated_at
FROM directory_users
WHERE entity_id = $1
  AND status <> 'deleted'
  AND (
    raw_profile->'department_ids' ? '0'
    OR raw_profile->'departmentIds' ? '0'
    OR raw_profile->'department_id_list' ? '0'
    OR raw_profile->>'department_id' = '0'
    OR raw_profile->>'departmentId' = '0'
    OR (
      NOT (raw_profile ? 'department_ids')
      AND NOT (raw_profile ? 'departmentIds')
      AND NOT (raw_profile ? 'department_id_list')
      AND NOT (raw_profile ? 'department_id')
      AND NOT (raw_profile ? 'departmentId')
    )
    OR (
      COALESCE(raw_profile->'department_ids', '[]'::jsonb) = '[]'::jsonb
      AND COALESCE(raw_profile->'departmentIds', '[]'::jsonb) = '[]'::jsonb
      AND COALESCE(raw_profile->'department_id_list', '[]'::jsonb) = '[]'::jsonb
      AND COALESCE(raw_profile->>'department_id', '') = ''
      AND COALESCE(raw_profile->>'departmentId', '') = ''
    )
  )
ORDER BY name ASC, created_at ASC
LIMIT $2 OFFSET $3;

-- name: CountRootDirectoryUsers :one
SELECT count(*)::bigint
FROM directory_users
WHERE entity_id = $1
  AND status <> 'deleted'
  AND (
    raw_profile->'department_ids' ? '0'
    OR raw_profile->'departmentIds' ? '0'
    OR raw_profile->'department_id_list' ? '0'
    OR raw_profile->>'department_id' = '0'
    OR raw_profile->>'departmentId' = '0'
    OR (
      NOT (raw_profile ? 'department_ids')
      AND NOT (raw_profile ? 'departmentIds')
      AND NOT (raw_profile ? 'department_id_list')
      AND NOT (raw_profile ? 'department_id')
      AND NOT (raw_profile ? 'departmentId')
    )
    OR (
      COALESCE(raw_profile->'department_ids', '[]'::jsonb) = '[]'::jsonb
      AND COALESCE(raw_profile->'departmentIds', '[]'::jsonb) = '[]'::jsonb
      AND COALESCE(raw_profile->'department_id_list', '[]'::jsonb) = '[]'::jsonb
      AND COALESCE(raw_profile->>'department_id', '') = ''
      AND COALESCE(raw_profile->>'departmentId', '') = ''
    )
  );

-- name: SearchOrganizationTreeUsers :many
SELECT id, entity_id, source_id, external_user_id, external_union_id, external_open_id, name, english_name, employee_no, job_title, email, phone, avatar_url, status, raw_profile, last_synced_at, created_at, updated_at
FROM directory_users
WHERE entity_id = $1
  AND status <> 'deleted'
  AND (
    name ILIKE '%' || $2::text || '%'
    OR english_name ILIKE '%' || $2::text || '%'
    OR employee_no ILIKE '%' || $2::text || '%'
    OR job_title ILIKE '%' || $2::text || '%'
    OR email ILIKE '%' || $2::text || '%'
    OR phone ILIKE '%' || $2::text || '%'
    OR external_user_id ILIKE '%' || $2::text || '%'
  )
ORDER BY name ASC, created_at ASC
LIMIT $3 OFFSET $4;

-- name: SearchOrganizationTreeDepartments :many
SELECT id, entity_id, organization_id, name, parent_id, source_id, external_department_id, created_at, updated_at
FROM departments
WHERE entity_id = $1
  AND (
    name ILIKE '%' || $2::text || '%'
    OR external_department_id ILIKE '%' || $2::text || '%'
  )
ORDER BY name ASC, created_at ASC
LIMIT $3 OFFSET $4;

-- name: UpsertDepartmentBySource :one
INSERT INTO departments (entity_id, organization_id, name, parent_id, source_id, external_department_id)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (entity_id, source_id, external_department_id) WHERE source_id IS NOT NULL AND external_department_id IS NOT NULL
DO UPDATE SET
    name = EXCLUDED.name,
    parent_id = CASE
      WHEN sqlc.arg('preserve_parent')::boolean THEN departments.parent_id
      ELSE EXCLUDED.parent_id
    END,
    organization_id = EXCLUDED.organization_id,
    updated_at = now()
RETURNING id, entity_id, organization_id, name, parent_id, source_id, external_department_id, created_at, updated_at;

-- name: DeleteMissingSourceDepartments :execrows
DELETE FROM departments
WHERE entity_id = $1
  AND source_id = $2
  AND NOT (external_department_id = ANY(sqlc.arg('present_external_department_ids')::text[]));

-- name: DeleteSourceDepartmentByExternalID :execrows
DELETE FROM departments
WHERE entity_id = $1
  AND source_id = $2
  AND external_department_id = $3;

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
