// SPDX-License-Identifier: MIT

package i18n

const (
	LocaleEnglishUS = "en-US"
	LocaleChineseCN = "zh-CN"
)

type Catalog struct {
	messages map[string]map[string]string
}

func NewCatalog() Catalog {
	return Catalog{
		messages: map[string]map[string]string{
			LocaleEnglishUS: {
				// Health
				"health.ok": "OK",

				// Audit action labels — auth
				"auth.login.success": "Login succeeded",
				"auth.login.failed":  "Login failed",
				"auth.logout":        "Logged out",
				"auth.token.revoke":  "Token revoked",

				// Audit action labels — SSO
				"sso.authorize.success": "SSO authorization granted",
				"sso.authorize.denied":  "SSO authorization denied",

				// Audit action labels — Feishu sync
				"sync.feishu.started":  "Feishu sync started",
				"sync.feishu.finished": "Feishu sync finished",
				"sync.feishu.failed":   "Feishu sync failed",

				// Audit action labels — user sync
				"sync.user.created":       "User created via sync",
				"sync.user.disabled":      "User disabled via sync",
				"sync.user.archived":      "User archived via sync",
				"sync.department.updated": "Department updated via sync",

				// Audit action labels — user management
				"user.updated":          "User updated",
				"user.disabled":         "User disabled",
				"user.archived":         "User archived",
				"user.bound_identity":   "Identity bound to user",
				"user.unbound_identity": "Identity unbound from user",

				// Audit action labels — roles
				"role.created":            "Role created",
				"role.updated":            "Role updated",
				"role.permission_changed": "Role permissions changed",

				// Audit action labels — applications & clients
				"application.assignment_changed": "Application assignment changed",
				"application.created":            "Application created",
				"oidc_client.updated":            "OIDC client updated",
				"secret.rotated":                 "Secret rotated",

				// Permission types
				"permission.type.api":    "API",
				"permission.type.menu":   "Menu",
				"permission.type.action": "Action",
				"permission.type.data":   "Data",

				// User lifecycle statuses
				"status.active":   "Active",
				"status.disabled": "Disabled",
				"status.locked":   "Locked",
				"status.deleted":  "Deleted",

				// User types
				"user.type.employee":        "Employee",
				"user.type.contractor":      "Contractor",
				"user.type.service_account": "Service Account",

				// Common error messages
				"error.unauthorized":          "Unauthorized",
				"error.forbidden":             "Forbidden",
				"error.not_found":             "Not found",
				"error.validation_failed":     "Validation failed",
				"error.internal_server_error": "Internal server error",
				"error.bad_request":           "Bad request",
				"error.conflict":              "Conflict",

				// Common UI labels
				"label.users":            "Users",
				"label.roles":            "Roles",
				"label.permissions":      "Permissions",
				"label.applications":     "Applications",
				"label.audit_logs":       "Audit Logs",
				"label.sync_jobs":        "Sync Jobs",
				"label.identity_sources": "Identity Sources",
				"label.resource_scopes":  "Resource Scopes",
				"label.settings":         "Settings",
				"label.dashboard":        "Dashboard",
			},

			LocaleChineseCN: {
				// Health
				"health.ok": "正常",

				// Audit action labels — auth
				"auth.login.success": "登录成功",
				"auth.login.failed":  "登录失败",
				"auth.logout":        "已登出",
				"auth.token.revoke":  "令牌已撤销",

				// Audit action labels — SSO
				"sso.authorize.success": "单点登录授权成功",
				"sso.authorize.denied":  "单点登录授权被拒",

				// Audit action labels — Feishu sync
				"sync.feishu.started":  "飞书同步已启动",
				"sync.feishu.finished": "飞书同步已完成",
				"sync.feishu.failed":   "飞书同步失败",

				// Audit action labels — user sync
				"sync.user.created":       "同步创建用户",
				"sync.user.disabled":      "同步禁用用户",
				"sync.user.archived":      "同步归档用户",
				"sync.department.updated": "同步更新部门",

				// Audit action labels — user management
				"user.updated":          "用户已更新",
				"user.disabled":         "用户已禁用",
				"user.archived":         "用户已归档",
				"user.bound_identity":   "已绑定身份",
				"user.unbound_identity": "已解绑身份",

				// Audit action labels — roles
				"role.created":            "角色已创建",
				"role.updated":            "角色已更新",
				"role.permission_changed": "角色权限已变更",

				// Audit action labels — applications & clients
				"application.assignment_changed": "应用分配已变更",
				"application.created":            "应用已创建",
				"oidc_client.updated":            "OIDC 客户端已更新",
				"secret.rotated":                 "密钥已轮换",

				// Permission types
				"permission.type.api":    "接口",
				"permission.type.menu":   "菜单",
				"permission.type.action": "操作",
				"permission.type.data":   "数据",

				// User lifecycle statuses
				"status.active":   "启用",
				"status.disabled": "禁用",
				"status.locked":   "锁定",
				"status.deleted":  "已删除",

				// User types
				"user.type.employee":        "正式员工",
				"user.type.contractor":      "外部人员",
				"user.type.service_account": "服务账号",

				// Common error messages
				"error.unauthorized":          "未授权",
				"error.forbidden":             "禁止访问",
				"error.not_found":             "未找到",
				"error.validation_failed":     "校验失败",
				"error.internal_server_error": "服务器内部错误",
				"error.bad_request":           "请求错误",
				"error.conflict":              "冲突",

				// Common UI labels
				"label.users":            "用户",
				"label.roles":            "角色",
				"label.permissions":      "权限",
				"label.applications":     "应用",
				"label.audit_logs":       "审计日志",
				"label.sync_jobs":        "同步任务",
				"label.identity_sources": "身份源",
				"label.resource_scopes":  "资源范围",
				"label.settings":         "设置",
				"label.dashboard":        "工作台",
			},
		},
	}
}

func (c Catalog) Message(locale string, code string) string {
	if messages, ok := c.messages[locale]; ok {
		if message, ok := messages[code]; ok {
			return message
		}
	}

	if messages, ok := c.messages[LocaleEnglishUS]; ok {
		if message, ok := messages[code]; ok {
			return message
		}
	}

	return code
}
