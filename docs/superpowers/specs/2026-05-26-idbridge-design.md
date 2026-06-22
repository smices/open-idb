# IdBridge (IDB) Design

## Positioning

IdBridge (IDB) is an enterprise identity infrastructure platform for one enterprise group. It connects business entities such as headquarters, branches, subsidiaries, factories, brands, and operating units with managed users, applications, permissions, and audit trails into one identity hub.

The product is not a generic login system and is not a SaaS platform sold to unrelated companies. It is the identity foundation for internal systems such as ERP, WMS, OMS, BI, CRM, finance systems, AI data platforms, admin tools, and approval/operations platforms inside the enterprise group.

## Product Name

- Product name: IdBridge
- Short name: IDB
- Category: Enterprise Identity Infrastructure
- Chinese positioning: 企业身份基础设施
- One-line description: IdBridge connects enterprise-group entities, organization, accounts, permissions, and system access into a unified identity hub.

## MVP Scope

The first version must support:

- OIDC SSO for internal applications.
- Feishu login.
- Feishu organization and user full sync.
- Extensible identity source model for DingTalk, WeCom, LDAP, and local identities.
- Managed user lifecycle.
- External directory user lifecycle.
- Account binding between external identities and managed users.
- RBAC.
- Minimal resource scope permissions for cross-border ecommerce scenarios.
- Application access assignment.
- Session and token management.
- Audit logs for login, sync, authorization, and administration.
- Built-in i18n from day one.

The first version will not implement:

- SAML.
- SCIM server.
- WebAuthn or Passkey.
- Full ABAC or OPA policy engine.
- External SaaS billing, customer plans, or public entity registration.
- Productized API Gateway plugins.
- Fully automatic conflict resolution across multiple identity sources.

## Business Entity Boundary Model

IdBridge uses a business-entity boundary model, not a SaaS tenant model.

Definitions:

- Business entity: a headquarters company, branch, subsidiary, factory, brand, or operating unit inside the same enterprise group.
- Entity boundary: the default isolation boundary for users, identity sources, departments, roles, resource scopes, sessions, audit logs, and synchronization jobs.
- Application: a shared or entity-owned system that may be accessed by users from one or more entities when explicit cross-entity authorization allows it.
- Enterprise administrator: manages the enterprise control plane through an entity URL. This role covers entity management, global configuration, and system-wide application/integration policy in the narrowed product scope.

Rules:

- Entities are isolated by default.
- Every entity-scoped table must include `entity_id`.
- APIs, authorization checks, sessions, audit logs, and sync jobs must respect entity boundaries by default.
- Application definitions may be global or entity-owned, but application access decisions can intentionally cross entity boundaries.
- Cross-entity access must be explicit through application assignments, RBAC/resource scopes, or a dedicated cross-entity access policy.
- A user from Entity A must not see or mutate Entity B data unless the request is an application access flow that has explicit cross-entity authorization.
- The first version does not need public entity signup, customer billing, customer plans, or external customer onboarding.
- New implementation does not need SaaS compatibility; future development should use business entity terminology and may rename existing tenant-shaped code/contracts.

## Architecture

IdBridge should start as a Go modular monolith.

```text
/internal
  /entity          Business entity boundaries and entity configuration
  /identity        Users, directory users, account bindings, departments
  /idp             Feishu/DingTalk OAuth login and directory sync
  /sso             OIDC provider, clients, tokens, sessions
  /rbac            Roles, permissions, resource scopes, Casbin enforcement
  /audit           Append-only audit logs
  /adminapi        Admin console APIs
  /worker          Sync jobs, scheduled jobs, async audit processing
```

Recommended infrastructure:

- Backend: Go
- HTTP router: Chi
- Database: PostgreSQL
- Cache/session support: Redis
- Data access: sqlc + pgx
- OAuth2/OIDC: ORY Fosite
- Authorization: Casbin
- Migrations: Goose or Atlas
- Logging: Zap
- API documentation: OpenAPI
- Admin UI: SvelteKit + Skeleton + Tailwind CSS

## I18n Baseline

Internationalization is a foundational requirement, not a later UI enhancement.

Defaults:

- Default locale: `en-US`
- Built-in locales from day one: `en-US`, `zh-CN`
- `zh-CN` is a default supported language shipped in the first version.
- Synced external user data must support Chinese names, department names, titles, and raw provider profile fields.
- API error responses must use stable machine-readable codes and localizable messages.
- Admin UI labels, menu names, permission names, validation messages, and audit action labels must be localizable.
- Business data that has display labels should support localized names when needed.

