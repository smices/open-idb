// SPDX-License-Identifier: MIT

package rbac

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/smices/open-idb/internal/db/generated"
)

// Service provides RBAC business logic with Casbin integration.
type Service struct {
	queries  *generated.Queries
	enforcer *Enforcer
}

// NewService creates a new RBAC service.
func NewService(queries *generated.Queries, enforcer *Enforcer) (*Service, error) {
	if queries == nil {
		return nil, fmt.Errorf("queries are required")
	}
	if enforcer == nil {
		return nil, fmt.Errorf("enforcer is required")
	}
	return &Service{queries: queries, enforcer: enforcer}, nil
}

// CreateRole creates a new role.
func (s *Service) CreateRole(ctx context.Context, entityID string, name string, code string, description string) (RoleResponse, error) {
	var entityULID string
	entityULID = entityID
	desc := pgtype.Text{String: description, Valid: description != ""}

	role, err := s.queries.CreateRole(ctx, generated.CreateRoleParams{
		EntityID:    entityULID,
		Name:        name,
		Code:        code,
		Description: desc,
	})
	if err != nil {
		return RoleResponse{}, fmt.Errorf("failed to create role: %w", err)
	}

	return toRoleResponse(role), nil
}

// ListRoles lists roles with pagination.
func (s *Service) ListRoles(ctx context.Context, entityID string, limit int32, offset int32) (ListResult, error) {
	var entityULID string
	entityULID = entityID

	roles, err := s.queries.ListRoles(ctx, generated.ListRolesParams{
		EntityID: entityULID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return ListResult{}, fmt.Errorf("failed to list roles: %w", err)
	}

	total, err := s.queries.CountRoles(ctx, entityULID)
	if err != nil {
		return ListResult{}, fmt.Errorf("failed to count roles: %w", err)
	}

	items := make([]RoleResponse, len(roles))
	for i, role := range roles {
		items[i] = toRoleResponse(role)
	}

	return ListResult{
		Items:  items,
		Total:  total,
		Limit:  int(limit),
		Offset: int(offset),
	}, nil
}

// GetRole retrieves a role by ID.
func (s *Service) GetRole(ctx context.Context, entityID string, id string) (RoleResponse, error) {
	var entityULID, roleULID string
	entityULID = entityID
	roleULID = id

	role, err := s.queries.GetRoleByID(ctx, generated.GetRoleByIDParams{
		EntityID: entityULID,
		ID:       roleULID,
	})
	if err != nil {
		return RoleResponse{}, fmt.Errorf("failed to get role: %w", err)
	}

	return toRoleResponse(role), nil
}

// UpdateRole updates a role.
func (s *Service) UpdateRole(ctx context.Context, entityID string, id string, name string, description string) (RoleResponse, error) {
	var entityULID, roleULID string
	entityULID = entityID
	roleULID = id

	role, err := s.queries.UpdateRole(ctx, generated.UpdateRoleParams{
		EntityID:    entityULID,
		ID:          roleULID,
		Name:        pgtype.Text{String: name, Valid: name != ""},
		Description: pgtype.Text{String: description, Valid: description != ""},
	})
	if err != nil {
		return RoleResponse{}, fmt.Errorf("failed to update role: %w", err)
	}

	return toRoleResponse(role), nil
}

// DeleteRole deletes a role.
func (s *Service) DeleteRole(ctx context.Context, entityID string, id string) error {
	var entityULID, roleULID string
	entityULID = entityID
	roleULID = id

	err := s.queries.DeleteRole(ctx, generated.DeleteRoleParams{
		EntityID: entityULID,
		ID:       roleULID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	return nil
}

// CreatePermission creates a new permission.
func (s *Service) CreatePermission(ctx context.Context, entityID string, code string, name string, permType string) (PermissionResponse, error) {
	var entityULID string
	entityULID = entityID

	perm, err := s.queries.CreatePermission(ctx, generated.CreatePermissionParams{
		EntityID: entityULID,
		Code:     code,
		Name:     name,
		Type:     permType,
	})
	if err != nil {
		return PermissionResponse{}, fmt.Errorf("failed to create permission: %w", err)
	}

	return toPermissionResponse(perm), nil
}

// ListPermissions lists permissions with pagination.
func (s *Service) ListPermissions(ctx context.Context, entityID string, limit int32, offset int32) (ListResult, error) {
	var entityULID string
	entityULID = entityID

	perms, err := s.queries.ListPermissions(ctx, generated.ListPermissionsParams{
		EntityID: entityULID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return ListResult{}, fmt.Errorf("failed to list permissions: %w", err)
	}

	total, err := s.queries.CountPermissions(ctx, entityULID)
	if err != nil {
		return ListResult{}, fmt.Errorf("failed to count permissions: %w", err)
	}

	items := make([]PermissionResponse, len(perms))
	for i, perm := range perms {
		items[i] = toPermissionResponse(perm)
	}

	return ListResult{
		Items:  items,
		Total:  total,
		Limit:  int(limit),
		Offset: int(offset),
	}, nil
}

// GetPermission retrieves a permission by ID.
func (s *Service) GetPermission(ctx context.Context, entityID string, id string) (PermissionResponse, error) {
	var entityULID, permULID string
	entityULID = entityID
	permULID = id

	perm, err := s.queries.GetPermissionByID(ctx, generated.GetPermissionByIDParams{
		EntityID: entityULID,
		ID:       permULID,
	})
	if err != nil {
		return PermissionResponse{}, fmt.Errorf("failed to get permission: %w", err)
	}

	return toPermissionResponse(perm), nil
}

// UpdatePermission updates a permission.
func (s *Service) UpdatePermission(ctx context.Context, entityID string, id string, name string) (PermissionResponse, error) {
	var entityULID, permULID string
	entityULID = entityID
	permULID = id

	perm, err := s.queries.UpdatePermission(ctx, generated.UpdatePermissionParams{
		EntityID: entityULID,
		ID:       permULID,
		Name:     pgtype.Text{String: name, Valid: name != ""},
	})
	if err != nil {
		return PermissionResponse{}, fmt.Errorf("failed to update permission: %w", err)
	}

	return toPermissionResponse(perm), nil
}

// DeletePermission deletes a permission.
func (s *Service) DeletePermission(ctx context.Context, entityID string, id string) error {
	var entityULID, permULID string
	entityULID = entityID
	permULID = id

	err := s.queries.DeletePermission(ctx, generated.DeletePermissionParams{
		EntityID: entityULID,
		ID:       permULID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete permission: %w", err)
	}

	return nil
}

// AssignPermissionToRole assigns a permission to a role and updates Casbin policy.
func (s *Service) AssignPermissionToRole(ctx context.Context, entityID string, roleID string, permissionID string) error {
	var entityULID, roleULID, permULID string
	entityULID = entityID
	roleULID = roleID
	permULID = permissionID

	// Get role and permission codes for Casbin
	role, err := s.queries.GetRoleByID(ctx, generated.GetRoleByIDParams{
		EntityID: entityULID,
		ID:       roleULID,
	})
	if err != nil {
		return fmt.Errorf("failed to get role: %w", err)
	}

	perm, err := s.queries.GetPermissionByID(ctx, generated.GetPermissionByIDParams{
		EntityID: entityULID,
		ID:       permULID,
	})
	if err != nil {
		return fmt.Errorf("failed to get permission: %w", err)
	}

	// Assign in database
	err = s.queries.AddPermissionToRole(ctx, generated.AddPermissionToRoleParams{
		EntityID:     entityULID,
		RoleID:       roleULID,
		PermissionID: permULID,
	})
	if err != nil {
		return fmt.Errorf("failed to assign permission to role: %w", err)
	}

	// Update Casbin policy
	err = s.enforcer.AddPermissionToRole(role.Code, perm.Code, entityID)
	if err != nil {
		return fmt.Errorf("failed to update Casbin policy: %w", err)
	}

	return nil
}

// RemovePermissionFromRole removes a permission from a role and updates Casbin policy.
func (s *Service) RemovePermissionFromRole(ctx context.Context, entityID string, roleID string, permissionID string) error {
	var entityULID, roleULID, permULID string
	entityULID = entityID
	roleULID = roleID
	permULID = permissionID

	// Get role and permission codes for Casbin
	role, err := s.queries.GetRoleByID(ctx, generated.GetRoleByIDParams{
		EntityID: entityULID,
		ID:       roleULID,
	})
	if err != nil {
		return fmt.Errorf("failed to get role: %w", err)
	}

	perm, err := s.queries.GetPermissionByID(ctx, generated.GetPermissionByIDParams{
		EntityID: entityULID,
		ID:       permULID,
	})
	if err != nil {
		return fmt.Errorf("failed to get permission: %w", err)
	}

	// Remove from database
	err = s.queries.RemovePermissionFromRole(ctx, generated.RemovePermissionFromRoleParams{
		EntityID:     entityULID,
		RoleID:       roleULID,
		PermissionID: permULID,
	})
	if err != nil {
		return fmt.Errorf("failed to remove permission from role: %w", err)
	}

	// Update Casbin policy
	err = s.enforcer.RemovePermissionFromRole(role.Code, perm.Code, entityID)
	if err != nil {
		return fmt.Errorf("failed to update Casbin policy: %w", err)
	}

	return nil
}

// AssignRoleToUser assigns a role to a user and updates Casbin policy.
func (s *Service) AssignRoleToUser(ctx context.Context, entityID string, userID string, roleID string) error {
	var entityULID, userULID, roleULID string
	entityULID = entityID
	userULID = userID
	roleULID = roleID

	// Get role code for Casbin
	role, err := s.queries.GetRoleByID(ctx, generated.GetRoleByIDParams{
		EntityID: entityULID,
		ID:       roleULID,
	})
	if err != nil {
		return fmt.Errorf("failed to get role: %w", err)
	}

	// Assign in database
	err = s.queries.AssignRoleToUser(ctx, generated.AssignRoleToUserParams{
		EntityID: entityULID,
		UserID:   userULID,
		RoleID:   roleULID,
	})
	if err != nil {
		return fmt.Errorf("failed to assign role to user: %w", err)
	}

	// Update Casbin policy
	err = s.enforcer.AddRoleToUser(userID, role.Code, entityID)
	if err != nil {
		return fmt.Errorf("failed to update Casbin policy: %w", err)
	}

	return nil
}

// RemoveRoleFromUser removes a role from a user and updates Casbin policy.
func (s *Service) RemoveRoleFromUser(ctx context.Context, entityID string, userID string, roleID string) error {
	var entityULID, userULID, roleULID string
	entityULID = entityID
	userULID = userID
	roleULID = roleID

	// Get role code for Casbin
	role, err := s.queries.GetRoleByID(ctx, generated.GetRoleByIDParams{
		EntityID: entityULID,
		ID:       roleULID,
	})
	if err != nil {
		return fmt.Errorf("failed to get role: %w", err)
	}

	// Remove from database
	err = s.queries.RemoveRoleFromUser(ctx, generated.RemoveRoleFromUserParams{
		EntityID: entityULID,
		UserID:   userULID,
		RoleID:   roleULID,
	})
	if err != nil {
		return fmt.Errorf("failed to remove role from user: %w", err)
	}

	// Update Casbin policy
	err = s.enforcer.RemoveRoleFromUser(userID, role.Code, entityID)
	if err != nil {
		return fmt.Errorf("failed to update Casbin policy: %w", err)
	}

	return nil
}

// CreateResourceScope creates a new resource scope.
func (s *Service) CreateResourceScope(ctx context.Context, entityID string, scopeType string, key string, name string) (ResourceScopeResponse, error) {
	var entityULID string
	entityULID = entityID

	scope, err := s.queries.CreateResourceScope(ctx, generated.CreateResourceScopeParams{
		EntityID: entityULID,
		Type:     scopeType,
		Key:      key,
		Name:     name,
	})
	if err != nil {
		return ResourceScopeResponse{}, fmt.Errorf("failed to create resource scope: %w", err)
	}

	return toResourceScopeResponse(scope), nil
}

// ListResourceScopes lists resource scopes with optional type filter and pagination.
func (s *Service) ListResourceScopes(ctx context.Context, entityID string, scopeType string, limit int32, offset int32) (ListResult, error) {
	var entityULID string
	entityULID = entityID

	scopes, err := s.queries.ListResourceScopes(ctx, generated.ListResourceScopesParams{
		EntityID: entityULID,
		Type:     pgtype.Text{String: scopeType, Valid: scopeType != ""},
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return ListResult{}, fmt.Errorf("failed to list resource scopes: %w", err)
	}

	total, err := s.queries.CountResourceScopes(ctx, generated.CountResourceScopesParams{
		EntityID: entityULID,
		Type:     pgtype.Text{String: scopeType, Valid: scopeType != ""},
	})
	if err != nil {
		return ListResult{}, fmt.Errorf("failed to count resource scopes: %w", err)
	}

	items := make([]ResourceScopeResponse, len(scopes))
	for i, scope := range scopes {
		items[i] = toResourceScopeResponse(scope)
	}

	return ListResult{
		Items:  items,
		Total:  total,
		Limit:  int(limit),
		Offset: int(offset),
	}, nil
}

// AssignResourceScopeToRole assigns a resource scope to a role.
func (s *Service) AssignResourceScopeToRole(ctx context.Context, entityID string, roleID string, scopeID string, effect string) error {
	var entityULID, roleULID, scopeULID string
	entityULID = entityID
	roleULID = roleID
	scopeULID = scopeID

	err := s.queries.AddResourceScopeToRole(ctx, generated.AddResourceScopeToRoleParams{
		EntityID:        entityULID,
		RoleID:          roleULID,
		ResourceScopeID: scopeULID,
		Effect:          effect,
	})
	if err != nil {
		return fmt.Errorf("failed to assign resource scope to role: %w", err)
	}

	return nil
}

// RemoveResourceScopeFromRole removes a resource scope from a role.
func (s *Service) RemoveResourceScopeFromRole(ctx context.Context, entityID string, roleID string, scopeID string) error {
	var entityULID, roleULID, scopeULID string
	entityULID = entityID
	roleULID = roleID
	scopeULID = scopeID

	err := s.queries.RemoveResourceScopeFromRole(ctx, generated.RemoveResourceScopeFromRoleParams{
		EntityID:        entityULID,
		RoleID:          roleULID,
		ResourceScopeID: scopeULID,
	})
	if err != nil {
		return fmt.Errorf("failed to remove resource scope from role: %w", err)
	}

	return nil
}

// CreateApplicationAssignment creates a new application assignment.
func (s *Service) CreateApplicationAssignment(ctx context.Context, entityID string, appID string, subjectType string, subjectID string, effect string) (AssignmentResponse, error) {
	var entityULID, appULID, subjectULID string
	entityULID = entityID
	appULID = appID
	subjectULID = subjectID

	assignment, err := s.queries.CreateApplicationAssignment(ctx, generated.CreateApplicationAssignmentParams{
		EntityID:      entityULID,
		ApplicationID: appULID,
		SubjectType:   subjectType,
		SubjectID:     subjectULID,
		Effect:        effect,
	})
	if err != nil {
		return AssignmentResponse{}, fmt.Errorf("failed to create application assignment: %w", err)
	}

	return toAssignmentResponse(assignment), nil
}

// ListApplicationAssignments lists application assignments with pagination.
func (s *Service) ListApplicationAssignments(ctx context.Context, entityID string, appID string, limit int32, offset int32) (ListResult, error) {
	var entityULID, appULID string
	entityULID = entityID
	appULID = appID

	assignments, err := s.queries.ListApplicationAssignments(ctx, generated.ListApplicationAssignmentsParams{
		EntityID:      entityULID,
		ApplicationID: appULID,
		Limit:         limit,
		Offset:        offset,
	})
	if err != nil {
		return ListResult{}, fmt.Errorf("failed to list application assignments: %w", err)
	}

	total, err := s.queries.CountApplicationAssignments(ctx, generated.CountApplicationAssignmentsParams{
		EntityID:      entityULID,
		ApplicationID: appULID,
	})
	if err != nil {
		return ListResult{}, fmt.Errorf("failed to count application assignments: %w", err)
	}

	items := make([]AssignmentResponse, len(assignments))
	for i, assignment := range assignments {
		items[i] = toAssignmentResponse(assignment)
	}

	return ListResult{
		Items:  items,
		Total:  total,
		Limit:  int(limit),
		Offset: int(offset),
	}, nil
}

// DeleteApplicationAssignment deletes an application assignment.
func (s *Service) DeleteApplicationAssignment(ctx context.Context, entityID string, id string) error {
	var entityULID, assignmentULID string
	entityULID = entityID
	assignmentULID = id

	err := s.queries.DeleteApplicationAssignment(ctx, generated.DeleteApplicationAssignmentParams{
		EntityID: entityULID,
		ID:       assignmentULID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete application assignment: %w", err)
	}

	return nil
}

// CheckAccess checks if a user has a specific permission using Casbin.
func (s *Service) CheckAccess(ctx context.Context, entityID string, userID string, permission string) (bool, error) {
	return s.enforcer.CheckPermission(userID, entityID, permission)
}

// GetUserRoles returns roles for a user.
func (s *Service) GetUserRoles(ctx context.Context, entityID string, userID string) ([]RoleResponse, error) {
	var entityULID, userULID string
	entityULID = entityID
	userULID = userID

	roles, err := s.queries.ListUserRoles(ctx, generated.ListUserRolesParams{
		EntityID: entityULID,
		UserID:   userULID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	items := make([]RoleResponse, len(roles))
	for i, role := range roles {
		items[i] = toRoleResponse(role)
	}

	return items, nil
}

// ListRolePermissions returns all permissions attached to a role.
func (s *Service) ListRolePermissions(ctx context.Context, entityID string, roleID string) ([]PermissionResponse, error) {
	var entityULID, roleULID string
	entityULID = entityID
	roleULID = roleID

	perms, err := s.queries.ListRolePermissions(ctx, generated.ListRolePermissionsParams{
		EntityID: entityULID,
		RoleID:   roleULID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list role permissions: %w", err)
	}

	items := make([]PermissionResponse, len(perms))
	for i, perm := range perms {
		items[i] = toPermissionResponse(perm)
	}

	return items, nil
}

// toRoleResponse converts a generated.Role to RoleResponse.
func toRoleResponse(role generated.Role) RoleResponse {
	desc := ""
	if role.Description.Valid {
		desc = role.Description.String
	}
	return RoleResponse{
		ID:          role.ID,
		EntityID:    role.EntityID,
		Name:        role.Name,
		Code:        role.Code,
		Description: desc,
		CreatedAt:   role.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// toPermissionResponse converts a generated.Permission to PermissionResponse.
func toPermissionResponse(perm generated.Permission) PermissionResponse {
	return PermissionResponse{
		ID:       perm.ID,
		EntityID: perm.EntityID,
		Code:     perm.Code,
		Name:     perm.Name,
		Type:     perm.Type,
	}
}

// toResourceScopeResponse converts a generated.ResourceScope to ResourceScopeResponse.
func toResourceScopeResponse(scope generated.ResourceScope) ResourceScopeResponse {
	return ResourceScopeResponse{
		ID:       scope.ID,
		EntityID: scope.EntityID,
		Type:     scope.Type,
		Key:      scope.Key,
		Name:     scope.Name,
	}
}

// toAssignmentResponse converts a generated.ApplicationAssignment to AssignmentResponse.
func toAssignmentResponse(assignment generated.ApplicationAssignment) AssignmentResponse {
	return AssignmentResponse{
		ID:            assignment.ID,
		EntityID:      assignment.EntityID,
		ApplicationID: assignment.ApplicationID,
		SubjectType:   assignment.SubjectType,
		SubjectID:     assignment.SubjectID,
		Effect:        assignment.Effect,
	}
}
