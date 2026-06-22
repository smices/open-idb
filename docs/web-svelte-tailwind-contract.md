# Svelte Web 前端硬约束（统一标准）

适用于新前端 `web/` 的提交、联调与验收，不允许与该标准冲突。  

**版本**：`web-svelte-tailwind-contract-v1.4`  
**更新日期**：`2026-05-29`

## 核心约束（必须满足）
- 非 SPA 主流程：页面主导航以 **SvelteKit 文件路由 + 服务端重定向/表单流程** 承载，不得使用状态机进行关键跳转编排；项目不采用 SPA 运行时主导航模式。
- 关键导航必须可回退到服务端流程，不以客户端路由状态机承载主流程。
- 不使用 SPA 导航 API 承担核心流程：禁止 `goto()`、`$app/navigation`、`window.history`，`window.location` 仅允许在 `web/src/lib/session.ts` 的会话跳转封装中集中处理。主导航必须依托 SvelteKit 文件路由和服务端流程。
- Tailwind 组件与 Skeleton 组件体系是项目核心 UI 基建：默认优先使用 `@skeletonlabs/skeleton` 与 Tailwind utility，不得在非边界场景自行搭建替代组件链路（菜单、弹窗、列表、卡片、按钮、表单、主题切换）。该项目要求 Tailwind/Skeleton 成为首选视觉与交互能力。允许的自定义仅用于组件边界不足的微场景。
- 充分使用 Svelte 原生能力组织视图：页面/列表/表单默认用 Svelte 的声明式语法（`{#if}`、`{#each}`、`form`、`store`、actions）完成；避免额外引入状态机去承担跨路由流程。关键视图优先通过 SvelteKit 的文件路由与组件边界表达。
- 样式优先级：业务页面与组件不新增内联 `style` / 局部 `<style>` 覆写；统一依赖 `web/src/app.css` 与 Skeleton 主题。
- 文案与国际化：默认文案基线为 `en-US`，页面文案通过 `i18n` key 提供，避免页面级中文硬编码。
- 布局与交互：页面采用双栏（侧边栏 + 主内容）结构，侧边栏支持折叠；主题/语言切换文本通过 `i18n` 输出。
- 核心原则再强调：该前端不采用 SPA 运行时主模式，优先以 SvelteKit 文件路由、服务端表单/重定向 + Svelte 原生组件化表达完成导航与界面行为；“Tailwind 组件是重要基础能力”是长期不变规则，不应被通用样式拼凑替代。

## 附加验收项（建议）
- 列表与面板优先卡片/块状结构，避免默认 `table` 驱动关键列表展示。
- 以可维护性为先：`theme`/`language` 等横向能力通过统一组件与 `i18n` 管理，不引入临时重复实现。

## 快速一致性检查命令
- `make web-frontend-contract`
- `rg -n "goto\\(|\\$app/navigation|window\\.history|window\\.location\\." web/src`
- `rg -n "style=\\\"|style='|<style" web/src/app.html web/src/routes web/src/lib`
- `rg -n "from ['\\\"]@skeletonlabs/skeleton['\\\"]|@skeletonlabs" web/src  # Tailwind + Skeleton 组件链路（关键约束）`
- `rg -n "style=|inline" web/src`

## 变更记录

- `web-svelte-tailwind-contract-v1.4`（2026-05-29）
  - 变更发起人：系统协作会话  
  - 变更原因：接到“Tailwind 组件为核心 + 不要 SPA 模式 + 充分用 Svelte 能力”的补充约束，避免后续复合实现回退。  
  - 变更影响：
    - 约束项：增加对 Tailwind/Skeleton 组件为关键底座的明确强调，强化“非 SPA 运行时主模式”与 Svelte 原生表达边界。
    - 风格约束：侧边栏布局、主题语言切换、列表与卡片应继续沿用 Skeleton/Tailwind 组件链路表达，不以自研替代。
    - 相关文档：本条约束更新同步到速读与前端验收文档。
  - 回归动作：`make web-frontend-contract` 与 `make web-check WEB_CHECK_STRICT=warn`。
  - 风险与兼容说明：文档级更新，无运行时行为变化。

