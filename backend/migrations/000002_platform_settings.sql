-- SPDX-License-Identifier: MIT

-- +goose Up
CREATE TABLE IF NOT EXISTS platform_settings (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    platform_name TEXT NOT NULL DEFAULT 'IdBridge',
    logo_url TEXT NOT NULL DEFAULT '',
    favicon_url TEXT NOT NULL DEFAULT '',
    title_suffix TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO platform_settings (id, platform_name, logo_url, favicon_url, title_suffix)
VALUES (1, 'IdBridge', '', '', '')
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS platform_settings;
