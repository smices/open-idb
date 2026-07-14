-- SPDX-License-Identifier: MIT

-- +goose Up
ALTER TABLE applications
    ADD COLUMN IF NOT EXISTS config JSONB NOT NULL DEFAULT '{}'::jsonb;

-- +goose Down
ALTER TABLE applications
    DROP COLUMN IF EXISTS config;
