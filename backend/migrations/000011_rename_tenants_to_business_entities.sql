-- SPDX-License-Identifier: MIT

-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF to_regclass('public.business_entities') IS NULL
       AND to_regclass('public.tenants') IS NOT NULL THEN
        ALTER TABLE public.tenants RENAME TO business_entities;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF to_regclass('public.tenants') IS NULL
       AND to_regclass('public.business_entities') IS NOT NULL THEN
        ALTER TABLE public.business_entities RENAME TO tenants;
    END IF;
END
$$;
-- +goose StatementEnd
