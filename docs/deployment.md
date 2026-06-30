# IdBridge 自托管部署指南

本文面向把 IdBridge 部署到自有 IDC、私有云或 Kubernetes 集群的使用方。目标是让部署方按步骤完成初始化、反向代理、飞书配置、OIDC 应用接入和上线验证。

## 1. 部署模型

IdBridge 由两个运行单元组成：

| 组件 | 说明 |
|---|---|
| `idbridge` backend | Go API 服务，提供 `/api/*`、`/sapi/*`、OIDC、登录、同步、审计等能力 |
| `idbridge-frontend` web | SvelteKit 静态前端，提供 `/admin/*`、`/portal`、`/auth/continue` 等页面 |

推荐生产形态：

```text
https://idbridge.example.com
  /api/*   -> backend:8080
  /sapi/*  -> backend:8080
  /        -> frontend:80
```

如果使用仓库内 `web/Dockerfile` 构建的前端镜像，镜像内 Caddy 已把 `/api*` 和 `/sapi*` 代理到 `http://idbridge:8080`。如果使用自己的 Nginx、Ingress、网关或前端静态托管，必须自行配置上面的路由。

## 2. 前置条件

生产或 IDC 环境至少需要：

- PostgreSQL 13+，并启用 `pgcrypto` 扩展。
- 一个稳定的 HTTPS 域名，例如 `https://idbridge.example.com`。
- 可以访问 Feishu OpenAPI 的出口网络，如果启用飞书登录或通讯录同步。
- 一个 Feishu 企业自建应用，如果启用飞书。
- 可安全存放数据库连接串和 Feishu App Secret 的 Secret 管理方式。

不建议生产使用：

- 明文 HTTP。
- 使用默认管理员密码长期运行。
- 把 `client_secret`、Feishu `app_secret` 写入公开仓库。

## 3. 必填配置

后端通过环境变量配置。

| 变量 | 必填 | 示例 | 说明 |
|---|---:|---|---|
| `DATABASE_URL` | 是 | `postgres://idbridge:***@postgres:5432/idbridge?sslmode=disable` | PostgreSQL 连接串 |
| `IDB_HTTP_ADDR` | 是 | `:8080` | 后端监听地址 |
| `IDB_OIDC_ISSUER` | 是 | `https://idbridge.example.com` | OIDC issuer，必须是用户和外部应用可访问的最终地址 |
| `IDB_WEB_BASE_URL` | 建议 | `https://idbridge.example.com` | Web 基准地址；未显式设置 Feishu 回调时用于生成回调地址 |
| `IDB_FEISHU_REDIRECT_URI` | 飞书必填 | `https://idbridge.example.com/api/auth/feishu/callback` | 飞书 OAuth 回调地址 |
| `IDB_OIDC_KEY_ID` | 建议 | `prod-key-1` | OIDC/JWKS key id |
| `IDB_DEFAULT_LOCALE` | 否 | `zh-CN` | `zh-CN` 或 `en-US` |
| `IDB_SESSION_TTL_SECONDS` | 否 | `86400` | 用户会话 TTL |
| `IDB_AUTH_CODE_TTL_SECONDS` | 否 | `300` | OIDC 授权码 TTL |
| `IDB_ACCESS_TOKEN_TTL_SECONDS` | 否 | `900` | access token TTL |
| `IDB_ID_TOKEN_TTL_SECONDS` | 否 | `900` | id token TTL |
| `IDB_REDIS_ENABLED` | 否 | `true` | 是否启用 Redis 读模型/缓存 |
| `IDB_REDIS_URL` | Redis 时必填 | `redis://redis:6379/0` | Redis 连接地址 |
| `IDB_FEISHU_BASE_URL` | 否 | `https://open.feishu.cn` | 飞书 OpenAPI 地址 |

注意：

- `IDB_OIDC_ISSUER` 必须和部署后发现文档中的 issuer 一致。
- `IDB_FEISHU_REDIRECT_URI` 必须在飞书开放平台中配置为 OAuth 回调地址。
- 如果后端在内网 `http://idbridge:8080`，但用户访问域名是 `https://idbridge.example.com`，`IDB_OIDC_ISSUER` 仍然必须写外部可访问地址。

## 4. 数据库初始化

当前版本使用一个首次初始化 SQL：

```text
backend/migrations/000001_schema_baseline.sql
```

执行迁移：

```bash
cd backend
go run github.com/pressly/goose/v3/cmd/goose@v3.22.1 \
  -dir migrations \
  postgres "$DATABASE_URL" up
```

