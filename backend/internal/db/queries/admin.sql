-- SPDX-License-Identifier: MIT

-- === Users ===

-- name: ListUsers :many
SELECT id, entity_id, username, display_name, email, phone, avatar_url, lifecycle_status, user_type, primary_source_id, locale, created_at, updated_at, english_name, employee_no, job_title
FROM users
WHERE entity_id = $1
  AND (sqlc.narg('lifecycle_status')::text IS NULL OR lifecycle_status = sqlc.narg('lifecycle_status')::text)
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountUsers :one
SELECT count(*)::bigint
FROM users
WHERE entity_id = $1
  AND (sqlc.narg('lifecycle_status')::text IS NULL OR lifecycle_status = sqlc.narg('lifecycle_status')::text);

-- name: GetUserByID :one
SELECT id, entity_id, username, display_name, email, phone, avatar_url, lifecycle_status, user_type, primary_source_id, locale, created_at, updated_at, english_name, employee_no, job_title
FROM users
WHERE entity_id = $1 AND id = $2;

-- name: UpdateUserLifecycle :one
UPDATE users
SET lifecycle_status = $3, updated_at = now()
WHERE entity_id = $1 AND id = $2
RETURNING id, entity_id, username, display_name, email, phone, avatar_url, lifecycle_status, user_type, primary_source_id, locale, created_at, updated_at, english_name, employee_no, job_title;

-- name: UpdateUser :one
UPDATE users
SET display_name = COALESCE(sqlc.narg('display_name'), display_name),
    email = COALESCE(sqlc.narg('email'), email),
    phone = COALESCE(sqlc.narg('phone'), phone),
    locale = COALESCE(sqlc.narg('locale'), locale),
    updated_at = now()
WHERE entity_id = $1 AND id = $2
RETURNING id, entity_id, username, display_name, email, phone, avatar_url, lifecycle_status, user_type, primary_source_id, locale, created_at, updated_at, english_name, employee_no, job_title;

-- === Directory Users ===

-- name: GetDirectoryUserByID :one
SELECT id, entity_id, source_id, external_user_id, external_union_id, external_open_id, name, email, phone, avatar_url, status, raw_profile, last_synced_at, created_at, updated_at, english_name, employee_no, job_title
FROM directory_users
WHERE entity_id = $1 AND id = $2;

-- === Applications ===

-- name: ListApplications :many
SELECT id, entity_id, name, type, status, created_at, updated_at
FROM applications
WHERE entity_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountApplications :one
SELECT count(*)::bigint
FROM applications
WHERE entity_id = $1;

-- name: GetApplicationByID :one
SELECT id, entity_id, name, type, status, created_at, updated_at
FROM applications
WHERE entity_id = $1 AND id = $2;

-- name: UpdateApplication :one
UPDATE applications
SET name = COALESCE(sqlc.narg('name'), name),
    status = COALESCE(sqlc.narg('status'), status),
    updated_at = now()
WHERE entity_id = $1 AND id = $2
RETURNING id, entity_id, name, type, status, created_at, updated_at;

-- name: DeleteApplication :exec
DELETE FROM applications
WHERE entity_id = $1 AND id = $2;

-- === Sync Jobs ===

-- name: ListAllSyncJobs :many
SELECT id, entity_id, source_id, type, provider, status, trace_id, started_at, finished_at, error_message, stats
FROM sync_jobs
WHERE entity_id = $1
ORDER BY started_at DESC
LIMIT $2 OFFSET $3;

-- name: CountAllSyncJobs :one
SELECT count(*)::bigint
FROM sync_jobs
WHERE entity_id = $1;
