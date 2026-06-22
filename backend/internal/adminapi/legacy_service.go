// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/smices/open-idb/internal/db/generated"
	"golang.org/x/crypto/bcrypt"
)

// LegacyAppUserResponse represents a legacy username/password mapping entry.
type LegacyAppUserResponse struct {
	ID                   string `json:"id"`
	EntityID             string `json:"entity_id"`
	ApplicationID        string `json:"application_id"`
	UserID               string `json:"user_id"`
	Username             string `json:"username"`
	LegacyUserIdentifier string `json:"legacy_user_identifier"`
	AuthScheme           string `json:"auth_scheme"`
	IsActive             bool   `json:"is_active"`
	LastUsedAt           string `json:"last_used_at"`
	CreatedAt            string `json:"created_at"`
	UpdatedAt            string `json:"updated_at"`
}

// LegacyAppUserCreateInput is the payload for creating a legacy mapping.
type LegacyAppUserCreateInput struct {
	Username             string
	UserID               string
	Password             string
	LegacyUserIdentifier string
	IsActive             *bool
}

// LegacyAppUserUpdateInput is the payload for updating a legacy mapping.
type LegacyAppUserUpdateInput struct {
	UserID               *string
	Password             *string
	LegacyUserIdentifier *string
	IsActive             *bool
}

func legacyTimestamp(v pgtype.Timestamptz) string {
	if !v.Valid {
		return ""
	}
	return v.Time.Format(time.RFC3339)
}

func legacyAppUserFromRow(row generated.LegacyAppUser) LegacyAppUserResponse {
	return LegacyAppUserResponse{
		ID:                   ulidString(row.ID),
		EntityID:             ulidString(row.EntityID),
		ApplicationID:        ulidString(row.ApplicationID),
		UserID:               ulidString(row.UserID),
		Username:             row.Username,
		LegacyUserIdentifier: textString(row.LegacyUserIdentifier),
		AuthScheme:           row.AuthScheme,
		IsActive:             row.IsActive,
		LastUsedAt:           legacyTimestamp(row.LastUsedAt),
		CreatedAt:            legacyTimestamp(row.CreatedAt),
		UpdatedAt:            legacyTimestamp(row.UpdatedAt),
	}
}

func legacyAppUserFromListRow(row generated.ListLegacyAppUsersByApplicationRow) LegacyAppUserResponse {
	return LegacyAppUserResponse{
		ID:                   ulidString(row.ID),
		EntityID:             ulidString(row.EntityID),
		ApplicationID:        ulidString(row.ApplicationID),
		UserID:               ulidString(row.UserID),
		Username:             row.Username,
		LegacyUserIdentifier: textString(row.LegacyUserIdentifier),
		AuthScheme:           row.AuthScheme,
		IsActive:             row.IsActive,
		LastUsedAt:           legacyTimestamp(row.LastUsedAt),
		CreatedAt:            legacyTimestamp(row.CreatedAt),
		UpdatedAt:            legacyTimestamp(row.UpdatedAt),
	}
}

func (s *AdminService) ListLegacyAppUsers(ctx context.Context, entityID, appID string, limit, offset int32) ([]LegacyAppUserResponse, error) {
	rows, err := s.queries.ListLegacyAppUsersByApplication(ctx, generated.ListLegacyAppUsersByApplicationParams{
		EntityID:      entityID,
		ApplicationID: appID,
		Limit:         limit,
		Offset:        offset,
	})
	if err != nil {
		return nil, err
	}
	users := make([]LegacyAppUserResponse, 0, len(rows))
	for _, row := range rows {
		users = append(users, legacyAppUserFromListRow(row))
	}
	return users, nil
}

func (s *AdminService) CountLegacyAppUsers(ctx context.Context, entityID, appID string) (int64, error) {
	return s.queries.CountLegacyAppUsersByApplication(ctx, generated.CountLegacyAppUsersByApplicationParams{
		EntityID:      entityID,
		ApplicationID: appID,
	})
}

func (s *AdminService) GetLegacyAppUser(ctx context.Context, entityID, appID string, username string) (LegacyAppUserResponse, error) {
	row, err := s.queries.GetLegacyAppUserByUsername(ctx, generated.GetLegacyAppUserByUsernameParams{
		EntityID:      entityID,
		ApplicationID: appID,
		Username:      username,
	})
	if err != nil {
		return LegacyAppUserResponse{}, err
	}
	return legacyAppUserFromRow(row), nil
}

