# Deleted User Archive Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace deleted managed-user rows with irreversible archived-user records so deleted accounts no longer occupy active usernames or bindings.

**Architecture:** Add an `archived_users` table and generated queries, then route all user deletion paths through an archive-and-delete transaction. Keep disabled and locked users in `users`; remove only deleted users from active account tables after their snapshots are stored.

**Tech Stack:** Go, PostgreSQL, Goose migrations, sqlc, React/Ant Design admin UI.

## Global Constraints

- Deleted users do not need to be restored to their original account.
- Deleted users only need to be retained as records, similar to audit/history.
- Disabled and deleted are different states.
- Disabled/locked users remain managed accounts.
- Deleted users are no longer managed accounts.
- Archived rows must not participate in login, sync, binding, or username uniqueness.
- Audit logs are immutable and must not be rewritten.

---

## File Structure

- Create `backend/migrations/000005_deleted_user_archive.sql`: production migration, archive table, backfill existing deleted users, remove deleted rows from active `users`.
- Modify `backend/migrations/000001_schema_baseline.sql`: align new installs with the archive model.
- Create `backend/internal/db/queries/archive.sql`: sqlc queries for archiving, listing archived users, and deleting dependent active rows.
- Modify `backend/internal/db/sqlc.yaml` only if the new query file is not included by the existing glob.
- Modify generated files under `backend/internal/db/generated/` using `make generate`.
- Modify `backend/internal/adminapi/admin_service.go`: add `ArchiveUser`, replace delete lifecycle behavior.
- Modify `backend/internal/adminapi/user_handlers.go`: expose archived-user list/read endpoints and route delete lifecycle to archive behavior.
- Modify `backend/internal/idp/sync.go`: full sync archives managed users missing from the directory instead of setting `lifecycle_status = 'deleted'`.
- Modify `web/src/lib/api.ts`: archived-user API methods and types.
- Modify `web/src/admin-pages.jsx`: optional read-only archived users page or audit-linked archive details.
- Modify `web/src/i18n/index.js`: labels for archived users.
- Add/update tests in `backend/internal/adminapi`, `backend/internal/idp`, and migration/schema tests.

---

### Task 1: Archive Schema And Queries

**Files:**
- Create: `backend/migrations/000005_deleted_user_archive.sql`
- Modify: `backend/migrations/000001_schema_baseline.sql`
- Create: `backend/internal/db/queries/archive.sql`
- Generated: `backend/internal/db/generated/archive.sql.go`
- Test: `backend/internal/id/ulid_policy_test.go`

**Interfaces:**
- Produces: `generated.ArchiveUser`, `generated.ListArchivedUsers`, `generated.GetArchivedUserByOriginalID`, `generated.DeleteUserActiveDependents`, `generated.DeleteUserActiveRow`.
- Consumes: current schema tables `users`, `account_bindings`, `sessions`, `oauth_authorization_codes`, `oauth_tokens`, `user_roles`, `local_credentials`, `application_assignments`.

- [ ] **Step 1: Write the schema expectation test**

Modify `backend/internal/id/ulid_policy_test.go` to assert the baseline schema contains `archived_users`.

```go
func TestSchemaDefinesArchivedUsersTable(t *testing.T) {
	content := readSchema(t)
	required := []string{
		"CREATE TABLE archived_users",
		"original_user_id CHAR(26) NOT NULL",
		"user_snapshot JSONB NOT NULL",
		"bindings_snapshot JSONB NOT NULL",
		"roles_snapshot JSONB NOT NULL",
		"UNIQUE (entity_id, original_user_id)",
		"CREATE INDEX idx_archived_users_entity_archived",
		"CREATE INDEX idx_archived_users_entity_username",
	}
	for _, snippet := range required {
		if !strings.Contains(content, snippet) {
			t.Fatalf("schema missing archived users snippet %q", snippet)
		}
	}
}
```

- [ ] **Step 2: Run the schema test and confirm failure**

Run:

```bash
cd backend && go test ./internal/id -run TestSchemaDefinesArchivedUsersTable -count=1
```

Expected: FAIL because `archived_users` does not exist in the baseline schema.

