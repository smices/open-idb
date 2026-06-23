-- SPDX-License-Identifier: MIT

-- === Account Bindings ===

-- name: ListAccountBindingsByUser :many
SELECT id, entity_id, user_id, source_id, directory_user_id, provider_uid, provider_union_id, is_primary, bound_at
FROM account_bindings
WHERE entity_id = $1 AND user_id = $2
ORDER BY bound_at DESC;

-- name: GetAccountBindingByID :one
SELECT id, entity_id, user_id, source_id, directory_user_id, provider_uid, provider_union_id, is_primary, bound_at
FROM account_bindings
WHERE entity_id = $1 AND id = $2;

-- name: DeleteAccountBindingByID :exec
DELETE FROM account_bindings
WHERE entity_id = $1 AND id = $2;

-- === Users (validation) ===

-- name: GetUserByEntityAndID :one
SELECT id, entity_id, username, display_name, email, phone, avatar_url, lifecycle_status, user_type, primary_source_id, locale, created_at, updated_at, english_name, employee_no, job_title
FROM users
WHERE entity_id = $1 AND id = $2;
