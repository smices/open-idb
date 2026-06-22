-- SPDX-License-Identifier: MIT

-- === Account Bindings (admin API queries) ===
-- DeleteAccountBindingByID, GetAccountBindingByID (without source JOIN),
-- and ListAccountBindingsByUser are in identity_store.sql.

-- name: ListAccountBindingsByUserID :many
SELECT ab.id, ab.entity_id, ab.user_id, ab.source_id, ab.directory_user_id,
       ab.provider_uid, ab.provider_union_id, ab.is_primary, ab.bound_at,
       s.type AS source_type, s.name AS source_name
FROM account_bindings ab
JOIN identity_sources s ON s.entity_id = ab.entity_id AND s.id = ab.source_id
WHERE ab.entity_id = $1 AND ab.user_id = $2
ORDER BY ab.is_primary DESC, ab.bound_at ASC;

-- name: GetAccountBindingWithSource :one
SELECT ab.id, ab.entity_id, ab.user_id, ab.source_id, ab.directory_user_id,
       ab.provider_uid, ab.provider_union_id, ab.is_primary, ab.bound_at,
       s.type AS source_type, s.name AS source_name
FROM account_bindings ab
JOIN identity_sources s ON s.entity_id = ab.entity_id AND s.id = ab.source_id
WHERE ab.entity_id = $1 AND ab.id = $2;

-- name: CountBindingsByUser :one
SELECT count(*)::bigint
FROM account_bindings
WHERE entity_id = $1 AND user_id = $2;

-- name: HasLocalCredential :one
SELECT EXISTS(
    SELECT 1 FROM local_credentials
    WHERE entity_id = $1 AND user_id = $2
)::boolean;
