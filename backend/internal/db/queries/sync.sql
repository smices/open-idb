-- SPDX-License-Identifier: MIT

-- name: CreateSyncJob :one
INSERT INTO sync_jobs (entity_id, source_id, type, provider, status, trace_id)
VALUES ($1, $2, $3, $4, 'running', $5)
RETURNING id, entity_id, source_id, type, provider, status, trace_id, started_at, finished_at, error_message, stats;

-- name: FinishSyncJob :one
UPDATE sync_jobs
SET status = 'succeeded',
    finished_at = now(),
    stats = $3
WHERE entity_id = $1 AND id = $2
RETURNING id, entity_id, source_id, type, provider, status, trace_id, started_at, finished_at, error_message, stats;

-- name: FailSyncJob :one
UPDATE sync_jobs
SET status = 'failed',
    finished_at = now(),
    error_message = $3,
    stats = $4
WHERE entity_id = $1 AND id = $2
RETURNING id, entity_id, source_id, type, provider, status, trace_id, started_at, finished_at, error_message, stats;

-- name: ListSyncJobsBySource :many
SELECT id, entity_id, source_id, type, provider, status, trace_id, started_at, finished_at, error_message, stats
FROM sync_jobs
WHERE entity_id = $1 AND source_id = $2
ORDER BY started_at DESC
LIMIT $3;