Recommended rules:

- Store stable codes in core fields, not translated text.
- Store translated display labels in explicit translation structures when needed.
- Keep audit `action` values stable and English-coded, such as `auth.login.success`.
- Render localized audit labels at the presentation layer.
- Preserve raw Feishu/DingTalk profiles as JSONB to avoid losing Chinese or provider-specific fields.
- Use UTF-8 everywhere.

Example localizable model:

```text
permissions
- id
- entity_id
- code              api:user.read
- type              api / menu / action / data
- default_name      User Read

permission_translations
- permission_id
- locale            en-US / zh-CN
- name
- description
```

For MVP, not every table needs a translation table. The required i18n baseline is:

- System messages are localizable.
- Admin UI is localizable.
- Synced Chinese directory data is stored correctly.
- Permission/menu/action display names can be localized.
- Audit actions use stable codes and can be localized in UI.

## Identity Model

IdBridge separates external directory identities from managed business users.

```text
Identity Source  External identity provider, such as Feishu or DingTalk
Directory User   User record synced from the external directory
Managed User     IdBridge user that can receive roles and application access
```

Core principles:

- Feishu/DingTalk users are first synced into `directory_users`.
- Full sync can automatically create managed `users` for all synced Feishu people.
- A managed user existing in IdBridge does not imply access to any business application.
- Only managed `users` can receive roles, permissions, and application access.
- `directory_users` do not directly receive business permissions.
- One managed user can bind multiple external identities.
- One external identity can bind to only one managed user.
- External directories answer "who is this person?"
- IdBridge answers "is this person one of our managed users, and what can they access?"

## Core Tables

Business entity and identity source:

```text
business_entities
- id
- name
- slug
- status
- default_locale
- created_at

identity_sources
- id
- entity_id
- type              feishu / dingtalk / wecom / ldap / local
- name
- config_encrypted
- status
- sync_enabled
- created_at
```

Directory data:

```text
directory_users
- id
- entity_id
- source_id
- external_user_id
- external_union_id
- external_open_id
- name
- email
- phone
- avatar_url
- status            active / disabled / deleted / unknown
- raw_profile
- last_synced_at

directory_departments
- id
- entity_id
- source_id
- external_department_id
- parent_external_department_id
- name
- raw_profile
- last_synced_at
```

Managed users and bindings:

```text
users
- id
- entity_id
- username
- display_name
- email
- phone
- avatar_url
- lifecycle_status  active / disabled / locked / deleted
- user_type         employee / contractor / service_account
- primary_source_id nullable
- locale            nullable; falls back to entity default locale
- created_at

account_bindings
- id
- entity_id
- user_id
- source_id
- directory_user_id
- provider_uid
- provider_union_id
- is_primary
- bound_at
```

Organization and groups:

```text
organizations
- id
- entity_id
- name
- parent_id

departments
- id
- entity_id
- organization_id
- name
- parent_id
- source_id nullable
- external_department_id nullable

groups
- id
- entity_id
- name
- type              manual / synced
```

SSO:

```text
applications
- id
- owner_entity_id nullable; null means group-level shared application
- name
- type              oidc_client / api_client / internal_app
- status
- cross_entity_mode same_entity_only / allow_policy / global_shared

oidc_clients
- id
- application_id
- client_id
- client_secret_hash
- redirect_uris
- allowed_scopes
- grant_types
- pkce_required
- status

sessions
- id
- entity_id
- application_id nullable
- user_id
- device_id
- ip
- user_agent
- login_method      feishu / dingtalk / password / token
- status            active / revoked / expired
- created_at
- expires_at

oauth_tokens
- id
- entity_id
- application_id
- user_id
- client_id
- token_type        access / refresh / id
- token_hash
- scopes
- revoked_at
- expires_at
```

Authorization:

```text
roles
- id
- entity_id
- name
- code
- description

permissions
- id
- entity_id
- code              api:user.read / menu:users / action:user.disable
- name
- type              api / menu / action / data

role_permissions
- role_id
- permission_id

user_roles
- user_id
- role_id

resource_scopes
- id
- entity_id
- type              store / warehouse / country / brand
- key
- name

role_resource_scopes
- role_id
- resource_scope_id
- effect            allow / deny

application_assignments
- id
- application_id
- subject_entity_id
- subject_type      user / group / department / role
- subject_id
- effect            allow / deny
- created_at

cross_entity_application_policies
- id
- application_id
- source_entity_id
- target_entity_id
- principal_type    user / group / department / role / all_users
- principal_id nullable
- effect            allow / deny
- created_at
```

