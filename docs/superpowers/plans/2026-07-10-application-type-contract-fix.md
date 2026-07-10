# Application Type Contract Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make application creation use the existing `oidc_client`, `api_client`, and `internal_app` contract and reject unsupported types with HTTP 400 before PostgreSQL.

**Architecture:** Keep the database constraint authoritative and add matching validation at the HTTP boundary. Expose the same three-value contract in the frontend API type, creation form, translations, and static UI contract checks so the browser cannot submit the obsolete values.

**Tech Stack:** Go 1.24, `net/http`, Chi, PostgreSQL, React 19, Ant Design 5, TypeScript, Vite, Node.js static contract checks.

## Global Constraints

- Application types are exactly `oidc_client`, `api_client`, and `internal_app`.
- Do not add a database migration or legacy aliases.
- Unsupported application types return HTTP 400 with error code `invalid_application_type`.
- Missing `name` or `type` retains the existing `invalid_request` behavior.
- Do not address the unrelated Chrome ad-blocking or permissions-policy warnings.

---

## File Structure

- Create `backend/internal/adminapi/application_handlers_test.go`: request-boundary regression tests for permitted and unsupported application types.
- Modify `backend/internal/adminapi/application_handlers.go`: centralized three-value predicate and pre-database validation.
- Modify `web/src/lib/api.ts`: exported `ApplicationType` union and typed create payload.
- Modify `web/scripts/check-ui-baseline.mjs`: static contract assertions for form values, default payload, API union, and translations.
- Modify `web/src/admin-pages.jsx`: canonical application-type constant, default, and select options.
- Modify `web/src/i18n/index.js`: Chinese and English labels keyed by canonical values.

### Task 1: Reject Invalid Application Types at the API Boundary

**Files:**
- Create: `backend/internal/adminapi/application_handlers_test.go`
- Modify: `backend/internal/adminapi/application_handlers.go:14-24,97-125`

**Interfaces:**
- Consumes: existing `userService.CreateApplication(ctx context.Context, entityID string, name, appType string) (ApplicationResponse, error)`.
- Produces: `isValidApplicationType(appType string) bool`; `POST /sapi/applications` returns 400/`invalid_application_type` for unsupported values.

- [ ] **Step 1: Write the failing handler regression tests**

Create `backend/internal/adminapi/application_handlers_test.go`:

```go
package adminapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

type applicationTestService struct {
	*mockUserService
	createApplicationFn func(context.Context, string, string, string) (ApplicationResponse, error)
}

func (s *applicationTestService) CreateApplication(ctx context.Context, entityID, name, appType string) (ApplicationResponse, error) {
	if s.createApplicationFn == nil {
		return ApplicationResponse{}, fmt.Errorf("unexpected CreateApplication call")
	}
	return s.createApplicationFn(ctx, entityID, name, appType)
}

func newApplicationTestRouter(service userService) *chi.Mux {
	router := chi.NewRouter()
	NewApplicationHandler(service).RegisterRoutes(router)
	return router
}

func TestApplicationHandlerCreateAcceptsContractTypes(t *testing.T) {
	for _, appType := range []string{"oidc_client", "api_client", "internal_app"} {
		t.Run(appType, func(t *testing.T) {
			service := &applicationTestService{mockUserService: &mockUserService{}}
			service.createApplicationFn = func(_ context.Context, entityID, name, gotType string) (ApplicationResponse, error) {
				if entityID != "01HZZZZZZZ0000000000000099" || name != "Contract App" || gotType != appType {
					t.Fatalf("CreateApplication args = (%q, %q, %q)", entityID, name, gotType)
				}
				return ApplicationResponse{Name: name, Type: gotType, Status: "active"}, nil
			}

			req := httptest.NewRequest(http.MethodPost, "/sapi/applications", strings.NewReader(fmt.Sprintf(`{"name":"Contract App","type":%q}`, appType)))
			req.AddCookie(adminTestSessionCookie())
			rr := httptest.NewRecorder()

			newApplicationTestRouter(service).ServeHTTP(rr, req)

			if rr.Code != http.StatusCreated {
				t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestApplicationHandlerCreateRejectsUnsupportedType(t *testing.T) {
	called := false
	service := &applicationTestService{mockUserService: &mockUserService{}}
	service.createApplicationFn = func(_ context.Context, _, _, _ string) (ApplicationResponse, error) {
		called = true
		return ApplicationResponse{}, nil
	}
	req := httptest.NewRequest(http.MethodPost, "/sapi/applications", strings.NewReader(`{"name":"Legacy App","type":"oidc"}`))
	req.AddCookie(adminTestSessionCookie())
	rr := httptest.NewRecorder()

	newApplicationTestRouter(service).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if called {
		t.Fatal("CreateApplication was called for an unsupported type")
	}
	if !strings.Contains(rr.Body.String(), `"error":"invalid_application_type"`) {
		t.Fatalf("body = %s", rr.Body.String())
	}
}
```

