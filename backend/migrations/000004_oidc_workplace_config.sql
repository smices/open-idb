-- SPDX-License-Identifier: MIT

-- +goose Up
ALTER TABLE oidc_clients
    ADD COLUMN IF NOT EXISTS workplace_provider TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS workplace_app_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS workplace_app_secret TEXT NOT NULL DEFAULT '';

UPDATE identity_sources
SET config = config - 'workplace_app_id' - 'workplace_app_secret' - 'workplace_exchange_mode'
WHERE type = 'feishu'
  AND config IS NOT NULL;

-- +goose Down
ALTER TABLE oidc_clients
    DROP COLUMN IF EXISTS workplace_app_secret,
    DROP COLUMN IF EXISTS workplace_app_id,
    DROP COLUMN IF EXISTS workplace_provider;