- [ ] **Step 3: Add the Goose migration**

Create `backend/migrations/000005_deleted_user_archive.sql`:

```sql
-- SPDX-License-Identifier: MIT

-- +goose Up
CREATE TABLE IF NOT EXISTS archived_users (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid() CHECK (id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    entity_id CHAR(26) NOT NULL CHECK (entity_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$') REFERENCES business_entities(id) ON DELETE CASCADE,
    original_user_id CHAR(26) NOT NULL CHECK (original_user_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    username TEXT NOT NULL,
    display_name TEXT NOT NULL,
    english_name TEXT NOT NULL DEFAULT '',
    employee_no TEXT NOT NULL DEFAULT '',
    job_title TEXT NOT NULL DEFAULT '',
    email TEXT,
    phone TEXT,
    avatar_url TEXT,
    user_type TEXT NOT NULL,
    primary_source_id CHAR(26) CHECK (primary_source_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    locale TEXT,
    original_created_at TIMESTAMPTZ NOT NULL,
    original_updated_at TIMESTAMPTZ NOT NULL,
    archived_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    archived_by_user_id CHAR(26) CHECK (archived_by_user_id ~ '^[0-9A-HJKMNP-TV-Z]{26}$'),
    archive_reason TEXT NOT NULL DEFAULT '',
    user_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    bindings_snapshot JSONB NOT NULL DEFAULT '[]'::jsonb,
    roles_snapshot JSONB NOT NULL DEFAULT '[]'::jsonb,
    UNIQUE (entity_id, original_user_id)
);

CREATE INDEX IF NOT EXISTS idx_archived_users_entity_archived
    ON archived_users(entity_id, archived_at DESC);

CREATE INDEX IF NOT EXISTS idx_archived_users_entity_username
    ON archived_users(entity_id, username);

WITH deleted_users AS (
    SELECT u.*
    FROM users u
    WHERE u.lifecycle_status = 'deleted'
),
inserted AS (
    INSERT INTO archived_users (
        entity_id,
        original_user_id,
        username,
        display_name,
        english_name,
        employee_no,
        job_title,
        email,
        phone,
        avatar_url,
        user_type,
        primary_source_id,
        locale,
        original_created_at,
        original_updated_at,
        archived_at,
        archive_reason,
        user_snapshot,
        bindings_snapshot,
        roles_snapshot
    )
    SELECT
        u.entity_id,
        u.id,
        u.username,
        u.display_name,
        u.english_name,
        u.employee_no,
        u.job_title,
        u.email,
        u.phone,
        u.avatar_url,
        u.user_type,
        u.primary_source_id,
        u.locale,
        u.created_at,
        u.updated_at,
        now(),
        'migrated deleted lifecycle user',
        to_jsonb(u),
        COALESCE((
            SELECT jsonb_agg(to_jsonb(ab) ORDER BY ab.bound_at)
            FROM account_bindings ab
            WHERE ab.entity_id = u.entity_id AND ab.user_id = u.id
        ), '[]'::jsonb),
        COALESCE((
            SELECT jsonb_agg(to_jsonb(ur) ORDER BY ur.role_id)
            FROM user_roles ur
            WHERE ur.entity_id = u.entity_id AND ur.user_id = u.id
        ), '[]'::jsonb)
    FROM deleted_users u
    ON CONFLICT (entity_id, original_user_id) DO NOTHING
    RETURNING entity_id, original_user_id
)
DELETE FROM users u
USING inserted i
WHERE u.entity_id = i.entity_id
  AND u.id = i.original_user_id
  AND u.lifecycle_status = 'deleted';

-- +goose Down
DROP TABLE IF EXISTS archived_users;
```

- [ ] **Step 4: Update baseline schema**

In `backend/migrations/000001_schema_baseline.sql`, add the same `archived_users` table and indexes immediately after `users` indexes or before `sessions`. Keep `users.lifecycle_status` unchanged in the baseline during this task to avoid a mixed migration with application behavior changes.

- [ ] **Step 5: Add sqlc archive queries**

Create `backend/internal/db/queries/archive.sql`:

