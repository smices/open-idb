-- +goose Up
-- Local accounts are not a primary enterprise identity source. Keep user records,
-- but remove the synthetic source and dependent directory artifacts.
WITH local_sources AS (
    SELECT entity_id, id
    FROM identity_sources
    WHERE type = 'local'
)
UPDATE users u
SET primary_source_id = NULL,
    updated_at = now()
FROM local_sources s
WHERE u.entity_id = s.entity_id
  AND u.primary_source_id = s.id;

WITH local_sources AS (
    SELECT entity_id, id
    FROM identity_sources
    WHERE type = 'local'
)
UPDATE departments d
SET source_id = NULL,
    external_department_id = NULL
FROM local_sources s
WHERE d.entity_id = s.entity_id
  AND d.source_id = s.id;

DELETE FROM identity_sources
WHERE type = 'local';

-- +goose Down
-- Intentionally no-op. Local accounts are login credentials, not identity sources.
