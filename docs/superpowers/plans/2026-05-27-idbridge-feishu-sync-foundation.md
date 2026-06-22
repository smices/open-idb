# IdBridge Feishu Sync Foundation Implementation Plan

> Superseded direction: business boundary semantics are now governed by [2026-06-02-business-entity-boundary-replan.md](2026-06-02-business-entity-boundary-replan.md). Future implementation should use business entity terminology and `business_entities` / `entity_id`, not SaaS tenant semantics. No compatibility layer is required.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first Feishu identity source foundation: department/user full sync into directory records, automatic managed-user provisioning, account binding, sync job visibility, and stable admin trigger APIs.

**Architecture:** Keep provider-specific Feishu API details behind `internal/idp/feishu`, and keep provider-neutral sync orchestration in `internal/idp`. The sync service consumes a Feishu client interface so tests can use fakes without calling Feishu. Database writes remain entity-scoped through sqlc queries.

**Tech Stack:** Go 1.22+, Chi, PostgreSQL, sqlc + pgx, Goose, built-in tests, Testcontainers PostgreSQL integration tests.

---

## Scope Boundary

In scope:

- Feishu identity source config model.
- Directory department schema and sqlc queries.
- Sync job schema and sqlc queries.
- Feishu client interface and HTTP implementation for tenant access token, departments, and users.
- Full sync service that upserts departments, directory users, managed users, account bindings, and sync job status.
- Admin endpoint to trigger a full sync for one source.
- Integration tests proving Chinese directory data is preserved and managed users do not receive application access by default.

Out of scope:

- Feishu OAuth login callback.
- Background scheduler.
- Real config encryption.
- Incremental/webhook sync.
- Department-to-managed-organization mapping.
- Audit logs; sync job rows are the visibility surface in this milestone.

## File Structure

Create or modify:

```text
migrations/000003_feishu_sync_core.sql
internal/db/queries/identity.sql
internal/db/queries/sync.sql
internal/db/generated/*
internal/idp/model.go
internal/idp/sync.go
internal/idp/sync_test.go
internal/idp/feishu/client.go
internal/idp/feishu/client_test.go
internal/adminapi/handlers.go
internal/adminapi/handlers_test.go
internal/httpserver/router.go
internal/app/app.go
tests/integration/feishu_sync_test.go
docs/development.md
```

## Task 1: Add Directory Department And Sync Job Schema

**Files:**
- Create: `migrations/000003_feishu_sync_core.sql`

- [ ] **Step 1: Write migration**

Create:

```sql
-- +goose Up
CREATE TABLE directory_departments (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid(),
    entity_id CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    source_id CHAR(26) NOT NULL,
    external_department_id TEXT NOT NULL,
    parent_external_department_id TEXT,
    name TEXT NOT NULL,
    raw_profile JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_synced_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, id),
    UNIQUE (entity_id, source_id, external_department_id),
    FOREIGN KEY (entity_id, source_id) REFERENCES identity_sources(entity_id, id) ON DELETE CASCADE
);

CREATE TABLE sync_jobs (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid(),
    entity_id CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    source_id CHAR(26) NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('full', 'incremental', 'webhook')),
    provider TEXT NOT NULL CHECK (provider IN ('feishu', 'dingtalk', 'wecom', 'ldap', 'local')),
    status TEXT NOT NULL CHECK (status IN ('running', 'succeeded', 'failed')),
    trace_id TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ,
    error_message TEXT,
    stats JSONB NOT NULL DEFAULT '{}'::jsonb,
    FOREIGN KEY (entity_id, source_id) REFERENCES identity_sources(entity_id, id) ON DELETE CASCADE
);

CREATE INDEX idx_directory_departments_tenant_source ON directory_departments(entity_id, source_id);
CREATE INDEX idx_sync_jobs_tenant_source_started ON sync_jobs(entity_id, source_id, started_at DESC);

-- +goose Down
DROP TABLE IF EXISTS sync_jobs;
DROP TABLE IF EXISTS directory_departments;
```

- [ ] **Step 2: Validate**

Run:

```bash
rtk go run github.com/pressly/goose/v3/cmd/goose@v3.22.1 -dir migrations validate
```

Expected: exit 0.

- [ ] **Step 3: Commit**

```bash
rtk git add migrations/000003_feishu_sync_core.sql
rtk git commit -m "feat: add feishu sync schema"
```

