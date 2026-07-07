## 快速导航（开发 / 接入 / 提交核验）

> 维护说明：该入口为统一源，后续文档入口统一引用此文件，避免重复维护失效。

- 开发环境与排障：`make dev-local-preflight`、`make dev-local`、`make web-frontend-contract`、`make web-check` / `make web-check-strict`、`make dev-local-quickfix`  
  文档：[development.md](development.md)
- 一键联调启动：`make dev-local-go-live`（执行预检后直接进入启动）  
  说明：启动 `web/` 前端与本地后端联调链路。  
- 自托管 / IDC 部署：环境变量、迁移、镜像、反向代理、飞书配置、上线验证  
  文档：[deployment.md](deployment.md)
- 应用接入：OIDC、飞书登录、飞书工作台 SSO、排查清单  
  文档：[integration-guide.md](integration-guide.md)
- 前端评审清单：React + Vite + Ant Design，通用能力包含 i18n、明暗主题、用户头像和菜单，能用 AntD 的地方优先用 AntD
  文档：[web-review-checklist.md](web-review-checklist.md)
- 前端统一约束文本：统一标准口径（React + Vite + Ant Design）
  文档：[web-react-antd-contract.md](web-react-antd-contract.md)
- 前端速读约束（1 页）：适合快速对齐  
  文档：[web-react-antd-contract-brief.md](web-react-antd-contract-brief.md)
- 标准版本：`web-react-antd-contract-v1.0`（更新于 `2026-07-07`）
- 前端最终交付清单：提交前最终核对动作（web 最终验收）  
  文档：[web-final-delivery-checklist.md](web-final-delivery-checklist.md)
- PR 提交前 2 分钟核验：`make web-frontend-contract` + 入口约束  
  模板：[.github/PULL_REQUEST_TEMPLATE/pull_request_template.md](../.github/PULL_REQUEST_TEMPLATE/pull_request_template.md)

统一输出码说明（dev-local / web 约束）：

- `[P0]`：阻断项，需先修复（如本地工具、K8S 不可达、关键依赖缺失）。
- `[P1]`：建议项，建议修复（如端口占用、可达性探测信息、非阻断风格提示）。
- 入口优先阅读：[development.md](development.md)
