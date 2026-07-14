// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminService struct {
	pool *pgxpool.Pool
}

type AdminManagementError struct {
	Status  int
	Code    string
	Message string
}

func (e AdminManagementError) Error() string {
	return e.Message
}

func NewAdminService(pool *pgxpool.Pool) *AdminService {
	return &AdminService{pool: pool}
}

func (s *AdminService) AuthenticateAdmin(ctx context.Context, username string, password string) (AdminLoginResult, error) {
	if s == nil || s.pool == nil {
		return AdminLoginResult{}, fmt.Errorf("admin auth service is not configured")
	}
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return AdminLoginResult{}, fmt.Errorf("username and password are required")
	}
	const query = `
SELECT
  au.id,
  COALESCE(
    au.entity_id,
    CASE
      WHEN au.role = 'platform_admin' THEN (
        SELECT be.id
        FROM business_entities be
        WHERE be.status = 'active'
        ORDER BY be.created_at, be.id
        LIMIT 1
      )
    END,
    ''
  ) AS entity_id,
  au.username,
  au.display_name,
  au.role
FROM admin_users au
JOIN admin_credentials ac ON ac.admin_user_id = au.id
WHERE au.username = $1
  AND au.status = 'active'
  AND ac.password_hash = crypt($2, ac.password_hash)
`
	var result AdminLoginResult
	if err := s.pool.QueryRow(ctx, query, username, password).Scan(&result.AdminID, &result.EntityID, &result.Username, &result.DisplayName, &result.Role); err != nil {
		return AdminLoginResult{}, err
	}
	return result, nil
}

func (s *AdminService) CreateAdminSession(ctx context.Context, result AdminLoginResult, meta SessionMetadata) (AdminSession, error) {
	if s == nil || s.pool == nil {
		return AdminSession{}, fmt.Errorf("admin auth service is not configured")
	}
	if meta.TTL <= 0 {
		meta.TTL = 24 * time.Hour
	}
	if meta.LoginMethod == "" {
		meta.LoginMethod = "password"
	}
	expiresAt := time.Now().Add(meta.TTL)
	const query = `
INSERT INTO admin_sessions (admin_user_id, device_id, ip, user_agent, login_method, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, expires_at
`
	session := AdminSession{
		AdminID:     result.AdminID,
		EntityID:    result.EntityID,
		Username:    result.Username,
		DisplayName: result.DisplayName,
		Role:        result.Role,
		ExpiresAt:   expiresAt,
	}
	var expires pgtype.Timestamptz
	err := s.pool.QueryRow(ctx, query, result.AdminID, meta.DeviceID, meta.IP, meta.UserAgent, meta.LoginMethod, expiresAt).Scan(&session.ID, &expires)
	if err != nil {
		return AdminSession{}, err
	}
	if expires.Valid {
		session.ExpiresAt = expires.Time
	}
	return session, nil
}

func (s *AdminService) RevokeAdminSession(ctx context.Context, sessionID string) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("admin auth service is not configured")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil
	}
	_, err := s.pool.Exec(ctx, `
UPDATE admin_sessions
SET revoked_at = COALESCE(revoked_at, now())
WHERE id = $1
`, sessionID)
	return err
}

func (s *AdminService) CurrentAdmin(ctx context.Context, session AdminSession) (AdminCurrentUser, error) {
	if s == nil || s.pool == nil {
		return AdminCurrentUser{}, fmt.Errorf("admin auth service is not configured")
	}
	const query = `
SELECT id, COALESCE(entity_id, '') AS entity_id, username, display_name, role
FROM admin_users
WHERE id = $1 AND status = 'active'
`
	var user AdminCurrentUser
	if err := s.pool.QueryRow(ctx, query, session.AdminID).Scan(&user.ID, &user.EntityID, &user.Username, &user.DisplayName, &user.Role); err != nil {
		return AdminCurrentUser{}, err
	}
	return user, nil
}

func (s *AdminService) UpdateAdminProfile(ctx context.Context, session AdminSession, displayName string) (AdminCurrentUser, error) {
	if s == nil || s.pool == nil {
		return AdminCurrentUser{}, fmt.Errorf("admin auth service is not configured")
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return AdminCurrentUser{}, fmt.Errorf("display_name is required")
	}
	const query = `
UPDATE admin_users
SET display_name = $2,
    updated_at = now()
WHERE id = $1 AND status = 'active'
RETURNING id, COALESCE(entity_id, '') AS entity_id, username, display_name, role
`
	var user AdminCurrentUser
	if err := s.pool.QueryRow(ctx, query, session.AdminID, displayName).Scan(&user.ID, &user.EntityID, &user.Username, &user.DisplayName, &user.Role); err != nil {
		return AdminCurrentUser{}, err
	}
	return user, nil
}

