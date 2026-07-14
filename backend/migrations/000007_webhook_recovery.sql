-- SPDX-License-Identifier: MIT

-- Keep the short ALTER TABLE locks separate from historical backfill and build
-- indexes concurrently so an online upgrade does not block sync_jobs traffic
-- for the duration of the migration. Every step is idempotent for safe retry.
-- +goose NO TRANSACTION

-- +goose Up
ALTER TABLE sync_jobs
    ADD COLUMN IF NOT EXISTS event_id TEXT,
    ADD COLUMN IF NOT EXISTS attempt_count INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN IF NOT EXISTS last_attempt_at TIMESTAMPTZ;

-- +goose StatementBegin
DO $block$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = 'sync_jobs'::regclass
          AND conname = 'sync_jobs_attempt_count_nonnegative'
    ) THEN
        ALTER TABLE sync_jobs
            ADD CONSTRAINT sync_jobs_attempt_count_nonnegative
            CHECK (attempt_count >= 0) NOT VALID;
    END IF;
END;
$block$;
-- +goose StatementEnd

ALTER TABLE sync_jobs
    VALIDATE CONSTRAINT sync_jobs_attempt_count_nonnegative;

-- Existing webhook rows remain immediately recoverable. Older releases stored
-- the serialized event in trace_id, so preserve its provider event ID when the
-- payload is valid JSON and use the row ULID as a collision-free fallback.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION idb_migration_webhook_event_id(raw_trace TEXT)
RETURNS TEXT
LANGUAGE plpgsql
IMMUTABLE
AS $function$
BEGIN
    RETURN COALESCE((raw_trace::jsonb)->>'EventID', (raw_trace::jsonb)->>'event_id');
EXCEPTION WHEN OTHERS THEN
    RETURN NULL;
END;
$function$;
-- +goose StatementEnd

UPDATE sync_jobs
SET event_id = COALESCE(NULLIF(btrim(idb_migration_webhook_event_id(trace_id)), ''), id)
WHERE type = 'webhook'
  AND event_id IS NULL;

DROP FUNCTION idb_migration_webhook_event_id(TEXT);

-- Older versions did not deduplicate deliveries, so the same provider event
-- may already exist more than once. Preserve every historical job while
-- keeping one canonical deduplication key; NULL rows remain processable but do
-- not prevent the partial unique index from being installed.
WITH ranked_webhook_events AS (
    SELECT id,
           row_number() OVER (
               PARTITION BY entity_id, source_id, event_id
               ORDER BY started_at, id
           ) AS duplicate_rank
    FROM sync_jobs
    WHERE type = 'webhook'
      AND event_id IS NOT NULL
)
UPDATE sync_jobs job
SET event_id = NULL
FROM ranked_webhook_events ranked
WHERE job.id = ranked.id
  AND ranked.duplicate_rank > 1;

DROP INDEX CONCURRENTLY IF EXISTS idx_sync_jobs_webhook_event;
CREATE UNIQUE INDEX CONCURRENTLY idx_sync_jobs_webhook_event
    ON sync_jobs(entity_id, source_id, event_id)
    WHERE type = 'webhook' AND event_id IS NOT NULL;

DROP INDEX CONCURRENTLY IF EXISTS idx_sync_jobs_due_webhook;
CREATE INDEX CONCURRENTLY idx_sync_jobs_due_webhook
    ON sync_jobs(next_attempt_at, entity_id, source_id)
    INCLUDE (provider)
    WHERE type = 'webhook' AND status = 'running';

CREATE TABLE IF NOT EXISTS webhook_sync_leases (
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    source_id CHAR(26) NOT NULL CHECK (source_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    claim_token TEXT NOT NULL CHECK (btrim(claim_token) <> ''),
    lease_expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (entity_id, source_id),
    FOREIGN KEY (entity_id, source_id) REFERENCES identity_sources(entity_id, id) ON DELETE CASCADE
);

DROP INDEX CONCURRENTLY IF EXISTS idx_webhook_sync_leases_expiry;
CREATE INDEX CONCURRENTLY idx_webhook_sync_leases_expiry
    ON webhook_sync_leases(lease_expires_at);

-- +goose Down
DROP TABLE IF EXISTS webhook_sync_leases;
DROP INDEX CONCURRENTLY IF EXISTS idx_sync_jobs_due_webhook;
DROP INDEX CONCURRENTLY IF EXISTS idx_sync_jobs_webhook_event;

ALTER TABLE sync_jobs
    DROP COLUMN IF EXISTS last_attempt_at,
    DROP COLUMN IF EXISTS next_attempt_at,
    DROP COLUMN IF EXISTS attempt_count,
    DROP COLUMN IF EXISTS event_id;