Application access rule:

- Entity-scoped administration can manage applications owned by that entity, and enterprise administrators may also manage group-level shared applications and global access policy.
- A global shared application can be visible to multiple entities, but access still requires assignment or cross-entity policy.
- A user in Entity A can access an application owned by Entity B only when `cross_entity_application_policies` or equivalent RBAC/resource-scope policy explicitly allows that crossing.
- Cross-entity application access must not grant general access to Entity B users, departments, roles, identity sources, or audit logs.

Audit:

```text
audit_logs
- id
- entity_id
- actor_user_id nullable
- actor_type        user / system / sync_job / api_client
- action
- resource_type
- resource_id
- before
- after
- ip
- user_agent
- trace_id
- created_at
```

## Provisioning And Access Rules

Full sync rules:

- Sync all Feishu departments and users.
- Upsert every synced person into `directory_users`.
- Automatically create or update managed `users`.
- Automatically create `account_bindings`.
- Default managed user lifecycle status: `active`.
- Default business application access: none.
- If the external directory user is disabled or resigned, disable or lock the managed user according to entity policy.
- Never physically delete managed users during sync.

Default policy:

```text
sync_all_directory_users = true
auto_create_managed_users = true
default_user_lifecycle_status = active
default_application_access = none
disable_user_when_directory_disabled = true
delete_user_when_directory_deleted = false
```

Application access is separate from user existence:

- A user can log into IdBridge without having access to ERP, BI, WMS, or other applications.
- A user can access an application only when assignment or role rules allow it.
- OIDC authorization must check application access before issuing tokens for the target client.

## Feishu Login Flow

```text
1. User clicks "Sign in with Feishu".
2. IdBridge redirects to Feishu OAuth.
3. Feishu callback returns the external identity.
4. IdBridge finds or creates the directory user.
5. IdBridge finds or creates the managed user according to policy.
6. IdBridge creates or refreshes the account binding.
7. IdBridge checks user lifecycle status.
8. IdBridge creates a session.
9. IdBridge continues the OIDC authorization flow if the login was initiated by an application.
10. IdBridge writes login audit logs.
```

If the user is logging into a business application, IdBridge must also check application access before signing tokens.

## Feishu Full Sync Flow

```text
1. Admin or worker triggers full sync.
2. Worker fetches Feishu departments.
3. Worker fetches Feishu users.
4. Worker upserts directory_departments.
5. Worker upserts directory_users.
6. Worker creates or updates managed users.
7. Worker creates or updates account bindings.
8. Worker maps departments where configured.
9. Worker disables or locks users whose source identity is disabled, according to policy.
10. Worker writes sync job audit and per-user audit events.
```

Failure behavior:

- Feishu API failure marks the sync job as failed.
- Successfully processed batches are not rolled back.
- Sync errors must be visible in admin UI.
- Sync jobs must include trace IDs and structured logs.

## OIDC Flow

The first version supports Authorization Code + PKCE.

```text
1. Internal application redirects to /oauth2/authorize.
2. IdBridge validates client_id, redirect_uri, response_type, scope, and PKCE challenge.
3. IdBridge checks whether the user has an active session.
4. If no session exists, IdBridge starts login.
5. If a session exists, IdBridge checks application access.
6. If access is denied, IdBridge rejects authorization and writes audit logs.
7. If access is allowed, IdBridge issues an authorization code.
8. Client exchanges code at /oauth2/token.
9. IdBridge signs ID token and access token.
10. Client establishes its own application session.
```

## API Boundaries

```text
/admin/v1/*       Admin console APIs
/oauth2/*         OIDC/OAuth2 protocol endpoints
/internal/v1/*    Internal application authorization, permission query, audit ingestion
```

## Application Integration Matrix (MVP)

V1 支持三类接入方式：

1. 标准 OIDC 应用（推荐）
   - 接入条件：`applications.type = oidc_client`
   - 协议：Authorization Code + PKCE
   - 认证目标：统一 IdBridge 会话与权限上下文
   - 典型场景：ERP、BI、WMS、CRM、运维、审批类系统与内部业务平台

2. API/M2M 客户端（后续逐步扩展）
   - 接入条件：`applications.type = api_client`
   - 用途：服务间调用、任务系统、网关服务
   - MVP 行为：先保持受控能力边界，仅支持最小化 client-level 调用能力

