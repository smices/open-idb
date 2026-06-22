# IdBridge 接入与集成指南

本文档按两个使用场景组织：  
1) 我去接入其他系统（如飞书、Google 等）  
2) 别的应用接入 IdBridge（OIDC / legacy 映射）

> 产品边界更新：IdBridge 的多实体边界不是对外售卖 SaaS 的客户租户模型，而是同一企业集团内的多业务实体模型。实体可以是总部、分公司、子公司、工厂、品牌或运营单元。实体之间默认隔离；应用访问可以通过显式授权或跨实体策略穿透。

> 基线：当前后端为 **Feishu** 提供了完整 OAuth 与目录同步链路；DingTalk/WeCom 在前端显示为规划占位，Google（目录类）未实现统一目录同步适配器。Google/Microsoft Entra 可先通过 OIDC 接入 IdBridge，再视场景补目录同步适配器。

## 快速导航（开发 / 接入 / 提交核验）

统一入口请见： [docs/quickstart-navigation.md](docs/quickstart-navigation.md)

## 一、我去接入其他系统（IdBridge 作为消费方）

### 1.1 接入飞书（已实现）

- 目标：  
  - 在登录页支持飞书登录（`/login`）  
  - 支持飞书目录/用户同步（`identity_sources` + `sync_jobs`）
- 前置：  
  - 先在飞书控制台创建企业自建应用，拿到 `app_id`、`app_secret`。
  - 对每个业务实体配置一个 `feishu` 的 IM Provider，并确保该实体存在可用的 Feishu 身份源。
  - 本项目不需要机器人 App ID / Secret；OAuth 登录与通讯录同步都使用同一套企业自建应用凭据。
- 可执行联调流程：
 1. 通过管理员接口写入飞书 IM 配置（`status=active`，`oauth_configured=true`，`sync_enabled` 按需）。
 2. `POST /api/admin/v1/identity-sources` 建 `type=feishu`（建议 `name=Feishu`）。
 3. 登录页带上 `entity_id` 打开：`/login?entity_id=<entity_id>`。
 4. 点击飞书二维码入口（登录成功后返回 `/?login_error` 不再出现）。
 5. 点击“触发全量同步”观察任务状态（`sync_jobs`）与目录用户落库。
6. 如需接入增量：在飞书应用里配置事件回调到
   `POST /api/webhooks/feishu/{entity_id}/{source_id}`，再通过
   `POST /api/admin/v1/identity-sources/{source_id}/sync/incremental` 触发一次增量窗口同步（通常用于周期性持续同步）。

- 配置接口（管理员侧）：

| 方法 | 路径 | 说明 |
|---|---|---|
  | `GET` | `/api/admin/v1/integrations/im` | 查询当前实体 IM Provider 配置 |
  | `PUT` | `/api/admin/v1/integrations/im/feishu` | 新增/更新飞书配置 |

- 配置 Body 示例（`feishu`）：
```json
{
  "display_name": "Feishu",
  "status": "active",
  "oauth_configured": true,
  "sync_enabled": true,
  "config": {
    "app_id": "cli_xxx",
    "app_secret": "xxxxxxxx"
  }
}
```

- 创建/确认 Identity Source（飞书）：
  - `POST /api/admin/v1/identity-sources`
  - 请求示例：`{"type":"feishu","name":"Feishu","sync_enabled":true}`
  - 可更新状态/同步开关：`PUT /api/admin/v1/identity-sources/{source_id}`

- 启动全量同步（后台异步）：
  - `POST /api/admin/v1/identity-sources/{source_id}/sync/full`
  - 返回：包含 `job_id` 的运行结果（字段见 `FullSyncResult`）

- Feishu 登录相关（供前端/回调用）：
  - `GET /api/admin/v1/auth/providers`：返回可用登录 provider（含 OAuth URL）
  - `GET /auth/feishu/login`：构造并重定向到飞书授权页
  - `GET /auth/feishu/callback`：OAuth 回调（code + state）
  - `POST /auth/feishu/exchange`：小程序/前端 AppCode 换取会话（`entity_id`, `auth_code`）

