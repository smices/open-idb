// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/smices/open-idb/internal/db/generated"
)

type Service struct {
	queries *generated.Queries
}

type LoginResult struct {
	UserID             string
	EntityID           string
	Username           string
	DisplayName        string
	MustChangePassword bool
	WeakPassword       bool
}

func NewService(queries *generated.Queries) (*Service, error) {
	if queries == nil {
		return nil, fmt.Errorf("queries are required")
	}
	return &Service{queries: queries}, nil
}

func (s *Service) AuthenticateLocal(ctx context.Context, username string, password string) (LoginResult, error) {
	if username == "" || password == "" {
		return LoginResult{}, fmt.Errorf("username and password are required")
	}
	row, err := s.queries.AuthenticateLocalUser(ctx, generated.AuthenticateLocalUserParams{
		Username: username,
		Crypt:    password,
	})
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{
		UserID:             ulidString(row.ID),
		EntityID:           ulidString(row.EntityID),
		Username:           row.Username,
		DisplayName:        row.DisplayName,
		MustChangePassword: row.MustChangePassword,
		WeakPassword:       row.WeakPassword,
	}, nil
}

func (s *Service) AuthenticateLocalWithEntity(ctx context.Context, entityID string, username string, password string) (LoginResult, error) {
	if entityID == "" || username == "" || password == "" {
		return LoginResult{}, fmt.Errorf("entity_id, username and password are required")
	}
	entityULID, err := resolveEntityRef(ctx, s.queries, entityID)
	if err != nil {
		return LoginResult{}, fmt.Errorf("invalid entity_id: %w", err)
	}
	row, err := s.queries.AuthenticateLocalUserByEntity(ctx, generated.AuthenticateLocalUserByEntityParams{
		EntityID: entityULID,
		Username: username,
		Crypt:    password,
	})
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{
		UserID:             ulidString(row.ID),
		EntityID:           ulidString(row.EntityID),
		Username:           row.Username,
		DisplayName:        row.DisplayName,
		MustChangePassword: row.MustChangePassword,
		WeakPassword:       row.WeakPassword,
	}, nil
}

func (s *Service) CreateLoginSession(ctx context.Context, result LoginResult, meta SessionMetadata) (Session, error) {
	session := Session{
		UserID:             result.UserID,
		EntityID:           result.EntityID,
		Username:           result.Username,
		DisplayName:        result.DisplayName,
		MustChangePassword: result.MustChangePassword,
		WeakPassword:       result.WeakPassword,
	}
	if meta.TTL <= 0 {
		meta.TTL = 24 * time.Hour
	}
	return createSessionValue(ctx, s.queries, session, meta)
}

func (s *Service) RevokeLoginSession(ctx context.Context, entityID, sessionID string) error {
	if strings.TrimSpace(entityID) == "" || strings.TrimSpace(sessionID) == "" {
		return fmt.Errorf("entity_id and session_id are required")
	}
	return s.queries.RevokeSession(ctx, generated.RevokeSessionParams{
		EntityID: entityID,
		ID:       sessionID,
	})
}

func (s *Service) GetLoginContextEntityBySlug(ctx context.Context, slug string) (LoginContextEntity, error) {
	entity, err := s.queries.GetEntityBySlug(ctx, slug)
	if err != nil {
		return LoginContextEntity{}, err
	}
	return LoginContextEntity{
		ID:           ulidString(entity.ID),
		Slug:         entity.Slug,
		Name:         entity.Name,
		BrandName:    entity.BrandName,
		LogoURL:      entity.LogoUrl,
		LoginMessage: entity.LoginMessage,
	}, nil
}

func (s *Service) GetLoginContextApplicationByClientID(ctx context.Context, clientID string) (LoginContext, error) {
	row, err := s.queries.GetLoginContextByOIDCClientID(ctx, clientID)
	if err != nil {
		return LoginContext{}, err
	}
	return LoginContext{
		Mode: LoginModeApp,
		Entity: &LoginContextEntity{
			ID:           ulidString(row.EntityID),
			Slug:         row.EntitySlug,
			Name:         row.EntityName,
			BrandName:    row.EntityBrandName,
			LogoURL:      row.EntityLogoUrl,
			LoginMessage: row.EntityLoginMessage,
		},
		Application: &LoginContextApplication{
			ID:   ulidString(row.ApplicationID),
			Name: row.ApplicationName,
		},
		Methods:              []string{"password", "feishu"},
		AllowEntitySelection: false,
		Reason:               "application",
	}, nil
}

func (s *Service) GetDefaultLoginContext(ctx context.Context) (LoginContext, error) {
	row, err := s.queries.GetDefaultLoginContextEntity(ctx)
	if err != nil {
		return LoginContext{}, err
	}
	return LoginContext{
		Mode: LoginModeUser,
		Entity: &LoginContextEntity{
			ID:           ulidString(row.EntityID),
			Slug:         row.EntitySlug,
			Name:         row.EntityName,
			BrandName:    row.EntityBrandName,
			LogoURL:      row.EntityLogoUrl,
			LoginMessage: row.EntityLoginMessage,
		},
		Methods:              []string{"password", "feishu"},
		AllowEntitySelection: false,
		Reason:               "default_entity",
	}, nil
}