func (s *AdminService) CreateLegacyAppUser(ctx context.Context, entityID, appID string, input LegacyAppUserCreateInput) (LegacyAppUserResponse, error) {
	username := strings.TrimSpace(input.Username)
	userInput := strings.TrimSpace(input.UserID)
	password := input.Password
	legacyIdentifier := optionalText(strings.TrimSpace(input.LegacyUserIdentifier))

	if username == "" {
		return LegacyAppUserResponse{}, fmt.Errorf("username is required")
	}
	if userInput == "" {
		return LegacyAppUserResponse{}, fmt.Errorf("user_id is required")
	}
	if password == "" {
		return LegacyAppUserResponse{}, fmt.Errorf("password is required")
	}

	userID, err := ulidValue(userInput)
	if err != nil {
		return LegacyAppUserResponse{}, err
	}
	if _, err := s.queries.GetUserByEntityAndID(ctx, generated.GetUserByEntityAndIDParams{
		EntityID: entityID,
		ID:       userID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LegacyAppUserResponse{}, fmt.Errorf("user not found")
		}
		return LegacyAppUserResponse{}, err
	}
	if _, err := s.queries.GetApplicationByID(ctx, generated.GetApplicationByIDParams{
		EntityID: entityID,
		ID:       appID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LegacyAppUserResponse{}, fmt.Errorf("application not found")
		}
		return LegacyAppUserResponse{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return LegacyAppUserResponse{}, err
	}
	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	row, err := s.queries.UpsertLegacyAppUser(ctx, generated.UpsertLegacyAppUserParams{
		EntityID:             entityID,
		ApplicationID:        appID,
		UserID:               userID,
		Username:             username,
		LegacyUserIdentifier: legacyIdentifier,
		AuthScheme:           "local",
		CredentialHash:       string(hash),
		IsActive:             isActive,
	})
	if err != nil {
		return LegacyAppUserResponse{}, err
	}
	resp := legacyAppUserFromRow(row)
	if err := s.audit.logCreate(ctx, ulidString(entityID), "", "legacy_app_user", resp.ID, resp); err != nil {
		return LegacyAppUserResponse{}, err
	}
	return resp, nil
}

func (s *AdminService) UpdateLegacyAppUser(ctx context.Context, entityID, appID string, username string, input LegacyAppUserUpdateInput) (LegacyAppUserResponse, error) {
	existing, err := s.queries.GetLegacyAppUserByUsername(ctx, generated.GetLegacyAppUserByUsernameParams{
		EntityID:      entityID,
		ApplicationID: appID,
		Username:      username,
	})
	if err != nil {
		return LegacyAppUserResponse{}, err
	}

	userID := existing.UserID
	if input.UserID != nil {
		userInput := strings.TrimSpace(*input.UserID)
		if userInput == "" {
			return LegacyAppUserResponse{}, fmt.Errorf("user_id cannot be empty")
		}
		parsedUserID, err := ulidValue(userInput)
		if err != nil {
			return LegacyAppUserResponse{}, err
		}
		if _, err := s.queries.GetUserByEntityAndID(ctx, generated.GetUserByEntityAndIDParams{
			EntityID: entityID,
			ID:       parsedUserID,
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return LegacyAppUserResponse{}, fmt.Errorf("user not found")
			}
			return LegacyAppUserResponse{}, err
		}
		userID = parsedUserID
	}

	legacyIdentifier := existing.LegacyUserIdentifier
	if input.LegacyUserIdentifier != nil {
		legacyIdentifier = optionalText(strings.TrimSpace(*input.LegacyUserIdentifier))
	}

	credentialHash := existing.CredentialHash
	if input.Password != nil {
		if strings.TrimSpace(*input.Password) == "" {
			return LegacyAppUserResponse{}, fmt.Errorf("password cannot be empty")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(*input.Password), bcrypt.DefaultCost)
		if err != nil {
			return LegacyAppUserResponse{}, err
		}
		credentialHash = string(hash)
	}

	isActive := existing.IsActive
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	before := legacyAppUserFromRow(existing)
	row, err := s.queries.UpsertLegacyAppUser(ctx, generated.UpsertLegacyAppUserParams{
		EntityID:             entityID,
		ApplicationID:        appID,
		UserID:               userID,
		Username:             existing.Username,
		LegacyUserIdentifier: legacyIdentifier,
		AuthScheme:           existing.AuthScheme,
		CredentialHash:       credentialHash,
		IsActive:             isActive,
	})
	if err != nil {
		return LegacyAppUserResponse{}, err
	}
	after := legacyAppUserFromRow(row)
	if err := s.audit.logUpdate(ctx, ulidString(entityID), "", "legacy_app_user", after.ID, before, after); err != nil {
		return LegacyAppUserResponse{}, err
	}
	return after, nil
}

func (s *AdminService) SetLegacyAppUserStatus(ctx context.Context, entityID, appID string, username string, isActive bool) (LegacyAppUserResponse, error) {
	before, err := s.GetLegacyAppUser(ctx, entityID, appID, username)
	if err != nil {
		return LegacyAppUserResponse{}, err
	}
	err = s.queries.SetLegacyAppUserStatus(ctx, generated.SetLegacyAppUserStatusParams{
		EntityID:      entityID,
		ApplicationID: appID,
		Username:      username,
		IsActive:      isActive,
	})
	if err != nil {
		return LegacyAppUserResponse{}, err
	}
	after, err := s.GetLegacyAppUser(ctx, entityID, appID, username)
	if err != nil {
		return LegacyAppUserResponse{}, err
	}
	if err := s.audit.logUpdate(ctx, ulidString(entityID), "", "legacy_app_user", before.ID, before, after); err != nil {
		return LegacyAppUserResponse{}, err
	}
	return after, nil
}

func (s *AdminService) DeleteLegacyAppUser(ctx context.Context, entityID, appID string, username string) error {
	before, err := s.GetLegacyAppUser(ctx, entityID, appID, username)
	if err != nil {
		return err
	}

	if err := s.queries.DeleteLegacyAppUser(ctx, generated.DeleteLegacyAppUserParams{
		EntityID:      entityID,
		ApplicationID: appID,
		Username:      username,
	}); err != nil {
		return err
	}

	if err := s.audit.logDelete(ctx, ulidString(entityID), "", "legacy_app_user", before.ID, before); err != nil {
		return err
	}
	return nil
}