## Task 2: Add Sync sqlc Queries

**Files:**
- Modify: `internal/db/queries/identity.sql`
- Create: `internal/db/queries/sync.sql`
- Modify: `internal/db/generated/*`

- [ ] **Step 1: Add identity upsert queries**

Append to `internal/db/queries/identity.sql`:

```sql
-- name: UpsertDirectoryDepartment :one
INSERT INTO directory_departments (
    entity_id,
    source_id,
    external_department_id,
    parent_external_department_id,
    name,
    raw_profile,
    last_synced_at
) VALUES (
    $1, $2, $3, $4, $5, $6, now()
)
ON CONFLICT (entity_id, source_id, external_department_id)
DO UPDATE SET
    parent_external_department_id = EXCLUDED.parent_external_department_id,
    name = EXCLUDED.name,
    raw_profile = EXCLUDED.raw_profile,
    last_synced_at = now(),
    updated_at = now()
RETURNING id, entity_id, source_id, external_department_id, parent_external_department_id, name, raw_profile, last_synced_at, created_at, updated_at;

-- name: GetAccountBindingByProviderUID :one
SELECT id, entity_id, user_id, source_id, directory_user_id, provider_uid, provider_union_id, is_primary, bound_at
FROM account_bindings
WHERE entity_id = $1 AND source_id = $2 AND provider_uid = $3;

-- name: UpdateManagedUserFromDirectory :one
UPDATE users
SET display_name = $4,
    email = $5,
    phone = $6,
    avatar_url = $7,
    lifecycle_status = $8,
    updated_at = now()
WHERE entity_id = $1 AND id = $2 AND primary_source_id = $3
RETURNING id, entity_id, username, display_name, email, phone, avatar_url, lifecycle_status, user_type, primary_source_id, locale, created_at, updated_at;
```

- [ ] **Step 2: Add sync job queries**

Create `internal/db/queries/sync.sql`:

```sql
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
```

- [ ] **Step 3: Generate and test**

Run:

```bash
rtk make generate
rtk env GOCACHE=/private/tmp/open-idb-gocache go test ./internal/db/generated ./...
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
rtk git add internal/db/queries internal/db/generated
rtk git commit -m "feat: add feishu sync queries"
```

## Task 3: Add Feishu Provider Models And Client

**Files:**
- Create: `internal/idp/model.go`
- Create: `internal/idp/feishu/client.go`
- Create: `internal/idp/feishu/client_test.go`

- [ ] **Step 1: Implement provider-neutral models**

Create:

```go
package idp

type DirectoryDepartment struct {
    ExternalDepartmentID       string
    ParentExternalDepartmentID string
    Name                       string
    RawProfile                 []byte
}

type DirectoryUser struct {
    ExternalUserID  string
    ExternalUnionID string
    ExternalOpenID  string
    Name            string
    Email           string
    Phone           string
    AvatarURL       string
    Status          string
    RawProfile      []byte
}

type FullSyncData struct {
    Departments []DirectoryDepartment
    Users       []DirectoryUser
}

type DirectoryProvider interface {
    FullSync(ctx context.Context) (FullSyncData, error)
}
```

- [ ] **Step 2: Implement Feishu HTTP client**

Create `internal/idp/feishu/client.go` with:

```go
type Config struct {
    AppID     string
    AppSecret string
    BaseURL   string
}

func NewClient(cfg Config, httpClient *http.Client) (*Client, error)
func (c *Client) FullSync(ctx context.Context) (idp.FullSyncData, error)
```

Rules:

- Default BaseURL is `https://open.feishu.cn`.
- Fetch tenant access token with `/open-apis/auth/v3/tenant_access_token/internal`.
- Fetch departments with `/open-apis/contact/v3/departments`.
- Fetch users with `/open-apis/contact/v3/users`.
- Preserve raw provider JSON bytes.
- Map Feishu active users to `active`; inactive/departed/disabled users to `disabled`; unknown states to `unknown`.

- [ ] **Step 3: Test with `httptest.Server`**

Tests must cover:

- tenant token is requested with app credentials.
- Chinese department/user names are preserved.
- provider error code returns error.
- missing app credentials rejected.

- [ ] **Step 4: Commit**

```bash
rtk git add internal/idp
rtk git commit -m "feat: add feishu directory client"
```

## Task 4: Add Full Sync Service

