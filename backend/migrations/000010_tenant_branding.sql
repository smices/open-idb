-- SPDX-License-Identifier: MIT

-- +goose Up
ALTER TABLE business_entities
    ADD COLUMN IF NOT EXISTS brand_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS logo_url TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS login_message TEXT NOT NULL DEFAULT '';

UPDATE business_entities
SET
    brand_name = COALESCE(NULLIF(brand_name, ''), name),
    login_message = COALESCE(NULLIF(login_message, ''), 'Enterprise identity access.')
WHERE status = 'active'
  AND (brand_name = '' OR login_message = '');

-- +goose Down
ALTER TABLE business_entities
    DROP COLUMN IF EXISTS login_message,
    DROP COLUMN IF EXISTS logo_url,
    DROP COLUMN IF EXISTS brand_name;
