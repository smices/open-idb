# React Web Frontend Contract

Applies to the production `web/` frontend.

**Version**: `web-react-antd-contract-v1.0`
**Updated**: `2026-07-07`

## Stack

- Runtime: React + Vite.
- UI system: Ant Design is the default component library. Use AntD components for forms, buttons, tables, menus, drawers, modals, cards, alerts, typography, tabs, segmented controls, tooltips, avatars, layout primitives, and theme tokens.
- Icons: use `lucide-react` where AntD does not already provide the needed symbol.
- Avatar: use the shared avatar implementation in `web/src/components/UserMenu.jsx`.
- i18n: user-facing copy must come from `web/src/i18n/index.js`.
- Theme: light/dark mode must use the shared AntD theme token flow in `web/src/main.jsx` and `web/src/styles.css`.

## Product Boundaries

- `/` is the user entry surface. It must not expose administrator copy, administrator navigation, admin login hints, or admin account recovery behavior.
- `/login` and `/auth/continue` are employee login surfaces. They prioritize Feishu SSO and Feishu workplace SSO; account/password login is only a user-side fallback.
- `/admin/login` and `/t/{entity}/admin/login` are administrator login surfaces. They use independent administrator accounts only.
- Administrator accounts live in `admin_users`. Employee/user accounts live in `users`. The two account systems are isolated and must not be counted or authenticated as each other.
- Dashboard user metrics count synced business users only. Administrator counts are exposed separately as `admin_users`.
- Feishu full sync must preserve existing ULID user IDs when merging by binding/provider identity, union identity, or stable username. Users missing from a full Feishu sync are soft-deleted, not physically deleted.

## Layout And Interaction

- Use a commercial admin-console layout: persistent left sidebar, grouped navigation, constrained content width, and dense but readable operational pages.
- Sidebar collapse is controlled by the rail button on the menu edge, vertically centered in the menu area. Do not place the collapse control in the logo block or bottom footer.
- Login pages must preserve the technology background image and the Feishu login button.
- Admin page titles and subtitles must remain on-screen and readable across desktop widths; avoid header overlap caused by fixed containers.
- Do not rebuild AntD primitives with custom CSS unless AntD cannot express the interaction.

## Checks

Run before shipping frontend changes:

```bash
make web-frontend-contract
cd web && npm run build
```

For full repository verification:

```bash
make test
```

## Change Log

- `web-react-antd-contract-v1.0` (`2026-07-07`)
  - Records the production React + Vite + Ant Design contract after the frontend migration.
  - Documents login isolation, Feishu SSO preservation, full-sync merge semantics, and dashboard count boundaries.
  - Replaces the retired legacy web contract as the active web standard.