- 同步约束：
  - 后端 DB 约束目前支持 provider/source type 为 `feishu|dingtalk|wecom|ldap|local`
  - 同步实现当前仅支持 `feishu`
  - 增量建议：通过飞书应用事件回调驱动，接口 `/api/webhooks/feishu/{entity_id}/{source_id}` -> `sync/incremental`。

### 1.2 接入 Google（当前状态）

- 若你要接入 Google Workspace/Microsoft Entra 这类目录：
- 这类目录同步在当前版本未内置统一适配器，不能直接使用 `/api/admin/v1/identity-sources/{id}/sync/full` 执行增/全量同步。
- 可行接入路线：
  1. 通过 OIDC 方式让 Google/第三方应用用户登录到 IdBridge（见“2.1 OIDC 标准接入”）；
  2. 或新增 `idp` 适配器（`DirectoryProvider`），并在 sync 服务中增加 `provider="google"` 分支 + 配置模型。

#### 1.2.1 Google/OIDC 作为消费方接入 IdBridge 的提示

- IdBridge 对 Google 的目录同步无专用适配器；
- 但对接外部系统时，可优先走 OIDC：对端应用作为 OIDC Client 注册到 IdBridge 后，直接复用标准授权码。

### 1.3 安全与联调检查清单（对外接入）

- 统一 HTTPS + 回调 URL 白名单固定；
- 记录与轮换 `app_secret`；
- 同步任务建议监控 `sync_jobs` 状态；
- `sync_enabled=false` 时禁止触发或在外部系统层面忽略；
- 首次联调建议只读验证：登录 URL、`state` 校验、实体上下文透传。

## 二、外部应用接入 IdBridge（IdBridge 作为身份服务）

### 2.1 OIDC 标准接入（推荐）

#### 2.1.1 在控制台创建应用与 OIDC Client

1. 创建应用（`type` 常用：`oidc_client`）  
   - `POST /api/admin/v1/applications`
   - 示例：`{"name":"MyApp","type":"oidc_client"}`
2. 创建 OIDC Client  
   - `POST /api/admin/v1/oidc-clients`
   - 示例请求：
```json
{
  "application_id": "<APP_ID>",
  "client_id": "my-app",
  "redirect_uris": ["https://app.example.com/callback"],
  "allowed_scopes": ["openid", "profile", "email", "phone"],
  "grant_types": ["authorization_code"],
  "response_types": ["code"],
  "pkce_required": true
}
```
   - 成功返回 `client_secret`（建议只展示一次并立即保存）
3. 将客户端分配给用户访问范围（应用权限），并确保用户具备 application access（通过 RBAC/应用授权链）

跨实体应用访问：

- 默认只允许同实体访问。
- 如果应用需要被多个分公司/子公司访问，必须在应用授权或跨实体策略中显式配置。
- 跨实体访问只授予应用访问权，不授予目标实体的用户、组织、身份源、审计等管理数据访问权。

#### 2.1.2 标准 OAuth/OIDC 端点

- 发现文档：`GET /.well-known/openid-configuration`
- 授权：`GET /oauth2/authorize`
- 换 Token：`POST /oauth2/token`
- 用户信息：`GET /oauth2/userinfo`
- 回收 Token：`POST /oauth2/revoke`

#### 2.1.3 授权码流程（最小闭环）

1. 外部应用重定向到 `/oauth2/authorize`
2. 用户登录后 IdBridge 回跳 `redirect_uri?code=...&state=...`
3. 应用后端用 `client_id + client_secret` + `code` 调 `/oauth2/token` 换 `access_token/id_token`
4. 用 `access_token` 调 `/oauth2/userinfo` 读取用户属性（`sub/profile/email/...`）

#### 2.1.4 飞书工作台/卡片自动登录

