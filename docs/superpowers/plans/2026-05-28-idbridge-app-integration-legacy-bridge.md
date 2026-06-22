# IdBridge Application Integration & Legacy Bridge Plan

> Superseded direction: business boundary semantics are now governed by [2026-06-02-business-entity-boundary-replan.md](2026-06-02-business-entity-boundary-replan.md). Future implementation should use business entity terminology and `business_entities` / `entity_id`, not SaaS tenant semantics. No compatibility layer is required.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the v1 application onboarding gap by formalizing standard app onboarding, OIDC hardening constraints, and legacy username/password mapped login bridge for non-OIDC applications.

**Architecture:** Keep the Go modular monolith. Add `internal/integration` for application onboarding/audit models and `internal/legacy` for legacy authentication bridge services. Reuse existing `internal/sso` token/session checks so legacy-authenticated principals obtain the same application access decision behavior as OIDC principals.

**Tech Stack:** Go 1.22+, Chi, PostgreSQL, sqlc + pgx, Goose, built-in Go tests, Testcontainers PostgreSQL integration tests.

## Scope Boundary

In scope:

- App onboarding contract for `applications`, `oidc_clients`, and onboarding audit.
- Legacy bridge domain model and SQL queries.
- Legacy bridge login endpoint/service with failure policy and lockout.
- Mapping management APIs for admin operations.
- Integration tests covering map lookup, success/failure, lockout, and access-denied path.
- Audit action coverage for legacy auth and mapping lifecycle.

Out of scope:

- Full LDAP/AD protocol adapter (placeholders only).
- API client token exchange extension (handled in later identity gateway milestone).
- MFA for legacy auth (add in security hardening milestone).

## Concurrent Execution Protocol (for 2+ AI)

- AI-Owner 标签建议（可直接用于提交信息）：
  - `owner:a`：Task 1 / Task 2
  - `owner:b`：Task 3 / Task 4 / Task 5 / Task 6 的 service/handler
  - `owner:c`：冲突调解与文档/对齐（我）
- 每次提交前必须包含一句“本提交影响范围 + 与其他 task 的依赖关系”。
- 不要跨 owner 修改重叠文件；若必须跨越，先在 plan 里提 `owner:coord` 的同步说明。

## Task 1 Handoff (Owner A)

Use this as the first execution contract when 你们并发运行 Task 1。

- Deliverable: `migrations/000004_legacy_app_integration.sql` 已生效并可回滚。
- Required checks in PR description:
  - 外键 `entity_id + application_id + user_id` 与 `users/tenants/applications` 的级联规则完整。
  - `legacy_user_identifier` 唯一约束不会导致误杀（建议允许 NULL 时多对一语义可接受）。
  - `legacy_app_users` 和 `legacy_password_events` 都带了 tenant 边界。
- Suggested commit message: `feat: add legacy integration schema`

## Task 2 Handoff (Owner A)

- Deliverable: `internal/db/queries/legacy.sql` 与 `internal/db/generated/*`。
- Validation checklist:
  - `GetLegacyAppUser` 用 tenant + application + username 查询，不能只按 username。
  - `UpsertLegacyAppUser` 包含 `legacy_user_identifier` 与 `is_active` 的更新逻辑。
  - `CreateLegacyPasswordEvent` 覆盖成功/失败/锁定的持久化。
  - 失败次数统计窗口参数来自上层传入时间参数（避免硬编码）。

### Owner A 快速协作规则（关键）

- Owner A 只提交 migration 和 queries 两个 patch，不改 service/handler。
- 如需变更字段名，先在该计划文档注释一条并请其他 Owner 确认。

## Task 3 Handoff (Owner B)

Deliverable: 应用接入合同实现（`internal/integration/*`）。

- 范围与目标：
  - 标准化 OIDC / API / legacy 的应用创建/更新规则。
  - 统一 `legacy_mode` 与 `app type` 的约束，不允许产生“模糊入口”。

