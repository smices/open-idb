# OrbStack Kubernetes

This project can run locally on OrbStack Kubernetes in the `open-idb` namespace.

## Commands

```bash
make k8s-build
make k8s-deploy
TRUST_LOCAL_CA=1 make k8s-local-tls
make k8s-status
```

Then check the app through the CoreDNS/Traefik host:

```bash
curl -k https://idbridge.local.test/login
```

## Notes

- The local image tag is `open-idb:dev`.
- The local frontend image tag is `open-idb-frontend:dev`, built from `web/`.
- The Kubernetes manifests use `imagePullPolicy: Never` so OrbStack uses the locally built image.
- PostgreSQL uses `emptyDir` storage for local development. Data is not durable.
- The migration job runs `goose` from the application image against `/app/migrations`.
- The Ingress host is `idbridge.local.test` and uses the `traefik` IngressClass.
- The Ingress points `/` to the `web/` frontend service. The frontend Caddy container only sends `/api*` to the backend.
- The `web` frontend follows `docs/web-svelte-tailwind-contract.md` (SvelteKit + Tailwind + Skeleton, non-SPA main flow).
- `make k8s-local-tls` creates an untracked local CA and wildcard certificate under `.local/tls/`, creates the Kubernetes `idbridge-tls` secret, and reapplies the Ingress.
- `TRUST_LOCAL_CA=1 make k8s-local-tls` also trusts the generated CA in the macOS login keychain so browsers stop warning about `https://idbridge.local.test`.

## Local HTTPS

Run this once on each Mac that should trust `*.local.test`:

```bash
TRUST_LOCAL_CA=1 make k8s-local-tls
```

The generated files stay in `.local/tls/` and are ignored by git. Safari and Chrome use the macOS keychain trust. Firefox may need `security.enterprise_roots.enabled=true` or a manual import of `.local/tls/dev-idbridge-ca.crt`.

## Live Refresh

For local backend development with automatic rebuild/restart:

```bash
make k8s-dev-watch
```

The watcher checks source files every two seconds, rebuilds `open-idb:dev`, and restarts only the `idbridge` deployment. It does not recreate PostgreSQL.

If only manifests or image changed and you do not need the watcher:

```bash
make k8s-build
make k8s-deploy-app
```

## Port Forward Fallback

If DNS or Ingress is unavailable:

```bash
make k8s-port-forward
curl -sS http://localhost:8080/readyz
```
