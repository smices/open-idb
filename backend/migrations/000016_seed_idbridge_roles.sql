-- SPDX-License-Identifier: MIT

-- +goose Up
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

-- +goose Down
WITH seeded_roles(code) AS (
    VALUES
        ('system_admin'),
        ('entity_admin'),
        ('identity_admin'),
        ('application_admin'),
        ('audit_viewer'),
        ('read_only')
),
seeded_permissions(code) AS (
    VALUES
        ('entities:read'),
        ('entities:manage'),
        ('identity_sources:read'),
        ('identity_sources:manage'),
        ('organization:read'),
        ('organization:sync'),
        ('users:read'),
        ('users:manage'),
        ('applications:read'),
        ('applications:manage'),
        ('roles:read'),
        ('roles:manage'),
        ('audit:read'),
        ('sync_jobs:read'),
        ('sync_jobs:manage')
)
DELETE FROM role_permissions rp
USING roles r, permissions p, seeded_roles sr, seeded_permissions sp
WHERE rp.entity_id = r.entity_id
  AND rp.role_id = r.id
  AND rp.entity_id = p.entity_id
  AND rp.permission_id = p.id
  AND r.code = sr.code
  AND p.code = sp.code;

DELETE FROM roles
WHERE code IN ('system_admin', 'entity_admin', 'identity_admin', 'application_admin', 'audit_viewer', 'read_only');

DELETE FROM permissions
WHERE code IN (
    'entities:read',
    'entities:manage',
    'identity_sources:read',
    'identity_sources:manage',
    'organization:read',
    'organization:sync',
    'users:read',
    'users:manage',
    'applications:read',
    'applications:manage',
    'roles:read',
    'roles:manage',
    'audit:read',
    'sync_jobs:read',
    'sync_jobs:manage'
);
