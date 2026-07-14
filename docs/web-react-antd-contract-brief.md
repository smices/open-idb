# Web Frontend Contract Brief

Applies to `web/`.

**Version**: `web-react-antd-contract-v1.0`
**Updated**: `2026-07-07`

- React + Vite is the production frontend stack.
- Ant Design is the default UI component system. Use AntD before custom components.
- Copy must use `web/src/i18n/index.js`; no page-level hardcoded UI text for translatable strings.
- Light/dark theme, language switching, user avatar, and user menu are shared frontend capabilities.
- `/` must not expose administrator entry points or administrator copy.
- Employee login uses Feishu SSO first. Admin login is isolated under `/admin/login` or `/t/{entity}/admin/login`.
- Dashboard user metrics exclude administrator accounts; admin count is a separate metric.
- Feishu full sync keeps stable ULID user IDs; missing directory records are marked deleted and missing managed users are archived out of the active account set.

Required checks:

```bash
make web-frontend-contract
cd web && npm run build
```

Full contract: [web-react-antd-contract.md](web-react-antd-contract.md)
