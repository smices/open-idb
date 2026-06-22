# IdBridge Foundation Implementation Plan

> Superseded direction: business boundary semantics are now governed by [2026-06-02-business-entity-boundary-replan.md](2026-06-02-business-entity-boundary-replan.md). Future implementation should use business entity terminology and `business_entities` / `entity_id`, not SaaS tenant semantics. No compatibility layer is required.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first executable IdBridge backend foundation: Go service scaffold, configuration, i18n baseline, PostgreSQL schema migrations, sqlc queries, core identity provisioning rules, health endpoints, and focused tests.

**Architecture:** Start with a Go modular monolith. The first milestone intentionally avoids OIDC, Feishu API calls, Casbin, Redis sessions, and the admin frontend; it creates the stable backend foundation those later plans will build on. Domain code lives under focused `internal/*` packages, SQL is explicit via sqlc + pgx, and tests exercise identity rules before HTTP integration expands.

**Tech Stack:** Go 1.22+, Chi, pgx, sqlc, Goose, Zap, Testcontainers for PostgreSQL integration tests, built-in Go `testing`, `go-cmp`.

---

## Scope Boundary

The full IdBridge design contains several independent subsystems. Implement them as separate milestones:

1. Foundation and identity core: this plan.
2. OIDC SSO provider with Fosite.
3. Feishu OAuth login and full directory sync.
4. RBAC, application assignments, and resource scopes.
5. Audit log ingestion and admin API expansion.
6. React + Ant Design Pro admin UI.

This plan must leave extension points for later milestones but must not implement them prematurely.

## File Structure

Create this structure:

```text
cmd/idbridge/main.go
internal/app/app.go
internal/config/config.go
internal/httpserver/router.go
internal/httpserver/health.go
internal/i18n/catalog.go
internal/i18n/catalog_test.go
internal/platform/postgres/pool.go
internal/tenant/model.go
internal/identity/model.go
internal/identity/provisioning.go
internal/identity/provisioning_test.go
internal/identity/store.go
internal/identity/store_test.go
internal/db/sqlc.yaml
internal/db/queries/identity.sql
internal/db/generated/.gitkeep
migrations/000001_identity_core.sql
tests/integration/postgres_test.go
docs/development.md
go.mod
go.sum
Makefile
```

Responsibilities:

- `cmd/idbridge/main.go`: process entrypoint only.
- `internal/app/app.go`: dependency wiring and graceful startup surface.
- `internal/config/config.go`: environment parsing and validation.
- `internal/httpserver/*`: Chi router and process health endpoints.
- `internal/i18n/*`: built-in `en-US` and `zh-CN` message catalog.
- `internal/platform/postgres/*`: pgx pool creation and ping.
- `internal/tenant/model.go`: tenant domain constants and IDs.
- `internal/identity/model.go`: managed users, directory users, identity sources, bindings.
- `internal/identity/provisioning.go`: pure provisioning rules from directory users to managed users.
- `internal/identity/store.go`: database-backed identity repository.
- `internal/db/queries/identity.sql`: sqlc queries for identity core.
- `migrations/000001_identity_core.sql`: database schema for the first milestone.
- `tests/integration/postgres_test.go`: PostgreSQL migration and repository smoke tests.
- `docs/development.md`: local setup and test commands.

## Task 1: Initialize Go Module And Tooling

**Files:**
- Create: `go.mod`
- Create: `Makefile`
- Create: `docs/development.md`

- [ ] **Step 1: Create `go.mod`**

Use module path `github.com/smices/open-idb`.

```go
module github.com/smices/open-idb

go 1.22
```

- [ ] **Step 2: Add baseline dependencies**

Run:

```bash
go get github.com/go-chi/chi/v5@latest
go get github.com/jackc/pgx/v5/pgxpool@latest
go get go.uber.org/zap@latest
go get github.com/oklog/ulid/v2@latest
go get github.com/google/go-cmp/cmp@latest
go get github.com/testcontainers/testcontainers-go@latest
go get github.com/testcontainers/testcontainers-go/modules/postgres@latest
```