```sql
-- SPDX-License-Identifier: MIT

-- name: ArchiveUser :one
INSERT INTO archived_users (
    entity_id,
    original_user_id,
    username,
    display_name,
    english_name,
    employee_no,
    job_title,
    email,
    phone,
    avatar_url,
    user_type,
    primary_source_id,
    locale,
    original_created_at,
    original_updated_at,
    archived_by_user_id,
    archive_reason,
    user_snapshot,
    bindings_snapshot,
    roles_snapshot
)
SELECT
    u.entity_id,
    u.id,
    u.username,
    u.display_name,
    u.english_name,
    u.employee_no,
    u.job_title,
    u.email,
    u.phone,
    u.avatar_url,
    u.user_type,
    u.primary_source_id,
    u.locale,
    u.created_at,
    u.updated_at,
    sqlc.narg('archived_by_user_id'),
    sqlc.arg('archive_reason'),
    to_jsonb(u),
    COALESCE((
        SELECT jsonb_agg(to_jsonb(ab) ORDER BY ab.bound_at)
        FROM account_bindings ab
        WHERE ab.entity_id = u.entity_id AND ab.user_id = u.id
    ), '[]'::jsonb),
    COALESCE((
        SELECT jsonb_agg(to_jsonb(ur) ORDER BY ur.role_id)
        FROM user_roles ur
        WHERE ur.entity_id = u.entity_id AND ur.user_id = u.id
    ), '[]'::jsonb)
FROM users u
WHERE u.entity_id = sqlc.arg('entity_id')
  AND u.id = sqlc.arg('user_id')
RETURNING id, entity_id, original_user_id, username, display_name, english_name, employee_no, job_title, email, phone, avatar_url, user_type, primary_source_id, locale, original_created_at, original_updated_at, archived_at, archived_by_user_id, archive_reason, user_snapshot, bindings_snapshot, roles_snapshot;

-- name: DeleteUserActiveDependents :exec
WITH deleted_application_assignments AS (
    DELETE FROM application_assignments
    WHERE entity_id = sqlc.arg('entity_id')
      AND subject_type = 'user'
      AND subject_id = sqlc.arg('user_id')
    RETURNING 1
)
SELECT count(*) FROM deleted_application_assignments;

-- name: DeleteUserActiveRow :exec
DELETE FROM users
WHERE entity_id = sqlc.arg('entity_id')
  AND id = sqlc.arg('user_id');

-- name: ListArchivedUsers :many
SELECT id, entity_id, original_user_id, username, display_name, english_name, employee_no, job_title, email, phone, avatar_url, user_type, primary_source_id, locale, original_created_at, original_updated_at, archived_at, archived_by_user_id, archive_reason, user_snapshot, bindings_snapshot, roles_snapshot
FROM archived_users
WHERE entity_id = sqlc.arg('entity_id')
  AND (sqlc.narg('username')::text IS NULL OR username ILIKE '%' || sqlc.narg('username')::text || '%')
ORDER BY archived_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountArchivedUsers :one
SELECT count(*)::bigint
FROM archived_users
WHERE entity_id = sqlc.arg('entity_id')
  AND (sqlc.narg('username')::text IS NULL OR username ILIKE '%' || sqlc.narg('username')::text || '%');

-- name: GetArchivedUserByID :one
SELECT id, entity_id, original_user_id, username, display_name, english_name, employee_no, job_title, email, phone, avatar_url, user_type, primary_source_id, locale, original_created_at, original_updated_at, archived_at, archived_by_user_id, archive_reason, user_snapshot, bindings_snapshot, roles_snapshot
FROM archived_users
WHERE entity_id = sqlc.arg('entity_id')
  AND id = sqlc.arg('id');

-- name: GetArchivedUserByOriginalID :one
SELECT id, entity_id, original_user_id, username, display_name, english_name, employee_no, job_title, email, phone, avatar_url, user_type, primary_source_id, locale, original_created_at, original_updated_at, archived_at, archived_by_user_id, archive_reason, user_snapshot, bindings_snapshot, roles_snapshot
FROM archived_users
WHERE entity_id = sqlc.arg('entity_id')
  AND original_user_id = sqlc.arg('original_user_id');
```