适用场景：用户在飞书工作台点击应用图标，或在飞书卡片中点击入口，自动完成飞书上游登录后回到 OIDC Client。

##### OIDC Client 需要提供的入口

OIDC Client 至少需要提供两个 HTTP 入口：

| 路径 | 谁访问 | 作用 |
|---|---|---|
| `/sso/start` | 飞书应用图标/卡片入口、业务应用未登录拦截器 | 创建 OIDC 授权请求，并重定向到 IdBridge `/oauth2/authorize` |
| `/oidc/callback` | IdBridge | 接收 `code` 和 `state`，用后端换取 `id_token/access_token` 并建立业务应用自己的会话 |

飞书开放平台的“应用入口地址”建议配置为业务应用自己的启动地址，而不是 IdBridge 登录页：

```text
https://app.example.com/sso/start
```

这样 OIDC Client 可以统一处理 `state`、`nonce`、PKCE、业务侧 `next` 跳转和自身会话。

##### Client 启动登录

`/sso/start` 需要完成：

1. 生成并保存 `state`，用于回调校验 CSRF。
2. 生成并保存 `nonce`，用于校验 `id_token`。
3. 如果 OIDC Client 是前端/public client，生成 PKCE `code_verifier` 和 `code_challenge`；如果是服务端 confidential client，也建议保留 PKCE。
4. 重定向到 IdBridge `/oauth2/authorize`。从飞书入口进入且希望无感使用飞书时，追加 `idp=feishu`。

示例：

```text
https://idb.example.com/oauth2/authorize?response_type=code&client_id=my-app&redirect_uri=https%3A%2F%2Fapp.example.com%2Foidc%2Fcallback&scope=openid%20profile%20email&state=<state>&nonce=<nonce>&code_challenge=<challenge>&code_challenge_method=S256&idp=feishu
```

参数说明：

| 参数 | 必填 | 说明 |
|---|---:|---|
| `response_type=code` | 是 | 使用授权码模式 |
| `client_id` | 是 | 在 IdBridge 注册的 OIDC Client ID |
| `redirect_uri` | 是 | 必须与 IdBridge 注册值完全一致 |
| `scope` | 是 | 至少包含 `openid`，按需追加 `profile email phone` |
| `state` | 是 | Client 随机生成，回调必须校验 |
| `nonce` | 建议 | Client 随机生成，校验 `id_token.nonce` |
| `code_challenge` / `code_challenge_method` | 视配置 | `pkce_required=true` 时必填 |
| `idp=feishu` | 飞书入口建议 | 指示 IdBridge 优先/自动使用飞书上游登录 |

##### IdBridge 登录阶段行为

当授权请求中带有 `idp=feishu` 时，IdBridge 登录上下文会从 `return_to=/oauth2/authorize?...&idp=feishu` 中解析首选登录方式，并返回：

```json
{
  "preferred_provider": "feishu",
  "auto_redirect_url": "/api/auth/feishu/login?entity_id=<entity_id>&return_to=<encoded authorize url>"
}
```

登录页或 BFF 收到 `auto_redirect_url` 后可直接跳转到该 URL。飞书上游登录完成后，IdBridge 继续原始 OIDC 授权请求，并最终回跳 OIDC Client 的 `redirect_uri`。

##### Client 回调换 Token

`/oidc/callback` 收到：

```text
https://app.example.com/oidc/callback?code=<authorization_code>&state=<state>
```

OIDC Client 后端需要：

1. 校验 `state` 与 `/sso/start` 保存值一致。
2. 使用 `code` 调 IdBridge `/oauth2/token`。
3. 校验 `id_token` 签名、`iss`、`aud`、`exp`、`nonce`。
4. 按 `sub` 或业务约定的 claim 映射本地用户。
5. 建立业务应用自己的 session/cookie，再跳回业务页面。

Token 请求示例：

