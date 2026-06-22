# PR Title

## 变更摘要
- 

## 变更范围
- [ ] Backend
- [ ] Web 前端（SvelteKit + Tailwind + Skeleton）
- [ ] Docs/脚本

## Web 前端提交前 2 分钟核对清单（可选）
- [ ] 已完成 1）`make web-check-strict`
- [ ] 已完成 2）关键导航约束：无关键流程 `goto`/`$app/navigation`/`window.history`（非关键流程例外：`web/src/lib/session.ts`）
- [ ] 已完成 3）架构约束：非 SPA 主流程，页面切换走 SvelteKit 文件路由 + 服务端/表单流程，不把状态机当页面导航承载
- [ ] 已完成 4）组件约束：Tailwind + Skeleton 组件链路为关键 UI 基建，优先 `@skeletonlabs/skeleton` 与 Tailwind utility，不得新增自定义样式/组件体系替代核心能力
- [ ] 已完成 5）样式约束：无内联 `style`，无新增业务级 `<style>` 覆写
- [ ] 已完成 6）i18n 约束：默认 `en-US`，文案不硬编码中文
- [ ] 已完成 7）联调可复现：`make dev-local-preflight` + `make dev-local` 可启动

> 注：如需同步更新核验入口文案，请优先更新 [docs/quickstart-navigation.md](/Volumes/FeWS/Projects/SMICES/open-idb/docs/quickstart-navigation.md)；其他文档统一引用该片段。
> 统一前端硬约束文本请参考：[docs/web-svelte-tailwind-contract.md](/Volumes/FeWS/Projects/SMICES/open-idb/docs/web-svelte-tailwind-contract.md)（web-svelte-tailwind-contract-v1.2）

对应命令：
```bash
make web-check-strict
rg -n "goto\\(|\\$app/navigation|window\\.history|window\\.location\\." web/src
rg -n "style=\\"|style='|<style" web/src/app.html web/src/routes web/src/lib
rg -n "from ['\\"]@skeletonlabs/skeleton['\\"]|@skeletonlabs" web/src
git diff --stat
```

## 关联文档
- 开发启动与排障：`docs/development.md`
- 前端评审清单：`docs/web-review-checklist.md`

## 备注
- 
