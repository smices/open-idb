-- SPDX-License-Identifier: MIT

-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION idb_generate_ulid()
RETURNS CHAR(26) AS $$
DECLARE
    alphabet CONSTANT TEXT := '0123456789ABCDEFGHJKMNPQRSTVWXYZ';
    ts BIGINT := floor(extract(epoch from clock_timestamp()) * 1000)::BIGINT;
    raw BYTEA := decode(lpad(to_hex(ts), 12, '0'), 'hex') || gen_random_bytes(10);
    value NUMERIC := 0;
    result TEXT := '';
    idx INT;
BEGIN
    FOR idx IN 0..15 LOOP
        value := value * 256 + get_byte(raw, idx);
    END LOOP;

    FOR idx IN 1..26 LOOP
        result := substr(alphabet, mod(value, 32)::INT + 1, 1) || result;
        value := floor(value / 32);
    END LOOP;

    RETURN result::CHAR(26);
END;
$$ LANGUAGE plpgsql VOLATILE;
-- +goose StatementEnd

CREATE TABLE business_entities (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled')),
    default_locale TEXT NOT NULL DEFAULT 'en-US' CHECK (default_locale IN ('en-US', 'zh-CN')),
    brand_name TEXT NOT NULL DEFAULT '',
    logo_url TEXT NOT NULL DEFAULT '',
    login_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE identity_sources (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK (type IN ('feishu', 'dingtalk', 'wecom', 'ldap', 'local')),
    name TEXT NOT NULL,
    config_encrypted BYTEA,
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled')),
    sync_enabled BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, id),
    UNIQUE (entity_id, type, name)
);

CREATE TABLE directory_users (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    source_id CHAR(26) NOT NULL CHECK (source_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    external_user_id TEXT NOT NULL,
    external_union_id TEXT,
    external_open_id TEXT,
    name TEXT NOT NULL,
    email TEXT,
    phone TEXT,
    avatar_url TEXT,
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled', 'deleted', 'unknown')),
    raw_profile JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_synced_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, id),
    UNIQUE (entity_id, source_id, id),
    UNIQUE (entity_id, source_id, external_user_id),
    FOREIGN KEY (entity_id, source_id) REFERENCES identity_sources(entity_id, id) ON DELETE CASCADE
);

