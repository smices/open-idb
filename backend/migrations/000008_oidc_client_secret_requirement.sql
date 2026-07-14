-- SPDX-License-Identifier: MIT

-- +goose Up
ALTER TABLE oidc_clients
    ADD COLUMN IF NOT EXISTS secret_required BOOLEAN;

-- Existing clients and writes from an older binary during a rolling upgrade
-- keep anonymous token-endpoint compatibility. The new CreateOIDCClient query
-- explicitly opts newly managed clients into secret verification.
UPDATE oidc_clients
SET secret_required = false
WHERE secret_required IS NULL;

ALTER TABLE oidc_clients
    ALTER COLUMN secret_required SET DEFAULT false,
    ALTER COLUMN secret_required SET NOT NULL;

-- +goose Down
ALTER TABLE oidc_clients
    DROP COLUMN IF EXISTS secret_required;
