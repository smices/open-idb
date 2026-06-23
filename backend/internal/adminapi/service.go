// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/smices/open-idb/internal/auth"
	"github.com/smices/open-idb/internal/db/generated"
	"github.com/smices/open-idb/internal/id"
)

type Service struct {
	queries *generated.Queries
}

func NewService(queries *generated.Queries) (*Service, error) {
	if queries == nil {
		return nil, fmt.Errorf("queries are required")
	}
	return &Service{queries: queries}, nil
}

func (s *Service) DashboardSummary(ctx context.Context, session auth.Session) (DashboardSummary, error) {
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		return DashboardSummary{}, err
	}
	row, err := s.queries.GetDashboardSummary(ctx, entityID)
	if err != nil {
		return DashboardSummary{}, err
	}
	return DashboardSummary{
		Users:                row.Users,
		ActiveUsers:          row.ActiveUsers,
		NewUsers:             row.NewUsers,
		ApplicationActivity:  row.ApplicationActivity,
		PendingAuthorization: row.PendingAuthorization,
		SyncHealth:           row.SyncHealth,
	}, nil
}

func (s *Service) CurrentUser(ctx context.Context, session auth.Session) (CurrentUser, error) {
	entityID, userID, err := sessionULIDs(session)
	if err != nil {
		return CurrentUser{}, err
	}
	row, err := s.queries.GetCurrentUserByID(ctx, generated.GetCurrentUserByIDParams{
		EntityID: entityID,
		ID:       userID,
	})
	if err != nil {
		return CurrentUser{}, err
	}
	roles, err := s.queries.GetUserRolesForToken(ctx, generated.GetUserRolesForTokenParams{
		EntityID: entityID,
		UserID:   userID,
	})
	if err != nil {
		return CurrentUser{}, err
	}
	consoleScope, capabilities := consoleAccessForUser(row.Username, roles)
	return CurrentUser{
		ID:                 ulidString(row.ID),
		EntityID:           ulidString(row.EntityID),
		Username:           row.Username,
		DisplayName:        row.DisplayName,
		Email:              textString(row.Email),
		Phone:              textString(row.Phone),
		AvatarURL:          textString(row.AvatarUrl),
		Locale:             localeOrDefault(row.Locale),
		MustChangePassword: boolValue(row.MustChangePassword),
		WeakPassword:       boolValue(row.WeakPassword),
		ConsoleScope:       consoleScope,
		Capabilities:       capabilities,
	}, nil
}

func (s *Service) UpdateProfile(ctx context.Context, input UpdateProfileInput) (CurrentUser, error) {
	entityID, err := ulidValue(input.EntityID)
	if err != nil {
		return CurrentUser{}, err
	}
	userID, err := ulidValue(input.UserID)
	if err != nil {
		return CurrentUser{}, err
	}
	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		return CurrentUser{}, fmt.Errorf("display_name is required")
	}
	if _, err := s.queries.UpdateUser(ctx, generated.UpdateUserParams{
		EntityID: entityID,
		ID:       userID,
		DisplayName: pgtype.Text{
			String: displayName,
			Valid:  true,
		},
	}); err != nil {
		return CurrentUser{}, err
	}
	return s.CurrentUser(ctx, auth.Session{
		EntityID: entityID,
		UserID:   userID,
	})
}

func consoleAccessForUser(username string, roles []string) (string, []string) {
	if username == "admin" {
		return "enterprise_admin", []string{"user", "enterprise", "system"}
	}
	for _, role := range roles {
		switch role {
		case "system_admin", "platform_admin", "super_admin":
			return "enterprise_admin", []string{"user", "enterprise", "system"}
		}
	}
	for _, role := range roles {
		switch role {
		case "entity_admin", "enterprise_admin", "org_admin", "admin", "operator":
			return "enterprise_admin", []string{"user", "enterprise", "system"}
		}
	}
	return "user", []string{"user"}
}

func (s *Service) UpdatePassword(ctx context.Context, input UpdatePasswordInput) error {
	entityID, err := ulidValue(input.EntityID)
	if err != nil {
		return err
	}
	userID, err := ulidValue(input.UserID)
	if err != nil {
		return err
	}
	if _, err := s.queries.VerifyLocalPasswordByUserID(ctx, generated.VerifyLocalPasswordByUserIDParams{
		EntityID: entityID,
		ID:       userID,
		Crypt:    input.CurrentPassword,
	}); err != nil {
		return fmt.Errorf("current password is invalid")
	}
	_, err = s.queries.UpdateLocalPassword(ctx, generated.UpdateLocalPasswordParams{
		EntityID:     entityID,
		UserID:       userID,
		Crypt:        input.NewPassword,
		WeakPassword: input.WeakPassword,
	})
	return err
}

func sessionULIDs(session auth.Session) (string, string, error) {
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		return "", "", err
	}
	userID, err := ulidValue(session.UserID)
	if err != nil {
		return "", "", err
	}
	return entityID, userID, nil
}

func ulidValue(value string) (string, error) {
	if err := id.ValidateULID(value); err != nil {
		return "", err
	}
	return value, nil
}

func ulidString(value string) string {
	return value
}

func textString(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func localeOrDefault(value pgtype.Text) string {
	if !value.Valid || value.String == "" {
		return "en-US"
	}
	return value.String
}

func boolValue(value pgtype.Bool) bool {
	return value.Valid && value.Bool
}
