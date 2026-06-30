# IdBridge

Open-source enterprise identity bridge for internal applications. IdBridge provides a focused identity control plane: company management, identity source configuration, Feishu directory sync, account authorization, OIDC application access, administrator management, and audit logs.

The product is designed for self-hosted deployments in a company IDC or private Kubernetes environment. It is not a hosted SaaS tenant model.

## What It Does

- Admin console: `/admin/*`
- User portal: `/portal`
- User/API surface: `/api/*`
- Admin/API surface: `/sapi/*`
- OIDC provider: authorization code + PKCE
- Current primary identity source: Feishu
- Planned identity source boundary: Feishu, WeCom, DingTalk, LDAP; current release only enables Feishu in production use

## Quick Start For Local Development

Requirements:

- Go 1.22+
- Node.js 20+
- npm
- Docker
- kubectl, when using the local Kubernetes flow

Run the local full stack:

```bash
make dev-web-local
```

Default local URLs:

- Web: `http://localhost:5180`
- Backend: `http://localhost:18080/healthz`
- Admin: `http://localhost:5180/admin/login`

Initial administrator after the baseline migration:

- Username: `admin`
- Password: `admin123`

Change this password immediately after first login.

## Production Deployment

Read the full deployment guide before exposing the service:

- [Self-hosted deployment guide](docs/deployment.md)
- [Application integration guide](docs/integration-guide.md)

Minimum production requirements:

- PostgreSQL with `pgcrypto` available
- HTTPS public or internal domain, for example `https://idbridge.example.com`
- Correct `IDB_OIDC_ISSUER`
- Reverse proxy routes for both `/api/*` and `/sapi/*`
- Feishu app credentials if Feishu login or directory sync is used

Critical environment variables:

```bash
DATABASE_URL='postgres://idbridge:***@postgres:5432/idbridge?sslmode=disable'
IDB_HTTP_ADDR=':8080'
IDB_OIDC_ISSUER='https://idbridge.example.com'
IDB_WEB_BASE_URL='https://idbridge.example.com'
IDB_FEISHU_REDIRECT_URI='https://idbridge.example.com/api/auth/feishu/callback'
```

Run database initialization before starting the backend:

```bash
cd backend
go run github.com/pressly/goose/v3/cmd/goose@v3.22.1 -dir migrations postgres "$DATABASE_URL" up
```

## Reverse Proxy Contract

If the frontend and backend are exposed under one domain, route:

- `/api/*` -> backend
- `/sapi/*` -> backend
- `/` -> frontend

The provided `web/Dockerfile` image uses Caddy and already proxies `/api*` and `/sapi*` to the backend service named `idbridge:8080`.

OIDC discovery for application integrators should use:

```text
https://idbridge.example.com/api/.well-known/openid-configuration
```

## Application Integration

Create applications from the admin console:

```text
/admin/applications
```

For OIDC applications, IdBridge automatically issues:

- `client_id`
- `client_secret`
- allowed scopes
- grant type
- response type
- PKCE requirement
- Feishu login template
- Feishu workplace SSO template

The application owner must provide a valid absolute callback URL, for example:

```text
https://app.example.com/auth/oidc/callback
```

OIDC clients that need people picker or organization search capabilities can enable the Directory API scope in IdBridge and call:

```text
GET /api/directory/organization-tree/search?q=<keyword>
GET /api/directory/organization-tree/root
GET /api/directory/organization-tree/children?id=<node_id>&kind=company|organization|department
```

## Verification

Before shipping an image or deploying to an IDC environment:

```bash
cd backend && go test ./...
cd web && npm run check && npm run build
```

After deployment:

```bash
curl -fsS https://idbridge.example.com/api/.well-known/openid-configuration
curl -fsS https://idbridge.example.com/api/oauth2/authorize
curl -fsS https://idbridge.example.com/admin/login
```

`/api/oauth2/authorize` is expected to redirect to login when there is no user session.

## License

MIT