```http
POST /oauth2/token HTTP/1.1
Host: idb.example.com
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code&
code=<authorization_code>&
redirect_uri=https%3A%2F%2Fapp.example.com%2Foidc%2Fcallback&
client_id=my-app&
client_secret=<client_secret>&
code_verifier=<pkce_code_verifier>
```

如果是 SPA/public client，不要使用 `client_secret`，必须使用 PKCE，并在后端或可信 BFF 中完成最终业务会话建立。

##### 端到端链路

推荐链路：

1. 飞书应用/卡片入口打开业务应用自己的登录启动地址，例如 `https://app.example.com/sso/start`。
2. OIDC Client 生成 `state`、`nonce`、PKCE，并重定向到 IdBridge 授权端点。需要强制使用飞书时，在授权请求中追加 `idp=feishu`：

```text
https://idb.example.com/oauth2/authorize?response_type=code&client_id=my-app&redirect_uri=https%3A%2F%2Fapp.example.com%2Foidc%2Fcallback&scope=openid%20profile%20email&state=<state>&nonce=<nonce>&code_challenge=<challenge>&code_challenge_method=S256&idp=feishu
```

3. IdBridge 登录上下文会从 `return_to=/oauth2/authorize?...&idp=feishu` 中解析首选登录方式，并返回：
   - `preferred_provider: "feishu"`
   - `auto_redirect_url: "/api/auth/feishu/login?entity_id=...&return_to=..."`
4. 登录页/BFF 收到 `auto_redirect_url` 后可直接跳转到该 URL；飞书回调完成后，IdBridge 继续原始 OIDC 授权请求，并最终回跳 OIDC Client 的 `redirect_uri?code=...&state=...`。
5. OIDC Client 校验 `state`，换取并校验 `id_token`，建立自身业务会话。

约束：

- OIDC Client 只信任 IdBridge 签发的 `id_token`，不要直接信任飞书 `user_access_token`。
- `client_secret` 只允许放在 OIDC Client 后端；前端/飞书卡片入口使用 PKCE。
- 飞书开放平台的 OAuth 重定向 URL 配置为 IdBridge 飞书回调地址，例如 `https://idb.example.com/auth/feishu/callback`。

#### 2.1.5 生产注意点

- 授权端点必须通过浏览器触发（会话依赖 `idb_session`）；  
- 代码交换与刷新 token（当前版本不支持 refresh token）；  
- 统一验证 `state`，防 CSRF；  
- 强制校验 `redirect_uri` 与注册值一致。

### 2.2 旧项目用户名密码映射接入（legacy 登录）

适用场景：老项目不具备 OIDC，对接 IdBridge 侧的用户名密码网关。

- 登录入口：  
  - `POST /login/legacy`
  - 管理面入口兼容：`POST /api/admin/v1/login/legacy`、`POST /admin/v1/login/legacy`
- Body：
```json
{
  "entity_id": "<entity ulid>",
  "application_id": "<application ulid>",
  "username": "old_user",
  "password": "raw_password"
}
```
- 成功响应：
```json
{
  "code": "legacy_auth_success",
  "user_id": "<ulid>",
  "entity_id": "<entity ulid>",
  "application_id": "<application ulid>",
  "username": "old_user",
  "display_name": "...",
  "session": "<session jwt-like base64>"
}
```
- 失败码（HTTP）：
  - `400` `entity_id` / `application_id` / 用户名密码缺失
  - `423` `legacy_auth_locked`（达到阈值后）
  - `401` 用户名不存在 / 密码错误 / 映射不存在 / 映射被禁用
  - `403` 映射存在但应用禁止访问
  - `500` 鉴权服务异常
- 服务端行为：
  - 登录失败落库审计（`legacy_password_events`）
  - 成功返回并写 `idb_session` cookie
  - 统一响应码语义：
    - 成功：`legacy_auth_success`
    - 失败：`legacy_auth_failed`
    - 锁定：`legacy_auth_locked`