- 验收要点：
  - `internal/integration/model.go` 中的枚举、校验、DTO 与错误码稳定。
  - `internal/integration/service.go` 能区分 `standard_oidc`、`api_client`、`legacy_bridge`。
  - 审计事件包含：`application.integration.onboarded`、`application.integration.updated`。

- 依赖约束：不得修改 `internal/db/queries/*`（由 Owner A 维护）。

- 建议提交信息：`feat: define application onboarding contract`

## Task 4 Handoff (Owner B)

Deliverable: 遗留应用映射与鉴权桥接（`internal/legacy/*`）。

- 范围与目标：
  - 映射查找、凭据校验（先支持 `local`）、失败计数、锁定策略。
  - 成功/失败事件落库（`legacy_password_events`）。

- 验收要点：
  - 查询映射必须带 `entity_id + application_id + username`。
  - 映射不存在或未激活统一返回不泄露存在性错误。
  - 达到阈值产生 `legacy.auth.locked`。
  - 成功鉴权仍调用应用访问策略分支（与 OIDC 共享决策）。

- 依赖约束：不得改 `internal/integration/*` 核心模型。

- 建议提交信息：`feat: add legacy password-mapped auth bridge`

## Task 5 Handoff (Owner B)

Deliverable: 入口与最小集成测试（`internal/adminapi/*`、`tests/integration/legacy_auth_test.go`）。

- 范围与目标：
  - admin 管理映射 CRUD 与事件查询。
  - legacy 外部调用入口（如 `POST /login/legacy`）返回稳定结构。
  - 集成测试覆盖成功/失败/锁定。

- 验收要点：
  - 入口返回码统一：`legacy_auth_failed`、`legacy_auth_locked`、`legacy_auth_success`。
  - `tests/integration/legacy_auth_test.go` 完整覆盖：映射成功、无映射失败、锁定阻断。

- 依赖约束：
  - 只在现有 OIDC 路径中做接入，不新增重复权限决策。
  - 遵循现有 admin auth 中间件与错误响应约定。

- 建议提交信息：`feat: add legacy mapping admin endpoints and legacy auth e2e`

## Task 6 Handoff (Owner B)

Deliverable: OIDC/legacy 统一应用授权决策链路（`internal/httpserver/*`、`internal/sso/*`）。

- 验收要点：
  - legacy 登录成功后调用与 OIDC 相同的应用授权判断。
  - legacy 与 OIDC 的审计动作码均可落审计。
  - 路由新增不影响既有 `/oauth2/*` 与 `/admin/v1/*` 健壮性。

- 依赖约束：若与既有 router 入口冲突，先发起 `owner:coord` 协调，再实现。

- 建议提交信息：`feat: align legacy auth with oidc authorization policy`

## Conflict log template（每次同步前可贴在本页最前）

- [ ] 时间：`YYYY-MM-DD HH:MM`
- [ ] Owner A commit：`<hash>`
- [ ] Owner B commit：`<hash>`
- [ ] 冲突文件：`<file>`（有/无）
- [ ] 冲突原因：`<reason>`
- [ ] 解决规则：按 P0/P1/P2/P3 顺序裁决
- [ ] 最终决定：`<merge/holdoff/rename-file/>`

## File Structure

Create or modify:

```text
migrations/000004_legacy_app_integration.sql
internal/db/queries/legacy.sql
internal/db/generated/*
internal/integration/model.go
internal/integration/queries.go
internal/integration/service.go
internal/integration/service_test.go
internal/legacy/model.go
internal/legacy/store.go
internal/legacy/auth.go
internal/legacy/auth_test.go
internal/adminapi/integration_handlers.go
internal/adminapi/integration_handlers_test.go
internal/httpserver/router.go
tests/integration/legacy_auth_test.go
docs/superpowers/specs/2026-05-26-idbridge-design.md
``` 

## Task 1: Add Legacy Integration Schema

