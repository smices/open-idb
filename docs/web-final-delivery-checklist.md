# Web Frontend Final Delivery Checklist

Updated: `2026-07-07`

## Target State

- `web/` is the production React + Vite + Ant Design frontend.
- The retired legacy implementation is not part of the runtime frontend.
- Shared frontend capabilities are present: i18n, light/dark theme, user avatar/menu, and AntD theme tokens.
- User login and administrator login are fully isolated.
- Feishu SSO and Feishu workplace SSO remain available for employee login.

## Route Verification

- `/`: user entry surface only; no administrator text or administrator link.
- `/login`: Feishu login button visible; account/password is fallback.
- `/auth/continue`: continues the user login flow and supports Feishu workplace context.
- `/admin/login`: administrator account login only.
- `/t/{entity}/admin/login`: entity administrator account login only.
- `/admin`: admin console after admin session authentication.

## Admin Console Verification

- Dashboard title and subtitle do not overlap.
- Dashboard user count excludes administrator accounts.
- Dashboard administrator count is separate.
- Sidebar collapse control is centered on the menu rail, not in the logo block or bottom footer.
- Identity sources, sync jobs, organization, users, applications, roles, and audit views remain reachable.

## Sync Verification

- Feishu full sync creates or updates synced users.
- Existing IdBridge ULID user IDs are preserved on merge.
- Directory records missing from a full Feishu snapshot are marked deleted; managed users are archived and removed from the active account set.
- Administrator accounts are not included in synced user counts.

## Required Checks

```bash
make web-frontend-contract
cd web && npm run build
make test
```

Rendered browser verification should be done with browser-act against the running local service.