容器镜像内已包含 `goose` 和 `/app/migrations`，Kubernetes Job 可直接执行：

```bash
goose -dir /app/migrations postgres "$DATABASE_URL" up
```

迁移完成后会初始化：

- 默认公司：`Default Enterprise`
- 默认平台管理员：`admin / admin123`
- 默认用户账号：`admin / admin123`
- 基础权限和角色

首次登录后必须立即修改默认管理员密码。

## 5. 构建镜像

后端：

```bash
docker build -t idbridge-backend:latest backend/
```

前端：

```bash
docker build -t idbridge-frontend:latest web/
```

前端镜像默认用 Caddy 暴露 80 端口，并把 `/api*`、`/sapi*` 转发到服务名 `idbridge:8080`。如果你的后端服务名不是 `idbridge`，请改 `web/Caddyfile` 或在部署层提供同名 Service。

## 6. Kubernetes 部署顺序

推荐顺序：

1. 创建 namespace。
2. 创建 PostgreSQL Secret。
3. 启动 PostgreSQL 或配置外部 PostgreSQL。
4. 执行 migration Job。
5. 部署 backend。
6. 部署 frontend。
7. 配置 Ingress/TLS。
8. 执行上线验证。

仓库内 OrbStack 示例：

```bash
make k8s-build
make k8s-build-frontend
make k8s-deploy
make k8s-deploy-frontend
```

这些示例面向本地开发，不应原样用于生产。IDC 部署时请至少替换：

- namespace
- 镜像仓库地址
- `DATABASE_URL`
- `IDB_OIDC_ISSUER`
- `IDB_WEB_BASE_URL`
- `IDB_FEISHU_REDIRECT_URI`
- TLS Secret
- Ingress host

## 7. 反向代理配置

### 7.1 Nginx 示例