func (s *AdminService) UpdateAdminPassword(ctx context.Context, session AdminSession, currentPassword string, newPassword string) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("admin auth service is not configured")
	}
	if currentPassword == "" || newPassword == "" {
		return fmt.Errorf("current password and new password are required")
	}
	const query = `
UPDATE admin_credentials
SET password_hash = crypt($3, gen_salt('bf')),
    updated_at = now()
WHERE admin_user_id = $1
  AND password_hash = crypt($2, password_hash)
`
	tag, err := s.pool.Exec(ctx, query, session.AdminID, currentPassword, newPassword)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("current password is invalid")
	}
	return nil
}

func (s *AdminService) ListManagedAdminUsers(ctx context.Context, session AdminSession) (AdminUserListResponse, error) {
	if err := s.requirePlatformAdmin(ctx, session); err != nil {
		return AdminUserListResponse{}, err
	}
	const query = `
SELECT
  au.id,
  COALESCE(au.entity_id, '') AS entity_id,
  COALESCE(be.name, '') AS entity_name,
  au.username,
  au.display_name,
  COALESCE(au.email, '') AS email,
  au.status,
  au.role,
  au.created_at,
  au.updated_at
FROM admin_users au
LEFT JOIN business_entities be ON be.id = au.entity_id
ORDER BY
  CASE WHEN au.username = 'admin' THEN 0 ELSE 1 END,
  au.created_at ASC,
  au.username ASC
`
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return AdminUserListResponse{}, err
	}
	defer rows.Close()
	items := make([]AdminUserSummary, 0)
	for rows.Next() {
		user, err := scanAdminUserSummary(rows)
		if err != nil {
			return AdminUserListResponse{}, err
		}
		items = append(items, user)
	}
	if err := rows.Err(); err != nil {
		return AdminUserListResponse{}, err
	}
	return AdminUserListResponse{Items: items, Total: len(items)}, nil
}

func (s *AdminService) ListAssignableAdminRoles(ctx context.Context, session AdminSession) ([]AdminRoleOption, error) {
	if err := s.requirePlatformAdmin(ctx, session); err != nil {
		return nil, err
	}
	return []AdminRoleOption{
		{
			Value:       "platform_admin",
			Label:       "系统管理员",
			Description: "管理平台级配置、公司、管理员和审计。",
		},
		{
			Value:          "enterprise_admin",
			Label:          "公司管理员",
			Description:    "管理绑定公司的账号、应用和身份接入。",
			RequiresEntity: true,
		},
	}, nil
}