- [ ] **Step 2: Run the focused test and verify RED**

Run from `backend/`:

```bash
go test ./internal/adminapi -run '^TestApplicationHandlerCreate' -count=1
```

Expected: `TestApplicationHandlerCreateRejectsUnsupportedType` fails because the current handler calls the service and returns 201 for `oidc`.

- [ ] **Step 3: Add the minimal application-type predicate and handler validation**

Add above `NewApplicationHandler` in `backend/internal/adminapi/application_handlers.go`:

```go
func isValidApplicationType(appType string) bool {
	switch appType {
	case "oidc_client", "api_client", "internal_app":
		return true
	default:
		return false
	}
}
```

In `createApplication`, immediately after the existing required-field check, add:

```go
	if !isValidApplicationType(body.Type) {
		writeError(w, http.StatusBadRequest, "invalid_application_type", "type must be one of oidc_client, api_client, or internal_app")
		return
	}
```

- [ ] **Step 4: Format and verify GREEN**

Run from `backend/`:

```bash
gofmt -w internal/adminapi/application_handlers.go internal/adminapi/application_handlers_test.go
go test ./internal/adminapi -run '^TestApplicationHandlerCreate' -count=1
go test ./internal/adminapi -count=1
```

Expected: all focused and package tests pass.

- [ ] **Step 5: Commit the backend boundary fix**

```bash
git add backend/internal/adminapi/application_handlers.go backend/internal/adminapi/application_handlers_test.go
git diff --cached --check
git commit -m "fix: validate application types before insert"
```

### Task 2: Align the Frontend Application-Type Contract

**Files:**
- Modify: `web/scripts/check-ui-baseline.mjs`
- Modify: `web/src/lib/api.ts:407-415,756-757`
- Modify: `web/src/admin-pages.jsx:471-510`
- Modify: `web/src/i18n/index.js:269-271,617-619`

**Interfaces:**
- Consumes: backend application-type values from Task 1.
- Produces: `ApplicationType = 'oidc_client' | 'api_client' | 'internal_app'`; the create form submits one of these values and defaults to `oidc_client`.

- [ ] **Step 1: Add failing frontend contract checks**

In `web/scripts/check-ui-baseline.mjs`, read the API and translation sources alongside the existing source reads:

```js
const apiTs = readFileSync(join(root, 'src/lib/api.ts'), 'utf8');
const i18nJs = readFileSync(join(root, 'src/i18n/index.js'), 'utf8');
```

Before the final `if (failures.length)` block, add:

```js
const applicationTypes = ['oidc_client', 'api_client', 'internal_app'];
const applicationTypeDeclaration = `const APPLICATION_TYPES = ['oidc_client', 'api_client', 'internal_app'];`;
const applicationTypeUnion = `export type ApplicationType = 'oidc_client' | 'api_client' | 'internal_app';`;

if (!adminPagesJs.includes(applicationTypeDeclaration)) {
  failures.push('Applications page must declare the canonical application type list');
}
if (!adminPagesJs.includes("type: values.type || 'oidc_client'")) {
  failures.push('Applications page must default new applications to oidc_client');
}
if (!apiTs.includes(applicationTypeUnion) || !apiTs.includes('payload: { name: string; type: ApplicationType }')) {
  failures.push('Frontend API must expose the canonical ApplicationType contract');
}
for (const appType of applicationTypes) {
  const key = `'applications.type.${appType}':`;
  if (i18nJs.split(key).length - 1 !== 2) {
    failures.push(`Missing bilingual application type label: ${appType}`);
  }
}
for (const obsoleteType of ['oidc', 'saml', 'custom']) {
  if (i18nJs.includes(`'applications.type.${obsoleteType}':`)) {
    failures.push(`Obsolete application type translation remains: ${obsoleteType}`);
  }
}
```