**Files:**
- Create: `migrations/000004_legacy_app_integration.sql`

- [ ] **Step 1: Write migration**

Create tables:

```sql
-- +goose Up
CREATE TABLE legacy_app_users (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid(),
    entity_id CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    application_id CHAR(26) NOT NULL,
    user_id CHAR(26) NOT NULL,
    username TEXT NOT NULL,
    legacy_user_identifier TEXT,
    auth_scheme TEXT NOT NULL CHECK (auth_scheme IN ('local', 'ldap', 'external_hash')),
    credential_ref TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, application_id, username),
    UNIQUE (entity_id, application_id, legacy_user_identifier),
    FOREIGN KEY (entity_id, application_id) REFERENCES applications(entity_id, id) ON DELETE CASCADE,
    FOREIGN KEY (entity_id, user_id) REFERENCES users(entity_id, id) ON DELETE CASCADE
);

CREATE TABLE legacy_password_events (
    id CHAR(26) PRIMARY KEY DEFAULT idb_generate_ulid(),
    entity_id CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    application_id CHAR(26) NOT NULL,
    user_id CHAR(26),
    username TEXT,
    event TEXT NOT NULL CHECK (event IN ('success', 'failed', 'locked', 'disabled')),
    client_ip TEXT,
    user_agent TEXT,
    trace_id TEXT,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (entity_id, id),
    FOREIGN KEY (entity_id, application_id) REFERENCES applications(entity_id, id) ON DELETE CASCADE,
    FOREIGN KEY (entity_id, user_id) REFERENCES users(entity_id, id) ON DELETE CASCADE
);

CREATE INDEX idx_legacy_app_users_tenant_app ON legacy_app_users(entity_id, application_id);
CREATE INDEX idx_legacy_password_events_tenant_app_user ON legacy_password_events(entity_id, application_id, username);

-- +goose Down
DROP TABLE IF EXISTS legacy_password_events;
DROP TABLE IF EXISTS legacy_app_users;
```

- [ ] **Step 2: Migrate and commit**

```bash
rtk go run github.com/pressly/goose/v3/cmd/goose@v3.22.1 -dir migrations validate
rtk git add migrations/000004_legacy_app_integration.sql
rtk git commit -m "feat: add legacy integration schema"
```

## Task 2: Add Legacy SQL Queries And Generate

**Files:**
- Create: `internal/db/queries/legacy.sql`
- Modify: `internal/db/generated/*`

- [ ] **Step 1: Add queries**

Create `internal/db/queries/legacy.sql`.

```sql
-- name: UpsertLegacyAppUser :one
INSERT INTO legacy_app_users (
    entity_id,
    application_id,
    user_id,
    username,
    legacy_user_identifier,
    auth_scheme,
    credential_ref,
    is_active
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
ON CONFLICT (entity_id, application_id, username)
DO UPDATE SET
    user_id = EXCLUDED.user_id,
    legacy_user_identifier = EXCLUDED.legacy_user_identifier,
    auth_scheme = EXCLUDED.auth_scheme,
    credential_ref = EXCLUDED.credential_ref,
    is_active = EXCLUDED.is_active,
    updated_at = now()
RETURNING id, entity_id, application_id, user_id, username, legacy_user_identifier, auth_scheme, credential_ref, is_active, last_used_at, created_at, updated_at;

-- name: GetLegacyAppUser :one
SELECT id, entity_id, application_id, user_id, username, legacy_user_identifier, auth_scheme, credential_ref, is_active, last_used_at, created_at, updated_at
FROM legacy_app_users
WHERE entity_id = $1 AND application_id = $2 AND username = $3;

-- name: ListLegacyAppUsersByApplication :many
SELECT id, entity_id, application_id, user_id, username, legacy_user_identifier, auth_scheme, is_active, last_used_at, created_at, updated_at
FROM legacy_app_users
WHERE entity_id = $1 AND application_id = $2
ORDER BY updated_at DESC
LIMIT $3;

-- name: TouchLegacyAppUserUsedAt :exec
UPDATE legacy_app_users
SET last_used_at = now()
WHERE entity_id = $1 AND id = $2;

-- name: SetLegacyAppUserStatus :exec
UPDATE legacy_app_users
SET is_active = $4, updated_at = now()
WHERE entity_id = $1 AND application_id = $2 AND username = $3;

-- name: CreateLegacyPasswordEvent :exec
INSERT INTO legacy_password_events (
    entity_id,
    application_id,
    user_id,
    username,
    event,
    client_ip,
    user_agent,
    trace_id
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: CountLegacyPasswordFailures :one
SELECT COUNT(*)
FROM legacy_password_events
WHERE entity_id = $1
  AND application_id = $2
  AND username = $3
  AND event = 'failed'
  AND occurred_at > $4;
```