func (s *AdminService) CreateManagedAdminUser(ctx context.Context, session AdminSession, request AdminUserCreateRequest) (AdminUserSummary, error) {
	if err := s.requirePlatformAdmin(ctx, session); err != nil {
		return AdminUserSummary{}, err
	}
	username := strings.ToLower(strings.TrimSpace(request.Username))
	if username == "" || strings.EqualFold(username, "admin") {
		return AdminUserSummary{}, adminBadRequest("invalid_admin_username", "username is not allowed")
	}
	displayName := strings.TrimSpace(request.DisplayName)
	if displayName == "" {
		displayName = username
	}
	email := strings.TrimSpace(request.Email)
	role, entityID, err := normalizeAdminRoleScope(request.Role, request.EntityID)
	if err != nil {
		return AdminUserSummary{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return AdminUserSummary{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const insertUser = `
INSERT INTO admin_users (username, display_name, email, status, role, entity_id, locale)
VALUES ($1, $2, NULLIF($3, ''), 'active', $4, NULLIF($5, ''), 'zh-CN')
RETURNING id, COALESCE(entity_id, '') AS entity_id, username, display_name, COALESCE(email, '') AS email, status, role, created_at, updated_at
`
	var created struct {
		id          string
		entityID    string
		username    string
		displayName string
		email       string
		status      string
		role        string
		createdAt   time.Time
		updatedAt   time.Time
	}
	if err := tx.QueryRow(ctx, insertUser, username, displayName, email, role, entityID).Scan(
		&created.id,
		&created.entityID,
		&created.username,
		&created.displayName,
		&created.email,
		&created.status,
		&created.role,
		&created.createdAt,
		&created.updatedAt,
	); err != nil {
		return AdminUserSummary{}, err
	}
	const insertCredential = `
INSERT INTO admin_credentials (admin_user_id, password_hash, must_change_password, weak_password)
VALUES ($1, crypt($2, gen_salt('bf')), true, false)
`
	if _, err := tx.Exec(ctx, insertCredential, created.id, request.Password); err != nil {
		return AdminUserSummary{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return AdminUserSummary{}, err
	}
	return s.GetManagedAdminUser(ctx, created.id)
}

func (s *AdminService) UpdateManagedAdminUser(ctx context.Context, session AdminSession, id string, request AdminUserUpdateRequest) (AdminUserSummary, error) {
	if err := s.requirePlatformAdmin(ctx, session); err != nil {
		return AdminUserSummary{}, err
	}
	target, err := s.GetManagedAdminUser(ctx, strings.TrimSpace(id))
	if err != nil {
		return AdminUserSummary{}, err
	}
	if target.Protected {
		return AdminUserSummary{}, adminForbidden("admin_user_protected", "built-in admin account can only change password")
	}
	if err := validateAdminUserUpdate(session, target, request); err != nil {
		return AdminUserSummary{}, err
	}
	displayName := strings.TrimSpace(request.DisplayName)
	if displayName == "" {
		displayName = target.Username
	}
	status := strings.TrimSpace(request.Status)
	if status == "" {
		status = "active"
	}
	if status != "active" && status != "disabled" && status != "locked" {
		return AdminUserSummary{}, adminBadRequest("invalid_admin_status", "status is invalid")
	}
	role, entityID, err := normalizeAdminRoleScope(request.Role, request.EntityID)
	if err != nil {
		return AdminUserSummary{}, err
	}
	const query = `
UPDATE admin_users
SET display_name = $2,
    email = NULLIF($3, ''),
    status = $4,
    role = $5,
    entity_id = NULLIF($6, ''),
    updated_at = now()
WHERE id = $1
RETURNING id
`
	var updatedID string
	if err := s.pool.QueryRow(ctx, query, target.ID, displayName, strings.TrimSpace(request.Email), status, role, entityID).Scan(&updatedID); err != nil {
		return AdminUserSummary{}, err
	}
	return s.GetManagedAdminUser(ctx, updatedID)
}

func (s *AdminService) DeleteManagedAdminUser(ctx context.Context, session AdminSession, id string) (AdminUserSummary, error) {
	if err := s.requirePlatformAdmin(ctx, session); err != nil {
		return AdminUserSummary{}, err
	}
	target, err := s.GetManagedAdminUser(ctx, strings.TrimSpace(id))
	if err != nil {
		return AdminUserSummary{}, err
	}
	if target.Protected {
		return AdminUserSummary{}, adminForbidden("admin_user_protected", "built-in admin account cannot be deleted")
	}
	if target.ID == session.AdminID {
		return AdminUserSummary{}, adminForbidden("admin_user_self_delete", "current admin account cannot delete itself")
	}
	const query = `DELETE FROM admin_users WHERE id = $1`
	tag, err := s.pool.Exec(ctx, query, target.ID)
	if err != nil {
		return AdminUserSummary{}, err
	}
	if tag.RowsAffected() == 0 {
		return AdminUserSummary{}, adminNotFound("admin_user_not_found", "admin user was not found")
	}
	return target, nil
}

func (s *AdminService) SetManagedAdminPassword(ctx context.Context, session AdminSession, id string, password string) error {
	if err := s.requirePlatformAdmin(ctx, session); err != nil {
		return err
	}
	target, err := s.GetManagedAdminUser(ctx, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	const query = `
UPDATE admin_credentials
SET password_hash = crypt($2, gen_salt('bf')),
    must_change_password = true,
    weak_password = false,
    password_updated_at = now(),
    updated_at = now()
WHERE admin_user_id = $1
`
	tag, err := s.pool.Exec(ctx, query, target.ID, password)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return adminNotFound("admin_credential_not_found", "admin credential was not found")
	}
	return nil
}

func (s *AdminService) GetManagedAdminUser(ctx context.Context, id string) (AdminUserSummary, error) {
	if s == nil || s.pool == nil {
		return AdminUserSummary{}, fmt.Errorf("admin auth service is not configured")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return AdminUserSummary{}, adminNotFound("admin_user_not_found", "admin user was not found")
	}
	const query = `
SELECT
  au.id,
  COALESCE(au.entity_id, '') AS entity_id,
  COALESCE(be.name, '') AS entity_name,
  au.username,
  au.display_name,
  COALESCE(au.email, '') AS email,
  au.status,
  au.role,
  au.created_at,
  au.updated_at
FROM admin_users au
LEFT JOIN business_entities be ON be.id = au.entity_id
WHERE au.id = $1
`
	user, err := scanAdminUserSummary(s.pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return AdminUserSummary{}, adminNotFound("admin_user_not_found", "admin user was not found")
		}
		return AdminUserSummary{}, err
	}
	return user, nil
}

func (s *AdminService) requirePlatformAdmin(ctx context.Context, session AdminSession) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("admin auth service is not configured")
	}
	current, err := s.CurrentAdmin(ctx, session)
	if err != nil {
		return adminForbidden("admin_session_invalid", "admin session is invalid")
	}
	if current.Role != "platform_admin" {
		return adminForbidden("system_admin_required", "system administrator role is required")
	}
	return nil
}

type adminUserScanner interface {
	Scan(dest ...any) error
}

func scanAdminUserSummary(scanner adminUserScanner) (AdminUserSummary, error) {
	var user AdminUserSummary
	var createdAt time.Time
	var updatedAt time.Time
	if err := scanner.Scan(
		&user.ID,
		&user.EntityID,
		&user.EntityName,
		&user.Username,
		&user.DisplayName,
		&user.Email,
		&user.Status,
		&user.Role,
		&createdAt,
		&updatedAt,
	); err != nil {
		return AdminUserSummary{}, err
	}
	user.Protected = user.Username == "admin"
	user.CreatedAt = createdAt.Format(time.RFC3339)
	user.UpdatedAt = updatedAt.Format(time.RFC3339)
	return user, nil
}

func normalizeAdminRoleScope(role string, entityID string) (string, string, error) {
	role = strings.TrimSpace(role)
	entityID = strings.TrimSpace(entityID)
	switch role {
	case "platform_admin":
		return role, "", nil
	case "enterprise_admin":
		if entityID == "" {
			return "", "", adminBadRequest("entity_required", "company is required for company administrator")
		}
		return role, entityID, nil
	default:
		return "", "", adminBadRequest("invalid_admin_role", "admin role is invalid")
	}
}

func validateAdminUserUpdate(session AdminSession, target AdminUserSummary, request AdminUserUpdateRequest) error {
	if target.ID != session.AdminID {
		return nil
	}
	requestedRole := strings.TrimSpace(request.Role)
	if requestedRole == "" {
		requestedRole = target.Role
	}
	requestedStatus := strings.TrimSpace(request.Status)
	if requestedStatus == "" {
		requestedStatus = target.Status
	}
	if requestedRole != target.Role {
		return adminForbidden("admin_user_self_role_change", "current admin account cannot change its own role")
	}
	if requestedStatus != target.Status {
		return adminForbidden("admin_user_self_status_change", "current admin account cannot change its own status")
	}
	return nil
}

func adminBadRequest(code string, message string) AdminManagementError {
	return AdminManagementError{Status: http.StatusBadRequest, Code: code, Message: message}
}

func adminForbidden(code string, message string) AdminManagementError {
	return AdminManagementError{Status: http.StatusForbidden, Code: code, Message: message}
}

func adminNotFound(code string, message string) AdminManagementError {
	return AdminManagementError{Status: http.StatusNotFound, Code: code, Message: message}
}

type DatabaseAdminSessionResolver struct {
	pool *pgxpool.Pool
}

func NewDatabaseAdminSessionResolver(pool *pgxpool.Pool) *DatabaseAdminSessionResolver {
	return &DatabaseAdminSessionResolver{pool: pool}
}

func (r *DatabaseAdminSessionResolver) ResolveAdminSession(ctx context.Context, sessionID string) (AdminSession, error) {
	if r == nil || r.pool == nil {
		return AdminSession{}, fmt.Errorf("admin session resolver is not configured")
	}
	const query = `
SELECT
  s.id,
  au.id,
  COALESCE(
    au.entity_id,
    CASE
      WHEN au.role = 'platform_admin' THEN (
        SELECT be.id
        FROM business_entities be
        WHERE be.status = 'active'
        ORDER BY be.created_at, be.id
        LIMIT 1
      )
    END,
    ''
  ) AS entity_id,
  au.username,
  au.display_name,
  au.role,
  s.expires_at
FROM admin_sessions s
JOIN admin_users au ON au.id = s.admin_user_id
WHERE s.id = $1
  AND s.revoked_at IS NULL
  AND s.expires_at > now()
  AND au.status = 'active'
`
	var session AdminSession
	var expires pgtype.Timestamptz
	err := r.pool.QueryRow(ctx, query, strings.TrimSpace(sessionID)).Scan(
		&session.ID,
		&session.AdminID,
		&session.EntityID,
		&session.Username,
		&session.DisplayName,
		&session.Role,
		&expires,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return AdminSession{}, err
		}
		return AdminSession{}, err
	}
	if expires.Valid {
		session.ExpiresAt = expires.Time
	}
	return session, nil
}