**Files:**
- Create: `internal/idp/sync.go`
- Create: `internal/idp/sync_test.go`

- [ ] **Step 1: Implement service**

Create a `SyncService` with:

```go
type SyncService struct {
    queries *generated.Queries
    provider DirectoryProvider
    traceID func() string
}

type FullSyncInput struct {
    TenantID string
    SourceID string
    Provider string
}

type FullSyncResult struct {
    JobID string
    DepartmentsUpserted int
    UsersUpserted int
    ManagedUsersCreated int
    ManagedUsersUpdated int
    BindingsCreated int
}

func (s *SyncService) RunFullSync(ctx context.Context, input FullSyncInput) (FullSyncResult, error)
```

Rules:

- Create `sync_jobs` row with status `running`.
- Upsert every department.
- Upsert every directory user.
- If account binding exists, update the managed user from directory data.
- If no account binding exists, create managed user and primary account binding.
- Default lifecycle: active for active provider users, disabled for disabled/deleted, locked for unknown.
- Default application access remains none; do not write application assignments.
- Finish job with JSON stats on success.
- Mark job failed with error message and partial stats on provider/write error.

- [ ] **Step 2: Unit tests**

Tests must cover:

- Chinese names and raw JSON are passed to generated query params through a fake store or real in-memory-free query abstraction.
- provider error creates/marks a failed job.
- existing account binding updates managed user instead of creating a new one.

- [ ] **Step 3: Commit**

```bash
rtk git add internal/idp/sync.go internal/idp/sync_test.go
rtk git commit -m "feat: add identity full sync service"
```

## Task 5: Add Admin Sync Trigger Endpoint

**Files:**
- Create: `internal/adminapi/handlers.go`
- Create: `internal/adminapi/handlers_test.go`
- Modify: `internal/httpserver/router.go`
- Modify: `internal/app/app.go`

- [ ] **Step 1: Add route**

Implement:

```text
POST /admin/v1/identity-sources/{source_id}/sync/full
```

Request headers:

```text
X-IDB-Tenant-ID
```

Response:

```json
{
  "job_id": "...",
  "departments_upserted": 1,
  "users_upserted": 2,
  "managed_users_created": 2,
  "managed_users_updated": 0,
  "bindings_created": 2
}
```

- [ ] **Step 2: Tests**

Tests must cover:

- missing tenant header returns `401` with stable JSON error.
- sync service error returns `500` with stable JSON error.
- successful trigger returns the sync result JSON.

- [ ] **Step 3: Commit**

```bash
rtk git add internal/adminapi internal/httpserver internal/app
rtk git commit -m "feat: add admin feishu sync trigger"
```

## Task 6: Add Integration Test

**Files:**
- Create: `tests/integration/feishu_sync_test.go`

- [ ] **Step 1: Write integration test**

Test must:

1. Start Postgres with Testcontainers.
2. Apply migrations.
3. Create business entity and Feishu identity source.
4. Run `SyncService` with a fake provider returning Chinese department/user data.
5. Assert `directory_departments`, `directory_users`, `users`, `account_bindings`, and `sync_jobs` rows exist.
6. Assert no application assignment rows are created.
7. Run sync again with changed name/status and assert update path works.

- [ ] **Step 2: Verify**

Run:

```bash
rtk env GOCACHE=/private/tmp/open-idb-gocache go test ./tests/integration -v -count=1
rtk env GOCACHE=/private/tmp/open-idb-gocache go test ./... -count=1
rtk env GOCACHE=/private/tmp/open-idb-gocache go vet ./...
rtk go mod tidy -diff
```

Expected: PASS / no diff.

- [ ] **Step 3: Commit**

```bash
rtk git add tests/integration/feishu_sync_test.go
rtk git commit -m "test: cover feishu full sync"
```

## Self-Review

Spec coverage:

- Feishu full sync is covered by Tasks 1-6.
- Automatic managed user creation and account binding are covered by Task 4 and Task 6.
- Chinese synced data preservation is covered by Task 3 and Task 6.
- Sync job visibility is covered by Tasks 1, 2, 4, and 6.
- Application access remains separate from managed-user existence and is asserted in Task 6.

Intentional gaps:

- Feishu OAuth login is not included; it should be the next plan after sync foundation.
- Audit logs are not implemented here; sync jobs provide milestone visibility.
- Real secret encryption is not included; config will remain out of database until encryption is designed.