CREATE TABLE users (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    username TEXT NOT NULL,
    display_name TEXT NOT NULL,
    email TEXT,
    phone TEXT,
    avatar_url TEXT,
    lifecycle_status TEXT NOT NULL CHECK (lifecycle_status IN ('active', 'disabled', 'locked', 'deleted')),
    user_type TEXT NOT NULL CHECK (user_type IN ('employee', 'contractor', 'service_account')),
    primary_source_id CHAR(26) CHECK (primary_source_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    locale TEXT CHECK (locale IN ('en-US', 'zh-CN')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, id),
    UNIQUE (entity_id, username),
    FOREIGN KEY (entity_id, primary_source_id) REFERENCES identity_sources(entity_id, id) ON DELETE SET NULL
);

CREATE TABLE account_bindings (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    user_id CHAR(26) NOT NULL CHECK (user_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    source_id CHAR(26) NOT NULL CHECK (source_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    directory_user_id CHAR(26) NOT NULL CHECK (directory_user_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    provider_uid TEXT NOT NULL,
    provider_union_id TEXT,
    is_primary BOOLEAN NOT NULL DEFAULT false,
    bound_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, source_id, provider_uid),
    UNIQUE (entity_id, directory_user_id),
    FOREIGN KEY (entity_id, user_id) REFERENCES users(entity_id, id) ON DELETE CASCADE,
    FOREIGN KEY (entity_id, source_id) REFERENCES identity_sources(entity_id, id) ON DELETE CASCADE,
    FOREIGN KEY (entity_id, directory_user_id) REFERENCES directory_users(entity_id, id) ON DELETE CASCADE,
    FOREIGN KEY (entity_id, source_id, directory_user_id) REFERENCES directory_users(entity_id, source_id, id) ON DELETE CASCADE
);

CREATE INDEX idx_directory_users_entity_source ON directory_users(entity_id, source_id);
CREATE INDEX idx_users_entity_status ON users(entity_id, lifecycle_status);
CREATE INDEX idx_account_bindings_user ON account_bindings(entity_id, user_id);

CREATE TABLE applications (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('oidc_client', 'api_client', 'internal_app')),
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, id),
    UNIQUE (entity_id, name)
);

CREATE TABLE oidc_clients (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    application_id CHAR(26) NOT NULL CHECK (application_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    client_id TEXT NOT NULL,
    client_secret_hash TEXT,
    redirect_uris TEXT[] NOT NULL,
    allowed_scopes TEXT[] NOT NULL DEFAULT ARRAY['openid', 'profile', 'email']::TEXT[],
    grant_types TEXT[] NOT NULL DEFAULT ARRAY['authorization_code']::TEXT[],
    response_types TEXT[] NOT NULL DEFAULT ARRAY['code']::TEXT[],
    pkce_required BOOLEAN NOT NULL DEFAULT true,
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, id),
    UNIQUE (entity_id, client_id),
    FOREIGN KEY (entity_id, application_id) REFERENCES applications(entity_id, id) ON DELETE CASCADE
);

CREATE TABLE oauth_authorization_codes (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    client_id TEXT NOT NULL,
    user_id CHAR(26) NOT NULL CHECK (user_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    code_hash TEXT NOT NULL,
    redirect_uri TEXT NOT NULL,
    scopes TEXT[] NOT NULL,
    code_challenge TEXT NOT NULL,
    code_challenge_method TEXT NOT NULL CHECK (code_challenge_method IN ('S256')),
    nonce TEXT,
    used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, code_hash),
    FOREIGN KEY (entity_id, client_id) REFERENCES oidc_clients(entity_id, client_id) ON DELETE CASCADE,
    FOREIGN KEY (entity_id, user_id) REFERENCES users(entity_id, id) ON DELETE CASCADE
);

CREATE TABLE oauth_tokens (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    user_id CHAR(26) NOT NULL CHECK (user_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    client_id TEXT NOT NULL,
    token_type TEXT NOT NULL CHECK (token_type IN ('access', 'id', 'refresh')),
    token_hash TEXT NOT NULL,
    scopes TEXT[] NOT NULL,
    revoked_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, token_hash),
    FOREIGN KEY (entity_id, user_id) REFERENCES users(entity_id, id) ON DELETE CASCADE,
    FOREIGN KEY (entity_id, client_id) REFERENCES oidc_clients(entity_id, client_id) ON DELETE CASCADE
);

CREATE INDEX idx_oidc_clients_entity_status ON oidc_clients(entity_id, status);
CREATE INDEX idx_oauth_authorization_codes_expiry ON oauth_authorization_codes(expires_at);
CREATE INDEX idx_oauth_tokens_user_client ON oauth_tokens(entity_id, user_id, client_id);

CREATE TABLE directory_departments (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    source_id CHAR(26) NOT NULL CHECK (source_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    external_department_id TEXT NOT NULL,
    parent_external_department_id TEXT,
    name TEXT NOT NULL,
    raw_profile JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_synced_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, id),
    UNIQUE (entity_id, source_id, external_department_id),
    FOREIGN KEY (entity_id, source_id) REFERENCES identity_sources(entity_id, id) ON DELETE CASCADE
);

CREATE TABLE sync_jobs (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    source_id CHAR(26) NOT NULL CHECK (source_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    type TEXT NOT NULL CHECK (type IN ('full', 'incremental', 'webhook')),
    provider TEXT NOT NULL CHECK (provider IN ('feishu', 'dingtalk', 'wecom', 'ldap', 'local')),
    status TEXT NOT NULL CHECK (status IN ('running', 'succeeded', 'failed')),
    trace_id TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ,
    error_message TEXT,
    stats JSONB NOT NULL DEFAULT '{}'::jsonb,
    FOREIGN KEY (entity_id, source_id) REFERENCES identity_sources(entity_id, id) ON DELETE CASCADE
);

CREATE INDEX idx_directory_departments_entity_source ON directory_departments(entity_id, source_id);
CREATE INDEX idx_sync_jobs_entity_source_started ON sync_jobs(entity_id, source_id, started_at DESC);

CREATE TABLE roles (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    code TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, id),
    UNIQUE (entity_id, code)
);

CREATE TABLE permissions (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('api', 'menu', 'action', 'data')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, id),
    UNIQUE (entity_id, code)
);

