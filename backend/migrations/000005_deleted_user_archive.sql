-- SPDX-License-Identifier: MIT

-- +goose Up
CREATE TABLE IF NOT EXISTS archived_users (
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

CREATE INDEX IF NOT EXISTS idx_archived_users_entity_archived
    ON archived_users(entity_id, archived_at DESC);

CREATE INDEX IF NOT EXISTS idx_archived_users_entity_username
    ON archived_users(entity_id, username);

WITH deleted_users AS (
    SELECT u.*
    FROM users u
    WHERE u.lifecycle_status = 'deleted'
),
inserted AS (
    INSERT INTO archived_users (
        entity_id,
        original_user_id,
        username,
        display_name,
        english_name,
        employee_no,
        job_title,
        email,
        phone,
        avatar_url,
        user_type,
        primary_source_id,
        locale,
        original_created_at,
        original_updated_at,
        archived_at,
        archive_reason,
        user_snapshot,
        bindings_snapshot,
        roles_snapshot
    )
    SELECT
        u.entity_id,
        u.id,
        u.username,
        u.display_name,
        u.english_name,
        u.employee_no,
        u.job_title,
        u.email,
        u.phone,
        u.avatar_url,
        u.user_type,
        u.primary_source_id,
        u.locale,
        u.created_at,
        u.updated_at,
        now(),
        'migrated deleted lifecycle user',
        to_jsonb(u),
        COALESCE((
            SELECT jsonb_agg(to_jsonb(ab) ORDER BY ab.bound_at)
            FROM account_bindings ab
            WHERE ab.entity_id = u.entity_id AND ab.user_id = u.id
        ), '[]'::jsonb),
        COALESCE((
            SELECT jsonb_agg(to_jsonb(ur) ORDER BY ur.role_id)
            FROM user_roles ur
            WHERE ur.entity_id = u.entity_id AND ur.user_id = u.id
        ), '[]'::jsonb)
    FROM deleted_users u
    ON CONFLICT (entity_id, original_user_id) DO NOTHING
    RETURNING entity_id, original_user_id
),
migrated_users AS (
    SELECT i.entity_id, i.original_user_id
    FROM inserted i
    UNION
    SELECT du.entity_id, du.id AS original_user_id
    FROM deleted_users du
    JOIN archived_users au
      ON au.entity_id = du.entity_id
     AND au.original_user_id = du.id
),
deleted_application_assignments AS (
    DELETE FROM application_assignments aa
    USING migrated_users mu
    WHERE aa.entity_id = mu.entity_id
      AND aa.subject_type = 'user'
      AND aa.subject_id = mu.original_user_id
    RETURNING aa.entity_id, aa.subject_id
)
DELETE FROM users u
USING migrated_users mu
WHERE u.entity_id = mu.entity_id
  AND u.id = mu.original_user_id
  AND u.lifecycle_status = 'deleted';

-- +goose Down
DROP TABLE IF EXISTS archived_users;
