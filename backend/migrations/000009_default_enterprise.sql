-- SPDX-License-Identifier: MIT

-- +goose Up
INSERT INTO business_entities (name, slug, status, default_locale, brand_name, logo_url, login_message)
VALUES ('Default Enterprise', 'default_enterprise', 'active', 'zh-CN', 'Default Enterprise', '', 'Enterprise identity access.')
ON CONFLICT (slug) DO UPDATE SET
    status = EXCLUDED.status;

-- +goose Down
DELETE FROM business_entities WHERE slug = 'default_enterprise';
