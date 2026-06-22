-- SPDX-License-Identifier: MIT

-- name: AuthenticateLocalUser :one
SELECT
    u.id,
    u.entity_id,
    u.username,
    u.display_name,
    u.email,
    u.phone,
    u.avatar_url,
    u.lifecycle_status,
    u.user_type,
    u.primary_source_id,
    u.locale,
    u.created_at,
    u.updated_at,
    lc.must_change_password,
    lc.weak_password
FROM users u
JOIN local_credentials lc ON lc.entity_id = u.entity_id AND lc.user_id = u.id
WHERE u.username = $1
  AND lc.password_hash = crypt($2, lc.password_hash)
  AND u.lifecycle_status = 'active';

-- name: AuthenticateLocalUserByEntity :one
SELECT
    u.id,
    u.entity_id,
    u.username,
    u.display_name,
    u.email,
    u.phone,
    u.avatar_url,
    u.lifecycle_status,
    u.user_type,
    u.primary_source_id,
    u.locale,
    u.created_at,
    u.updated_at,
    lc.must_change_password,
    lc.weak_password
FROM users u
JOIN local_credentials lc ON lc.entity_id = u.entity_id AND lc.user_id = u.id
WHERE u.entity_id = $1
  AND u.username = $2
  AND lc.password_hash = crypt($3, lc.password_hash)
  AND u.lifecycle_status = 'active';

-- name: UpdateLocalPassword :one
UPDATE local_credentials
SET password_hash = crypt($3, gen_salt('bf')),
    must_change_password = false,
    weak_password = $4,
    password_updated_at = now(),
    updated_at = now()
WHERE entity_id = $1 AND user_id = $2
RETURNING id, entity_id, user_id, password_hash, must_change_password, weak_password, password_updated_at, created_at, updated_at;

-- name: GetLoginContextByOIDCClientID :one
SELECT
    be.id AS entity_id,
    be.slug AS entity_slug,
    be.name AS entity_name,
    be.brand_name AS entity_brand_name,
    be.logo_url AS entity_logo_url,
    be.login_message AS entity_login_message,
    a.id AS application_id,
    a.name AS application_name
FROM oidc_clients oc
JOIN applications a ON a.entity_id = oc.entity_id AND a.id = oc.application_id
JOIN business_entities be ON be.id = oc.entity_id
WHERE oc.client_id = $1
  AND oc.status = 'active'
  AND a.status = 'active'
  AND be.status = 'active'
LIMIT 1;

-- name: GetDefaultLoginContextEntity :one
SELECT
    be.id AS entity_id,
    be.slug AS entity_slug,
    be.name AS entity_name,
    be.brand_name AS entity_brand_name,
    be.logo_url AS entity_logo_url,
    be.login_message AS entity_login_message
FROM business_entities be
JOIN identity_sources src ON src.entity_id = be.id
WHERE be.status = 'active'
  AND src.type = 'feishu'
  AND src.status = 'active'
ORDER BY be.created_at ASC
LIMIT 1;
