## What I implemented

- Added `ArchivedUserResponse` plus `archivedUserFromRow` in `backend/internal/adminapi/user_handlers.go`.
- Added admin service methods for archived users:
  - `ArchiveUser`
  - `ListArchivedUsers`
  - `CountArchivedUsers`
  - `GetArchivedUser`
- Changed admin deletion behavior so `POST /sapi/users/{id}/delete` routes through `setLifecycle(..., "deleted")`, which now calls `ArchiveUser` instead of `UpdateUserLifecycle`.
- Added read-only archived-user endpoints:
  - `GET /sapi/archived-users`
  - `GET /sapi/archived-users/{id}`
- Added transaction support to `AdminService` via `SetTxStarter`, and wired the real app service to the Postgres pool in `backend/internal/app/app.go`.
- Ensured archive insert, dependent cleanup, active-row delete, and DB-backed audit write happen in one transaction when the configured audit logger is the standard `*audit.Service`. For non-DB audit loggers, archive/delete still commit first and audit runs after commit.

## What I tested and exact results

- Focused RED check:
  - Command: `cd backend && go test ./internal/adminapi -run TestArchiveUserWritesArchiveAndDeletesActiveRow -count=1`
  - Result: `FAIL` with `svc.ArchiveUser undefined`
- Focused GREEN check:
  - Command: `cd backend && go test ./internal/adminapi -run TestArchiveUserWritesArchiveAndDeletesActiveRow -count=1`
  - Result: `ok  	github.com/smices/open-idb/internal/adminapi	0.472s`
- Full package check:
  - Command: `cd backend && go test ./internal/adminapi`
  - Result: `ok  	github.com/smices/open-idb/internal/adminapi	0.486s`

## TDD Evidence

### RED

- Added `backend/internal/adminapi/admin_service_test.go` with `TestArchiveUserWritesArchiveAndDeletesActiveRow`.
- Ran the focused test before implementation.
- Failure summary:
  - package build failed
  - compiler error: `internal/adminapi/admin_service_test.go:93:22: svc.ArchiveUser undefined (type *AdminService has no field or method ArchiveUser)`

### GREEN

- Implemented `ArchiveUser`, handler branching, archived-user endpoints, route coverage, and transaction wiring.
- Re-ran the focused test successfully.
- Re-ran the full `./internal/adminapi` package successfully.

## Files changed

- `backend/internal/adminapi/admin_service.go`
- `backend/internal/adminapi/admin_service_test.go`
- `backend/internal/adminapi/user_handlers.go`
- `backend/internal/adminapi/user_handlers_test.go`
- `backend/internal/app/app.go`

## Self-review findings

- The archive path now deletes the active `users` row after the archive record is written, which frees username reuse as required.
- Disabled and active lifecycle paths still use the existing `UpdateUserLifecycle` behavior.
- The app-level wiring change in `backend/internal/app/app.go` is required so runtime uses a real transaction starter.
- Handler tests cover the deleted branch and archived-user read endpoints.
- The service integration test exercises the critical user-visible invariant: archive row written, active row gone, username reusable.

## Issues or concerns

- None blocking.
