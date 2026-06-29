-- SPDX-License-Identifier: MIT

-- name: GetFeishuIdentitySourceConfig :one
SELECT id,
       entity_id,
       type AS provider,
       name AS display_name,
       status,
       (COALESCE(length(config_encrypted), 0) > 0)::boolean AS oauth_configured,
       sync_enabled,
       COALESCE(config_encrypted, '{}'::bytea) AS config,
       created_at
FROM identity_sources
WHERE entity_id = $1 AND type = 'feishu'
ORDER BY created_at DESC
LIMIT 1;

-- name: UpdateFeishuIdentitySourceConfig :one
UPDATE identity_sources
SET name = $2,
    status = $3,
    sync_enabled = $4,
    config_encrypted = $5
WHERE id = (
    SELECT src.id
    FROM identity_sources src
    WHERE src.entity_id = $1 AND src.type = 'feishu'
    ORDER BY src.created_at DESC
    LIMIT 1
)
RETURNING id,
          entity_id,
          type AS provider,
          name AS display_name,
          status,
          (COALESCE(length(config_encrypted), 0) > 0)::boolean AS oauth_configured,
          sync_enabled,
          COALESCE(config_encrypted, '{}'::bytea) AS config,
          created_at;

-- name: ListLoginProviders :many
SELECT provider, display_name, status, oauth_configured
     , config
FROM (
    SELECT type AS provider,
           name AS display_name,
           status,
           (COALESCE(length(config_encrypted), 0) > 0)::boolean AS oauth_configured,
           COALESCE(config_encrypted, '{}'::bytea) AS config,
           created_at
    FROM identity_sources
    WHERE entity_id = $1 AND type = 'feishu' AND status = 'active'
) providers
WHERE oauth_configured = true
ORDER BY created_at DESC;

-- name: GetFeishuSourceByEntity :one
SELECT id, entity_id, type, name, status, sync_enabled, created_at
FROM identity_sources
WHERE entity_id = $1 AND type = 'feishu' AND status = 'active'
LIMIT 1;