3. 遗留应用桥接（你提到的“老旧项目”）
   - 接入条件：`applications.type = internal_app` 或专门 legacy 桥接类型
   - 协议：本地用户名+密码由桥接器校验，成功后映射到受管用户
   - 核心要求：不允许创建“影子用户”；所有授权仍从 `users` / `application_assignments` 生效

## Standard OIDC Integration Contract (MVP)

Onboard 时必须提供：

- application 元信息（所属实体、名称、状态、类型）
- oidc_client 元信息（client_id、redirect_uris、allowed_scopes、grant_types）
- `pkce_required=true`
- `client_secret_hash`（MVP 可由运维先置空后补充）

运行时检查：

- application + client 必须 active
- redirect_uri 完整匹配白名单
- scope 为允许集合子集
- PKCE 校验通过
- 登录用户 session 已激活且生命周期为 active
- 应用访问策略允许该主体访问该应用

输出要求：

- token issuance 包含 `id_token` 与最小 `access_token`
- access token 包含 entity_id、aud、scope、sid 等最小上下文
- 不返回任何大体量权限列表（按钮级/菜单级/商家清单）

边界约束：

- 不在 MVP 期间实现 provider 侧撤销发现接口（可保留 501 语义）
- OIDC client 变更与停用必须写审计，禁用后拒绝登录、可根据策略回收会话

## Legacy Password-Mapped Login Bridge

对于仍然只支持用户名密码登录的老旧应用，新增统一桥接模型：

- 应用端提交本地用户名/密码到 IdBridge 桥接入口
- IdBridge 在 `legacy_app_users` 中按 `(entity_id, application_id, username)` 查找映射
- 成功后映射到受管 `user_id`，复用同一套会话与应用访问授权判定
- 失败仅返回本应用内登录失败，不回退到 OIDC 路径

建议新增字段：

- `legacy_app_users`
  - `id`
  - `entity_id`
  - `application_id`
  - `username`（应用内账号）
  - `legacy_user_identifier`（可存储外部系统 ID）
  - `auth_scheme`（`local`/`ldap`/`external_hash`）
  - `credential_ref`（加盐哈希或外部凭证引用）
  - `is_active`
  - `last_used_at`
  - `created_at`
  - `updated_at`

- `legacy_password_events`
  - `id`
  - `entity_id`
  - `application_id`
  - `user_id`
  - `username`
  - `event`（`success`/`failed`/`locked`/`disabled`）
  - `client_ip`
  - `user_agent`
  - `trace_id`
  - `occurred_at`

行为约束：

- 同一应用内同名用户唯一
- 映射失败不允许创建新用户
- 达到失败阈值后触发锁定（支持实体级配置）
- 成功登录仍必须通过应用权限检查，否则拒发应用令牌或会话票据

Important endpoints:

```text
/admin/v1/users
/admin/v1/directory-users
/admin/v1/identity-sources
/admin/v1/applications
/admin/v1/roles
/admin/v1/permissions
/admin/v1/resource-scopes
/admin/v1/audit-logs
/admin/v1/sync-jobs

/oauth2/authorize
/oauth2/token
/oauth2/userinfo
/oauth2/revoke
/.well-known/openid-configuration
/.well-known/jwks.json

/internal/v1/introspect
/internal/v1/permissions/check
/internal/v1/users/{id}/access
/internal/v1/audit-events
```

## Token Claims

`id_token` should contain identity claims, not large permission payloads.

```json
{
  "iss": "https://idbridge.example.com",
  "sub": "user_01H...",
  "aud": "client_erp",
  "entity_id": "entity_01H...",
  "name": "Zhang San",
  "email": "zhangsan@example.com",
  "phone_number": "138****0000",
  "picture": "https://...",
  "preferred_username": "zhangsan",
  "locale": "zh-CN",
  "identity_sources": ["feishu"],
  "session_id": "sess_01H..."
}
```

`access_token` should contain only minimal authorization context.

```json
{
  "iss": "https://idbridge.example.com",
  "sub": "user_01H...",
  "aud": "client_erp",
  "entity_id": "entity_01H...",
  "scope": "openid profile email",
  "roles": ["erp_operator"],
  "permissions_version": 12,
  "resource_scopes_version": 5,
  "sid": "sess_01H..."
}
```

Do not put full menu permissions, button permissions, store lists, warehouse lists, or country lists directly into JWTs. Applications that need fine-grained checks should call internal permission APIs.

## Permission Model

Permission types:

