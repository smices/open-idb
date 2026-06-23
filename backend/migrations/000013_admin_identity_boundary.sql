-- SPDX-License-Identifier: MIT

-- +goose Up
CREATE TABLE admin_users (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    entity_id CHAR(26) CHECK (entity_id IS NULL OR entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    username TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    email TEXT,
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled', 'locked')),
    role TEXT NOT NULL CHECK (role IN ('platform_admin', 'enterprise_admin')),
    locale TEXT NOT NULL DEFAULT 'en-US' CHECK (locale IN ('en-US', 'zh-CN')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK ((role = 'platform_admin' AND entity_id IS NULL) OR (role = 'enterprise_admin' AND entity_id IS NOT NULL))
);

CREATE TABLE admin_credentials (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    admin_user_id CHAR(26) NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
    password_hash TEXT NOT NULL,
    must_change_password BOOLEAN NOT NULL DEFAULT false,
    weak_password BOOLEAN NOT NULL DEFAULT false,
    password_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (admin_user_id)
);

CREATE TABLE admin_sessions (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    admin_user_id CHAR(26) NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
    device_id TEXT,
    ip TEXT,
    user_agent TEXT,
    login_method TEXT NOT NULL CHECK (login_method IN ('password', 'token')),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_admin_sessions_active ON admin_sessions(admin_user_id, expires_at) WHERE revoked_at IS NULL;

WITH admin_row AS (
    INSERT INTO admin_users (username, display_name, email, status, role, locale)
    VALUES ('admin', 'Administrator', 'admin@idbridge.local', 'active', 'platform_admin', 'en-US')
    ON CONFLICT (username) DO UPDATE SET
        display_name = EXCLUDED.display_name,
        status = 'active',
        role = 'platform_admin',
        entity_id = NULL
    RETURNING id
)
INSERT INTO admin_credentials (admin_user_id, password_hash, must_change_password, weak_password)
SELECT id, crypt('admin123', gen_salt('bf')), true, true
FROM admin_row
ON CONFLICT (admin_user_id) DO NOTHING;

-- +goose Down
DROP INDEX IF EXISTS idx_admin_sessions_active;
DROP TABLE IF EXISTS admin_sessions;
DROP TABLE IF EXISTS admin_credentials;
DROP TABLE IF EXISTS admin_users;
