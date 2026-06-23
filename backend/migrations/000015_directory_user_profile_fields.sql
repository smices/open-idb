-- SPDX-License-Identifier: MIT

-- +goose Up
ALTER TABLE directory_users
    ADD COLUMN english_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN employee_no TEXT NOT NULL DEFAULT '',
    ADD COLUMN job_title TEXT NOT NULL DEFAULT '';

ALTER TABLE users
    ADD COLUMN english_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN employee_no TEXT NOT NULL DEFAULT '',
    ADD COLUMN job_title TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE users
    DROP COLUMN IF EXISTS job_title,
    DROP COLUMN IF EXISTS employee_no,
    DROP COLUMN IF EXISTS english_name;

ALTER TABLE directory_users
    DROP COLUMN IF EXISTS job_title,
    DROP COLUMN IF EXISTS employee_no,
    DROP COLUMN IF EXISTS english_name;