```text
api       api:user.read / api:order.export
menu      menu:erp.orders / menu:admin.users
action    action:user.disable / action:order.refund
data      data:store / data:warehouse / data:country / data:brand
```

Relationships:

```text
Role -> Permission
Role -> ResourceScope
User/Group/Department -> Role
Application -> Assignment
```

Authorization order:

```text
1. User lifecycle_status is active.
2. Application is active.
3. Client redirect_uri and scope are valid.
4. Application assignment allows access.
5. Explicit deny does not match.
6. Role permission allows the operation.
7. Resource scope allows the target business resource.
```

The first version uses explicit allow/deny and Casbin enforcement. It does not include a full policy language. A future policy engine can be inserted after role and resource checks.

## Cross-Border Ecommerce Resource Scopes

The MVP must include minimal data permission primitives for common cross-border ecommerce resources:

- Store
- Warehouse
- Country
- Brand

Examples:

```text
store:shopify-us-001
store:amazon-de-001
warehouse:cn-sh-01
country:US
brand:acme
```

These scopes are assigned to roles and checked by applications through the permission API.

## Audit Strategy

Audit logs are append-only.

Required audit actions:

```text
auth.login.success
auth.login.failed
auth.logout
auth.token.revoke
sso.authorize.success
sso.authorize.denied

sync.feishu.started
sync.feishu.finished
sync.feishu.failed
sync.user.created
sync.user.disabled
sync.department.updated

user.updated
user.disabled
user.bound_identity
user.unbound_identity

role.created
role.updated
role.permission_changed
application.assignment_changed

application.created
oidc_client.updated
secret.rotated

legacy.auth.success
legacy.auth.failed
legacy.auth.locked
legacy_mapping.created
legacy_mapping.updated
legacy_mapping.disabled
application.integration.onboarded
application.integration.updated
```

Rules:

- All admin changes must include `before` and `after`.
- System tasks use `actor_type = sync_job` or `system`.
- Login events include IP, user agent, session ID, and trace ID.
- Secrets and tokens are never stored in audit logs.
- Phone numbers and sensitive profile fields should be masked or hashed where appropriate.
- Audit action codes remain stable and are localized in UI.
- PostgreSQL JSONB is enough for MVP; OpenSearch or ClickHouse can be added later.

## Error Handling

- Feishu API failure: mark the sync job failed and keep processed batches.
- OIDC client config error: reject authorize/token request and audit the denial.
- No application access: allow IdBridge login but deny token issuance for the target application.
- Permission check failure: default deny.
- Redis unavailable: fail session/token operations rather than bypassing security.
- Critical admin audit write failure: fail the admin operation.
- Login audit write failure: implementation may use an internal degraded queue, but it must not silently disappear.

## Testing Strategy

Unit tests:

- Feishu user to directory user mapping.
- Directory user to managed user provisioning.
- Account binding uniqueness.
- Application assignment allow/deny.
- Role permission enforcement.
- Resource scope enforcement.
- Locale fallback and localized message selection.

Integration tests:

- OIDC Authorization Code + PKCE.
- Feishu full sync with mocked provider API.
- Session revoke.
- Token revoke.
- Application access denial during authorize.
- Audit before/after for admin changes.

Contract tests:

- `/.well-known/openid-configuration`
- `/.well-known/jwks.json`
- `/oauth2/token`
- `/oauth2/userinfo`
- `/internal/v1/permissions/check`

## Future Phases

Phase 2:

- DingTalk directory sync.
- WeCom directory sync.
- LDAP/AD sync.
- Incremental sync and webhook sync.
- SCIM client support.
- More complete admin UI.

Phase 3:

- SAML.
- SCIM server.
- MFA with TOTP and email code.
- WebAuthn/Passkey.
- Policy engine.
- API Gateway integration.
- AI risk audit and anomaly detection.
- Cross-entity policy simulation and impact analysis.
- Cross-entity application access approval workflow.

## Open Implementation Notes

- Use `en-US` as default locale in seed data and entity defaults.
- Ship `en-US` and `zh-CN` translations for admin UI and built-in system resources in the first version.
- Prefer stable IDs and stable codes over mutable display names.
- Never treat external directory membership as business application access.
- Never delete managed users from directory sync.
- Keep business authorization decisions inside IdBridge, not inside Feishu or DingTalk.
- Default every entity boundary to isolated.
- Treat cross-entity application access as an explicit exception, never as inherited visibility.
- Do not design public SaaS customer onboarding, billing, or plan enforcement for this product line.