- [ ] **Step 6: Generate sqlc code**

Run:

```bash
make generate
```

Expected: generated archive query file appears under `backend/internal/db/generated/`.

- [ ] **Step 7: Run schema and generated package tests**

Run:

```bash
cd backend && go test ./internal/id ./internal/db/generated
```

Expected: PASS.

- [ ] **Step 8: Commit Task 1**

Run:

```bash
git add backend/migrations/000005_deleted_user_archive.sql backend/migrations/000001_schema_baseline.sql backend/internal/db/queries/archive.sql backend/internal/db/generated backend/internal/id/ulid_policy_test.go
git commit -m "feat: add archived user storage"
```

---

### Task 2: Admin Archive Service

**Files:**
- Modify: `backend/internal/adminapi/admin_service.go`
- Modify: `backend/internal/adminapi/user_handlers.go`
- Modify: `backend/internal/adminapi/user_handlers_test.go`
- Add or modify: `backend/internal/adminapi/admin_service_test.go`

**Interfaces:**
- Consumes: `generated.ArchiveUser`, `generated.DeleteUserActiveDependents`, `generated.DeleteUserActiveRow`, `generated.ListArchivedUsers`, `generated.GetArchivedUserByID`.
- Produces: `AdminService.ArchiveUser(ctx, entityID, userID, actorUserID, reason string) (ArchivedUserResponse, error)` and read-only archived-user admin APIs.

- [ ] **Step 1: Add response structs**

In `backend/internal/adminapi/user_handlers.go`, add:

```go
type ArchivedUserResponse struct {
	ID                string          `json:"id"`
	EntityID          string          `json:"entity_id"`
	OriginalUserID    string          `json:"original_user_id"`
	Username          string          `json:"username"`
	DisplayName       string          `json:"display_name"`
	Email             string          `json:"email"`
	Phone             string          `json:"phone"`
	UserType          string          `json:"user_type"`
	ArchivedAt        string          `json:"archived_at"`
	ArchivedByUserID  string          `json:"archived_by_user_id"`
	ArchiveReason     string          `json:"archive_reason"`
	UserSnapshot      json.RawMessage `json:"user_snapshot"`
	BindingsSnapshot  json.RawMessage `json:"bindings_snapshot"`
	RolesSnapshot     json.RawMessage `json:"roles_snapshot"`
}
```

- [ ] **Step 2: Write service test for archiving**

Create or extend `backend/internal/adminapi/admin_service_test.go` with a test that uses a fake query layer or a database-backed integration helper already used in this package. The assertion must verify:

```go
func TestArchiveUserWritesArchiveAndDeletesActiveRow(t *testing.T) {
	// Arrange one active user with username "ada@example.test".
	// Arrange one account binding and one role assignment.
	// Call svc.ArchiveUser(ctx, entityID, userID, adminID, "admin deleted user").
	// Assert archive.OriginalUserID == userID.
	// Assert archive.Username == "ada@example.test".
	// Assert users lookup by userID returns pgx.ErrNoRows.
	// Assert a new active user can be created with username "ada@example.test".
}
```

- [ ] **Step 3: Run the service test and confirm failure**

Run:

```bash
cd backend && go test ./internal/adminapi -run TestArchiveUserWritesArchiveAndDeletesActiveRow -count=1
```

Expected: FAIL because `ArchiveUser` does not exist.

- [ ] **Step 4: Implement archive response mapper**

Add to `backend/internal/adminapi/user_handlers.go`:

```go
func archivedUserFromRow(row generated.ArchivedUser) ArchivedUserResponse {
	return ArchivedUserResponse{
		ID:               ulidString(row.ID),
		EntityID:         ulidString(row.EntityID),
		OriginalUserID:   ulidString(row.OriginalUserID),
		Username:         row.Username,
		DisplayName:      row.DisplayName,
		Email:            textString(row.Email),
		Phone:            textString(row.Phone),
		UserType:         row.UserType,
		ArchivedAt:       row.ArchivedAt.Time.Format(time.RFC3339),
		ArchivedByUserID: textString(row.ArchivedByUserID),
		ArchiveReason:    row.ArchiveReason,
		UserSnapshot:     row.UserSnapshot,
		BindingsSnapshot: row.BindingsSnapshot,
		RolesSnapshot:    row.RolesSnapshot,
	}
}
```

