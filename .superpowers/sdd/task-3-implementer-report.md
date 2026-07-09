# Task 3 Implementer Report

## What I implemented
- Replaced the full-sync managed-user soft-delete path with archive-and-delete behavior in `backend/internal/idp/sync.go`.
- Added `ListManagedUsersForDeletedDirectoryUsers` to `backend/internal/db/queries/identity.sql` and regenerated sqlc output.
- Added per-user transaction handling in sync so archive insert, active dependent cleanup, and active user row delete commit atomically.
- Preserved sync stats semantics: `ManagedUsersDeleted` increments once per archived managed user.
- Updated sync audit emission to point at `archived_user` resources for full-sync removals.
- Added a real DB-backed regression test in `backend/internal/idp/sync_test.go`.
- Wired the app and existing Feishu sync integration tests to provide the sync service transaction starter.

## What I tested and exact results
- `cd backend && go test ./internal/idp -run TestFullSyncArchivesMissingManagedUsers -count=1 -v`
  - Result with `DOCKER_HOST=unix:///Users/jacky/.orbstack/run/docker.sock`: `PASS`
- `make generate`
  - Result: `cd backend && CGO_ENABLED=0 go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.27.0 generate -f internal/db/sqlc.yaml`
- `cd backend && go test ./internal/db/generated -run TestIdentitySQL -count=1`
  - Result: `ok  	github.com/smices/open-idb/internal/db/generated	0.451s [no tests to run]`
- `cd backend && go test ./internal/idp`
  - Result: `ok  	github.com/smices/open-idb/internal/idp	0.170s`
- `cd backend && DOCKER_HOST=unix:///Users/jacky/.orbstack/run/docker.sock go test ./internal/idp`
  - Result: `ok  	github.com/smices/open-idb/internal/idp	2.098s`

## TDD Evidence
### RED
- Command:
  - `cd backend && DOCKER_HOST=unix:///Users/jacky/.orbstack/run/docker.sock go test ./internal/idp -run TestFullSyncArchivesMissingManagedUsers -count=1 -v`
- Summary:
  - Test failed before implementation.
  - Failure: `expected active user row to be deleted, got <nil>`
  - Result: `FAIL`

### GREEN
- Command:
  - `cd backend && DOCKER_HOST=unix:///Users/jacky/.orbstack/run/docker.sock go test ./internal/idp -run TestFullSyncArchivesMissingManagedUsers -count=1 -v`
- Summary:
  - Test passed after implementation.
  - Result: `ok  	github.com/smices/open-idb/internal/idp	2.364s`

## Files changed
- `backend/internal/idp/sync.go`
- `backend/internal/idp/sync_test.go`
- `backend/internal/db/queries/identity.sql`
- `backend/internal/db/generated/identity.sql.go`
- `backend/internal/app/app.go`
- `backend/tests/integration/feishu_sync_test.go`

## Self-review findings
- The archive/delete sequence now runs inside a transaction per managed user, which satisfies the no-half-archived/no-half-active invariant for each removal.
- The sync service now requires a transaction starter when the full-sync deletion/archive path is exercised; app wiring and sync-related tests were updated accordingly.
- Integration coverage outside `./internal/idp` was updated for the changed behavior, but I did not run the full integration suite in this task.

## Issues or concerns
- In this environment, plain container-backed Go tests do not see Docker automatically. Setting `DOCKER_HOST=unix:///Users/jacky/.orbstack/run/docker.sock` was required to collect real RED/GREEN evidence instead of a skipped test.
