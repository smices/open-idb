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

CREATE TABLE platform_settings (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    platform_name TEXT NOT NULL DEFAULT 'IdBridge',
    logo_url TEXT NOT NULL DEFAULT '',
    favicon_url TEXT NOT NULL DEFAULT '',
    title_suffix TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
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
    english_name TEXT NOT NULL DEFAULT '',
    employee_no TEXT NOT NULL DEFAULT '',
    job_title TEXT NOT NULL DEFAULT '',
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
    english_name TEXT NOT NULL DEFAULT '',
    employee_no TEXT NOT NULL DEFAULT '',
    job_title TEXT NOT NULL DEFAULT '',
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

CREATE TABLE archived_users (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    original_user_id CHAR(26) NOT NULL CHECK (original_user_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    username TEXT NOT NULL,
    display_name TEXT NOT NULL,
    english_name TEXT NOT NULL DEFAULT '',
    employee_no TEXT NOT NULL DEFAULT '',
    job_title TEXT NOT NULL DEFAULT '',
    email TEXT,
    phone TEXT,
    avatar_url TEXT,
    user_type TEXT NOT NULL,
    primary_source_id CHAR(26) CHECK (primary_source_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    locale TEXT,
    original_created_at TIMESTAMPTZ NOT NULL,
    original_updated_at TIMESTAMPTZ NOT NULL,
    archived_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    archived_by_user_id CHAR(26) CHECK (archived_by_user_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    archive_reason TEXT NOT NULL DEFAULT '',
    user_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    bindings_snapshot JSONB NOT NULL DEFAULT '[]'::jsonb,
    roles_snapshot JSONB NOT NULL DEFAULT '[]'::jsonb,
    UNIQUE (entity_id, original_user_id)
);

CREATE INDEX idx_archived_users_entity_archived
    ON archived_users(entity_id, archived_at DESC);

CREATE INDEX idx_archived_users_entity_username
    ON archived_users(entity_id, username);

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
    workplace_provider TEXT NOT NULL DEFAULT '',
    workplace_app_id TEXT NOT NULL DEFAULT '',
    workplace_app_secret TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, id),
    UNIQUE (entity_id, application_id),
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

INSERT INTO platform_settings (id, platform_name, logo_url, favicon_url, title_suffix)
VALUES (1, 'IdBridge', '', '', '')
ON CONFLICT (id) DO UPDATE SET
    platform_name = EXCLUDED.platform_name,
    logo_url = EXCLUDED.logo_url,
    favicon_url = EXCLUDED.favicon_url,
    title_suffix = EXCLUDED.title_suffix,
    updated_at = now();

WITH entity_row AS (
    INSERT INTO business_entities (name, slug, status, default_locale, brand_name, logo_url, login_message)
    VALUES ('Default Enterprise', 'default_enterprise', 'active', 'zh-CN', 'Default Enterprise', '', 'Enterprise identity access.')
    ON CONFLICT (slug) DO UPDATE SET
        status = EXCLUDED.status
    RETURNING id
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
        id,
        'admin',
        'Administrator',
        'admin@idbridge.local',
        'active',
        'employee',
        NULL,
        'en-US'
    FROM entity_row
    ON CONFLICT (entity_id, username) DO UPDATE SET
        display_name = EXCLUDED.display_name,
        lifecycle_status = 'active'
    RETURNING id, entity_id
)
INSERT INTO local_credentials (entity_id, user_id, password_hash, must_change_password, weak_password)
SELECT entity_id, id, crypt('admin123', gen_salt('bf')), true, true
FROM user_row
ON CONFLICT (entity_id, user_id) DO NOTHING;

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

WITH permission_seed(code, name, type) AS (
    VALUES
        ('entities:read', '查看公司', 'action'),
        ('entities:manage', '管理公司', 'action'),
        ('identity_sources:read', '查看身份源', 'action'),
        ('identity_sources:manage', '管理身份源', 'action'),
        ('organization:read', '查看组织架构', 'action'),
        ('organization:sync', '同步组织架构', 'action'),
        ('users:read', '查看账号', 'action'),
        ('users:manage', '管理账号', 'action'),
        ('applications:read', '查看应用', 'action'),
        ('applications:manage', '管理应用', 'action'),
        ('roles:read', '查看角色', 'action'),
        ('roles:manage', '管理角色', 'action'),
        ('audit:read', '查看审计日志', 'action'),
        ('sync_jobs:read', '查看同步任务', 'action'),
        ('sync_jobs:manage', '管理同步任务', 'action')
)
INSERT INTO permissions (entity_id, code, name, type)
SELECT be.id, ps.code, ps.name, ps.type
FROM business_entities be
CROSS JOIN permission_seed ps
ON CONFLICT (entity_id, code) DO UPDATE SET
    name = EXCLUDED.name,
    type = EXCLUDED.type,
    updated_at = now();

WITH role_seed(code, name, description) AS (
    VALUES
        ('employee', '员工', '默认员工角色；用于登录后访问已授权的业务应用。'),
        ('system_admin', '系统管理员', '管理公司、身份源、组织、账号、应用、角色、同步任务和审计日志。'),
        ('entity_admin', '公司管理员', '负责本公司身份源、组织架构、账号、应用和同步任务管理。'),
        ('identity_admin', '身份管理员', '负责身份源、组织架构、账号和同步任务管理。'),
        ('application_admin', '应用管理员', '负责应用接入配置和应用授权相关管理。'),
        ('audit_viewer', '审计员', '只读查看审计日志和关键运行记录。'),
        ('read_only', '只读查看员', '只读查看公司、组织、账号、应用、角色和同步任务。')
)
INSERT INTO roles (entity_id, code, name, description)
SELECT be.id, rs.code, rs.name, rs.description
FROM business_entities be
CROSS JOIN role_seed rs
ON CONFLICT (entity_id, code) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    updated_at = now();

WITH role_permission_seed(role_code, permission_code) AS (
    VALUES
        ('system_admin', 'entities:read'),
        ('system_admin', 'entities:manage'),
        ('system_admin', 'identity_sources:read'),
        ('system_admin', 'identity_sources:manage'),
        ('system_admin', 'organization:read'),
        ('system_admin', 'organization:sync'),
        ('system_admin', 'users:read'),
        ('system_admin', 'users:manage'),
        ('system_admin', 'applications:read'),
        ('system_admin', 'applications:manage'),
        ('system_admin', 'roles:read'),
        ('system_admin', 'roles:manage'),
        ('system_admin', 'audit:read'),
        ('system_admin', 'sync_jobs:read'),
        ('system_admin', 'sync_jobs:manage'),
        ('entity_admin', 'identity_sources:read'),
        ('entity_admin', 'identity_sources:manage'),
        ('entity_admin', 'organization:read'),
        ('entity_admin', 'organization:sync'),
        ('entity_admin', 'users:read'),
        ('entity_admin', 'users:manage'),
        ('entity_admin', 'applications:read'),
        ('entity_admin', 'applications:manage'),
        ('entity_admin', 'roles:read'),
        ('entity_admin', 'audit:read'),
        ('entity_admin', 'sync_jobs:read'),
        ('entity_admin', 'sync_jobs:manage'),
        ('identity_admin', 'identity_sources:read'),
        ('identity_admin', 'identity_sources:manage'),
        ('identity_admin', 'organization:read'),
        ('identity_admin', 'organization:sync'),
        ('identity_admin', 'users:read'),
        ('identity_admin', 'users:manage'),
        ('identity_admin', 'sync_jobs:read'),
        ('identity_admin', 'sync_jobs:manage'),
        ('application_admin', 'applications:read'),
        ('application_admin', 'applications:manage'),
        ('application_admin', 'users:read'),
        ('application_admin', 'roles:read'),
        ('audit_viewer', 'audit:read'),
        ('audit_viewer', 'sync_jobs:read'),
        ('read_only', 'entities:read'),
        ('read_only', 'identity_sources:read'),
        ('read_only', 'organization:read'),
        ('read_only', 'users:read'),
        ('read_only', 'applications:read'),
        ('read_only', 'roles:read'),
        ('read_only', 'audit:read'),
        ('read_only', 'sync_jobs:read')
)
INSERT INTO role_permissions (entity_id, role_id, permission_id)
SELECT r.entity_id, r.id, p.id
FROM role_permission_seed rps
JOIN roles r ON r.code = rps.role_code
JOIN permissions p ON p.entity_id = r.entity_id AND p.code = rps.permission_code
ON CONFLICT DO NOTHING;

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

CREATE INDEX idx_mcp_connectors_entity_status ON mcp_connectors(entity_id, status);

CREATE TABLE audit_logs (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    actor_user_id CHAR(26) CHECK (actor_user_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    actor_type TEXT NOT NULL CHECK (actor_type IN ('user', 'admin', 'system', 'sync_job', 'api_client')),
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


-- +goose Down
DROP INDEX IF EXISTS idx_departments_source_external;
DROP TABLE IF EXISTS group_members;
DROP TABLE IF EXISTS groups;
DROP TABLE IF EXISTS departments;
DROP TABLE IF EXISTS organizations;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS mcp_connectors;
DROP TABLE IF EXISTS admin_sessions;
DROP TABLE IF EXISTS admin_credentials;
DROP TABLE IF EXISTS admin_users;
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
DROP TABLE IF EXISTS archived_users;
DROP TABLE IF EXISTS account_bindings;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS directory_users;
DROP TABLE IF EXISTS identity_sources;
DROP TABLE IF EXISTS business_entities;
DROP FUNCTION IF EXISTS idb_generate_ulid;