- [ ] **Step 5: Implement `ArchiveUser`**

Add to `backend/internal/adminapi/admin_service.go`:

```go
func (s *AdminService) ArchiveUser(ctx context.Context, entityID, userID, actorUserID, reason string) (ArchivedUserResponse, error) {
	before, err := s.GetUserByID(ctx, entityID, userID)
	if err != nil {
		return ArchivedUserResponse{}, err
	}
	if reason == "" {
		reason = "admin deleted user"
	}
	actor := pgtype.Text{}
	if actorUserID != "" {
		actor = pgtype.Text{String: actorUserID, Valid: true}
	}
	archive, err := s.queries.ArchiveUser(ctx, generated.ArchiveUserParams{
		EntityID:         entityID,
		UserID:           userID,
		ArchivedByUserID: actor,
		ArchiveReason:    reason,
	})
	if err != nil {
		return ArchivedUserResponse{}, err
	}
	if err := s.queries.DeleteUserActiveDependents(ctx, generated.DeleteUserActiveDependentsParams{
		EntityID: entityID,
		UserID:   userID,
	}); err != nil {
		return ArchivedUserResponse{}, err
	}
	if err := s.queries.DeleteUserActiveRow(ctx, generated.DeleteUserActiveRowParams{
		EntityID: entityID,
		UserID:   userID,
	}); err != nil {
		return ArchivedUserResponse{}, err
	}
	resp := archivedUserFromRow(archive)
	if err := s.audit.logAction(ctx, audit.Event{
		EntityID:     entityID,
		ActorUserID:  actorUserID,
		ActorType:    "user",
		Action:       "user.archived",
		ResourceType: "archived_user",
		ResourceID:   resp.ID,
		Before:       before,
		After:        resp,
	}); err != nil {
		return ArchivedUserResponse{}, err
	}
	return resp, nil
}
```

If `AdminService` does not run queries inside a transaction, add a focused transaction wrapper before this method is merged. The final implementation must make archive insert, dependent delete, user delete, and audit event atomic or must defer audit after commit with a guaranteed archive transaction. Prefer atomic archive and delete; audit can be after commit only if audit failure should not rollback deletion.

- [ ] **Step 6: Replace delete lifecycle handler**

In `setLifecycle`, route `status == "deleted"` to `ArchiveUser`:

```go
if status == "deleted" {
	archived, err := h.service.ArchiveUser(r.Context(), entityID, id, session.UserID, "admin deleted user")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "user_archive_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, archived)
	return
}
```

If `setLifecycle` currently lacks session details beyond `entityID`, extend the service interface and handler setup so the actor user id is available. If not available, pass an empty actor id and keep the audit actor type consistent with existing admin operations.

- [ ] **Step 7: Add archived-user read endpoints**

Add routes:

```go
r.Get("/sapi/archived-users", h.listArchivedUsers)
r.Get("/sapi/archived-users/{id}", h.getArchivedUser)
```

Add handler methods:

```go
func (h Handler) listArchivedUsers(w http.ResponseWriter, r *http.Request) {
	entityID, ok := readEntityID(w, r)
	if !ok {
		return
	}
	limit := parseIntQuery(r, "limit", 50)
	offset := parseIntQuery(r, "offset", 0)
	items, total, err := h.service.ListArchivedUsers(r.Context(), entityID, r.URL.Query().Get("username"), int32(limit), int32(offset))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "archived_users_list_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"items": items, "total": total, "limit": limit, "offset": offset})
}
```

Implement `ListArchivedUsers` and `GetArchivedUser` on `AdminService` using generated queries.

- [ ] **Step 8: Run admin tests**

Run:

```bash
cd backend && go test ./internal/adminapi
```

Expected: PASS.

- [ ] **Step 9: Commit Task 2**

Run:

```bash
git add backend/internal/adminapi backend/internal/db/generated
git commit -m "feat: archive users from admin deletion"
```

---

### Task 3: Directory Sync Archive Path

