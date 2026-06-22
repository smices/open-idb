// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/smices/open-idb/internal/db/generated"
)

// InternalService provides data-access methods for the internal v1 API handlers.
// It satisfies the internalService interface defined in internal_handlers.go.
type InternalService struct {
	queries *generated.Queries
}

// NewInternalService creates a new InternalService.
func NewInternalService(queries *generated.Queries) (*InternalService, error) {
	if queries == nil {
		return nil, fmt.Errorf("queries are required")
	}
	return &InternalService{queries: queries}, nil
}

// IntrospectToken looks up an OAuth token by its hash and returns introspection data.
// The entityID is required because the query is scoped to the entity.
func (s *InternalService) IntrospectToken(ctx context.Context, entityID string, tokenHash string) (IntrospectResponse, error) {
	token, err := s.queries.GetOAuthTokenByHash(ctx, generated.GetOAuthTokenByHashParams{
		EntityID:  entityID,
		TokenHash: tokenHash,
	})
	if err != nil {
		// Token not found is not an error — it's simply inactive.
		return IntrospectResponse{Active: false}, nil
	}

	// Check if the token has been revoked.
	if token.RevokedAt.Valid {
		return IntrospectResponse{Active: false}, nil
	}

	// Check if the token has expired.
	if token.ExpiresAt.Valid && token.ExpiresAt.Time.Before(time.Now()) {
		return IntrospectResponse{Active: false}, nil
	}

	resp := IntrospectResponse{
		Active:    true,
		UserID:    ulidString(token.UserID),
		ClientID:  token.ClientID,
		Scopes:    token.Scopes,
		TokenType: token.TokenType,
	}
	if token.ExpiresAt.Valid {
		resp.ExpiresAt = token.ExpiresAt.Time.Format(time.RFC3339)
	}
	return resp, nil
}

// CheckPermission verifies whether a user holds a specific permission, optionally
// scoped to a resource type and key.
func (s *InternalService) CheckPermission(ctx context.Context, entityID string, input CheckPermissionInput) (CheckPermissionResult, error) {
	userID, err := ulidValue(input.UserID)
	if err != nil {
		return CheckPermissionResult{}, fmt.Errorf("invalid user_id: %w", err)
	}

	// Step 1: Verify user lifecycle status is active.
	lifecycleStatus, err := s.queries.GetUserLifecycleStatus(ctx, generated.GetUserLifecycleStatusParams{
		EntityID: entityID,
		ID:       userID,
	})
	if err != nil {
		return CheckPermissionResult{Allowed: false, Reason: "user not found"}, nil
	}
	if lifecycleStatus != "active" {
		return CheckPermissionResult{Allowed: false, Reason: "user is not active"}, nil
	}

	// Step 2: Check the user holds the requested permission.
	perms, err := s.queries.GetUserPermissions(ctx, generated.GetUserPermissionsParams{
		EntityID: entityID,
		UserID:   userID,
	})
	if err != nil {
		return CheckPermissionResult{}, fmt.Errorf("failed to query user permissions: %w", err)
	}

	hasPermission := false
	for _, p := range perms {
		if p.Code == input.PermissionCode {
			hasPermission = true
			break
		}
	}
	if !hasPermission {
		return CheckPermissionResult{Allowed: false, Reason: "user does not have the requested permission"}, nil
	}

	// Step 3: If a resource scope was specified, verify the user's roles grant access.
	// Explicit deny takes precedence over allow.
	if input.ResourceType != "" && input.ResourceKey != "" {
		scopes, err := s.queries.GetUserResourceScopes(ctx, generated.GetUserResourceScopesParams{
			EntityID: entityID,
			UserID:   userID,
		})
		if err != nil {
			return CheckPermissionResult{}, fmt.Errorf("failed to query user resource scopes: %w", err)
		}

		hasAllow := false
		hasDeny := false
		for _, rs := range scopes {
			if rs.Type == input.ResourceType && rs.Key == input.ResourceKey {
				if rs.Effect == "deny" {
					hasDeny = true
				}
				if rs.Effect == "allow" {
					hasAllow = true
				}
			}
		}
		// Explicit deny takes precedence over allow.
		if hasDeny {
			return CheckPermissionResult{Allowed: false, Reason: "explicit deny on resource scope"}, nil
		}
		if !hasAllow {
			return CheckPermissionResult{Allowed: false, Reason: "user does not have the required resource scope"}, nil
		}
	}

	return CheckPermissionResult{Allowed: true, Reason: "permission granted"}, nil
}