- [ ] **Step 2: Generate code**

```bash
rtk make generate
```

- [ ] **Step 3: Commit**

```bash
rtk git add internal/db/queries/legacy.sql internal/db/generated
rtk git commit -m "feat: add legacy integration queries"
```

## Task 3: Standardize Application Onboarding Contract

**Files:**
- Create: `internal/integration/model.go`
- Create: `internal/integration/queries.go`
- Create: `internal/integration/service.go`
- Create: `internal/integration/service_test.go`

- [ ] **Step 1: Add domain model helpers**

Implement application contract checks for onboarding and mutation:

- `ApplicationType` enum values (`oidc_client`, `api_client`, `internal_app`)
- `OnboardingType` (`standard_oidc`, `api_client`, `legacy_bridge`)
- `ApplicationIntegrationSpec` with required/optional fields
- Validate at create/update time:
  - name non-empty and entity-scoped unique
  - type constraints
  - `oidc_client` requires `client_id`, `redirect_uris`, `allowed_scopes`, `grant_types`
  - `internal_app` legacy requires `legacy_mode_enabled`

- [ ] **Step 2: Add service behavior**

Service functions:

- `CreateOIDCApplication(ctx, tenantID, req)`
- `CreateAPIClientApplication(ctx, tenantID, req)`
- `CreateLegacyBridgeApplication(ctx, tenantID, req)`
- `SetApplicationStatus(ctx, tenantID, applicationID, status)`
- `UpdateApplicationIntegration(ctx, tenantID, applicationID, patch)`

Rules:

- Every create/update writes audit actions:
  - `application.integration.onboarded`
  - `application.integration.updated`
- `oidc_client_updated` (alias use `oidc_client.updated`) when OIDC fields change

- [ ] **Step 3: Add admin API endpoint tests**

At least:

- onboarding input validation for each application type
- duplicate redirect URI and invalid OIDC config rejected
- legacy app can be onboarded only with explicit legacy mode
- status transition active/disabled updates effective immediately

- [ ] **Step 4: Commit**

```bash
rtk git add internal/integration
rtk git commit -m "feat: define application onboarding contract"
```

## Task 4: Implement Legacy Authentication Bridge

**Files:**
- Create: `internal/legacy/model.go`
- Create: `internal/legacy/store.go`
- Create: `internal/legacy/auth.go`
- Create: `internal/legacy/auth_test.go`
- Modify: `internal/adminapi/integration_handlers.go`
- Modify: `internal/adminapi/integration_handlers_test.go`

- [ ] **Step 1: Add auth service**

Implement `legacy/auth.go`:

```go
func AuthenticateLegacyUser(ctx context.Context, tenantID ulid.ULID, applicationID ulid.ULID, username, password string, reqMeta AuthRequestMeta) (AuthResult, error)
```

Rules:

- Authenticate by mapping lookup `(entity_id, application_id, username)`
- if mapping not found -> `failed` event and generic error
- if mapping inactive/disabled -> `disabled` event
- verify credential by `auth_scheme` (initially only `local` salted-hash)
- success -> mark last used, emit `success` event
- failure counter window (e.g. last 10 min > threshold) -> emit `locked`, reject login
- success still executes application access checks before returning identity result

