# PR Title

## 变更摘要
- 

## 变更范围
- [ ] Backend
- [ ] Web 前端（React + Vite + Ant Design）
- [ ] Docs/脚本

## Web 前端提交前 2 分钟核对清单（可选）
- [ ] 已完成 1）`make web-frontend-contract`
- [ ] 已完成 2）组件约束：能用 Ant Design 的场景优先使用 AntD，不自建通用控件体系
- [ ] 已完成 3）通用能力：i18n、明暗主题、用户头像/菜单保持可用
- [ ] 已完成 4）登录边界：`/` 不暴露管理员入口；用户飞书 SSO 与管理员账号登录隔离
- [ ] 已完成 5）仪表盘边界：用户数不包含管理员账号，管理员数量单独展示
- [ ] 已完成 6）飞书同步边界：全量同步保留既有 ULID，缺失用户软删除
- [ ] 已完成 7）联调可复现：`make dev-local-preflight` + `make dev-local` 可启动

> 注：如需同步更新核验入口文案，请优先更新 [docs/quickstart-navigation.md](../../docs/quickstart-navigation.md)；其他文档统一引用该片段。
> 统一前端硬约束文本请参考：[docs/web-react-antd-contract.md](../../docs/web-react-antd-contract.md)（web-react-antd-contract-v1.0）

对应命令：
```bash
make web-frontend-contract
cd web && npm run build
git diff --stat
```

## 关联文档
- 开发启动与排障：`docs/development.md`
- 前端评审清单：`docs/web-review-checklist.md`

## 备注
- 