- `web-svelte-tailwind-contract-v1.3`（2026-05-29）
  - 变更发起人：系统协作会话
  - 变更原因：强化“Tailwind/Skeleton 组件优先 + 非 SPA + 两栏可折叠布局 + 明确主题/语言 i18n 交互 + Svelte 原生表达”的执行边界。
  - 变更影响：
    - 约束项：强调两栏导航场景、组件化优先级边界、主题/语言交互统一化与非 SPA 导航；
    - 校验命令：保持 `make web-frontend-contract` 与 `web-review-checklist` 的非 SPA/Svelte 原生检查。
    - 相关文档：`docs/web-svelte-tailwind-contract.md`、`docs/web-svelte-tailwind-contract-brief.md`、`web/README.md`、`docs/web-review-checklist.md`、`Makefile`。
  - 回归动作：
    - 执行 `make web-frontend-contract`
    - 执行 `rg -n "goto\\(|\\$app/navigation|window\\.history|window\\.location\\." web/src`
  - 风险与兼容说明：文档级约束增强，无运行时行为变化。

- `web-svelte-tailwind-contract-v1.1`（2026-05-29）
  - 变更发起人：系统协作会话
  - 变更原因：补充版本升级示例机制，形成长期可追溯的标准变更流程。
  - 变更影响：
    - 约束项：补充“版本更新清单模板”及其维护要求；
    - 校验命令：`make web-frontend-contract`、统一路径 `rg` 检查；
    - 相关文档：`docs/README.md`, `docs/quickstart-navigation.md`, `docs/development.md`, `docs/integration-guide.md`, `web/README.md`, `docs/web-review-checklist.md`, `docs/web-final-delivery-checklist.md`, `.github/PULL_REQUEST_TEMPLATE/pull_request_template.md`。
  - 回归动作：
    - 执行 `make web-frontend-contract`
    - 执行 `rg -n "web-svelte-tailwind-contract-v1\\.1" docs .github web/README.md Makefile`
  - 风险与兼容说明：纯文档级变更，对 runtime 行为无侵入影响。

- `web-svelte-tailwind-contract-v1.2`（2026-05-29）
  - 变更发起人：系统协作会话
  - 变更原因：再次强化“Tailwind/Skeleton 组件优先 + 非 SPA 主导航 + 深入使用 Svelte 原生能力”的硬约束。
  - 变更影响：
    - 约束项：Tailwind/Skeleton 组件能力写入“关键流程必须先用组件化能力实现”；
    - 校验命令：保持 `make web-frontend-contract` 与 `web-review-checklist` 的非 SPA/Svelte 原生检查。
    - 相关文档：`docs/web-svelte-tailwind-contract.md`、`docs/web-svelte-tailwind-contract-brief.md`、`web/README.md`、`docs/web-review-checklist.md`、`Makefile`。
  - 回归动作：
    - 执行 `make web-frontend-contract`
    - 执行 `rg -n "goto\\(|\\$app/navigation|window\\.history|window\\.location\\." web/src`
  - 风险与兼容说明：文档级约束增强，无运行时行为变化。

- `web-svelte-tailwind-contract-v1.0`（2026-05-29）
  - 首版：建立统一非 SPA + SvelteKit + Tailwind/Skeleton 前端硬约束；
  - 明确 `window.location` 仅在 `web/src/lib/session.ts` 集中封装；
  - 统一新增版本号与交叉引用入口；
  - 增加快速一致性检查命令与速读版映射。

## 版本更新清单模板（每次改动后追加）

请在每次提交新版本时新增一条，按此结构维护：

- `web-svelte-tailwind-contract-vX.Y`（YYYY-MM-DD）
  - 变更发起人：
  - 变更原因：
  - 变更影响：
    - 约束项：
    - 校验命令：
    - 相关文档：
  - 回归动作：
  - 风险与兼容说明：

> 变更时请同时更新 `Makefile` 中的 `WEB_FRONTEND_CONTRACT_VERSION` 与 `WEB_FRONTEND_CONTRACT_DATE`。