- [ ] **Step 2: Add admin management handlers**

Add endpoint ideas:

- `POST /admin/v1/applications/{id}/legacy-mappings`
- `PATCH /admin/v1/applications/{id}/legacy-mappings/{mapping_id}`
- `GET /admin/v1/applications/{id}/legacy-mappings`
- `GET /admin/v1/applications/{id}/legacy-mappings/{mapping_id}/events`

All admin responses should be stable-code JSON with machine-readable `code` values.

- [ ] **Step 3: Add security controls**

- password hash algorithm: bcrypt with cost config
- no password/logins returned in responses
- do not disclose whether user exists when auth fails (统一错误码)
- optional configurable lockout threshold per tenant (default 5/10 minutes)

- [ ] **Step 4: Tests**

Test cases:

- valid mapping success
- invalid password failure
- mapping inactive
- too many failures -> locked
- success path still denied by app access policy
- locked account unlocks after window expires

- [ ] **Step 5: Commit**

```bash
rtk git add internal/legacy internal/adminapi
rtk git commit -m "feat: add legacy password-mapped auth bridge"
```

## Task 5: Add Legacy Authorization Entry Endpoint And Integration Test

**Files:**
- Create: `tests/integration/legacy_auth_test.go`

- [ ] **Step 1: Add API endpoint**

Implement one legacy auth exchange endpoint for external apps (path to finalize by app owner):

- `POST /login/legacy` (or `POST /oauth2/legacy/authorize` if you need OIDC-compatible semantics)

Request contract example:

```json
{
  "entity_id": "ulid",
  "application_id": "ulid",
  "username": "legacy_user",
  "password": "***"
}
```

Success returns:

- mapped managed `user_id`
- `session_id`
- `authorized_applications`

Failure returns stable `code` (`legacy_auth_failed` / `legacy_auth_locked`).

- [ ] **Step 2: Integration test**

- create one entity, one legacy app, one managed user, one mapping
- verify bad credential triggers event + failure
- verify success returns session and app access context
- verify locked state
- verify mapping failure cannot login even with existing managed user id

- [ ] **Step 3: Commit**

```bash
rtk git add tests/integration/legacy_auth_test.go
rtk git commit -m "test: cover legacy password bridge auth flow"
```

## Task 6: Link With OIDC/Session Access Path

**Files:**
- Modify: `internal/httpserver/router.go`
- Modify: `internal/sso/service.go`
- Modify: `internal/sso/handlers.go`

- [ ] **Step 1: Ensure same access policy path**

Legacy success result must call the same application access check branch used by OIDC authorize decisions. Do not maintain duplicate policy logic.

- [ ] **Step 2: Unified audit codes**

Make sure legacy auth emits:

- `legacy.auth.success`
- `legacy.auth.failed`
- `legacy.auth.locked`
- `legacy_mapping.updated` / `legacy_mapping.created` where mapping state changes

- [ ] **Step 3: Commit**

```bash
rtk git add internal/httpserver internal/sso internal/legacy
rtk git commit -m "feat: align legacy auth with oidc authorization policy"
```

## Final Checklist

- [ ] 所有新增 schema 可回滚。
- [ ] legacy 与 OIDC 可并存，不互斥。
- [ ] 老旧应用不再依赖直接管理员本地凭证，全部通过映射桥接。
- [ ] 所有审计动作在设计中有稳定 code，并在管理变更/登录失败路径均可观测。

## Execution Order (suggested)

1. 先做 Task 1 + Task 2（数据库与 SQL）
   - 确保 `legacy_app_users`、`legacy_password_events` 的数据模型先可落地。
   - 这一步会影响后续 service/handler 的测试夹具。