运营说明：
- 目前已提供了登录入口与审计链路；
  - 可通过管理 API 直接维护 `legacy_app_users`，完成“老项目 -> IdBridge”用户名密码映射。
  - 映射变更会产生日志（create/update/delete/status）用于审计。
  - 控制台入口在“应用管理” -> 应用设置里同步提供：OIDC Client 配置与 Legacy 映射维护（同一弹窗）。

#### 2.2.2 legacy 映射管理 API（已支持）

- 按应用配置映射：`/api/admin/v1/applications/{application_id}/legacy-users`
- 入口示例（建议走 `/admin/v1/...`）：

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/api/admin/v1/applications/{application_id}/legacy-users` | 分页查询该应用 legacy 映射列表（`limit`/`offset`） |
| `GET` | `/api/admin/v1/applications/{application_id}/legacy-users/{username}` | 查询单个 legacy 映射 |
| `POST` | `/api/admin/v1/applications/{application_id}/legacy-users` | 新建映射 |
| `PUT` | `/api/admin/v1/applications/{application_id}/legacy-users/{username}` | 更新映射（密码、目标用户、legacy id、状态） |
| `POST` | `/api/admin/v1/applications/{application_id}/legacy-users/{username}/enable` | 启用映射 |
| `POST` | `/api/admin/v1/applications/{application_id}/legacy-users/{username}/disable` | 禁用映射 |
| `DELETE` | `/api/admin/v1/applications/{application_id}/legacy-users/{username}` | 删除映射 |

- 新建映射 Body 示例：
```json
{
  "username": "old_user",
  "user_id": "<ulid of managed user>",
  "password": "raw-password",
  "legacy_user_identifier": "optional",
  "is_active": true
}
```

- 更新映射 Body 示例：
```json
{
  "user_id": "<ulid>",
  "password": "new-password",
  "legacy_user_identifier": "optional",
  "is_active": false
}
```

### 2.3 与前端/控制台的对接关系（现状）

- 登录页支持：
  - 本地账号登录：`/api/login/account`
  - 飞书入口：`/auth/feishu/login?entity_id=...`
  - OIDC 参数回传：`/login?client_id=...&redirect_uri=...` 会记录为 `return_to`
- 管理页面支持：
  - 应用管理：`/api/admin/v1/applications` + `/api/admin/v1/oidc-clients`
  - Legacy 映射管理：`/api/admin/v1/applications/{application_id}/legacy-users`
  - 身份源管理：`/api/admin/v1/identity-sources`
  - 身份源同步：`/api/admin/v1/identity-sources/{id}/sync/full`
  - IM Provider 配置：`/api/admin/v1/integrations/im`

### 2.4 前端与后端契约对齐（当前实现）

- OIDC 应用接入（标准）
  - 前端入口：
    - 登录页解析 OIDC 参数，发起 `/oauth2/authorize`
    - 应用管理页提供应用与 OIDC Client 的创建、查询、更新、密钥轮换
  - 后端接口：
    - `GET /oauth2/authorize`
    - `POST /oauth2/token`
    - `GET /oauth2/userinfo`
    - `POST /oauth2/revoke`
    - `POST /api/admin/v1/applications`
    - `POST /api/admin/v1/oidc-clients`
- Legacy 老项目映射登录（已打通）
  - 前端入口：
    - 应用设置页提供 legacy 映射列表与 CRUD（启停、删除）
  - 后端接口：
    - `POST /login/legacy`（兼容 `/api/admin/v1/login/legacy`、`/admin/v1/login/legacy`）
    - `GET/POST/PUT/DELETE /api/admin/v1/applications/{application_id}/legacy-users`
    - `POST .../legacy-users/{username}/enable|disable`
- Feishu 接入/同步
  - 前端入口：
    - 集成页 Feishu Tab 提供应用配置与同步触发
    - 前端实现遵循 `web/`：SvelteKit 文件路由、非 SPA 主流程、Tailwind/Skeleton 组件链路（非定制 CSS 为主）
  - 后端接口：
    - `GET/PUT /api/admin/v1/integrations/im/feishu`
    - `POST/GET/PUT /api/admin/v1/identity-sources`
    - `POST /api/admin/v1/identity-sources/{id}/sync/full`
    - `POST /api/admin/v1/identity-sources/{id}/sync/incremental`
    - `GET /api/admin/v1/auth/providers`
  - 同步模式说明：
    - `full`：导入并 upsert 全量目录用户，适合首次导入或修复偏差后重新对齐。
    - `incremental`：按增量窗口抓取变更并 upsert 到本地目录，优先用于周期性/准实时更新；对于飞书，可通过事件与游标结合实现增量持续同步。

### 2.5 当前未对齐（该版本）

- Google/微软目录全量同步：未在当前 `idp` 提供商实现。若需支持，需新增：
  - `identity_sources.type = google` 的配置约束
  - `DirectoryProvider` 的 `google` 实现
  - sync 服务分支与配置页字段补齐

### 2.6 接入完成后的 2 分钟核验（可直接复制）

以下命令用于接入联调后快速确认接口与页面链路是否打通（尤其适用于飞书与 OIDC 的接入验证）：

```bash
# 1) 前端/后端开发链路就绪
make web-check-strict
make dev-local