CREATE TABLE role_permissions (
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    role_id CHAR(26) NOT NULL CHECK (role_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    permission_id CHAR(26) NOT NULL CHECK (permission_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (entity_id, role_id, permission_id),
    FOREIGN KEY (entity_id, role_id) REFERENCES roles(entity_id, id) ON DELETE CASCADE,
    FOREIGN KEY (entity_id, permission_id) REFERENCES permissions(entity_id, id) ON DELETE CASCADE
);

CREATE TABLE user_roles (
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    user_id CHAR(26) NOT NULL CHECK (user_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    role_id CHAR(26) NOT NULL CHECK (role_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (entity_id, user_id, role_id),
    FOREIGN KEY (entity_id, user_id) REFERENCES users(entity_id, id) ON DELETE CASCADE,
    FOREIGN KEY (entity_id, role_id) REFERENCES roles(entity_id, id) ON DELETE CASCADE
);

CREATE TABLE resource_scopes (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK (type IN ('store', 'warehouse', 'country', 'brand')),
    key TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, id),
    UNIQUE (entity_id, type, key)
);

CREATE TABLE role_resource_scopes (
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    role_id CHAR(26) NOT NULL CHECK (role_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    resource_scope_id CHAR(26) NOT NULL CHECK (resource_scope_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    effect TEXT NOT NULL CHECK (effect IN ('allow', 'deny')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (entity_id, role_id, resource_scope_id, effect),
    FOREIGN KEY (entity_id, role_id) REFERENCES roles(entity_id, id) ON DELETE CASCADE,
    FOREIGN KEY (entity_id, resource_scope_id) REFERENCES resource_scopes(entity_id, id) ON DELETE CASCADE
);

CREATE TABLE application_assignments (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    application_id CHAR(26) NOT NULL CHECK (application_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    subject_type TEXT NOT NULL CHECK (subject_type IN ('user', 'group', 'department', 'role')),
    subject_id CHAR(26) NOT NULL CHECK (subject_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    effect TEXT NOT NULL CHECK (effect IN ('allow', 'deny')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, application_id, subject_type, subject_id, effect),
    FOREIGN KEY (entity_id, application_id) REFERENCES applications(entity_id, id) ON DELETE CASCADE
);

CREATE INDEX idx_user_roles_user ON user_roles(entity_id, user_id);
CREATE INDEX idx_application_assignments_lookup ON application_assignments(entity_id, application_id, subject_type, subject_id, effect);

CREATE TABLE local_credentials (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    user_id CHAR(26) NOT NULL CHECK (user_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    password_hash TEXT NOT NULL,
    must_change_password BOOLEAN NOT NULL DEFAULT false,
    weak_password BOOLEAN NOT NULL DEFAULT false,
    password_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, user_id),
    FOREIGN KEY (entity_id, user_id) REFERENCES users(entity_id, id) ON DELETE CASCADE
);

WITH entity_row AS (
    INSERT INTO business_entities (name, slug, status, default_locale, brand_name, logo_url, login_message)
    VALUES ('Default Enterprise', 'default_enterprise', 'active', 'zh-CN', 'Default Enterprise', '', 'Enterprise identity access.')
    ON CONFLICT (slug) DO UPDATE SET
        status = EXCLUDED.status
    RETURNING id
),
source_row AS (
    INSERT INTO identity_sources (entity_id, type, name, status, sync_enabled)
    SELECT id, 'local', 'Local Accounts', 'active', false
    FROM entity_row
    ON CONFLICT (entity_id, type, name) DO UPDATE SET status = EXCLUDED.status
    RETURNING id, entity_id
),
user_row AS (
    INSERT INTO users (
        entity_id,
        username,
        display_name,
        email,
        lifecycle_status,
        user_type,
        primary_source_id,
        locale
    )
    SELECT
        entity_id,
        'admin',
        'Administrator',
        'admin@idbridge.local',
        'active',
        'employee',
        id,
        'en-US'
    FROM source_row
    ON CONFLICT (entity_id, username) DO UPDATE SET
        display_name = EXCLUDED.display_name,
        lifecycle_status = 'active'
    RETURNING id, entity_id
)
INSERT INTO local_credentials (entity_id, user_id, password_hash, must_change_password, weak_password)
SELECT entity_id, id, crypt('admin123', gen_salt('bf')), true, true
FROM user_row
ON CONFLICT (entity_id, user_id) DO NOTHING;

CREATE TABLE im_provider_configs (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    provider TEXT NOT NULL CHECK (provider IN ('feishu', 'dingtalk', 'wecom')),
    display_name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled')),
    oauth_configured BOOLEAN NOT NULL DEFAULT false,
    bot_configured BOOLEAN NOT NULL DEFAULT false,
    sync_enabled BOOLEAN NOT NULL DEFAULT false,
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, provider)
);

CREATE TABLE mcp_connectors (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    endpoint_url TEXT NOT NULL,
    auth_type TEXT NOT NULL CHECK (auth_type IN ('none', 'bearer', 'basic', 'api_key')),
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled')),
    description TEXT NOT NULL DEFAULT '',
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, name)
);

CREATE INDEX idx_im_provider_configs_entity ON im_provider_configs(entity_id);
CREATE INDEX idx_mcp_connectors_entity_status ON mcp_connectors(entity_id, status);

CREATE TABLE audit_logs (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    actor_user_id CHAR(26) CHECK (actor_user_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    actor_type TEXT NOT NULL CHECK (actor_type IN ('user', 'system', 'sync_job', 'api_client')),
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL DEFAULT '',
    resource_id TEXT NOT NULL DEFAULT '',
    before_state JSONB NOT NULL DEFAULT '{}'::jsonb,
    after_state JSONB NOT NULL DEFAULT '{}'::jsonb,
    ip TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    trace_id TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_entity_created ON audit_logs(entity_id, created_at DESC);
CREATE INDEX idx_audit_logs_entity_action ON audit_logs(entity_id, action);
CREATE INDEX idx_audit_logs_entity_resource ON audit_logs(entity_id, resource_type, resource_id);

CREATE TABLE sessions (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    user_id CHAR(26) NOT NULL CHECK (user_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    device_id TEXT NOT NULL DEFAULT '',
    ip TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    login_method TEXT NOT NULL CHECK (login_method IN ('feishu', 'dingtalk', 'password', 'token')),
    status TEXT NOT NULL CHECK (status IN ('active', 'revoked', 'expired')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    UNIQUE (entity_id, id),
    FOREIGN KEY (entity_id, user_id) REFERENCES users(entity_id, id) ON DELETE CASCADE
);

CREATE INDEX idx_sessions_entity_user ON sessions(entity_id, user_id);
CREATE INDEX idx_sessions_entity_status ON sessions(entity_id, status);
CREATE INDEX idx_sessions_expiry ON sessions(expires_at);

CREATE TABLE organizations (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    parent_id CHAR(26) CHECK (parent_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES organizations(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, id)
);

CREATE TABLE departments (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    organization_id CHAR(26) NOT NULL CHECK (organization_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    parent_id CHAR(26) CHECK (parent_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES departments(id) ON DELETE SET NULL,
    source_id CHAR(26) CHECK (source_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    external_department_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, id),
    UNIQUE (entity_id, organization_id, id),
    FOREIGN KEY (entity_id, source_id) REFERENCES identity_sources(entity_id, id) ON DELETE SET NULL
);

CREATE TABLE groups (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('manual', 'synced')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, id),
    UNIQUE (entity_id, name)
);

CREATE TABLE group_members (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    group_id CHAR(26) NOT NULL CHECK (group_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES groups(id) ON DELETE CASCADE,
    user_id CHAR(26) NOT NULL CHECK (user_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    added_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, group_id, user_id),
    FOREIGN KEY (entity_id, user_id) REFERENCES users(entity_id, id) ON DELETE CASCADE
);

CREATE INDEX idx_organizations_entity ON organizations(entity_id);
CREATE INDEX idx_departments_entity_org ON departments(entity_id, organization_id);
CREATE INDEX idx_departments_entity_source ON departments(entity_id, source_id);
CREATE INDEX idx_groups_entity ON groups(entity_id);
CREATE INDEX idx_group_members_group ON group_members(entity_id, group_id);
CREATE INDEX idx_group_members_user ON group_members(entity_id, user_id);

-- Add unique index for department mapping from external identity sources.
-- Only applies when source_id and external_department_id are present.
CREATE UNIQUE INDEX IF NOT EXISTS idx_departments_source_external
ON departments (entity_id, source_id, external_department_id)
WHERE source_id IS NOT NULL AND external_department_id IS NOT NULL;


CREATE TABLE legacy_app_users (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    application_id CHAR(26) NOT NULL CHECK (application_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    user_id CHAR(26) NOT NULL CHECK (user_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    username TEXT NOT NULL,
    legacy_user_identifier TEXT,
    auth_scheme TEXT NOT NULL CHECK (auth_scheme IN ('local')),
    credential_hash TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, application_id, username),
    UNIQUE (entity_id, application_id, legacy_user_identifier),
    FOREIGN KEY (entity_id, application_id) REFERENCES applications(entity_id, id) ON DELETE CASCADE,
    FOREIGN KEY (entity_id, user_id) REFERENCES users(entity_id, id) ON DELETE CASCADE
);

CREATE TABLE legacy_password_events (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    application_id CHAR(26) NOT NULL CHECK (application_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    user_id CHAR(26) CHECK (user_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    username TEXT,
    event TEXT NOT NULL CHECK (event IN ('success', 'failed', 'locked', 'disabled', 'access_denied')),
    client_ip TEXT,
    user_agent TEXT,
    trace_id TEXT,
    reason TEXT,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (entity_id, application_id) REFERENCES applications(entity_id, id) ON DELETE CASCADE,
    FOREIGN KEY (entity_id, user_id) REFERENCES users(entity_id, id) ON DELETE CASCADE
);

CREATE INDEX idx_legacy_app_users_entity_app ON legacy_app_users(entity_id, application_id);
CREATE UNIQUE INDEX idx_legacy_app_users_entity_app_legacy_identifier ON legacy_app_users(entity_id, application_id, legacy_user_identifier)
    WHERE legacy_user_identifier IS NOT NULL;
CREATE INDEX idx_legacy_password_events_lookup
    ON legacy_password_events(entity_id, application_id, username, occurred_at);
CREATE INDEX idx_legacy_password_events_entity_app_user
    ON legacy_password_events(entity_id, application_id, user_id, occurred_at);

-- +goose Down
DROP INDEX IF EXISTS idx_legacy_password_events_entity_app_user;
DROP INDEX IF EXISTS idx_legacy_password_events_lookup;
DROP INDEX IF EXISTS idx_legacy_app_users_entity_app_legacy_identifier;
DROP INDEX IF EXISTS idx_legacy_app_users_entity_app;
DROP TABLE IF EXISTS legacy_password_events;
DROP TABLE IF EXISTS legacy_app_users;

DROP INDEX IF EXISTS idx_departments_source_external;
DROP TABLE IF EXISTS group_members;
DROP TABLE IF EXISTS groups;
DROP TABLE IF EXISTS departments;
DROP TABLE IF EXISTS organizations;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS mcp_connectors;
DROP TABLE IF EXISTS im_provider_configs;
DROP TABLE IF EXISTS local_credentials;
DROP TABLE IF EXISTS application_assignments;
DROP TABLE IF EXISTS role_resource_scopes;
DROP TABLE IF EXISTS resource_scopes;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS sync_jobs;
DROP TABLE IF EXISTS directory_departments;
DROP TABLE IF EXISTS oauth_tokens;
DROP TABLE IF EXISTS oauth_authorization_codes;
DROP TABLE IF EXISTS oidc_clients;
DROP TABLE IF EXISTS applications;
DROP TABLE IF EXISTS account_bindings;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS directory_users;
DROP TABLE IF EXISTS identity_sources;
DROP TABLE IF EXISTS business_entities;
DROP FUNCTION IF EXISTS idb_generate_ulid;
