-- SPDX-License-Identifier: MIT

-- +goose Up
DELETE FROM oidc_clients
WHERE id IN (
    SELECT id
    FROM (
        SELECT
            id,
            row_number() OVER (
                PARTITION BY entity_id, application_id
                ORDER BY created_at DESC, id DESC
            ) AS rn
        FROM oidc_clients
    ) ranked
    WHERE rn > 1
);

ALTER TABLE oidc_clients
    ADD CONSTRAINT oidc_clients_entity_application_unique UNIQUE (entity_id, application_id);

-- +goose Down
ALTER TABLE oidc_clients
    DROP CONSTRAINT IF EXISTS oidc_clients_entity_application_unique;
