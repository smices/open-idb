// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/smices/open-idb/internal/db/generated"
)

type Session struct {
	ID                 string
	UserID             string
	EntityID           string
	Username           string
	DisplayName        string
	MustChangePassword bool
	WeakPassword       bool
	ExpiresAt          time.Time
}

type AdminSession struct {
	ID          string
	AdminID     string
	EntityID     string
	Username    string
	DisplayName string
	Role        string
	ExpiresAt   time.Time
}

type SessionMetadata struct {
	LoginMethod string
	DeviceID    string
	IP          string
	UserAgent   string
	TTL         time.Duration
}

type SessionResolver interface {
	ResolveSession(ctx context.Context, sessionID string) (Session, error)
}

type AdminSessionResolver interface {
	ResolveAdminSession(ctx context.Context, sessionID string) (AdminSession, error)
}

var configuredSessionResolver struct {
	mu            sync.RWMutex
	resolver      SessionResolver
	adminResolver AdminSessionResolver
}

func SetSessionResolver(resolver SessionResolver) {
	configuredSessionResolver.mu.Lock()
	defer configuredSessionResolver.mu.Unlock()
	configuredSessionResolver.resolver = resolver
}

func SetAdminSessionResolver(resolver AdminSessionResolver) {
	configuredSessionResolver.mu.Lock()
	defer configuredSessionResolver.mu.Unlock()
	configuredSessionResolver.adminResolver = resolver
}

func ResolveSession(ctx context.Context, value string) (Session, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return Session{}, fmt.Errorf("session id is required")
	}
	configuredSessionResolver.mu.RLock()
	resolver := configuredSessionResolver.resolver
	configuredSessionResolver.mu.RUnlock()
	if resolver != nil {
		return resolver.ResolveSession(ctx, value)
	}
	return DecodeSession(value)
}

func ResolveAdminSession(ctx context.Context, value string) (AdminSession, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return AdminSession{}, fmt.Errorf("admin session id is required")
	}
	configuredSessionResolver.mu.RLock()
	resolver := configuredSessionResolver.adminResolver
	configuredSessionResolver.mu.RUnlock()
	if resolver != nil {
		return resolver.ResolveAdminSession(ctx, value)
	}
	return DecodeAdminSession(value)
}

type DatabaseSessionResolver struct {
	queries *generated.Queries
}

func NewDatabaseSessionResolver(queries *generated.Queries) *DatabaseSessionResolver {
	return &DatabaseSessionResolver{queries: queries}
}

func (r *DatabaseSessionResolver) ResolveSession(ctx context.Context, sessionID string) (Session, error) {
	if r == nil || r.queries == nil {
		return Session{}, fmt.Errorf("session resolver is not configured")
	}
	row, err := r.queries.GetActiveSessionIdentity(ctx, sessionID)
	if err != nil {
		return Session{}, err
	}
	session := Session{
		ID:                 row.ID,
		UserID:             row.UserID,
		EntityID:           row.EntityID,
		Username:           row.Username,
		DisplayName:        row.DisplayName,
		MustChangePassword: row.MustChangePassword,
		WeakPassword:       row.WeakPassword,
	}
	if row.ExpiresAt.Valid {
		session.ExpiresAt = row.ExpiresAt.Time
	}
	return session, nil
}

type sessionCreator interface {
	CreateSession(ctx context.Context, arg generated.CreateSessionParams) (generated.Session, error)
}

func createSessionValue(ctx context.Context, queries any, session Session, meta SessionMetadata) (Session, error) {
	if meta.TTL <= 0 {
		meta.TTL = 24 * time.Hour
	}
	if meta.LoginMethod == "" {
		meta.LoginMethod = "password"
	}
	session.ExpiresAt = time.Now().Add(meta.TTL)
	if creator, ok := queries.(sessionCreator); ok {
		row, err := creator.CreateSession(ctx, generated.CreateSessionParams{
			EntityID:    session.EntityID,
			UserID:      session.UserID,
			DeviceID:    meta.DeviceID,
			Ip:          meta.IP,
			UserAgent:   meta.UserAgent,
			LoginMethod: meta.LoginMethod,
			ExpiresAt: pgtype.Timestamptz{
				Time:  session.ExpiresAt,
				Valid: true,
			},
		})
		if err != nil {
			return Session{}, err
		}
		session.ID = row.ID
		session.EntityID = row.EntityID
		session.UserID = row.UserID
		if row.ExpiresAt.Valid {
			session.ExpiresAt = row.ExpiresAt.Time
		}
		return session, nil
	}
	value, err := EncodeSession(session)
	if err != nil {
		return Session{}, err
	}
	session.ID = value
	return session, nil
}

func EncodeSession(session Session) (string, error) {
	payload, err := json.Marshal(session)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func EncodeAdminSession(session AdminSession) (string, error) {
	payload, err := json.Marshal(session)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func DecodeSession(value string) (Session, error) {
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return Session{}, err
	}
	var session Session
	if err := json.Unmarshal(payload, &session); err != nil {
		return Session{}, err
	}
	if session.UserID == "" || session.EntityID == "" || session.Username == "" {
		return Session{}, fmt.Errorf("session is missing required identity fields")
	}
	return session, nil
}

func DecodeAdminSession(value string) (AdminSession, error) {
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return AdminSession{}, err
	}
	var session AdminSession
	if err := json.Unmarshal(payload, &session); err != nil {
		return AdminSession{}, err
	}
	if session.AdminID == "" || session.Username == "" {
		return AdminSession{}, fmt.Errorf("admin session is missing required identity fields")
	}
	return session, nil
}