**Files:**
- Modify: `backend/internal/idp/sync.go`
- Modify: `backend/internal/idp/sync_test.go`
- Modify: `backend/internal/db/queries/identity.sql`
- Generated: `backend/internal/db/generated/identity.sql.go`

**Interfaces:**
- Consumes: `ArchiveUser` and delete-active queries from Task 1.
- Produces: sync behavior where missing directory users archive active managed users.

- [ ] **Step 1: Write failing sync test**

In `backend/internal/idp/sync_test.go`, add:

```go
func TestFullSyncArchivesMissingManagedUsers(t *testing.T) {
	// Arrange a full sync where one directory user becomes status "deleted".
	// Arrange an account binding pointing from that directory user to a managed user.
	// Run sync.
	// Assert the managed user no longer exists in users.
	// Assert archived_users has original_user_id equal to the managed user id.
	// Assert stats.ManagedUsersDeleted increments by 1.
}
```

- [ ] **Step 2: Run test and confirm failure**

Run:

```bash
cd backend && go test ./internal/idp -run TestFullSyncArchivesMissingManagedUsers -count=1
```

Expected: FAIL because sync still calls `SoftDeleteManagedUsersByDirectoryStatus`.

- [ ] **Step 3: Replace soft-delete query with archive candidates**

Add a generated query in `backend/internal/db/queries/identity.sql`:

```sql
-- name: ListManagedUsersForDeletedDirectoryUsers :many
SELECT DISTINCT u.id, u.entity_id, u.username, u.display_name, u.english_name, u.employee_no, u.job_title, u.email, u.phone, u.avatar_url, u.lifecycle_status, u.user_type, u.primary_source_id, u.locale, u.created_at, u.updated_at
FROM users u
JOIN account_bindings ab
  ON ab.entity_id = u.entity_id
 AND ab.user_id = u.id
JOIN directory_users du
  ON du.entity_id = ab.entity_id
 AND du.source_id = ab.source_id
 AND du.id = ab.directory_user_id
WHERE u.entity_id = sqlc.arg('entity_id')
  AND ab.source_id = sqlc.arg('source_id')
  AND du.status = 'deleted';
```

Run `make generate`.

- [ ] **Step 4: Implement archive loop**

In `runSync`, replace `SoftDeleteManagedUsersByDirectoryStatus` with:

```go
deletedManagedUsers, err := s.queries.ListManagedUsersForDeletedDirectoryUsers(ctx, generated.ListManagedUsersForDeletedDirectoryUsersParams{
	EntityID: entityID,
	SourceID: sourceID,
})
if err != nil {
	_ = s.finishWebhookJobs(ctx, entityID, webhookJobs, result, err)
	_ = s.failJob(ctx, entityID, job.ID, result, err)
	return result, err
}
for _, deletedUser := range deletedManagedUsers {
	archive, err := s.queries.ArchiveUser(ctx, generated.ArchiveUserParams{
		EntityID:         entityID,
		UserID:           ulidString(deletedUser.ID),
		ArchivedByUserID: pgtype.Text{},
		ArchiveReason:    "directory full sync removed user",
	})
	if err != nil {
		_ = s.finishWebhookJobs(ctx, entityID, webhookJobs, result, err)
		_ = s.failJob(ctx, entityID, job.ID, result, err)
		return result, err
	}
	if err := s.queries.DeleteUserActiveDependents(ctx, generated.DeleteUserActiveDependentsParams{
		EntityID: entityID,
		UserID:   ulidString(deletedUser.ID),
	}); err != nil {
		_ = s.finishWebhookJobs(ctx, entityID, webhookJobs, result, err)
		_ = s.failJob(ctx, entityID, job.ID, result, err)
		return result, err
	}
	if err := s.queries.DeleteUserActiveRow(ctx, generated.DeleteUserActiveRowParams{
		EntityID: entityID,
		UserID:   ulidString(deletedUser.ID),
	}); err != nil {
		_ = s.finishWebhookJobs(ctx, entityID, webhookJobs, result, err)
		_ = s.failJob(ctx, entityID, job.ID, result, err)
		return result, err
	}
	result.ManagedUsersDeleted++
	s.writeAudit(ctx, audit.Event{
		EntityID:     input.EntityID,
		ActorType:    "sync_job",
		Action:       audit.ActionSyncUserDisabled,
		ResourceType: "archived_user",
		ResourceID:   ulidString(archive.ID),
		After:        map[string]string{"username": deletedUser.Username, "archive_reason": "directory full sync removed user"},
		TraceID:      traceID,
	})
}
```

