# Feishu Workplace Diagnostics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Feishu Workplace and Portal loading failures diagnosable by users without exposing credentials.

**Architecture:** A framework-neutral diagnostics helper creates a redacted support bundle. The standalone Workplace page and React Portal consume the helper; API errors retain only response metadata. Backend login classification returns precise stable codes for unsynchronized and inactive identities.

**Tech Stack:** JavaScript, React 19, Ant Design, Node test runner, Go.

---

### Task 1: Add redacted frontend diagnostic primitives

**Files:**
- Create: `web/src/lib/diagnostics.js`
- Create: `web/src/lib/diagnostics.test.mjs`

- [ ] **Step 1: Write failing tests for redaction and formatting**

```js
test('formats a redacted support bundle', () => {
  const text = formatDiagnostic({ code: 'auth_code_failed', route: '/auth/continue?code=secret', error: { status: 502, message: 'upstream failed' } });
  assert.match(text, /code=auth_code_failed/);
  assert.doesNotMatch(text, /secret/);
});
```

- [ ] **Step 2: Run the test and verify it fails because the module is absent**

Run: `cd web && node --test src/lib/diagnostics.test.mjs`

- [ ] **Step 3: Implement `formatDiagnostic`, sensitive-key redaction, and `copyDiagnostic`**

```js
export function formatDiagnostic(input) { /* emits key=value lines only */ }
export async function copyDiagnostic(input) { return navigator.clipboard.writeText(formatDiagnostic(input)); }
```

- [ ] **Step 4: Run the focused test and verify it passes**

Run: `cd web && node --test src/lib/diagnostics.test.mjs`

### Task 2: Keep Workplace failures visible and copyable

**Files:**
- Modify: `web/src/workplace-continue.js:550-640`
- Test: `web/src/lib/diagnostics.test.mjs`

- [ ] **Step 1: Add a failing source-level behavior test**

```js
assert.match(source, /copyDiagnostic/);
assert.doesNotMatch(source, /setTimeout\(\(\) =>[\s\S]*redirectToLogin/);
```

- [ ] **Step 2: Run the test and verify it fails against the redirecting implementation**

Run: `cd web && node --test src/lib/diagnostics.test.mjs`

- [ ] **Step 3: Implement an inline error panel with retry, return-to-login, and copy diagnostics**

```js
showError(copy.error, formatDiagnostic({ code, stage, route: window.location.href, error }));
```

- [ ] **Step 4: Run the focused test and verify it passes**

Run: `cd web && node --test src/lib/diagnostics.test.mjs`

### Task 3: Add Portal diagnostic copy action

**Files:**
- Create: `web/src/portal/DiagnosticDetails.jsx`
- Modify: `web/src/portal/PortalHomePage.jsx:34-64`
- Modify: `web/src/portal/portal-domain.test.mjs`

- [ ] **Step 1: Write a failing Portal source test**

```js
assert.match(source('PortalHomePage.jsx'), /DiagnosticDetails/);
assert.match(source('DiagnosticDetails.jsx'), /copyDiagnostic/);
```

- [ ] **Step 2: Run the test and verify it fails**

Run: `cd web && node --test src/portal/portal-domain.test.mjs`

- [ ] **Step 3: Add an accessible Ant Design details panel and copy button**

```jsx
<Button onClick={copy} aria-label={t('diagnostics.copy')}>{t('diagnostics.copy')}</Button>
```

- [ ] **Step 4: Run Portal tests and verify they pass**

Run: `cd web && node --test src/portal/portal-domain.test.mjs`

### Task 4: Preserve safe API metadata and classify identity state

**Files:**
- Modify: `web/src/lib/api.ts:9-43`
- Modify: `backend/internal/auth/feishu_login.go:775-822`
- Modify: `backend/internal/auth/feishu_login_test.go`

- [ ] **Step 1: Write failing Go classification tests**

```go
if got := classifyFeishuLoginError(errors.New("identity_not_found")); got != "identity_not_found" { t.Fatal(got) }
```

- [ ] **Step 2: Run the test and verify it fails**

Run: `cd backend && go test ./internal/auth -run TestClassifyFeishuLoginError -count=1`

- [ ] **Step 3: Add exact classifications and metadata properties to API errors**

```go
case message == "identity_not_found": return "identity_not_found"
case message == "identity_inactive": return "identity_inactive"
```

- [ ] **Step 4: Run the focused backend and frontend tests**

Run: `cd backend && go test ./internal/auth -run TestClassifyFeishuLoginError -count=1`
Run: `cd web && npm run test:unit && npm run typecheck`

### Task 5: Verify the release surface

**Files:**
- Modify: `web/src/i18n/index.js`
- Modify: `docs/superpowers/plans/2026-07-16-portal-application-directory.md`

- [ ] **Step 1: Add localized labels for diagnostic copy, success, and safe fallback**
- [ ] **Step 2: Mark completed Portal plan checks accurately**
- [ ] **Step 3: Run full verification**

Run: `cd backend && go test ./...`
Run: `cd web && npm run check`

- [ ] **Step 4: Commit the implementation**

```bash
git add web/src backend/internal/auth docs/superpowers
git commit -m "fix(feishu): expose redacted workplace diagnostics"
```
