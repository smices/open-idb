-- SPDX-License-Identifier: MIT

-- name: CreateSyncJob :one
INSERT INTO sync_jobs (entity_id, source_id, type, provider, status, trace_id)
VALUES ($1, $2, $3, $4, 'running', $5)
RETURNING id, entity_id, source_id, type, provider, status, trace_id, started_at, finished_at, error_message, stats, event_id, attempt_count, next_attempt_at, last_attempt_at;

-- name: CreateWebhookSyncJob :one
INSERT INTO sync_jobs (entity_id, source_id, type, provider, status, trace_id, event_id)
VALUES ($1, $2, 'webhook', $3, 'running', $4, $5)
ON CONFLICT (entity_id, source_id, event_id)
    WHERE type = 'webhook' AND event_id IS NOT NULL
DO UPDATE SET event_id = EXCLUDED.event_id
RETURNING id, entity_id, source_id, type, provider, status, trace_id, started_at, finished_at, error_message, stats, event_id, attempt_count, next_attempt_at, last_attempt_at;

-- name: FinishSyncJob :one
UPDATE sync_jobs
SET status = 'succeeded',
    finished_at = now(),
    error_message = NULL,
    stats = $3
WHERE entity_id = $1 AND id = $2
RETURNING id, entity_id, source_id, type, provider, status, trace_id, started_at, finished_at, error_message, stats, event_id, attempt_count, next_attempt_at, last_attempt_at;

-- name: FailSyncJob :one
UPDATE sync_jobs
SET status = 'failed',
    finished_at = now(),
    error_message = $3,
    stats = $4
WHERE entity_id = $1 AND id = $2
RETURNING id, entity_id, source_id, type, provider, status, trace_id, started_at, finished_at, error_message, stats, event_id, attempt_count, next_attempt_at, last_attempt_at;

-- name: ClaimDueWebhookSyncSources :many
WITH due_sources AS MATERIALIZED (
    SELECT
        job.entity_id,
        job.source_id,
        min(job.provider)::text AS provider,
        min(job.started_at) AS oldest_started_at
    FROM sync_jobs job
    LEFT JOIN webhook_sync_leases lease
      ON lease.entity_id = job.entity_id
     AND lease.source_id = job.source_id
    WHERE job.type = 'webhook'
      AND job.status = 'running'
      AND job.next_attempt_at <= now()
      AND (lease.entity_id IS NULL OR lease.lease_expires_at <= now())
    GROUP BY job.entity_id, job.source_id
    ORDER BY oldest_started_at
    LIMIT sqlc.arg('batch_size')::integer
), claimed AS (
    INSERT INTO webhook_sync_leases (entity_id, source_id, claim_token, lease_expires_at)
    SELECT
        entity_id,
        source_id,
        sqlc.arg('claim_token')::text,
        now() + make_interval(secs => sqlc.arg('lease_seconds')::integer)
    FROM due_sources
    ON CONFLICT (entity_id, source_id) DO UPDATE
    SET claim_token = EXCLUDED.claim_token,
        lease_expires_at = EXCLUDED.lease_expires_at,
        updated_at = now()
    WHERE webhook_sync_leases.lease_expires_at <= now()
    RETURNING entity_id, source_id, claim_token
)
SELECT claimed.entity_id, claimed.source_id, due_sources.provider, claimed.claim_token
FROM claimed
JOIN due_sources
  ON due_sources.entity_id = claimed.entity_id
 AND due_sources.source_id = claimed.source_id
ORDER BY due_sources.oldest_started_at;

-- name: ClaimDueWebhookJobsBySource :many
WITH due_jobs AS (
    SELECT job.id
    FROM sync_jobs job
    WHERE job.entity_id = sqlc.arg('entity_id')
      AND job.source_id = sqlc.arg('source_id')
      AND job.type = 'webhook'
      AND job.status = 'running'
      AND job.next_attempt_at <= now()
      AND (
          sqlc.arg('claim_token')::text = ''
          OR EXISTS (
              SELECT 1
              FROM webhook_sync_leases lease
              WHERE lease.entity_id = job.entity_id
                AND lease.source_id = job.source_id
                AND lease.claim_token = sqlc.arg('claim_token')::text
                AND lease.lease_expires_at > now()
          )
      )
    ORDER BY job.started_at
    LIMIT sqlc.arg('batch_size')::integer
    FOR UPDATE SKIP LOCKED
)
UPDATE sync_jobs job
SET attempt_count = job.attempt_count + 1,
    last_attempt_at = now(),
    next_attempt_at = now() + make_interval(secs => sqlc.arg('lease_seconds')::integer)
FROM due_jobs
WHERE job.id = due_jobs.id
RETURNING job.id, job.entity_id, job.source_id, job.type, job.provider, job.status, job.trace_id, job.started_at, job.finished_at, job.error_message, job.stats, job.event_id, job.attempt_count, job.next_attempt_at, job.last_attempt_at;

-- name: RescheduleWebhookSyncJob :one
UPDATE sync_jobs
SET status = 'running',
    finished_at = NULL,
    error_message = sqlc.arg('error_message'),
    stats = sqlc.arg('stats'),
    next_attempt_at = now() + make_interval(secs => sqlc.arg('delay_seconds')::integer)
WHERE entity_id = sqlc.arg('entity_id')
  AND id = sqlc.arg('id')
  AND type = 'webhook'
  AND status = 'running'
RETURNING id, entity_id, source_id, type, provider, status, trace_id, started_at, finished_at, error_message, stats, event_id, attempt_count, next_attempt_at, last_attempt_at;

-- name: ReleaseWebhookSyncLease :execrows
DELETE FROM webhook_sync_leases
WHERE entity_id = $1
  AND source_id = $2
  AND claim_token = $3;

-- name: ListSyncJobsBySource :many
SELECT id, entity_id, source_id, type, provider, status, trace_id, started_at, finished_at, error_message, stats, event_id, attempt_count, next_attempt_at, last_attempt_at
FROM sync_jobs
WHERE entity_id = $1 AND source_id = $2
ORDER BY started_at DESC
LIMIT $3;