2. 并行推进 Task 3（应用接入合同）与 Task 4（legacy 认证逻辑）
   - Task 3 负责入驻约束与审计语义。
   - Task 4 负责映射查找与认证流程。
   - 两者共享的模型与错误码要尽量一次性对齐。

3. 做 Task 5（对外桥接入口）+ Task 6（与 OIDC/应用访问策略对齐）
   - 将桥接入口挂到 API 边界（如 `/oauth2/legacy` 或 `/login/legacy`）。
   - 确认返回结构不会与 OIDC token 响应混淆。

4. 最后补 Task 5（集成测试）
   - 只在 Task 1~6 全量可运行后才跑，避免假阳性。

## 里程碑验收条件

- legacy 映射链路：`应用账号 -> 映射 -> 受管用户 -> 应用访问策略` 全链路可观测。
- 一致性规则：同应用同名账号唯一、未映射不可登录、锁定/失败有事件记录。
- OIDC 路径不受影响：OIDC 授权决策继续只由 `users` + `application_assignments` 生效。
- 审计可观测：登录成功/失败/锁定与 mapping 创建/更新都有稳定 action code。

## 风险与默认值建议（MVP）

- 默认密码方案：`local` + bcrypt，后续再接入 LDAP/外部 hash provider。
- 默认锁定策略：`5 次失败 / 10 分钟`（可落在租户配置中可配）。
- 返回错误码默认简化：
  - 失败统一返回 `legacy_auth_failed`（不要泄露账号是否存在）。
  - 触发锁定时返回 `legacy_auth_locked`。
- 默认映射入口建议仅支持管理员创建映射与密码初始化，不支持用户自助注册。

## 与现有计划的衔接

- OIDC 基础计划维持不变： [2026-05-27-idbridge-oidc-foundation.md](docs/superpowers/plans/2026-05-27-idbridge-oidc-foundation.md)
- 设计层补齐点已在：[2026-05-26-idbridge-design.md](docs/superpowers/specs/2026-05-26-idbridge-design.md)

## Multi-Agent Conflict-Handling (重要)

### 规则（执行前先读）

1. 文件拆分
   - 一人只改一个“文档职责域”：
     - 人 A：架构与数据库 Schema
     - 人 B：服务与鉴权逻辑
     - 人 C（我）：计划与接口边界
   - 禁止两个 AI 同时修改同一文件中的同一章节。

2. 任务领地（建议）
   - Task 1/2（migrations + sql）只由一人负责。
   - Task 3（应用接入合同）只改 `internal/integration/*`。
   - Task 4（legacy auth）只改 `internal/legacy/*`。
   - Task 5/6（入口与对齐）只改 `internal/adminapi/*`、`internal/httpserver/*`、`internal/sso/*`。
- Task 5（集成测试）只改 `tests/integration/*`。

3. 文件争用处理
   - 先检查未提交改动（`git status --short`）。
   - 若同一文件有冲突修改，优先保留：
     1) 现有功能正确性不退化
     2) 与 spec 一致性
     3) 新增功能最小化侵入
   - 必须同步更新：对应 plan 的任务清单状态、未完成项和交付说明。

4. 统一验收口径
   - 所有改动只能合并为：
     - 数据一致性（tenant/应用/映射）
     - 鉴权一致（OIDC 与 legacy 使用同一访问判定)
     - 日志可观测（legacy/auth/mapping action code）
   - 任何一步若未满足，延后后续任务，先补完依赖。

5. 冲突仲裁优先级（按时间递进）
   - P0：功能回归（不能破坏已有 OIDC 路径）
   - P1：租户隔离与映射正确性
   - P2：安全审计与锁定策略
   - P3：文案/命名和可观测性增强

### 我作为第三方的处理方式

- 我只接手没有明确 owner 的未冲突段，优先处理计划与对接规范文档。
- 若发现同文件已被其他 AI 变更且无冲突说明：我改最后追加页（append-only）并标注来源。
- 在确认你接受前，不做破坏式合并与回滚其他人的实现。
