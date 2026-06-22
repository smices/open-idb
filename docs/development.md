# Development

## Requirements

- Go 1.22 or newer
- Node.js and npm
- Docker
- kubectl connected to the local development cluster

The backend lives under `backend/`. The only frontend is the SvelteKit web app under `web/`.

## Common Commands

```bash
make test
make lint
make generate
DATABASE_URL=postgres://postgres:postgres@localhost:5432/idbridge?sslmode=disable make migrate-up
make run
```

Direct backend commands:

```bash
cd backend
go test ./...
go run ./cmd/idbridge
```

## Local Full-Stack Development

Use `make dev-local` to run the local backend and `web/` frontend with hot reload while using PostgreSQL from the local k8s dev environment.

```bash
make dev-local
```

What it does:

- Checks required local tools: `kubectl`, `go`, `node`, `npm`.
- Validates the k8s namespace and `svc/postgres`.
- Starts a PostgreSQL port-forward to `127.0.0.1:15432`.
- Starts the Go backend at `http://localhost:18080`.
- Starts the SvelteKit web frontend at `http://localhost:5180`.
- Runs the frontend contract checks in preflight/quickfix flows.

Useful overrides:

```bash
DEV_LOCAL_NAMESPACE=open-idb make dev-local
DEV_LOCAL_PG_PORT=15433 DEV_LOCAL_BACKEND_PORT=18081 DEV_LOCAL_WEB_PORT=5181 make dev-local
IDB_ETCD_SERVICE=etcd make dev-local
```

## Web Frontend

```bash
make web-dev
make web-build
make web-frontend-contract
make web-check WEB_CHECK_STRICT=error
```

Or run Vite directly:

```bash
cd web
npm run dev -- --host 0.0.0.0 --port 5180
```

## Local HTTPS on OrbStack k8s

```bash
make k8s-build
make k8s-build-frontend
make k8s-deploy
make k8s-deploy-frontend
TRUST_LOCAL_CA=1 make k8s-local-tls
```

Current local domain:

```bash
https://idbridge.local.test/
```

The k8s ingress points `/` at the `web/` frontend service. The frontend Caddy container only proxies `/api*` to the backend; everything else is served by the frontend.

## Troubleshooting

Run the preflight first:

```bash
make dev-local-preflight
```

If startup fails:

```bash
make dev-local-quickfix
```

Check logs:

```bash
tail -n 80 .local/dev-web-local-backend.log
tail -n 80 .local/dev-web-local-web.log
tail -n 80 .local/dev-web-local-postgres.log
```

Check k8s dependencies:

```bash
kubectl -n open-idb get namespace open-idb
kubectl -n open-idb get svc postgres
kubectl -n open-idb get pod -l app.kubernetes.io/name=postgres
```

Check ports:

```bash
lsof -nP -iTCP:15432,18080,5180 -sTCP:LISTEN
```

## Submission Checks

Before opening a PR:

```bash
make web-frontend-contract
make web-check WEB_CHECK_STRICT=error
make test
```

Quick route verification after a k8s frontend deploy:

```bash
curl -k --resolve idbridge.local.test:443:192.168.139.2 https://idbridge.local.test/login
curl -k --resolve idbridge.local.test:443:192.168.139.2 -X POST -d 'account=admin&password=wrong' https://idbridge.local.test/api/login/account
```
