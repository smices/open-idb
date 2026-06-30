-- SPDX-License-Identifier: MIT

-- +goose Up
INSERT INTO roles (entity_id, code, name, description)
SELECT be.id, 'employee', '员工', '默认员工角色；用于登录后访问已授权的业务应用。'
FROM business_entities be
ON CONFLICT (entity_id, code) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    updated_at = now();

INSERT INTO user_roles (entity_id, user_id, role_id)
SELECT u.entity_id, u.id, r.id
FROM users u
JOIN roles r ON r.entity_id = u.entity_id AND r.code = 'employee'
WHERE u.user_type = 'employee'
ON CONFLICT DO NOTHING;

INSERT INTO application_assignments (entity_id, application_id, subject_type, subject_id, effect)
SELECT a.entity_id, a.id, 'role', r.id, 'allow'
FROM applications a
JOIN roles r ON r.entity_id = a.entity_id AND r.code = 'employee'
WHERE a.type = 'oidc_client'
  AND a.status = 'active'
ON CONFLICT (entity_id, application_id, subject_type, subject_id, effect) DO NOTHING;

-- +goose Down
DELETE FROM application_assignments aa
USING roles r
WHERE aa.entity_id = r.entity_id
  AND aa.subject_type = 'role'
  AND aa.subject_id = r.id
  AND r.code = 'employee';

DELETE FROM user_roles ur
USING roles r
WHERE ur.entity_id = r.entity_id
  AND ur.role_id = r.id
  AND r.code = 'employee';

DELETE FROM roles
WHERE code = 'employee';
