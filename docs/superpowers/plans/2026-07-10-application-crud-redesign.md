# Application CRUD Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver type-aware application CRUD, complete JSON view/copy, and recoverable encrypted OIDC secrets.

**Architecture:** Persist common metadata and non-secret type configuration on `applications`; retain OIDC protocol configuration in `oidc_clients`. Use AES-256-GCM encrypted secret copies for administrator detail responses while retaining hashes for authentication.

**Tech Stack:** PostgreSQL, Goose, sqlc, Go, React, Ant Design, Vite.

## Global Constraints

- Types are `oidc_client`, `api_client`, and `internal_app`.
- Never log plaintext secrets; list/edit responses never include them.
- Detail JSON includes a decrypted secret only for authenticated administrators and only when the encrypted copy exists.
- Existing hash-only clients report `client_secret_available: false`.

---

### Task 1: Add Persistent Application and Encrypted-Secret Fields

**Files:** `backend/migrations/0000xx_application_config.sql`, `backend/internal/db/queries/*.sql`, generated sqlc output.

- [ ] Write migration tests that apply the migration and assert `applications.description`, `applications.config`, and `oidc_clients.client_secret_encrypted` defaults.
- [ ] Add the Goose migration:

```sql
ALTER TABLE applications ADD COLUMN description TEXT NOT NULL DEFAULT '';
ALTER TABLE applications ADD COLUMN config JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE oidc_clients ADD COLUMN client_secret_encrypted BYTEA;
```

- [ ] Extend create/get/update queries to round-trip description, config, and encrypted secret; regenerate with `make generate`.
- [ ] Run `go test ./internal/db/generated ./... -count=1` and commit `feat: persist application configuration`.

### Task 2: Encrypt and Recover OIDC Client Secrets

**Files:** `backend/internal/config/config.go`, `backend/internal/adminapi/oidc_secret.go`, `backend/internal/adminapi/oidc_secret_test.go`, OIDC service/query code.

- [ ] Write failing tests for a base64 32-byte key, AES-GCM round trip, malformed key rejection, random nonce, and legacy nil ciphertext.
- [ ] Implement `encryptOIDCSecret(key, plaintext)` and `decryptOIDCSecret(key, blob)` using `crypto/aes`, `cipher.NewGCM`, and `nonce || ciphertext` storage.
- [ ] Add required `IDB_OIDC_SECRET_ENCRYPTION_KEY` config validation; create/rotate writes hash and encrypted copy together.
- [ ] Run focused tests and `go test ./... -count=1`; commit `feat: encrypt recoverable oidc secrets`.

### Task 3: Add Atomic Typed Application CRUD API

**Files:** `backend/internal/adminapi/application_handlers.go`, `admin_service.go`, tests, generated transaction/query helpers.

- [ ] Write failing handler/service tests for typed create/update/detail, tenant isolation, invalid configs, legacy details, and secret omission from list/edit responses.
- [ ] Add request/detail contracts: common `name,type,status,description,config`; OIDC includes client fields; API config is `client_id,audience,allowed_scopes`; internal config is `app_id,entry_url`.
- [ ] Implement transaction-backed OIDC create; detail decrypts only for admin sessions; keep compatibility endpoints functional.
- [ ] Run `go test ./internal/adminapi -count=1`, `go test ./... -count=1`, `go vet ./...`; commit `feat: add typed application CRUD api`.

### Task 4: Replace Application Page with Pure CRUD

**Files:** `web/src/admin-pages.jsx`, `web/src/lib/api.ts`, `web/src/i18n/index.js`, `web/scripts/check-ui-baseline.mjs`.

- [ ] Add failing UI-contract checks for View/Edit/Delete only, no access drawer, type-specific fields, and detail JSON copy warning.
- [ ] Implement a shared Create/Edit form with common fields and dynamic OIDC/API/Internal sections; type read-only on edit.
- [ ] Implement read-only View modal with `navigator.clipboard.writeText(JSON.stringify(detail, null, 2))`, success feedback, and inline clipboard error.
- [ ] Remove role/OIDC drawer loaders and access UI; retain delete confirmation.
- [ ] Run `npm run check:ui` and `npm run build`; commit `feat: redesign application CRUD`.

### Task 5: End-to-End Verification

- [ ] Run `go test ./... -count=1`, `go vet ./...`, `npm run check:ui`, and `npm run build`.
- [ ] Verify the final diff contains only migration, encrypted-secret support, typed CRUD, and application UI changes; commit any verification fixes separately.

## Self-Review

Tasks 1–3 cover persistence, encryption, atomic API behavior, and legacy clients. Task 4 covers separate New/Edit/View/Delete, dynamic fields, and JSON copying. Task 5 supplies fresh full-suite evidence.
