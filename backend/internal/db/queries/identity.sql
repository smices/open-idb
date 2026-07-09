-- SPDX-License-Identifier: MIT

-- name: CreateEntity :one
INSERT INTO business_entities (name, slug, status, default_locale, brand_name, logo_url, login_message)
VALUES ($1, $2, 'active', $3, $4, $5, $6)
RETURNING id, name, slug, status, default_locale, brand_name, logo_url, login_message, created_at;

-- name: ListEntities :many
SELECT id, name, slug, status, default_locale, brand_name, logo_url, login_message, created_at
FROM business_entities
ORDER BY created_at DESC, name ASC
LIMIT $1 OFFSET $2;

-- name: CountEntities :one
SELECT count(*) FROM business_entities;

-- name: GetEntityByID :one
SELECT id, name, slug, status, default_locale, brand_name, logo_url, login_message, created_at
FROM business_entities
WHERE id = $1;

-- name: GetEntityBySlug :one
SELECT id, name, slug, status, default_locale, brand_name, logo_url, login_message, created_at
FROM business_entities
WHERE slug = $1;

-- name: UpdateEntity :one
UPDATE business_entities
SET
    name = COALESCE(sqlc.narg('name'), name),
    status = COALESCE(sqlc.narg('status'), status),
    default_locale = COALESCE(sqlc.narg('default_locale'), default_locale),
    brand_name = COALESCE(sqlc.narg('brand_name'), brand_name),
    logo_url = COALESCE(sqlc.narg('logo_url'), logo_url),
    login_message = COALESCE(sqlc.narg('login_message'), login_message)
WHERE id = sqlc.arg('id')
RETURNING id, name, slug, status, default_locale, brand_name, logo_url, login_message, created_at;

-- name: CreateIdentitySource :one
INSERT INTO identity_sources (entity_id, type, name, status, sync_enabled)
VALUES ($1, $2, $3, 'active', $4)
RETURNING id, entity_id, type, name, status, sync_enabled, created_at;

-- name: UpsertDirectoryUser :one
INSERT INTO directory_users (
    entity_id,
    source_id,
    external_user_id,
    external_union_id,
    external_open_id,
    name,
    english_name,
    employee_no,
    job_title,
    email,
    phone,
    avatar_url,
    status,
    raw_profile,
    last_synced_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, now()
)
ON CONFLICT (entity_id, source_id, external_user_id)
DO UPDATE SET
    external_union_id = EXCLUDED.external_union_id,
    external_open_id = EXCLUDED.external_open_id,
    name = EXCLUDED.name,
    english_name = EXCLUDED.english_name,
    employee_no = EXCLUDED.employee_no,
    job_title = EXCLUDED.job_title,
    email = EXCLUDED.email,
    phone = EXCLUDED.phone,
    avatar_url = EXCLUDED.avatar_url,
    status = EXCLUDED.status,
    raw_profile = EXCLUDED.raw_profile,
    last_synced_at = now(),
    updated_at = now()
RETURNING id, entity_id, source_id, external_user_id, external_union_id, external_open_id, name, english_name, employee_no, job_title, email, phone, avatar_url, status, raw_profile, last_synced_at, created_at, updated_at;

-- name: CreateManagedUser :one
INSERT INTO users (
    entity_id,
    username,
    display_name,
    english_name,
    employee_no,
    job_title,
    email,
    phone,
    avatar_url,
    lifecycle_status,
    user_type,
    primary_source_id,
    locale
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)
RETURNING id, entity_id, username, display_name, english_name, employee_no, job_title, email, phone, avatar_url, lifecycle_status, user_type, primary_source_id, locale, created_at, updated_at;

