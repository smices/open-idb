What you implemented

- Added archived-user frontend API types and methods in `web/src/lib/api.ts` for:
  - `listArchivedUsers({ username, limit, offset })`
  - `getArchivedUser(id)`
- Added a read-only archived users admin page in `web/src/admin-pages.jsx`:
  - Username filter
  - Dense table with username, display name, email, archive reason, and archived time
  - Detail drawer that loads `/sapi/archived-users/{id}`
  - JSON snapshot display with copy-to-clipboard action
- Added archived users admin navigation and route wiring in `web/src/main.jsx` and `web/src/admin-pages.jsx`.
- Added Chinese and English translations for archived users labels and navigation copy in `web/src/i18n/index.js`.

What you tested and exact results

- Ran `cd web && npm run build`
  - Result: PASS
  - Output summary: Vite production build completed successfully in about 2 seconds.
- Ran `cd web && npm run check:ui`
  - Result: PASS
  - Output summary: `UI baseline checks passed.`

Files changed

- `web/src/lib/api.ts`
- `web/src/admin-pages.jsx`
- `web/src/i18n/index.js`
- `web/src/main.jsx`

Self-review findings

- The archived users UI is read-only and does not expose any recovery or mutation actions.
- The detail drawer uses the archived-user read endpoint instead of relying on partial list-row data.
- The new menu item follows the existing grouped admin navigation pattern and uses the lucide `Archive` icon as requested.
- The page stays within current admin UI conventions: toolbar + table + drawer, no nested cards, no explainer copy.

Issues or concerns

- The archived-user snapshot types in `web/src/lib/api.ts` follow the task brief exactly. If the backend later returns deeper nested JSON structures inside snapshot payloads, the runtime UI will still work because the drawer renders raw JSON, but the TypeScript shape may become too narrow.