# 2) 关键约束扫描（导航、样式、组件）
rg -n "goto\\(|\\$app/navigation|window\\.history|window\\.location\\." web/src
rg -n "style=\\\"|style='|<style" web/src/app.html web/src/routes web/src/lib
rg -n "from ['\\\"]@skeletonlabs/skeleton['\\\"]|@skeletonlabs" web/src
rg -n "from ['\"]react['\"]|ReactDOM|\\bclassName=" web/src

# 3) 常见问题一键复盘
make dev-local-quickfix
```

联调说明：
- 飞书接入：验证 `GET /api/admin/v1/auth/providers` 能返回 `feishu` provider，并完成一次 `/auth/feishu/login` -> 回调流程。
- OIDC 接入：验证 `/.well-known/openid-configuration` 与 `POST /oauth2/token` 可达。
- Legacy 映射接入：验证 `/api/admin/v1/applications/{application_id}/legacy-users` 的增删改查与应用登录。
- 相关核验清单：
- [web 评审清单](docs/web-review-checklist.md)
- [PR 提交前 2 分钟模板](.github/PULL_REQUEST_TEMPLATE/pull_request_template.md)

前端标准口径（非 SPA + SvelteKit + Tailwind/Skeleton，版本：`web-svelte-tailwind-contract-v1.4`）统一约束见：  
[web-svelte-tailwind-contract.md](docs/web-svelte-tailwind-contract.md)。

### 2.7 前端实现补充约束（给接入开发的约定）

- 新前端统一在 `web/` 落地，强调：
  - Tailwind 与 Skeleton 是核心 UI 组件基础设施，不以 SPA 状态机承载主流程；
  - 主流程不得使用 `goto`/`$app/navigation` 进行页面切换，优先使用 `a href`、`form` 与服务端重定向；
  - 列表与卡片优先使用 `card/article` 等语义结构，不默认表格化；
  - 文案默认英文（`en-US`），通过 i18n key 管理。

## 三、迁移到新供应商（如 Google）时的最小改造清单

若你要让 IdBridge 做 Google 等新系统的目录同步，建议按以下顺序落地：

1. 新增/复用 `identity_sources.type = google` 的 source 配置约束与校验；
2. 在 `backend/internal/idp` 新增 provider adapter，实现在 API 侧获取部门/用户的分页同步；
3. 在 `NewSyncService` 的 `ProviderFactory` 中添加 `provider="google"` 分支；
4. 在 IM Provider 配置页面/接口补充 Google 的配置 schema 与校验；
5. 写入端到端回归：查询用户 -> 映射绑定 -> 全量同步 -> 审计链路。
