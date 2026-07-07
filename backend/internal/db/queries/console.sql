-- SPDX-License-Identifier: MIT

-- name: GetDashboardSummary :one
WITH synced_users AS (
    SELECT id, lifecycle_status, created_at
    FROM users
    WHERE entity_id = sqlc.arg('entity_id')
      AND primary_source_id IS NOT NULL
      AND lifecycle_status <> 'deleted'
)
SELECT
    (SELECT count(*)::bigint FROM synced_users) AS users,
    (SELECT count(*)::bigint FROM synced_users WHERE lifecycle_status = 'active') AS active_users,
    (SELECT count(*)::bigint FROM synced_users WHERE created_at >= now() - interval '7 days') AS new_users,
    (
        SELECT count(*)::bigint
        FROM admin_users au
        WHERE au.entity_id = sqlc.arg('entity_id')::char(26)
           OR au.entity_id IS NULL
    ) AS admin_users,
    (
        SELECT count(*)::bigint
        FROM oauth_tokens ot
        WHERE ot.entity_id = sqlc.arg('entity_id')
          AND ot.created_at >= now() - interval '24 hours'
    ) AS application_activity,
    0::bigint AS pending_authorization,
    CASE
        WHEN EXISTS (
            SELECT 1
            FROM sync_jobs sj
            WHERE sj.entity_id = sqlc.arg('entity_id')
              AND sj.status = 'failed'
              AND sj.started_at >= now() - interval '24 hours'
        ) THEN 'degraded'
        ELSE 'ready'
    END AS sync_health
;

-- name: GetCurrentUserByID :one
SELECT
    u.id,
    u.entity_id,
    u.username,
    u.display_name,
    u.email,
    u.phone,
    u.avatar_url,
    u.locale,
    lc.must_change_password,
    lc.weak_password
FROM users u
LEFT JOIN local_credentials lc ON lc.entity_id = u.entity_id AND lc.user_id = u.id
WHERE u.entity_id = $1
  AND u.id = $2
  AND u.lifecycle_status = 'active';

-- name: VerifyLocalPasswordByUserID :one
SELECT u.id
FROM users u
JOIN local_credentials lc ON lc.entity_id = u.entity_id AND lc.user_id = u.id
WHERE u.entity_id = $1
  AND u.id = $2
  AND lc.password_hash = crypt($3, lc.password_hash)
  AND u.lifecycle_status = 'active';