Expected: `go.mod` and `go.sum` contain Chi, pgx, Zap, ulid, go-cmp, and Testcontainers dependencies.

- [ ] **Step 3: Create `Makefile`**

```makefile
.PHONY: test lint generate migrate-up run

test:
	go test ./...

lint:
	go test ./...

generate:
	sqlc generate -f internal/db/sqlc.yaml

migrate-up:
	goose -dir migrations postgres "$$DATABASE_URL" up

run:
	go run ./cmd/idbridge
```

- [ ] **Step 4: Create `docs/development.md`**

```markdown
# Development

## Requirements

- Go 1.22 or newer
- PostgreSQL 15 or newer for local manual testing
- Docker for integration tests that use Testcontainers
- `sqlc` for query generation
- `goose` for database migrations

## Common Commands

```bash
make test
make generate
DATABASE_URL=postgres://postgres:postgres@localhost:5432/idbridge?sslmode=disable make migrate-up
make run
```

## Locale Baseline

The default locale is `en-US`. The first version ships both `en-US` and `zh-CN` messages.
```

- [ ] **Step 5: Verify**

Run:

```bash
go test ./...
```

Expected: PASS or `go test` reports no packages before source files are added.

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum Makefile docs/development.md
git commit -m "chore: initialize idbridge go module"
```

## Task 2: Add Configuration And Health Server

**Files:**
- Create: `cmd/idbridge/main.go`
- Create: `internal/app/app.go`
- Create: `internal/config/config.go`
- Create: `internal/httpserver/router.go`
- Create: `internal/httpserver/health.go`

- [ ] **Step 1: Write `internal/config/config.go`**

```go
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr        string
	DatabaseURL     string
	DefaultLocale   string
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:        getEnv("IDB_HTTP_ADDR", ":8080"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		DefaultLocale:   getEnv("IDB_DEFAULT_LOCALE", "en-US"),
		ShutdownTimeout: getDurationSeconds("IDB_SHUTDOWN_TIMEOUT_SECONDS", 10*time.Second),
	}

	if cfg.DefaultLocale != "en-US" && cfg.DefaultLocale != "zh-CN" {
		return Config{}, fmt.Errorf("IDB_DEFAULT_LOCALE must be en-US or zh-CN")
	}

	return cfg, nil
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getDurationSeconds(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}
```

- [ ] **Step 2: Write `internal/httpserver/health.go`**

```go
package httpserver

import (
	"encoding/json"
	"net/http"
)

type HealthResponse struct {
	Status string `json:"status"`
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(HealthResponse{Status: "ok"})
}
```

- [ ] **Step 3: Write `internal/httpserver/router.go`**

```go
package httpserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewRouter() http.Handler {
	r := chi.NewRouter()
	r.Get("/healthz", HealthHandler)
	r.Get("/readyz", HealthHandler)
	return r
}
```

- [ ] **Step 4: Write `internal/app/app.go`**

```go
package app

import (
	"context"
	"net/http"
	"time"

	"github.com/smices/open-idb/internal/config"
	"github.com/smices/open-idb/internal/httpserver"
	"go.uber.org/zap"
)

type App struct {
	cfg    config.Config
	logger *zap.Logger
	server *http.Server
}

