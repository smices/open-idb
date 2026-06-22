# Storage Architecture

IdBridge uses PostgreSQL as the authoritative data store for production. Redis
is optional and must only be used for ephemeral state, cache, and rate limiting.

## Production Source of Truth

PostgreSQL is the source of truth for:

- business entities, users, identity sources, account bindings, and directory data
- OIDC clients and application assignments
- authorization codes, token revocation state, and token expiry metadata
- login sessions, session revocation, and session expiry
- roles, permissions, resource scopes, and audit logs

Security decisions must be valid from PostgreSQL alone. Redis data loss,
expiration, failover, or flush operations must not grant access, revive revoked
tokens, or bypass authorization checks.

## Session Model

The browser cookie stores only an opaque `idb_session` session id. User identity
and validity are resolved server-side from the `sessions` table.

Every authenticated request must verify:

- the session id exists
- `sessions.status = 'active'`
- `sessions.expires_at > now()`
- the referenced user is still active

Session revocation updates PostgreSQL. Any Redis session cache entry, when
introduced, is a best-effort acceleration layer and must be deleted on revoke.

## OIDC Authorization State

Authorization code and token validity is represented in PostgreSQL:

- `oauth_authorization_codes.expires_at`
- `oauth_authorization_codes.used_at`
- `oauth_tokens.expires_at`
- `oauth_tokens.revoked_at`

Authorization code exchange must atomically mark a code as used with a predicate
that requires the code to be unused and unexpired. Redis TTL may be used as a
short-lived cache, but the PostgreSQL fields remain the security boundary.

A background cleanup runner periodically:

- deletes expired authorization codes
- deletes expired OAuth token records
- marks expired active sessions as `expired`

## Redis Scope

Redis is optional. Suitable Redis keys include:

- `oidc:state:{state_hash}` for short-lived login state
- `oidc:login_context:{context_hash}` for temporary redirect/login context
- `rate:login:{hash}` for login rate limiting
- `rate:token_exchange:{hash}` for token endpoint rate limiting
- `session_cache:{session_id}` for short-lived session lookup cache

Redis must not store:

- cleartext access tokens, ID tokens, refresh tokens, or client secrets
- authoritative OIDC client configuration
- token revocation facts
- audit logs
- user, role, permission, or resource-scope source data

Redis is enabled through configuration:

- `IDB_REDIS_URL=redis://redis.open-idb.svc.cluster.local:6379/0`
- `IDB_REDIS_ENABLED=true`

Setting `IDB_REDIS_URL` enables Redis even when `IDB_REDIS_ENABLED` is not set.
When Redis is not configured, IdBridge uses an in-process memory store for
ephemeral helpers. The memory store is intentionally not a durable or shared
cluster coordination mechanism.

Current Redis-backed behavior:

- Feishu OAuth redirect state: opaque state id, 5-minute TTL, deleted after callback
- local account login rate limit: 10 attempts per entity/account/IP per 15 minutes
- OIDC token exchange rate limit: 30 attempts per client/IP per minute

## Database Portability

The project is PostgreSQL-only. Multi-database support is not a current goal.
The schema intentionally uses PostgreSQL behavior such as `TEXT[]`,
`TIMESTAMPTZ`, pgcrypto password hashing, SQL constraints, and sqlc generated
`pgx/v5` bindings.

Future storage work should improve service-level Store interfaces without
committing to MySQL, SQLite, or other SQL dialect support.

Current code exposes service-level Store interfaces for OIDC flows so service
logic depends on domain persistence contracts rather than directly on the sqlc
query container. The production implementation remains PostgreSQL/sqlc.
