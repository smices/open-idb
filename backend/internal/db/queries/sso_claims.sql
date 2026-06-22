-- SPDX-License-Identifier: MIT

-- name: GetUserClaimsForToken :one
SELECT u.id, u.entity_id, u.username, u.display_name, u.email, u.phone, u.avatar_url, u.locale,
       COALESCE(
           (SELECT json_agg(DISTINCT isrc.type)
            FROM account_bindings ab
            JOIN identity_sources isrc ON isrc.entity_id = ab.entity_id AND isrc.id = ab.source_id
            WHERE ab.entity_id = u.entity_id AND ab.user_id = u.id),
           '[]'::json
       ) AS identity_sources
FROM users u
WHERE u.entity_id = $1 AND u.id = $2;

-- name: GetUserRolesForToken :many
SELECT r.code
FROM roles r
JOIN user_roles ur ON ur.entity_id = r.entity_id AND ur.role_id = r.id
WHERE ur.entity_id = $1 AND ur.user_id = $2;