- [ ] **Step 5: Run sync tests**

Run:

```bash
cd backend && go test ./internal/idp
```

Expected: PASS.

- [ ] **Step 6: Commit Task 3**

Run:

```bash
git add backend/internal/idp backend/internal/db/queries/identity.sql backend/internal/db/generated
git commit -m "feat: archive directory-deleted users during sync"
```

---

### Task 4: Admin UI For Archived Users

**Files:**
- Modify: `web/src/lib/api.ts`
- Modify: `web/src/admin-pages.jsx`
- Modify: `web/src/i18n/index.js`
- Modify: `web/src/main.jsx` if navigation items are declared there.

**Interfaces:**
- Consumes: `/sapi/archived-users`, `/sapi/archived-users/{id}`.
- Produces: read-only archived users page with JSON copy.

- [ ] **Step 1: Add API methods**

In `web/src/lib/api.ts`, add types:

```ts
export type ArchivedUser = {
  id: string;
  entity_id: string;
  original_user_id: string;
  username: string;
  display_name: string;
  email: string;
  phone: string;
  user_type: string;
  archived_at: string;
  archived_by_user_id: string;
  archive_reason: string;
  user_snapshot: Record<string, string | number | boolean | null>;
  bindings_snapshot: Array<Record<string, string | number | boolean | null>>;
  roles_snapshot: Array<Record<string, string | number | boolean | null>>;
};

export type ArchivedUserListResponse = {
  items: ArchivedUser[];
  total: number;
  limit: number;
  offset: number;
};
```

Add API methods:

```ts
listArchivedUsers: (params?: { username?: string; limit?: number; offset?: number }) => {
  const suffix = queryString({
    username: params?.username,
    limit: params?.limit,
    offset: params?.offset,
  });
  return apiRequest<ArchivedUserListResponse>(`/sapi/archived-users${suffix}`);
},
getArchivedUser: (id: string) => apiRequest<ArchivedUser>(`/sapi/archived-users/${encodeURIComponent(id)}`),
```

- [ ] **Step 2: Add page component**

In `web/src/admin-pages.jsx`, add:

```jsx
function ArchivedUsersPage() {
  const { t } = useTranslation();
  const { message } = AntApp.useApp();
  const [filters, setFilters] = useState({});
  const [selected, setSelected] = useState(null);
  const { loading, data, reload } = useLoader(() => api.listArchivedUsers({ ...filters, limit: 100 }), [JSON.stringify(filters)]);
  const copySelected = async () => {
    await navigator.clipboard.writeText(JSON.stringify(selected, null, 2));
    message.success(t('common.copied'));
  };
  return (
    <div className="page-stack">
      <div className="toolbar-left">
        <Input placeholder={t('archivedUsers.username')} style={{ width: 220 }} onChange={(e) => setFilters((f) => ({ ...f, username: e.target.value || undefined }))} />
        <Button icon={<RefreshCw size={16} />} onClick={reload}>{t('common.refresh')}</Button>
      </div>
      <Table rowKey="id" loading={loading} dataSource={data?.items || []} columns={[
        { title: t('archivedUsers.username'), dataIndex: 'username' },
        { title: t('archivedUsers.displayName'), dataIndex: 'display_name' },
        { title: t('archivedUsers.email'), dataIndex: 'email' },
        { title: t('archivedUsers.reason'), dataIndex: 'archive_reason', ellipsis: true },
        { title: t('archivedUsers.archivedAt'), dataIndex: 'archived_at', render: formatDate },
        { title: t('common.details'), render: (_, row) => <Button size="small" onClick={() => setSelected(row)}>JSON</Button> },
      ]} />
      <Drawer
        width={720}
        open={Boolean(selected)}
        onClose={() => setSelected(null)}
        title={t('archivedUsers.detailTitle')}
        extra={<Button onClick={copySelected}>{t('common.copy')}</Button>}
      >
        <pre className="json-box">{JSON.stringify(selected, null, 2)}</pre>
      </Drawer>
    </div>
  );
}
```

