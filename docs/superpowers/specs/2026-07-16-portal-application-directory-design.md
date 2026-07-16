# Portal Application Directory Design

## Status

Accepted on 2026-07-16.

## Context

The authenticated user Portal currently shares page-level implementation with the
admin console. This caused `PortalPage` to call `useLoader`, a hook defined only
inside the lazily loaded admin module, producing a runtime `ReferenceError`.

The Portal is a user-facing product surface with a different purpose from the
admin console: employees view their own profile and discover their enterprise's
applications. It must not depend on admin page modules, admin-only APIs, or
authorization presentation logic.

## Decision

Create a dedicated Portal feature domain. It may share the HTTP client and
common visual primitives with the rest of the web application, but it owns its
data hooks, pages, routes, and Portal-specific UI state.

The first Portal release presents an enterprise application directory. It lists
every enabled application configured for the current user's enterprise. It does
not determine whether a user may access an application and does not expose role,
permission, or access-assignment data. Authorization-aware application access
will be introduced as a separate subsequent capability.

## Frontend Boundary

Add a `web/src/portal/` domain with the following responsibilities:

- `PortalShell`: authenticated user navigation and Portal layout.
- `PortalHomePage`: enterprise application directory.
- `ProfilePage`: self-service profile and password changes.
- `usePortalApplications`: loading, refresh, cancellation, and error state for
  the Portal application directory only.
- `portal-routes`: route selection for Portal pages.

`web/src/admin-pages.jsx` remains an independently lazy-loaded admin domain.
Portal code must not import it or reuse hooks declared in it. `web/src/lib/api.ts`
remains the shared HTTP transport boundary.

## API Contract

Add an authenticated, user-session-scoped endpoint:

`GET /api/portal/applications`

The endpoint returns all enabled applications for the current session's
enterprise, sorted deterministically by application name and ID. It returns only
safe catalogue fields:

```json
{
  "applications": [
    {
      "id": "01...",
      "name": "Expense",
      "type": "internal_app",
      "description": "Submit and review expenses",
      "logo_url": "https://...",
      "entry_url": "https://..."
    }
  ]
}
```

`description`, `logo_url`, and `entry_url` are optional. The response must not
include client secrets, secret hashes, provider credentials, redirect URIs,
application role assignments, permissions, or `has_access`.

Disabled applications are omitted. A missing or invalid user session receives
the existing unauthenticated response semantics.

The existing `GET /api/me/access` remains an access-summary API and is not used
by the Portal directory.

## User Experience

The Portal home page renders the enabled enterprise application catalogue as
cards or a responsive list. Each item shows its name, type, optional logo and
description, and an entry action when `entry_url` is present. The page shows an
empty state when no enabled applications exist and a retryable error state when
the directory request fails.

The Portal navigation contains only Portal-owned destinations: My Applications
and My Profile. It does not expose admin console navigation or access decisions.

## Future Access Capability

The future authorization capability may extend each application item with an
optional access field such as `access_status`. It must not change the identity
or catalogue fields in this response. Entry actions can then be gated or routed
through an authorization flow without re-coupling Portal pages to the admin
domain.

## Testing and Acceptance Criteria

- A user who completes login can render the Portal application directory without
  loading or importing any admin-page hook.
- The API returns all and only enabled applications in the current enterprise.
- The response excludes all secret and authorization-assignment fields.
- The UI renders application, empty, loading, error, and retry states.
- Existing `/api/me/access` behavior remains unchanged.
- Frontend production build and backend test suite pass.

## Alternatives Rejected

### Define `useLoader` in `main.jsx`

This removes the immediate exception but preserves an accidental shared
implementation between unrelated product domains.

### Export the admin `useLoader` for Portal use

This makes the Portal depend on an admin module and defeats lazy-loading and
domain boundaries.

### Use `/api/me/access` as the Portal directory

That endpoint is access-oriented and returns a user-specific subset plus
authorization information. It conflicts with the first-release requirement to
show the full enabled enterprise catalogue before access control is introduced.
