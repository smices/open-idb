# Web Frontend Final Delivery Checklist (2026-05-29)

## 目标状态
- 新前端路径：`web/`
- 关键要求：
- 非 SPA 主流程；基于 SvelteKit 文件路由
  - Tailwind + Skeleton 组件能力优先（Tailwind + Skeleton 组件链路是项目关键 UI 基建）
- 与执行约束保持一致（版本 `web-svelte-tailwind-contract-v1.4`）：查看 [web-svelte-tailwind-contract.md](docs/web-svelte-tailwind-contract.md)
- 页面层无硬编码中文（i18n 兜底）
- 列表默认卡片化，不以 `table` 为主布局

## 已完成变更对照（web）

### 路由页面
- `web/src/routes/+page.svelte`：登录页面卡片化重构（去掉自定义边框壳）
- `web/src/routes/+layout.svelte`：
  - 登录态、未授权态提示改为 `card`
  - Theme / Language Popover 使用 `card` 外壳
- `web/src/routes/application-assignments/+page.svelte`：列表项使用 `article + card`
- `web/src/routes/applications/+page.svelte`：应用列表、OIDC 子列表、Legacy 子列表使用 `card/article`
- `web/src/routes/dashboard/+page.svelte`：卡片化 KPI 与状态视图
- `web/src/routes/identity-sources/+page.svelte`：身份源列表与条目使用 `card/article`
- `web/src/routes/integrations/+page.svelte`：集成说明块改为 `card`
- `web/src/routes/mcp/+page.svelte`：连接器列表改为 `article/card`
- `web/src/routes/profile/+page.svelte`：Profile 页面卡片分块
- `web/src/routes/roles/+page.svelte`：角色/权限列表改为 `article/card`
- `web/src/routes/users/+page.svelte`：用户列表、授权弹窗列表改为 `card/article`
- `web/src/routes/users/[id]/+page.svelte`：用户详情子列表改为 `card/article`

### 非功能性约束清理
- `web/src/lib/session.ts` 是 `window.location.*` 的唯一业务外置封装点（不在页面内分散）

## 约束门禁（建议提审前必跑）
- `make web-frontend-contract`
- `make web-check WEB_CHECK_STRICT=error`

## 与后端前端接入配套核对
- `web/` 页面仅消费 `web/src/lib/api.ts` 的现有接口，未引入新后端行为分支
- 登录页保留标准表单与 OAuth 跳转路径，沿用服务端会话机制
- 与后端会话/鉴权流程耦合点集中在：`$lib/session.ts` 与后端接口返回控制
- 若后续需新增前端行为，优先在 `web/src/lib/api.ts` 增加接口封装，并保持页面通过 `t('...')` 完成文案映射

## 快速回归清单（PR 前 2 分钟）
1. `make web-frontend-contract`
2. `make web-check WEB_CHECK_STRICT=warn`（观测新增告警）
3. `make web-check WEB_CHECK_STRICT=error`（确认零阻断）
4. `make dev-local`
5. 执行关键路径手工核验：
   - 登录页（含错误码提示）
   - 左侧菜单可访问 9 个主页面
   - 角色/用户/应用/身份源/集成页增删改查
6. 若失败先执行：`make dev-local-quickfix`

## 风险与剩余动作
- 当前集中为 UI 与接入约束收口；后续可继续做以下非阻断增强：
  - 侧边栏/头部交互组件化抽离
  - 深层样式的可复用基线组件化（弹窗/列表项）
  - 持续补齐 `docs/integration-guide.md` 的接入核验说明