- [ ] **Step 3: Wire route and navigation**

Add route:

```jsx
if (pathname === '/admin/archived-users') return <ArchivedUsersPage />;
```

Add a navigation item where admin items are declared:

```jsx
{ key: '/admin/archived-users', icon: <Archive size={16} />, label: t('archivedUsers.title'), path: '/admin/archived-users' }
```

Import `Archive` from `lucide-react` if needed.

- [ ] **Step 4: Add translations**

In `web/src/i18n/index.js`, add Chinese:

```js
'archivedUsers.title': '已归档用户',
'archivedUsers.username': '用户名',
'archivedUsers.displayName': '显示名称',
'archivedUsers.email': '邮箱',
'archivedUsers.reason': '归档原因',
'archivedUsers.archivedAt': '归档时间',
'archivedUsers.detailTitle': '归档用户详情',
'common.copy': '复制',
'common.copied': '已复制',
```

Add English:

```js
'archivedUsers.title': 'Archived Users',
'archivedUsers.username': 'Username',
'archivedUsers.displayName': 'Display name',
'archivedUsers.email': 'Email',
'archivedUsers.reason': 'Archive reason',
'archivedUsers.archivedAt': 'Archived at',
'archivedUsers.detailTitle': 'Archived user detail',
'common.copy': 'Copy',
'common.copied': 'Copied',
```

- [ ] **Step 5: Run frontend checks**

Run:

```bash
cd web && npm run build && npm run check:ui
```

Expected: PASS.

- [ ] **Step 6: Commit Task 4**

Run:

```bash
git add web/src/lib/api.ts web/src/admin-pages.jsx web/src/i18n/index.js web/src/main.jsx
git commit -m "feat: add archived users admin view"
```

---

### Task 5: Final Verification And Cleanup

**Files:**
- Review: all files changed by Tasks 1-4.
- Modify only if tests reveal a concrete issue.

**Interfaces:**
- Consumes all tasks.
- Produces a verified branch ready for deployment.

- [ ] **Step 1: Run full backend tests**

Run:

```bash
cd backend && go test ./...
```

Expected: PASS.

- [ ] **Step 2: Run frontend checks**

Run:

```bash
cd web && npm run build && npm run check:ui
```

Expected: PASS.

- [ ] **Step 3: Run whitespace and generated diff checks**

Run:

```bash
git diff --check
git status --short
```

Expected: no whitespace errors; status only shows intentional files if there is a final cleanup commit pending.

- [ ] **Step 4: Manual data-flow verification**

Use local or staging database with one entity:

1. Create user `ada@example.test`.
2. Delete the user through admin API.
3. Confirm `users` no longer has that user id.
4. Confirm `archived_users` has `original_user_id`.
5. Create a new user with username `ada@example.test`.
6. Confirm creation succeeds.
7. Confirm login and sync queries ignore archived rows.

Record SQL probes:

```sql
SELECT id, username FROM users WHERE entity_id = '<entity_id>' AND username = 'ada@example.test';
SELECT original_user_id, username, archive_reason FROM archived_users WHERE entity_id = '<entity_id>' AND username = 'ada@example.test';
```

- [ ] **Step 5: Commit verification cleanup if needed**

If verification required code edits:

```bash
git add <changed-files>
git commit -m "test: verify deleted user archive flow"
```

If no edits were needed, do not create an empty commit.

---

## Self-Review

- Spec coverage: archive table, irreversible deletion, disabled-vs-deleted separation, sync behavior, login behavior, audit retention, and admin read-only access are covered by Tasks 1-5.
- Placeholder scan: no task depends on an unnamed file, unspecified endpoint, or missing command.
- Type consistency: `ArchivedUserResponse`, `ArchiveUser`, `ListArchivedUsers`, and `DeleteUserActiveRow` names are used consistently across backend and frontend tasks.