- [ ] **Step 2: Run the UI check and verify RED**

Run from `web/`:

```bash
npm run check:ui
```

Expected: failure messages report the missing canonical declaration/default/API union/labels and the obsolete translations.

- [ ] **Step 3: Type the frontend API contract**

Immediately before `export interface Application` in `web/src/lib/api.ts`, add:

```ts
export type ApplicationType = 'oidc_client' | 'api_client' | 'internal_app';
```

Change the `Application` field and create payload to:

```ts
export interface Application {
  id: string;
  entity_id: string;
  name: string;
  type: ApplicationType;
  status: string;
  created_at: string;
  updated_at: string;
}
```

```ts
  createApplication: (payload: { name: string; type: ApplicationType }) =>
    apiRequest<Application>('/sapi/applications', { method: 'POST', body: payload }),
```

- [ ] **Step 4: Update the form values and default**

Add near the other top-level constants in `web/src/admin-pages.jsx`:

```js
const APPLICATION_TYPES = ['oidc_client', 'api_client', 'internal_app'];
```

Change the create call to:

```js
    else await api.createApplication({ name: values.name, type: values.type || 'oidc_client' });
```

Change the type select options to:

```jsx
<Form.Item name="type" label={t('applications.type')}>
  <Select
    disabled={Boolean(selected)}
    options={APPLICATION_TYPES.map((value) => ({ value, label: t(`applications.type.${value}`, value) }))}
  />
</Form.Item>
```

Keep the surrounding form fields and behavior unchanged.

- [ ] **Step 5: Replace obsolete bilingual translation keys**

In the Chinese catalog in `web/src/i18n/index.js`, replace the three old keys with:

```js
  'applications.type.oidc_client': 'OIDC 客户端',
  'applications.type.api_client': 'API 客户端',
  'applications.type.internal_app': '内部应用',
```

In the English catalog, use:

```js
  'applications.type.oidc_client': 'OIDC Client',
  'applications.type.api_client': 'API Client',
  'applications.type.internal_app': 'Internal Application',
```

- [ ] **Step 6: Verify GREEN and build the frontend**

Run from `web/`:

```bash
npm run check:ui
npm run build
```

Expected: `UI baseline checks passed.` and Vite exits 0 with production assets built.

- [ ] **Step 7: Commit the frontend contract fix**

```bash
git add web/scripts/check-ui-baseline.mjs web/src/lib/api.ts web/src/admin-pages.jsx web/src/i18n/index.js
git diff --cached --check
git commit -m "fix: align application type options with schema"
```

### Task 3: Full Verification and Scope Review

**Files:**
- Verify only; modify files only if a command exposes a defect in this plan's changes.

**Interfaces:**
- Consumes: Tasks 1 and 2.
- Produces: fresh evidence that backend tests/vet and frontend checks/build pass together.

- [ ] **Step 1: Run all backend tests**

Run from `backend/`:

```bash
go test ./... -count=1
go vet ./...
```

Expected: both commands exit 0. Docker-backed integration tests may report the repository's established clean skip when Docker is unavailable; no test may fail.

- [ ] **Step 2: Run all frontend verification**

Run from `web/`:

```bash
npm run check:ui
npm run build
```

Expected: both commands exit 0.

- [ ] **Step 3: Review the final diff and scope**

Run from the repository root:

```bash
git status --short --branch
git diff HEAD~2 --check
git diff HEAD~2 -- backend/internal/adminapi/application_handlers.go backend/internal/adminapi/application_handlers_test.go web/scripts/check-ui-baseline.mjs web/src/lib/api.ts web/src/admin-pages.jsx web/src/i18n/index.js
```

Confirm the diff contains only the application-type validation, frontend contract values, translations, and their tests. Confirm no migration or Chrome-warning work is present.

## Self-Review

- Spec coverage: backend validation/error behavior is covered by Task 1; canonical frontend values/default/translations are covered by Task 2; complete verification is covered by Task 3.
- Contract consistency: every layer uses `oidc_client`, `api_client`, and `internal_app`; the stable API error is `invalid_application_type`.
- Scope: no database migration, compatibility alias, SAML feature, onboarding redesign, or Chrome-warning change is included.
