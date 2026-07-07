# IdBridge Web

React + Vite + Ant Design frontend for IdBridge.

## Scripts

- `npm run dev` starts the local dev server on Vite.
- `npm run build` creates the production build in `dist/`.
- `npm run preview` serves the production build locally.
- `npm run check:ui` verifies the frontend baseline rules for the AntD migration.
- `npm run check` runs the production build.

The API proxy target defaults to `http://localhost:18080` and can be changed with `PUBLIC_API_TARGET` or `VITE_API_TARGET`.

## Product Boundaries

- `/`, `/login`, and `/auth/continue` are user-side surfaces. They must keep Feishu SSO visible and must not expose administrator entry points.
- `/admin/login` and `/t/{entity}/admin/login` are administrator-only account login surfaces.
- Shared UI capabilities live in the React app: i18n, light/dark theme, user avatar/menu, and Ant Design tokens.
