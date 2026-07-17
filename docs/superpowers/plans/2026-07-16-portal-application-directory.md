# Portal Application Directory Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver an independent authenticated Portal that lists every enabled application configured for the current enterprise, without evaluating application access.

**Architecture:** Add a session-scoped catalogue endpoint with a deliberately small application projection, then move Portal shell, profile, page, and resource hook into `web/src/portal/`. The Portal shares only the API client and common presentation components; it does not import admin pages or their hooks.

**Tech Stack:** Go, chi, pgx/sqlc, React, Vite, Ant Design.

## Global Constraints

- List only enabled applications in the active session's enterprise.
- Do not return secrets, secret hashes, redirect URIs, provider credentials, role assignments, permissions, or access decisions.
- Keep `GET /api/me/access` unchanged and do not use it for the Portal catalogue.
- Portal code must not import `admin-pages.jsx`.
- Preserve existing authenticated Portal profile and password behavior.

---

### Task 1: Add the user-session application catalogue API

**Files:** Create `backend/internal/portal/handler.go` and `handler_test.go`; modify the application query package and `backend/internal/httpserver/router.go`.

**Produces:** `GET /api/portal/applications` with `{ applications: [{ id, name, type, description?, logo_url?, entry_url? }] }`.

- [x] Write failing handler tests for session entity scope, enabled-only filtering, deterministic ordering, unauthenticated rejection, and safe response projection.
- [x] Run `go test ./internal/portal -run TestListApplications -count=1`; it must fail because the handler and route are absent.
- [x] Add a dedicated query scoped by `entity_id` with `status = 'active'` and `ORDER BY name, id`; select only safe catalogue fields.
- [x] Implement the session-authenticated handler with a response-specific struct; do not call admin list endpoints or return admin application models.
- [x] Register the route alongside user-session routes, not under `/sapi` or admin authorization middleware.
- [x] Re-run `go test ./internal/portal -run TestListApplications -count=1`; it must pass.
- [x] Commit with `feat(portal): add enterprise application catalogue`.

### Task 2: Create the independent Portal frontend domain

**Files:** Create `web/src/portal/usePortalApplications.js`, `PortalHomePage.jsx`, and `PortalShell.jsx`; modify `web/src/lib/api.ts` and `web/src/main.jsx`; add frontend tests following the existing project convention.

**Consumes:** `api.portalApplications(): Promise<{ applications: PortalApplication[] }>`.

**Produces:** `usePortalApplications(): { loading, applications, error, reload }` and an independent Portal shell.

- [x] Write failing tests for applications, empty state, request error with retry, and absence of imports from `admin-pages.jsx` or `useLoader`.
- [x] Run the repository's focused frontend test command; it must fail because the Portal API method, hook, and components do not exist.
- [x] Add the typed API client method for `/api/portal/applications`.
- [x] Implement `usePortalApplications` with local loading, error, retry, and stale-request cancellation; it must not reuse the admin hook.
- [x] Implement the Portal home page as a card or responsive list directory using only catalogue fields. Render an entry action only when `entry_url` is present. Do not display roles, permissions, `has_access`, or authorization wording.
- [x] Move Portal navigation, profile routing, and home-page rendering out of `main.jsx`; retain admin lazy-loading exclusively for admin routes.
- [x] Run focused frontend tests and `npm run build` in `web/`; both must pass without an unresolved `useLoader` reference.
- [x] Commit with `feat(portal): isolate enterprise application directory`.

### Task 3: Integration verification

**Files:** Modify the closest authenticated Portal integration test under `backend/tests/integration/`.

- [x] Write an integration test that creates enabled and disabled applications for two enterprises, authenticates one user, and asserts that the Portal endpoint returns only that enterprise's enabled catalogue.
- [x] Assert in the same test that `/api/me/access` retains its access-summary behavior.
- [x] Run `go test ./tests/integration -run Portal -count=1`; it must pass.
- [x] Run final verification: `(cd backend && go test ./...)`, `(cd web && npm run build)`, and `git diff --check`.
- [x] Commit with `test(portal): cover enterprise application catalogue`.
