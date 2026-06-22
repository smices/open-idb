# web 前端评审清单（Svelte + Tailwind + Skeleton）

## 快速导航（开发 / 接入 / 提交核验）

统一入口请见： [docs/quickstart-navigation.md](docs/quickstart-navigation.md)

## 基础规则
- [ ] 默认语言为 `en-US`；文案不得在页面内中文硬编码。
- [ ] 导航策略以 SvelteKit 文件路由与服务端流程为主，不使用 SPA 状态机承载关键页面切换；不得引入 `window.history`、`goto()`、`$app/navigation`。
- [ ] Tailwind + Skeleton 为关键 UI 组件体系，页面主流程不得以自定义组件/样式系统替代标准组件链路。
- [ ] 禁止在业务主流程使用 `goto()`、`$app/navigation`、`window.history`。
- [ ] 禁止新增业务级 `window.location` 跳转（允许仅存在 `web/src/lib/session.ts` 的会话封装）。
- [ ] 统一约束口径请见：[web-svelte-tailwind-contract.md](docs/web-svelte-tailwind-contract.md)（v1.4）。

## 组件与样式规则
- [ ] 优先采用 Skeleton 组件 + Tailwind utility 实现界面（Button、Card、Modal、Table、Popover 等）。`Tailwind` 与 `Skeleton` 组件链路是项目关键 UI 体系，页面布局优先组件化，不以大量自定义原子组合替代；关键列表要避免重 table 化实现，优先卡片/块状呈现以体现 SvelteKit + Svelte 原生视图编排（仅允许边界场景补齐样式）。
- [ ] 避免主列表默认使用 `<table>`，除明确列导向场景外优先卡片/列表组件结构承载关键数据。
- [ ] 允许的自定义应只用于边界场景组件，不得替代 Skeleton 的既有组件能力或破坏 Tailwind/Skeleton 体系。
- [ ] 避免内联样式（`style="..."` / `style='...'`）；避免页面内 `<style>` 覆写。
- [ ] 主题风格应使用 Skeleton 主题系统并保持一致性。

## 校验命令（提交前）
```bash
git status --short | head -n 30
make web-frontend-contract
rg -n "goto\(|\$app/navigation|window\.history|window\.location\." web/src
rg -n "<table\\b|</table>" web/src/routes web/src/lib --glob '*.svelte' --glob '*.ts'
rg -n "style=\"|style='|<style" web/src/app.html web/src/routes web/src/lib
rg -n "from ['\"]@skeletonlabs/skeleton['\"]|@skeletonlabs" web/src
rg -n "\bclassName=" web/src
rg -n "[\u4e00-\u9fff]" web/src/routes web/src/lib --glob '!web/src/lib/i18n.ts'
```

## 快速联调
- `make dev-local`
- 遇到问题先执行 `make dev-local-quickfix`

## 最终交付对照

本次改造的最终交付与验收清单见：[web-final-delivery-checklist.md](docs/web-final-delivery-checklist.md)

## PR 提交前模板（可直接复制）

- [ ] 默认语言与文案
  - [ ] 默认文案基线为英文（`en-US`）
  - [ ] 页面无中文硬编码，文案来源于 `web/src/lib/i18n.ts`
- [ ] 导航与页面架构
  - [ ] 使用 SvelteKit 文件路由与服务端重定向流程
  - [ ] 未在关键流程中使用 `goto()` / `$app/navigation` / `window.history` / `window.location`
- [ ] 组件与样式
  - [ ] 关键页面主列表不使用 `<table>`；采用卡片/块状组织
  - [ ] 优先使用 Tailwind + Skeleton 样式与组件能力（不以自定义组件系统替代）
  - [ ] 无页面级 `<style>` 覆写、无内联 style
- [ ] 前端自动校验
  - [ ] `make web-frontend-contract`
  - [ ] `make web-check WEB_CHECK_STRICT=error`
- [ ] 联调与排障
  - [ ] `make dev-local` 本地通路验收
  - [ ] 如有失败，已执行 `make dev-local-quickfix` 并按输出修复
