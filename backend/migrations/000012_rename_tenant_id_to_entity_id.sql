-- SPDX-License-Identifier: MIT

-- +goose Up
-- +goose StatementBegin
DO $$
DECLARE
    current_table TEXT;
    tables TEXT[] := ARRAY[
        'account_bindings',
        'application_assignments',
        'applications',
        'audit_logs',
        'departments',
        'directory_departments',
        'directory_users',
        'group_members',
        'groups',
        'identity_sources',
        'im_provider_configs',
        'legacy_app_users',
        'legacy_password_events',
        'local_credentials',
        'mcp_connectors',
        'oauth_authorization_codes',
        'oauth_tokens',
        'oidc_clients',
        'organizations',
        'permissions',
        'resource_scopes',
        'role_permissions',
        'role_resource_scopes',
        'roles',
        'sessions',
        'sync_jobs',
        'user_roles',
        'users'
    ];
BEGIN
    FOREACH current_table IN ARRAY tables LOOP
        IF EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = current_table
              AND column_name = 'tenant_id'
        ) AND NOT EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = current_table
              AND column_name = 'entity_id'
        ) THEN
            EXECUTE format('ALTER TABLE public.%I RENAME COLUMN tenant_id TO entity_id', current_table);
        END IF;
    END LOOP;
END
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
DECLARE
    current_table TEXT;
    tables TEXT[] := ARRAY[
        'account_bindings',
        'application_assignments',
        'applications',
        'audit_logs',
        'departments',
        'directory_departments',
        'directory_users',
        'group_members',
        'groups',
        'identity_sources',
        'im_provider_configs',
        'legacy_app_users',
        'legacy_password_events',
        'local_credentials',
        'mcp_connectors',
        'oauth_authorization_codes',
        'oauth_tokens',
        'oidc_clients',
        'organizations',
        'permissions',
        'resource_scopes',
        'role_permissions',
        'role_resource_scopes',
        'roles',
        'sessions',
        'sync_jobs',
        'user_roles',
        'users'
    ];
BEGIN
    FOREACH current_table IN ARRAY tables LOOP
        IF EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = current_table
              AND column_name = 'entity_id'
        ) AND NOT EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = current_table
              AND column_name = 'tenant_id'
        ) THEN
            EXECUTE format('ALTER TABLE public.%I RENAME COLUMN entity_id TO tenant_id', current_table);
        END IF;
    END LOOP;
END
$$;
-- +goose StatementEnd
