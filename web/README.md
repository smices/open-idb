# IdBridge Web Frontend (SvelteKit)

> 该目录为新前端（web）实现。Tailwind 组件体系与 Skeleton 是默认前端基础能力与关键交付标准，不走 SPA 模式。Tailwind 组件先于自定义样式，页面编排优先使用 SvelteKit 文件路由与 Svelte 原生能力，以充分利用 Svelte 的原生视图能力。

## 开发与核验导航

- 统一入口请见： [docs/quickstart-navigation.md](docs/quickstart-navigation.md)
- 统一前端硬约束： [docs/web-svelte-tailwind-contract.md](docs/web-svelte-tailwind-contract.md)（v1.3）  
  速读版：[docs/web-svelte-tailwind-contract-brief.md](docs/web-svelte-tailwind-contract-brief.md)（v1.3）
- 开发启动命令：`cd web && npm run dev -- --host 0.0.0.0 --port 5180`
- 约束验收命令：`make web-frontend-contract`
- 全量链路联调请使用：`make dev-local`

本目录开发约定：
- 非 SPA 状态机主导航，优先使用文件路由与服务端流程；
- 使用 Skeleton + Tailwind 组件体系（组件优先）；
- 优先以 Svelte 原生能力实现页面逻辑（`{#if}` / `{#each}` / `form` / store / actions）。
- Tailwind/Skeleton 是本项目前端基础设施，不以自定义样式系统重建组件；
- 主题、路由切换与列表组织以 Svelte 与 SvelteKit 原生能力为第一优先。
