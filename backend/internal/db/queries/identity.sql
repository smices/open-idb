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
    email,
    phone,
    avatar_url,
    status,
    raw_profile,
    last_synced_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, now()
)
ON CONFLICT (entity_id, source_id, external_user_id)
DO UPDATE SET
    external_union_id = EXCLUDED.external_union_id,
    external_open_id = EXCLUDED.external_open_id,
    name = EXCLUDED.name,
    email = EXCLUDED.email,
    phone = EXCLUDED.phone,
    avatar_url = EXCLUDED.avatar_url,
    status = EXCLUDED.status,
    raw_profile = EXCLUDED.raw_profile,
    last_synced_at = now(),
    updated_at = now()
RETURNING id, entity_id, source_id, external_user_id, external_union_id, external_open_id, name, email, phone, avatar_url, status, raw_profile, last_synced_at, created_at, updated_at;

-- name: CreateManagedUser :one
INSERT INTO users (
    entity_id,
    username,
    display_name,
    email,
    phone,
    avatar_url,
    lifecycle_status,
    user_type,
    primary_source_id,
    locale
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING id, entity_id, username, display_name, email, phone, avatar_url, lifecycle_status, user_type, primary_source_id, locale, created_at, updated_at;

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

-- name: GetDirectoryUserByExternalID :one
SELECT id, entity_id, source_id, external_user_id, external_union_id, external_open_id, name, email, phone, avatar_url, status, raw_profile, last_synced_at, created_at, updated_at
FROM directory_users
WHERE entity_id = $1 AND source_id = $2 AND external_user_id = $3;

-- name: GetManagedUserByBinding :one
SELECT u.id, u.entity_id, u.username, u.display_name, u.email, u.phone, u.avatar_url, u.lifecycle_status, u.user_type, u.primary_source_id, u.locale, u.created_at, u.updated_at
FROM users u
JOIN account_bindings ab ON ab.entity_id = u.entity_id AND ab.user_id = u.id
WHERE ab.entity_id = $1 AND ab.source_id = $2 AND ab.provider_uid = $3;

-- name: UpdateManagedUserFromDirectory :one
UPDATE users
SET display_name = $4,
    email = $5,
    phone = $6,
    avatar_url = $7,
    lifecycle_status = $8,
    updated_at = now()
WHERE entity_id = $1 AND id = $2 AND primary_source_id = $3
RETURNING id, entity_id, username, display_name, email, phone, avatar_url, lifecycle_status, user_type, primary_source_id, locale, created_at, updated_at;