```nginx
server {
    listen 443 ssl http2;
    server_name idbridge.example.com;

    ssl_certificate     /etc/nginx/tls/tls.crt;
    ssl_certificate_key /etc/nginx/tls/tls.key;

    location /api/ {
        proxy_pass http://idbridge-backend:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto https;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    location /sapi/ {
        proxy_pass http://idbridge-backend:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto https;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    location / {
        proxy_pass http://idbridge-frontend:80;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto https;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

### 7.2 Ingress 路由要求

如果 Ingress 直接分流，而不是让前端 Caddy 代理 API，请配置：

| 路径 | 后端服务 |
|---|---|
| `/api` | `idbridge` backend |
| `/sapi` | `idbridge` backend |
| `/` | frontend |

OIDC 集成推荐使用 `/api` 前缀下的发现文档：

```text
https://idbridge.example.com/api/.well-known/openid-configuration
```

## 8. 飞书配置

在飞书开放平台创建企业自建应用，至少准备：

- App ID
- App Secret
- OAuth 回调地址：

```text
https://idbridge.example.com/api/auth/feishu/callback
```

在 IdBridge 后台配置：

1. 登录 `/admin/login`。
2. 进入“身份源”。
3. 添加飞书身份源。
4. 填入 Feishu App ID 和 App Secret。
5. 保存后触发全量同步。
6. 在“同步任务”查看进度和日志。
7. 在“组织架构”检查部门树和人员。

身份源边界：

- 当前版本只开放飞书作为主身份源。
- 同一公司内主身份源互斥。
- 本地账号只是登录兜底能力，不作为身份源配置。

## 9. 管理员和账号

访问：

```text
https://idbridge.example.com/admin/login
```

初始管理员：

```text
admin / admin123
```

首次操作建议：

1. 修改 `admin` 密码。
2. 在“管理员管理”创建真实平台管理员或公司管理员。
3. 禁用或仅保留 `admin` 作为 break-glass 账号。
4. 配置公司信息。
5. 配置身份源。
6. 同步组织架构和目录用户。
7. 配置应用。

管理员账号和普通用户账号是隔离的：

- 管理员：`admin_users`，登录 `/admin/*`，使用 `idb_admin_session`。
- 用户：`users`，登录 `/portal` 或应用登录流，使用 `idb_session`。

## 10. 应用接入

在后台进入：

```text
/admin/applications
```

添加 OIDC 应用时，IdBridge 会自动签发：

- `client_id`
- `client_secret`
- 默认授权范围：`openid profile email`
- 授权类型：`authorization_code`
- 响应类型：`code`
- PKCE：启用

如果业务应用需要人员选择、部门选择或组织搜索，在 OIDC 配置中勾选“目录 API”。这会把 `directory:read` 加入应用允许的授权范围：

```text
directory:read
```

IdBridge 会按应用已勾选的授权范围签发 access token；业务应用侧不需要同步维护这份授权范围。

管理员需要填写业务应用回调地址，必须是绝对 URL：

```text
https://app.example.com/auth/oidc/callback
```

保存后应用抽屉会展示：

- Discovery URL
- Authorization Endpoint
- Token Endpoint
- UserInfo Endpoint
- Feishu login template
- Feishu workplace SSO template

业务应用需要自己生成 `state`、`nonce`、PKCE `code_verifier/code_challenge`，不能复用固定值。

## 11. 飞书工作台 SSO

适用场景：用户从飞书工作台点击业务应用，业务应用跳转到 IdBridge 完成飞书上游登录，再回到业务应用。

推荐链路：

1. 飞书工作台入口配置为业务应用自己的地址：

```text
https://app.example.com/sso/start
```

2. 业务应用生成 `state`、`nonce`、PKCE。
3. 业务应用重定向到 IdBridge 授权端点，并带：

```text
idp=feishu&workplace=feishu
```

4. IdBridge 在飞书工作台环境中通过飞书 JS bridge 获取 `auth_code`。
5. IdBridge 用 `auth_code` 建立自身用户会话。
6. IdBridge 继续 OIDC 授权请求，回调业务应用 `redirect_uri?code=...&state=...`。
7. 业务应用用 `code` 换 token 并建立自己的业务会话。

普通浏览器没有飞书 JS bridge，因此本地普通 Chrome 测试工作台入口时，看到“登录码交换失败”是预期现象。真实验证必须在飞书工作台内完成。

## 12. 上线验证

基础健康检查：

```bash
curl -fsS https://idbridge.example.com/api/.well-known/openid-configuration
curl -fsS https://idbridge.example.com/admin/login
```

OIDC authorize 未登录时应返回 302 到登录页：

```bash
curl -i "https://idbridge.example.com/api/oauth2/authorize"
```

后台页面验证：

1. 打开 `/admin/login`。
2. 使用管理员账号登录。
3. 打开“身份源”，确认能加载。
4. 打开“应用管理”，创建一个临时 OIDC 应用。
5. 确认自动生成 `client_id` 和 `client_secret`。
6. 使用生成的授权 URL 检查是否跳到登录页。
7. 删除临时应用。

Feishu 验证：

1. 配置飞书身份源。
2. 触发全量同步。
3. 检查“同步任务”日志。
4. 检查“组织架构”是否显示公司、部门、人员和英文名。
5. 使用飞书登录一个已同步用户。
6. 在飞书工作台真实打开业务应用入口，验证 `workplace=feishu` 链路。

## 13. 常见问题

### 后台登录后页面加载 API 失败

检查 `/sapi/*` 是否代理到 backend。只代理 `/api/*` 不够。

### OIDC discovery 地址返回前端 HTML

说明路由转发错误。推荐使用：

```text
/api/.well-known/openid-configuration
```

并确保 `/api/*` 到 backend。

### 飞书回调失败

检查：

- 飞书开放平台回调地址是否等于 `IDB_FEISHU_REDIRECT_URI`。
- 回调地址是否为 HTTPS。
- `/api/auth/feishu/callback` 是否能从飞书访问到。

### 工作台 SSO 在普通浏览器失败

普通浏览器没有飞书 JS bridge，无法获取飞书工作台 `auth_code`。请在飞书工作台内验证。

### authorize 一直提示登录态缺失

这是未登录时的正常行为。OIDC authorize 会先要求 IdBridge 用户会话，登录成功后继续授权请求。

### token 换取失败

检查：

- `redirect_uri` 是否和后台登记值完全一致。
- PKCE `code_verifier` 是否对应授权请求里的 `code_challenge`。
- `client_secret` 是否放在服务端请求中，且没有泄露到前端。
- `IDB_OIDC_ISSUER` 是否是最终用户可访问域名。

## 14. 发布前检查清单

- [ ] PostgreSQL 已初始化并完成迁移。
- [ ] `admin / admin123` 已修改密码。
- [ ] `IDB_OIDC_ISSUER` 是最终 HTTPS 域名。
- [ ] `/api/*` 和 `/sapi/*` 已正确转发到 backend。
- [ ] Feishu OAuth 回调地址已在飞书开放平台配置。
- [ ] 应用回调 URI 是 HTTPS 绝对地址。
- [ ] OIDC discovery 可访问。
- [ ] 管理后台、身份源、应用管理、同步任务、组织架构页面可加载。
- [ ] 已完成一次飞书全量同步。
- [ ] 已完成至少一个 OIDC 应用授权码登录闭环。
