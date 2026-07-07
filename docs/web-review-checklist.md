# Web Frontend Review Checklist

Unified entry: [quickstart-navigation.md](quickstart-navigation.md)

## Baseline

- [ ] Frontend stack is React + Vite.
- [ ] Ant Design is used for standard controls: menu, layout, buttons, forms, tables, cards, drawers, modals, tabs, segmented controls, tooltips, alerts, avatars, and typography.
- [ ] Custom CSS is limited to shell layout, responsive sizing, token mapping, and product-specific composition.
- [ ] Theme uses AntD token flow and supports light/dark mode.
- [ ] User-facing copy comes from `web/src/i18n/index.js`.
- [ ] User avatar and account menu use the shared implementation.

## Login And Account Isolation

- [ ] `/` does not show administrator text, administrator links, or admin login hints.
- [ ] `/login` and `/auth/continue` preserve Feishu SSO and Feishu workplace SSO as the primary employee login path.
- [ ] Account/password login on the user side remains a fallback only.
- [ ] `/admin/login` and `/t/{entity}/admin/login` authenticate independent administrator accounts only.
- [ ] Admin account state never falls back to user account state, and user sessions never count as admin sessions.

## Admin Console

- [ ] Sidebar collapse control is on the menu edge and vertically centered in the menu area.
- [ ] Logo block does not contain the collapse control.
- [ ] Page titles and subtitles render without overlap or clipping on desktop and mobile widths.
- [ ] Dashboard user count excludes administrator accounts.
- [ ] Dashboard administrator count is displayed separately when needed.

## Feishu And Sync

- [ ] Feishu login button is visible on user login surfaces.
- [ ] Feishu workplace SSO flow remains available through `workplace=feishu`.
- [ ] Full sync preserves existing IdBridge ULID user IDs when merging.
- [ ] Full sync soft-deletes synced users missing from the Feishu snapshot.

## Required Checks

```bash
make web-frontend-contract
cd web && npm run build
make test
```

For rendered UI verification, use browser-act against `http://localhost:5180`.