func New(cfg config.Config, logger *zap.Logger) *App {
	return &App{
		cfg:    cfg,
		logger: logger,
		server: &http.Server{
			Addr:              cfg.HTTPAddr,
			Handler:           httpserver.NewRouter(),
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		a.logger.Info("starting http server", zap.String("addr", a.cfg.HTTPAddr))
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.ShutdownTimeout)
		defer cancel()
		return a.server.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
```

- [ ] **Step 5: Write `cmd/idbridge/main.go`**

```go
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/smices/open-idb/internal/app"
	"github.com/smices/open-idb/internal/config"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer func() { _ = logger.Sync() }()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("load config", zap.Error(err))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.New(cfg, logger).Run(ctx); err != nil {
		logger.Fatal("run app", zap.Error(err))
	}
}
```

- [ ] **Step 6: Verify health server**

Run:

```bash
go test ./...
```

Expected: PASS.

Run:

```bash
go run ./cmd/idbridge
```

Expected: server logs `starting http server` on `:8080`.

In another shell:

```bash
curl -s http://localhost:8080/healthz
```

Expected:

```json
{"status":"ok"}
```

- [ ] **Step 7: Commit**

```bash
git add cmd internal go.mod go.sum
git commit -m "feat: add idbridge health server"
```

## Task 3: Add I18n Catalog

**Files:**
- Create: `internal/i18n/catalog.go`
- Create: `internal/i18n/catalog_test.go`

- [ ] **Step 1: Write failing tests in `internal/i18n/catalog_test.go`**

```go
package i18n

import "testing"

func TestCatalogReturnsEnglishByDefault(t *testing.T) {
	catalog := NewCatalog()

	got := catalog.Message("en-US", "health.ok")
	if got != "OK" {
		t.Fatalf("expected OK, got %q", got)
	}
}

func TestCatalogReturnsChineseMessage(t *testing.T) {
	catalog := NewCatalog()

	got := catalog.Message("zh-CN", "health.ok")
	if got != "正常" {
		t.Fatalf("expected Chinese message, got %q", got)
	}
}

func TestCatalogFallsBackToEnglish(t *testing.T) {
	catalog := NewCatalog()

	got := catalog.Message("fr-FR", "health.ok")
	if got != "OK" {
		t.Fatalf("expected English fallback, got %q", got)
	}
}

func TestCatalogReturnsCodeForMissingMessage(t *testing.T) {
	catalog := NewCatalog()

	got := catalog.Message("zh-CN", "missing.code")
	if got != "missing.code" {
		t.Fatalf("expected missing code fallback, got %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify failure**

Run:

```bash
go test ./internal/i18n -run TestCatalog -v
```

Expected: FAIL because `NewCatalog` is undefined.

- [ ] **Step 3: Implement `internal/i18n/catalog.go`**

```go
package i18n

const (
	LocaleEnglishUS = "en-US"
	LocaleChineseCN = "zh-CN"
)

type Catalog struct {
	messages map[string]map[string]string
}

func NewCatalog() Catalog {
	return Catalog{
		messages: map[string]map[string]string{
			LocaleEnglishUS: {
				"health.ok": "OK",
			},
			LocaleChineseCN: {
				"health.ok": "正常",
			},
		},
	}
}

func (c Catalog) Message(locale string, code string) string {
	if messages, ok := c.messages[locale]; ok {
		if message, ok := messages[code]; ok {
			return message
		}
	}

	if messages, ok := c.messages[LocaleEnglishUS]; ok {
		if message, ok := messages[code]; ok {
			return message
		}
	}

	return code
}
```

- [ ] **Step 4: Run tests**

Run:

```bash
go test ./internal/i18n -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/i18n
git commit -m "feat: add built-in locale catalog"
```

## Task 4: Add Identity Domain Models And Provisioning Rules

**Files:**
- Create: `internal/tenant/model.go`
- Create: `internal/identity/model.go`
- Create: `internal/identity/provisioning.go`
- Create: `internal/identity/provisioning_test.go`

- [ ] **Step 1: Write `internal/tenant/model.go`**

```go
package tenant

type ID string

type Status string

const (
	StatusActive   Status = "active"
	StatusDisabled Status = "disabled"
)
```

- [ ] **Step 2: Write failing tests in `internal/identity/provisioning_test.go`**

```go
package identity

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/smices/open-idb/internal/tenant"
)

func TestProvisionManagedUserFromDirectoryUser(t *testing.T) {
	sourceID := SourceID("src_feishu")
	dir := DirectoryUser{
		ID:              DirectoryUserID("dir_1"),
		TenantID:        tenant.ID("tenant_1"),
		SourceID:        sourceID,
		ExternalUserID:  "ou_1",
		ExternalUnionID: "union_1",
		Name:            "张三",
		Email:           "zhangsan@example.com",
		Phone:           "13800000000",
		AvatarURL:       "https://example.com/avatar.png",
		Status:          DirectoryUserStatusActive,
	}

	got := ProvisionManagedUser(dir, ProvisionPolicy{
		AutoCreateManagedUsers: true,
		DefaultLifecycleStatus: UserLifecycleActive,
		DefaultLocale:          "zh-CN",
	})

	want := ManagedUserDraft{
		TenantID:        tenant.ID("tenant_1"),
		Username:        "zhangsan@example.com",
		DisplayName:     "张三",
		Email:           "zhangsan@example.com",
		Phone:           "13800000000",
		AvatarURL:       "https://example.com/avatar.png",
		LifecycleStatus: UserLifecycleActive,
		UserType:        UserTypeEmployee,
		PrimarySourceID: &sourceID,
		Locale:          "zh-CN",
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("draft mismatch (-want +got):\n%s", diff)
	}
}

func TestProvisionManagedUserUsesExternalIDWhenEmailMissing(t *testing.T) {
	dir := DirectoryUser{
		TenantID:       tenant.ID("tenant_1"),
		SourceID:       SourceID("src_feishu"),
		ExternalUserID: "ou_1",
		Name:           "No Email",
		Status:         DirectoryUserStatusActive,
	}

	got := ProvisionManagedUser(dir, ProvisionPolicy{
		AutoCreateManagedUsers: true,
		DefaultLifecycleStatus: UserLifecycleActive,
		DefaultLocale:          "en-US",
	})

	if got.Username != "ou_1" {
		t.Fatalf("expected external user id username, got %q", got.Username)
	}
}

func TestProvisionManagedUserReturnsZeroWhenAutoCreateDisabled(t *testing.T) {
	dir := DirectoryUser{
		TenantID:       tenant.ID("tenant_1"),
		SourceID:       SourceID("src_feishu"),
		ExternalUserID: "ou_1",
		Status:         DirectoryUserStatusActive,
	}

	got := ProvisionManagedUser(dir, ProvisionPolicy{
		AutoCreateManagedUsers: false,
		DefaultLifecycleStatus: UserLifecycleActive,
		DefaultLocale:          "en-US",
	})

	if got.Username != "" {
		t.Fatalf("expected zero draft, got %#v", got)
	}
}
```

- [ ] **Step 3: Run test to verify failure**

Run:

```bash
go test ./internal/identity -run TestProvisionManagedUser -v
```

Expected: FAIL because identity types are undefined.

- [ ] **Step 4: Implement `internal/identity/model.go`**

```go
package identity

import "github.com/smices/open-idb/internal/tenant"

type SourceID string
type DirectoryUserID string
type UserID string

type SourceType string

const (
	SourceTypeFeishu   SourceType = "feishu"
	SourceTypeDingTalk SourceType = "dingtalk"
	SourceTypeWeCom    SourceType = "wecom"
	SourceTypeLDAP     SourceType = "ldap"
	SourceTypeLocal    SourceType = "local"
)

type DirectoryUserStatus string

const (
	DirectoryUserStatusActive   DirectoryUserStatus = "active"
	DirectoryUserStatusDisabled DirectoryUserStatus = "disabled"
	DirectoryUserStatusDeleted  DirectoryUserStatus = "deleted"
	DirectoryUserStatusUnknown  DirectoryUserStatus = "unknown"
)

type UserLifecycleStatus string

const (
	UserLifecycleActive   UserLifecycleStatus = "active"
	UserLifecycleDisabled UserLifecycleStatus = "disabled"
	UserLifecycleLocked   UserLifecycleStatus = "locked"
	UserLifecycleDeleted  UserLifecycleStatus = "deleted"
)

type UserType string

const (
	UserTypeEmployee       UserType = "employee"
	UserTypeContractor     UserType = "contractor"
	UserTypeServiceAccount UserType = "service_account"
)

type DirectoryUser struct {
	ID              DirectoryUserID
	TenantID        tenant.ID
	SourceID        SourceID
	ExternalUserID  string
	ExternalUnionID string
	ExternalOpenID  string
	Name            string
	Email           string
	Phone           string
	AvatarURL       string
	Status          DirectoryUserStatus
}

type ManagedUserDraft struct {
	TenantID        tenant.ID
	Username        string
	DisplayName     string
	Email           string
	Phone           string
	AvatarURL       string
	LifecycleStatus UserLifecycleStatus
	UserType        UserType
	PrimarySourceID *SourceID
	Locale          string
}

type ProvisionPolicy struct {
	AutoCreateManagedUsers bool
	DefaultLifecycleStatus UserLifecycleStatus
	DefaultLocale          string
}
```

- [ ] **Step 5: Implement `internal/identity/provisioning.go`**

```go
package identity

func ProvisionManagedUser(dir DirectoryUser, policy ProvisionPolicy) ManagedUserDraft {
	if !policy.AutoCreateManagedUsers {
		return ManagedUserDraft{}
	}

	username := dir.Email
	if username == "" {
		username = dir.ExternalUserID
	}

	sourceID := dir.SourceID

	return ManagedUserDraft{
		TenantID:        dir.TenantID,
		Username:        username,
		DisplayName:     dir.Name,
		Email:           dir.Email,
		Phone:           dir.Phone,
		AvatarURL:       dir.AvatarURL,
		LifecycleStatus: policy.DefaultLifecycleStatus,
		UserType:        UserTypeEmployee,
		PrimarySourceID: &sourceID,
		Locale:          policy.DefaultLocale,
	}
}
```

- [ ] **Step 6: Run tests**

Run:

```bash
go test ./internal/identity -v
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/tenant internal/identity
git commit -m "feat: add identity provisioning rules"
```

## Task 5: Add PostgreSQL Schema Migration

**Files:**
- Create: `migrations/000001_identity_core.sql`

- [ ] **Step 1: Create migration**

```sql
-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE tenants (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid(),
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled')),
    default_locale TEXT NOT NULL DEFAULT 'en-US' CHECK (default_locale IN ('en-US', 'zh-CN')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE identity_sources (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid(),
    entity_id CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK (type IN ('feishu', 'dingtalk', 'wecom', 'ldap', 'local')),
    name TEXT NOT NULL,
    config_encrypted BYTEA,
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled')),
    sync_enabled BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, type, name)
);

CREATE TABLE directory_users (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid(),
    entity_id CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    source_id CHAR(26) NOT NULL REFERENCES identity_sources(id) ON DELETE CASCADE,
    external_user_id TEXT NOT NULL,
    external_union_id TEXT,
    external_open_id TEXT,
    name TEXT NOT NULL,
    email TEXT,
    phone TEXT,
    avatar_url TEXT,
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled', 'deleted', 'unknown')),
    raw_profile JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_synced_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, source_id, external_user_id)
);

CREATE TABLE users (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid(),
    entity_id CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    username TEXT NOT NULL,
    display_name TEXT NOT NULL,
    email TEXT,
    phone TEXT,
    avatar_url TEXT,
    lifecycle_status TEXT NOT NULL CHECK (lifecycle_status IN ('active', 'disabled', 'locked', 'deleted')),
    user_type TEXT NOT NULL CHECK (user_type IN ('employee', 'contractor', 'service_account')),
    primary_source_id CHAR(26) REFERENCES identity_sources(id) ON DELETE SET NULL,
    locale TEXT CHECK (locale IN ('en-US', 'zh-CN')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, username)
);

CREATE TABLE account_bindings (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid(),
    entity_id CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id CHAR(26) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source_id CHAR(26) NOT NULL REFERENCES identity_sources(id) ON DELETE CASCADE,
    directory_user_id CHAR(26) NOT NULL REFERENCES directory_users(id) ON DELETE CASCADE,
    provider_uid TEXT NOT NULL,
    provider_union_id TEXT,
    is_primary BOOLEAN NOT NULL DEFAULT false,
    bound_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, source_id, provider_uid),
    UNIQUE (entity_id, directory_user_id)
);

CREATE INDEX idx_directory_users_tenant_source ON directory_users(entity_id, source_id);
CREATE INDEX idx_users_tenant_status ON users(entity_id, lifecycle_status);
CREATE INDEX idx_account_bindings_user ON account_bindings(entity_id, user_id);

-- +goose Down
DROP TABLE IF EXISTS account_bindings;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS directory_users;
DROP TABLE IF EXISTS identity_sources;
DROP TABLE IF EXISTS tenants;
DROP EXTENSION IF EXISTS pgcrypto;
```

- [ ] **Step 2: Verify migration manually against local PostgreSQL**

Run:

```bash
DATABASE_URL=postgres://postgres:postgres@localhost:5432/idbridge?sslmode=disable make migrate-up
```

Expected: Goose applies `000001_identity_core.sql`.

- [ ] **Step 3: Commit**

```bash
git add migrations/000001_identity_core.sql
git commit -m "feat: add identity core migration"
```

## Task 6: Add sqlc Queries And Identity Store

**Files:**
- Create: `internal/db/sqlc.yaml`
- Create: `internal/db/queries/identity.sql`
- Create: `internal/db/generated/.gitkeep`
- Create: `internal/platform/postgres/pool.go`
- Create: `internal/identity/store.go`
- Create: `internal/identity/store_test.go`

- [ ] **Step 1: Create `internal/db/sqlc.yaml`**

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "queries"
    schema: "../../migrations"
    gen:
      go:
        package: "generated"
        out: "generated"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_empty_slices: true
```

- [ ] **Step 2: Create `internal/db/queries/identity.sql`**

```sql
-- name: CreateTenant :one
INSERT INTO tenants (name, slug, status, default_locale)
VALUES ($1, $2, 'active', $3)
RETURNING id, name, slug, status, default_locale, created_at;

-- name: CreateIdentitySource :one
INSERT INTO identity_sources (entity_id, type, name, status, sync_enabled)
VALUES ($1, $2, $3, 'active', $4)
RETURNING id, entity_id, type, name, status, sync_enabled, created_at;

-- name: UpsertDirectoryUser :one
INSERT INTO directory_users (
    entity_id,
    source_id,
    external_user_id,
    external_union_id,
    external_open_id,
    name,
    email,
    phone,
    avatar_url,
    status,
    raw_profile,
    last_synced_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, now()
)
ON CONFLICT (entity_id, source_id, external_user_id)
DO UPDATE SET
    external_union_id = EXCLUDED.external_union_id,
    external_open_id = EXCLUDED.external_open_id,
    name = EXCLUDED.name,
    email = EXCLUDED.email,
    phone = EXCLUDED.phone,
    avatar_url = EXCLUDED.avatar_url,
    status = EXCLUDED.status,
    raw_profile = EXCLUDED.raw_profile,
    last_synced_at = now(),
    updated_at = now()
RETURNING id, entity_id, source_id, external_user_id, external_union_id, external_open_id, name, email, phone, avatar_url, status, raw_profile, last_synced_at, created_at, updated_at;

-- name: CreateManagedUser :one
INSERT INTO users (
    entity_id,
    username,
    display_name,
    email,
    phone,
    avatar_url,
    lifecycle_status,
    user_type,
    primary_source_id,
    locale
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING id, entity_id, username, display_name, email, phone, avatar_url, lifecycle_status, user_type, primary_source_id, locale, created_at, updated_at;

-- name: CreateAccountBinding :one
INSERT INTO account_bindings (
    entity_id,
    user_id,
    source_id,
    directory_user_id,
    provider_uid,
    provider_union_id,
    is_primary
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING id, entity_id, user_id, source_id, directory_user_id, provider_uid, provider_union_id, is_primary, bound_at;
```

- [ ] **Step 3: Generate sqlc code**

Run:

```bash
make generate
```

Expected: `internal/db/generated` contains generated Go files.

- [ ] **Step 4: Create `internal/platform/postgres/pool.go`**

```go
package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}
```

- [ ] **Step 5: Create `internal/identity/store.go`**

```go
package identity

import "github.com/smices/open-idb/internal/db/generated"

type Store struct {
	q *generated.Queries
}

func NewStore(q *generated.Queries) Store {
	return Store{q: q}
}
```

- [ ] **Step 6: Write repository tests after generated types exist**

Create `internal/identity/store_test.go` with a focused compile-time test:

```go
package identity

import (
	"testing"

	"github.com/smices/open-idb/internal/db/generated"
)

func TestNewStore(t *testing.T) {
	store := NewStore(&generated.Queries{})
	if store.q == nil {
		t.Fatal("expected queries")
	}
}
```

- [ ] **Step 7: Run tests**

Run:

```bash
go test ./internal/identity ./internal/platform/postgres
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/db internal/platform/postgres internal/identity go.mod go.sum
git commit -m "feat: add identity sqlc store"
```

## Task 7: Add PostgreSQL Integration Test Harness

**Files:**
- Create: `tests/integration/postgres_test.go`

- [ ] **Step 1: Write `tests/integration/postgres_test.go`**

```go
package integration

import (
	"context"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestPostgresContainerStarts(t *testing.T) {
	ctx := context.Background()

	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("idbridge"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("5432/tcp")),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Fatalf("terminate postgres container: %v", err)
		}
	})

	conn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	if conn == "" {
		t.Fatal("expected connection string")
	}
}
```

- [ ] **Step 2: Run integration test**

Run:

```bash
go test ./tests/integration -v
```

Expected: PASS if Docker is available. If Docker is unavailable, record that integration tests require Docker and keep unit tests passing.

- [ ] **Step 3: Commit**

```bash
git add tests/integration go.mod go.sum
git commit -m "test: add postgres integration harness"
```

## Task 8: Final Verification For Milestone 1

**Files:**
- Modify only if verification finds compile or test failures.

- [ ] **Step 1: Format all Go files**

Run:

```bash
gofmt -w cmd internal tests
```

Expected: no output.

- [ ] **Step 2: Run all unit tests**

Run:

```bash
go test ./...
```

Expected: PASS, except integration tests may require Docker.

- [ ] **Step 3: Run the service**

Run:

```bash
go run ./cmd/idbridge
```

Expected: server starts on `:8080`.

In another shell:

```bash
curl -s http://localhost:8080/readyz
```

Expected:

```json
{"status":"ok"}
```

- [ ] **Step 4: Review git state**

Run:

```bash
git status --short
git log --oneline -5
```

Expected: only intentional files are modified or untracked. Recent commits should correspond to the tasks above.

- [ ] **Step 5: Commit final fixes if any**

```bash
git add .
git commit -m "chore: verify idbridge foundation"
```

Skip this commit if there are no final fixes.

## Self-Review

Spec coverage in this milestone:

- Go modular monolith foundation: covered by Tasks 1 and 2.
- i18n baseline with `en-US` default and `zh-CN` built in: covered by Task 3.
- Managed user, directory user, identity source, and account binding model: covered by Tasks 4, 5, and 6.
- PostgreSQL, sqlc, and pgx baseline: covered by Tasks 5, 6, and 7.
- Feishu full sync, OIDC, RBAC, audit, Redis session management, and admin frontend: intentionally deferred to later milestone plans.

Ambiguity scan:

- This plan avoids open-ended instructions for the core tasks.
- The first milestone intentionally keeps the identity store as a generated-query wrapper. Repository methods that map pgx CHAR(26) types to domain IDs belong in the next implementation plan after generated code is present.

Type consistency:

- Domain types use `tenant.ID`, `identity.SourceID`, `identity.DirectoryUser`, and `identity.ManagedUserDraft`.
- Status strings match the design spec and migration check constraints.
- Locale strings use `en-US` and `zh-CN`.
