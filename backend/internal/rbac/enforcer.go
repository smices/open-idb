// SPDX-License-Identifier: MIT

package rbac

import (
	"context"
	"fmt"

	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
)

// casbinModel defines the RBAC model with domains (entities).
// Request: sub (user), dom (entity), obj (permission), act (action)
// policy: sub (role), dom (entity), obj (permission), act (allow)
// role_definition: g = user, role, domain
const casbinModelText = `
[request_definition]
r = sub, dom, obj, act

[policy_definition]
p = sub, dom, obj, act

[role_definition]
g = _, _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub, r.dom) && r.dom == p.dom && r.obj == p.obj && r.act == p.act
`

// Enforcer wraps Casbin v3 enforcer with IdBridge-specific logic.
type Enforcer struct {
	enforcer *casbin.Enforcer
}

// PolicyLoader defines the interface for loading RBAC policies from the database.
type PolicyLoader interface {
	// ListAllUserRoles loads all user roles for a entity.
	ListAllUserRoles(ctx context.Context, entityID string) ([]UserRolePolicy, error)
	// ListAllRolePermissions loads all role permissions for a entity.
	ListAllRolePermissions(ctx context.Context, entityID string) ([]RolePermissionPolicy, error)
}

// UserRolePolicy represents a user-role assignment for policy loading.
type UserRolePolicy struct {
	UserID   string
	RoleCode string
}

// RolePermissionPolicy represents a role-permission assignment for policy loading.
type RolePermissionPolicy struct {
	RoleCode       string
	PermissionCode string
}

// NewEnforcer creates a new Casbin enforcer with the RBAC model.
func NewEnforcer() (*Enforcer, error) {
	m, err := model.NewModelFromString(casbinModelText)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin model: %w", err)
	}

	e, err := casbin.NewEnforcer(m)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin enforcer: %w", err)
	}

	return &Enforcer{enforcer: e}, nil
}

// LoadPolicy loads RBAC policies from database for a entity.
// For each user_role: g, user_id, role_code, entity_id
// For each role_permission: p, role_code, entity_id, permission_code, allow
func (e *Enforcer) LoadPolicy(ctx context.Context, entityID string, loader PolicyLoader) error {
	// Clear existing policies for this entity
	e.enforcer.ClearPolicy()

	// Load user roles
	userRoles, err := loader.ListAllUserRoles(ctx, entityID)
	if err != nil {
		return fmt.Errorf("failed to load user roles: %w", err)
	}

	for _, ur := range userRoles {
		_, err := e.enforcer.AddGroupingPolicy(ur.UserID, ur.RoleCode, entityID)
		if err != nil {
			return fmt.Errorf("failed to add user role policy: %w", err)
		}
	}

	// Load role permissions
	rolePerms, err := loader.ListAllRolePermissions(ctx, entityID)
	if err != nil {
		return fmt.Errorf("failed to load role permissions: %w", err)
	}

	for _, rp := range rolePerms {
		_, err := e.enforcer.AddPolicy(rp.RoleCode, entityID, rp.PermissionCode, "allow")
		if err != nil {
			return fmt.Errorf("failed to add role permission policy: %w", err)
		}
	}

	return nil
}

// CheckPermission checks if a user has a specific permission in a entity.
func (e *Enforcer) CheckPermission(userID string, entityID string, permission string) (bool, error) {
	result, err := e.enforcer.Enforce(userID, entityID, permission, "allow")
	if err != nil {
		return false, fmt.Errorf("failed to enforce permission: %w", err)
	}
	return result, nil
}

// AddRoleToUser adds a role to a user in the Casbin policy.
func (e *Enforcer) AddRoleToUser(userID string, roleCode string, entityID string) error {
	_, err := e.enforcer.AddGroupingPolicy(userID, roleCode, entityID)
	if err != nil {
		return fmt.Errorf("failed to add role to user: %w", err)
	}
	return nil
}

// AddPermissionToRole adds a permission to a role in the Casbin policy.
func (e *Enforcer) AddPermissionToRole(roleCode string, permissionCode string, entityID string) error {
	_, err := e.enforcer.AddPolicy(roleCode, entityID, permissionCode, "allow")
	if err != nil {
		return fmt.Errorf("failed to add permission to role: %w", err)
	}
	return nil
}

// RemoveRoleFromUser removes a role from a user in the Casbin policy.
func (e *Enforcer) RemoveRoleFromUser(userID string, roleCode string, entityID string) error {
	_, err := e.enforcer.RemoveGroupingPolicy(userID, roleCode, entityID)
	if err != nil {
		return fmt.Errorf("failed to remove role from user: %w", err)
	}
	return nil
}

// RemovePermissionFromRole removes a permission from a role in the Casbin policy.
func (e *Enforcer) RemovePermissionFromRole(roleCode string, permissionCode string, entityID string) error {
	_, err := e.enforcer.RemovePolicy(roleCode, entityID, permissionCode, "allow")
	if err != nil {
		return fmt.Errorf("failed to remove permission from role: %w", err)
	}
	return nil
}
