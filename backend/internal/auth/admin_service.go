// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminService struct {
	pool *pgxpool.Pool
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
