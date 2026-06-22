-- SPDX-License-Identifier: MIT

-- === Legacy username/password integration ===

-- name: UpsertLegacyAppUser :one
INSERT INTO legacy_app_users (
    entity_id,
    application_id,
    user_id,
    username,
    legacy_user_identifier,
    auth_scheme,
    credential_hash,
    is_active
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
ON CONFLICT (entity_id, application_id, username)
DO UPDATE SET
    user_id = EXCLUDED.user_id,
    legacy_user_identifier = EXCLUDED.legacy_user_identifier,
    auth_scheme = EXCLUDED.auth_scheme,
    credential_hash = EXCLUDED.credential_hash,
    is_active = EXCLUDED.is_active,
    updated_at = now()
RETURNING id, entity_id, application_id, user_id, username, legacy_user_identifier, auth_scheme, credential_hash, is_active, last_used_at, created_at, updated_at;

-- name: GetLegacyAppUserByUsername :one
SELECT id, entity_id, application_id, user_id, username, legacy_user_identifier, auth_scheme, credential_hash, is_active, last_used_at, created_at, updated_at
FROM legacy_app_users
WHERE entity_id = $1
  AND application_id = $2
  AND username = $3;

-- name: VerifyLegacyAppUserCredential :one
SELECT id, entity_id, application_id, user_id, username, legacy_user_identifier, auth_scheme, credential_hash, is_active, last_used_at, created_at, updated_at
FROM legacy_app_users
WHERE entity_id = $1
  AND application_id = $2
  AND username = $3
  AND is_active = true
  AND auth_scheme = 'local'
  AND credential_hash = crypt($4, credential_hash);

-- name: ListLegacyAppUsersByApplication :many
SELECT id, entity_id, application_id, user_id, username, legacy_user_identifier, auth_scheme, is_active, last_used_at, created_at, updated_at
FROM legacy_app_users
WHERE entity_id = $1 AND application_id = $2
ORDER BY updated_at DESC
LIMIT $3 OFFSET $4;

-- name: CountLegacyAppUsersByApplication :one
SELECT count(*)::bigint
FROM legacy_app_users
WHERE entity_id = $1 AND application_id = $2;

-- name: TouchLegacyAppUserUsedAt :exec
UPDATE legacy_app_users
SET last_used_at = now(), updated_at = now()
WHERE entity_id = $1 AND id = $2;

-- name: SetLegacyAppUserStatus :exec
UPDATE legacy_app_users
SET is_active = $4, updated_at = now()
WHERE entity_id = $1 AND application_id = $2 AND username = $3;

-- name: DeleteLegacyAppUser :exec
DELETE FROM legacy_app_users
WHERE entity_id = $1 AND application_id = $2 AND username = $3;

-- name: CreateLegacyPasswordEvent :exec
INSERT INTO legacy_password_events (
    entity_id,
    application_id,
    user_id,
    username,
    event,
    client_ip,
    user_agent,
    trace_id,
    reason
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: CountLegacyPasswordFailures :one
SELECT COUNT(*)
FROM legacy_password_events
WHERE entity_id = $1
  AND application_id = $2
  AND username = $3
  AND event = 'failed'
  AND occurred_at > $4;
