-- SPDX-License-Identifier: MIT

-- name: GetPermissionsVersion :one
SELECT COALESCE(
    (SELECT EXTRACT(EPOCH FROM MAX(sub.ts))::bigint
     FROM (
         SELECT r.updated_at AS ts FROM roles r WHERE r.entity_id = $1
         UNION ALL
         SELECT p.updated_at AS ts FROM permissions p WHERE p.entity_id = $1
         UNION ALL
         SELECT ur.created_at AS ts FROM user_roles ur WHERE ur.entity_id = $1
         UNION ALL
         SELECT rp.created_at AS ts FROM role_permissions rp WHERE rp.entity_id = $1
     ) sub),
    0
)::bigint AS version;

-- name: GetResourceScopesVersion :one
SELECT COALESCE(
    (SELECT EXTRACT(EPOCH FROM MAX(sub.ts))::bigint
     FROM (
         SELECT rs.updated_at AS ts FROM resource_scopes rs WHERE rs.entity_id = $1
         UNION ALL
         SELECT rrs.created_at AS ts FROM role_resource_scopes rrs WHERE rrs.entity_id = $1
     ) sub),
    0
)::bigint AS version;