// GetUserAccess returns a full access summary for a user, including their
// accessible applications, roles, permissions, and resource scopes.
func (s *InternalService) GetUserAccess(ctx context.Context, entityID string, userID string) (UserAccessSummary, error) {
	// Get user lifecycle status.
	lifecycleStatus, err := s.queries.GetUserLifecycleStatus(ctx, generated.GetUserLifecycleStatusParams{
		EntityID: entityID,
		ID:       userID,
	})
	if err != nil {
		return UserAccessSummary{}, fmt.Errorf("user not found: %w", err)
	}

	// Get applications the user can access.
	appRows, err := s.queries.GetUserApplicationAccess(ctx, generated.GetUserApplicationAccessParams{
		EntityID:  entityID,
		SubjectID: userID,
	})
	if err != nil {
		return UserAccessSummary{}, fmt.Errorf("failed to query application access: %w", err)
	}

	// Get user's roles.
	roles, err := s.queries.ListUserRoles(ctx, generated.ListUserRolesParams{
		EntityID: entityID,
		UserID:   userID,
	})
	if err != nil {
		return UserAccessSummary{}, fmt.Errorf("failed to query user roles: %w", err)
	}

	// Build role details (permissions + resource scopes per role).
	roleInfos := make([]RoleAccessInfo, 0, len(roles))
	for _, role := range roles {
		permRows, err := s.queries.ListRolePermissions(ctx, generated.ListRolePermissionsParams{
			EntityID: entityID,
			RoleID:   role.ID,
		})
		if err != nil {
			return UserAccessSummary{}, fmt.Errorf("failed to query role permissions: %w", err)
		}
		permCodes := make([]string, 0, len(permRows))
		for _, p := range permRows {
			permCodes = append(permCodes, p.Code)
		}

		scopeRows, err := s.queries.ListRoleResourceScopes(ctx, generated.ListRoleResourceScopesParams{
			EntityID: entityID,
			RoleID:   role.ID,
		})
		if err != nil {
			return UserAccessSummary{}, fmt.Errorf("failed to query role resource scopes: %w", err)
		}
		scopeInfos := make([]ResourceScopeInfo, 0, len(scopeRows))
		for _, rs := range scopeRows {
			scopeInfos = append(scopeInfos, ResourceScopeInfo{
				Type:   rs.Type,
				Key:    rs.Key,
				Effect: rs.Effect,
			})
		}

		roleInfos = append(roleInfos, RoleAccessInfo{
			RoleID:         role.ID,
			RoleCode:       role.Code,
			Permissions:    permCodes,
			ResourceScopes: scopeInfos,
		})
	}

	// Build application access entries.
	applications := make([]UserApplicationAccess, 0, len(appRows))
	for _, app := range appRows {
		applications = append(applications, UserApplicationAccess{
			ApplicationID:   app.ID,
			ApplicationName: app.Name,
			ApplicationType: app.Type,
			HasAccess:       true,
			Roles:           roleInfos,
		})
	}

	return UserAccessSummary{
		UserID:       ulidString(userID),
		EntityID:     ulidString(entityID),
		Status:       lifecycleStatus,
		Applications: applications,
	}, nil
}

// CreateAuditEvent writes an audit event to the audit_logs table on behalf
// of another service.
func (s *InternalService) CreateAuditEvent(ctx context.Context, entityID string, input AuditEventInput) (AuditEventResult, error) {
	actorUserID := pgtype.Text{}
	if input.ActorUserID != "" {
		parsedActorUserID, err := ulidValue(input.ActorUserID)
		if err != nil {
			return AuditEventResult{}, fmt.Errorf("invalid actor_user_id: %w", err)
		}
		actorUserID = pgtype.Text{String: parsedActorUserID, Valid: true}
	}

	var beforeState []byte
	if input.BeforeState != nil {
		beforeState = []byte(input.BeforeState)
	}
	var afterState []byte
	if input.AfterState != nil {
		afterState = []byte(input.AfterState)
	}

	log, err := s.queries.CreateAuditLog(ctx, generated.CreateAuditLogParams{
		EntityID:     entityID,
		ActorUserID:  actorUserID,
		ActorType:    input.ActorType,
		Action:       input.Action,
		ResourceType: input.ResourceType,
		ResourceID:   input.ResourceID,
		BeforeState:  beforeState,
		AfterState:   afterState,
		Ip:           input.Ip,
		UserAgent:    input.UserAgent,
		TraceID:      input.TraceID,
	})
	if err != nil {
		return AuditEventResult{}, fmt.Errorf("failed to create audit log: %w", err)
	}

	return AuditEventResult{
		ID:        log.ID,
		CreatedAt: log.CreatedAt.Time.Format(time.RFC3339),
	}, nil
}