-- name: CreateAccountBinding :one
INSERT INTO account_bindings (
    entity_id,
    user_id,
    source_id,
    directory_user_id,
    provider_uid,
    provider_union_id,
    is_primary
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING id, entity_id, user_id, source_id, directory_user_id, provider_uid, provider_union_id, is_primary, bound_at;

-- name: UpsertDirectoryDepartment :one
INSERT INTO directory_departments (
    entity_id,
    source_id,
    external_department_id,
    parent_external_department_id,
    name,
    raw_profile,
    last_synced_at
) VALUES (
    $1, $2, $3, $4, $5, $6, now()
)
ON CONFLICT (entity_id, source_id, external_department_id)
DO UPDATE SET
    parent_external_department_id = EXCLUDED.parent_external_department_id,
    name = EXCLUDED.name,
    raw_profile = EXCLUDED.raw_profile,
    last_synced_at = now(),
    updated_at = now()
RETURNING id, entity_id, source_id, external_department_id, parent_external_department_id, name, raw_profile, last_synced_at, created_at, updated_at;

-- name: GetAccountBindingByProviderUID :one
SELECT id, entity_id, user_id, source_id, directory_user_id, provider_uid, provider_union_id, is_primary, bound_at
FROM account_bindings
WHERE entity_id = $1 AND source_id = $2 AND provider_uid = $3;

-- name: GetAccountBindingByProviderUnionID :one
SELECT id, entity_id, user_id, source_id, directory_user_id, provider_uid, provider_union_id, is_primary, bound_at
FROM account_bindings
WHERE entity_id = $1 AND source_id = $2 AND provider_union_id = $3 AND provider_union_id <> ''
LIMIT 1;

-- name: UpdateAccountBindingFromDirectory :one
UPDATE account_bindings
SET directory_user_id = $4,
    provider_uid = $5,
    provider_union_id = $6,
    is_primary = true
WHERE entity_id = $1 AND source_id = $2 AND id = $3
RETURNING id, entity_id, user_id, source_id, directory_user_id, provider_uid, provider_union_id, is_primary, bound_at;

-- name: GetDirectoryUserByExternalID :one
SELECT id, entity_id, source_id, external_user_id, external_union_id, external_open_id, name, english_name, employee_no, job_title, email, phone, avatar_url, status, raw_profile, last_synced_at, created_at, updated_at
FROM directory_users
WHERE entity_id = $1 AND source_id = $2 AND external_user_id = $3;

-- name: GetManagedUserByUsername :one
SELECT id, entity_id, username, display_name, english_name, employee_no, job_title, email, phone, avatar_url, lifecycle_status, user_type, primary_source_id, locale, created_at, updated_at
FROM users
WHERE entity_id = $1 AND username = $2
LIMIT 1;

-- name: GetManagedUserByBinding :one
SELECT u.id, u.entity_id, u.username, u.display_name, u.english_name, u.employee_no, u.job_title, u.email, u.phone, u.avatar_url, u.lifecycle_status, u.user_type, u.primary_source_id, u.locale, u.created_at, u.updated_at
FROM users u
JOIN account_bindings ab ON ab.entity_id = u.entity_id AND ab.user_id = u.id
WHERE ab.entity_id = $1 AND ab.source_id = $2 AND ab.provider_uid = $3;

-- name: UpdateManagedUserFromDirectory :one
UPDATE users
SET username = COALESCE(sqlc.narg('username'), username),
    display_name = sqlc.arg('display_name'),
    english_name = sqlc.arg('english_name'),
    employee_no = sqlc.arg('employee_no'),
    job_title = sqlc.arg('job_title'),
    email = sqlc.arg('email'),
    phone = sqlc.arg('phone'),
    avatar_url = sqlc.arg('avatar_url'),
    lifecycle_status = sqlc.arg('lifecycle_status'),
    primary_source_id = sqlc.arg('primary_source_id'),
    updated_at = now()
WHERE entity_id = sqlc.arg('entity_id') AND id = sqlc.arg('id')
RETURNING id, entity_id, username, display_name, english_name, employee_no, job_title, email, phone, avatar_url, lifecycle_status, user_type, primary_source_id, locale, created_at, updated_at;

-- name: MarkMissingDirectoryUsersDeleted :many
UPDATE directory_users
SET status = 'deleted',
    last_synced_at = now(),
    updated_at = now()
WHERE entity_id = $1
  AND source_id = $2
  AND status <> 'deleted'
  AND NOT (external_user_id = ANY(sqlc.arg('present_external_user_ids')::text[]))
RETURNING id, entity_id, source_id, external_user_id, external_union_id, external_open_id, name, english_name, employee_no, job_title, email, phone, avatar_url, status, raw_profile, last_synced_at, created_at, updated_at;

-- name: SoftDeleteManagedUsersByDirectoryStatus :many
UPDATE users u
SET lifecycle_status = 'deleted',
    updated_at = now()
FROM account_bindings ab
JOIN directory_users du
  ON du.entity_id = ab.entity_id
 AND du.source_id = ab.source_id
 AND du.id = ab.directory_user_id
WHERE u.entity_id = $1
  AND u.id = ab.user_id
  AND ab.entity_id = $1
  AND ab.source_id = $2
  AND du.status = 'deleted'
  AND u.lifecycle_status <> 'deleted'
RETURNING u.id, u.entity_id, u.username, u.display_name, u.english_name, u.employee_no, u.job_title, u.email, u.phone, u.avatar_url, u.lifecycle_status, u.user_type, u.primary_source_id, u.locale, u.created_at, u.updated_at;
