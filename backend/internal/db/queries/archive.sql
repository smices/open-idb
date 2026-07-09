-- SPDX-License-Identifier: MIT

-- name: ArchiveUser :one
INSERT INTO archived_users (
    entity_id,
    original_user_id,
    username,
    display_name,
    english_name,
    employee_no,
    job_title,
    email,
    phone,
    avatar_url,
    user_type,
    primary_source_id,
    locale,
    original_created_at,
    original_updated_at,
    archived_by_user_id,
    archive_reason,
    user_snapshot,
    bindings_snapshot,
    roles_snapshot
)
SELECT
    u.entity_id,
    u.id,
    u.username,
    u.display_name,
    u.english_name,
    u.employee_no,
    u.job_title,
    u.email,
    u.phone,
    u.avatar_url,
    u.user_type,
    u.primary_source_id,
    u.locale,
    u.created_at,
    u.updated_at,
    sqlc.narg('archived_by_user_id'),
    sqlc.arg('archive_reason'),
    to_jsonb(u),
    COALESCE((
        SELECT jsonb_agg(to_jsonb(ab) ORDER BY ab.bound_at)
        FROM account_bindings ab
        WHERE ab.entity_id = u.entity_id AND ab.user_id = u.id
    ), '[]'::jsonb),
    COALESCE((
        SELECT jsonb_agg(to_jsonb(ur) ORDER BY ur.role_id)
        FROM user_roles ur
        WHERE ur.entity_id = u.entity_id AND ur.user_id = u.id
    ), '[]'::jsonb)
FROM users u
WHERE u.entity_id = sqlc.arg('entity_id')
  AND u.id = sqlc.arg('user_id')
RETURNING id, entity_id, original_user_id, username, display_name, english_name, employee_no, job_title, email, phone, avatar_url, user_type, primary_source_id, locale, original_created_at, original_updated_at, archived_at, archived_by_user_id, archive_reason, user_snapshot, bindings_snapshot, roles_snapshot;

-- name: DeleteUserActiveDependents :exec
WITH deleted_application_assignments AS (
    DELETE FROM application_assignments
    WHERE entity_id = sqlc.arg('entity_id')
      AND subject_type = 'user'
      AND subject_id = sqlc.arg('user_id')
    RETURNING 1
)
SELECT count(*) FROM deleted_application_assignments;

-- name: DeleteUserActiveRow :exec
DELETE FROM users
WHERE entity_id = sqlc.arg('entity_id')
  AND id = sqlc.arg('user_id');

-- name: ListArchivedUsers :many
SELECT id, entity_id, original_user_id, username, display_name, english_name, employee_no, job_title, email, phone, avatar_url, user_type, primary_source_id, locale, original_created_at, original_updated_at, archived_at, archived_by_user_id, archive_reason, user_snapshot, bindings_snapshot, roles_snapshot
FROM archived_users
WHERE entity_id = sqlc.arg('entity_id')
  AND (sqlc.narg('username')::text IS NULL OR username ILIKE '%' || sqlc.narg('username')::text || '%')
ORDER BY archived_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountArchivedUsers :one
SELECT count(*)::bigint
FROM archived_users
WHERE entity_id = sqlc.arg('entity_id')
  AND (sqlc.narg('username')::text IS NULL OR username ILIKE '%' || sqlc.narg('username')::text || '%');

-- name: GetArchivedUserByID :one
SELECT id, entity_id, original_user_id, username, display_name, english_name, employee_no, job_title, email, phone, avatar_url, user_type, primary_source_id, locale, original_created_at, original_updated_at, archived_at, archived_by_user_id, archive_reason, user_snapshot, bindings_snapshot, roles_snapshot
FROM archived_users
WHERE entity_id = sqlc.arg('entity_id')
  AND id = sqlc.arg('id');

-- name: GetArchivedUserByOriginalID :one
SELECT id, entity_id, original_user_id, username, display_name, english_name, employee_no, job_title, email, phone, avatar_url, user_type, primary_source_id, locale, original_created_at, original_updated_at, archived_at, archived_by_user_id, archive_reason, user_snapshot, bindings_snapshot, roles_snapshot
FROM archived_users
WHERE entity_id = sqlc.arg('entity_id')
  AND original_user_id = sqlc.arg('original_user_id');
