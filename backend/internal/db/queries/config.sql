-- SPDX-License-Identifier: MIT

-- name: ListIMProviderConfigs :many
SELECT id, entity_id, provider, display_name, status, oauth_configured, bot_configured, sync_enabled, config, created_at, updated_at
FROM im_provider_configs
WHERE entity_id = $1
ORDER BY provider;

-- name: UpsertIMProviderConfig :one
INSERT INTO im_provider_configs (
    entity_id,
    provider,
    display_name,
    status,
    oauth_configured,
    bot_configured,
    sync_enabled,
    config
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
ON CONFLICT (entity_id, provider)
DO UPDATE SET
    display_name = EXCLUDED.display_name,
    status = EXCLUDED.status,
    oauth_configured = EXCLUDED.oauth_configured,
    bot_configured = EXCLUDED.bot_configured,
    sync_enabled = EXCLUDED.sync_enabled,
    config = EXCLUDED.config,
    updated_at = now()
RETURNING id, entity_id, provider, display_name, status, oauth_configured, bot_configured, sync_enabled, config, created_at, updated_at;

-- name: ListLoginProviders :many
SELECT provider, display_name, status, oauth_configured
     , config
FROM im_provider_configs
WHERE entity_id = $1 AND status = 'active' AND oauth_configured = true
ORDER BY provider;

-- name: GetFeishuSourceByEntity :one
SELECT id, entity_id, type, name, status, sync_enabled, created_at
FROM identity_sources
WHERE entity_id = $1 AND type = 'feishu' AND status = 'active'
LIMIT 1;

-- name: ListMCPConnectors :many
SELECT id, entity_id, name, endpoint_url, auth_type, status, description, config, created_at, updated_at
FROM mcp_connectors
WHERE entity_id = $1
ORDER BY name;

-- name: CreateMCPConnector :one
INSERT INTO mcp_connectors (
    entity_id,
    name,
    endpoint_url,
    auth_type,
    status,
    description,
    config
) VALUES (
    $1, $2, $3, $4, $5, $6, '{}'::jsonb
)
RETURNING id, entity_id, name, endpoint_url, auth_type, status, description, config, created_at, updated_at;
