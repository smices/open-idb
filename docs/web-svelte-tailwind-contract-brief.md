# web 前端约束（1 页速读）

适用范围：`web/`（新前端）  

**版本**：`web-svelte-tailwind-contract-v1.4`  
**更新日期**：`2026-05-29`

- 1）**非 SPA 主流程**：关键页面导航用 SvelteKit 文件路由与服务端/表单流程，不用状态机/`goto` 承载关键跳转。
- 2）**组件优先**：Tailwind 组件与 Skeleton 组件体系是项目核心 UI 组件能力，属于项目关键基础设施，不以自定义 CSS 体系重建组件。
- 3）**充分使用 Svelte 原生能力**：列表/状态主流程用 `{#if}`、`{#each}`、`form`、`store`、actions。
- 4）**非 SPA 主模式**：禁 `goto()`、`$app/navigation`、`window.history`；`window.location` 只允许在 `web/src/lib/session.ts` 统一封装。
- 5）**样式与文案**：业务层无内联 `style`，无新增业务级 `<style>`；默认文案 `en-US`，页面文案走 i18n key。

补充：
- Tailwind 与 Skeleton 是项目最重要的 UI 组件能力，不以手写布局替代官方组件体系；能直接用 Skeleton 的场景必须优先用组件（包括弹窗、菜单、卡片、列表、表单控件、主题切换）。
- `web` 采用文件路由型前端，不以前端状态机接管主导航；尽量把路由与重定向交给 SvelteKit 与服务端能力。
- 重点口径：Tailwind 组件 + Skeleton 是本项目 UI 的基础能力；非 SPA 运行时主流程，所有关键页面与交互优先靠 Svelte 原生 `{#if}/{#each}` 与文件路由实现。

**提交前强制核验**：`make web-frontend-contract`

**统一完整口径**：`docs/web-svelte-tailwind-contract.md`

> 版本更新请按完整版 `docs/web-svelte-tailwind-contract.md` 的「版本更新清单模板」追加后，再同步更新本页版本节。

## 变更记录

- `web-svelte-tailwind-contract-v1.4`（2026-05-29）
  - 首次补充：明确 Tailwind/Skeleton 组件优先与“非 SPA 主模式”双重底线，加入“充分使用 Svelte 原生能力”执行语义，便于后续接入/验收一致。

- `web-svelte-tailwind-contract-v1.3`（2026-05-29）
  - 首次补充：把 Tailwind/Skeleton 组件能力定义为项目核心，强调两栏可折叠侧边栏、主题/语言切换、非 SPA 主导航与 Svelte 原生表达。

- `web-svelte-tailwind-contract-v1.2`（2026-05-29）
  - 首版速写同步：加入版本迭代示例入口与统一约束版本口径提醒。

- `web-svelte-tailwind-contract-v1.0`（2026-05-29）
  - 首版速读同步：非 SPA、Tailwind+Skeleton 组件优先、Svelte 原生、`window.location` 限制。

> 变更时请同步更新 [web-svelte-tailwind-contract.md](docs/web-svelte-tailwind-contract.md) 的版本元信息。
