// SPDX-License-Identifier: MIT

package rbac

import (
	"context"
	"testing"
)

func TestNewEnforcer(t *testing.T) {
	enforcer, err := NewEnforcer()
	if err != nil {
		t.Fatalf("failed to create enforcer: %v", err)
	}
	if enforcer == nil {
		t.Fatal("enforcer is nil")
	}
	if enforcer.enforcer == nil {
		t.Fatal("casbin enforcer is nil")
	}
}

func TestEnforcer_AddRoleToUser(t *testing.T) {
	enforcer, err := NewEnforcer()
	if err != nil {
		t.Fatalf("failed to create enforcer: %v", err)
	}

	entityID := "test-entity"
	userID := "user-1"
	roleCode := "admin"

	err = enforcer.AddRoleToUser(userID, roleCode, entityID)
	if err != nil {
		t.Fatalf("failed to add role to user: %v", err)
	}
}

func TestEnforcer_AddPermissionToRole(t *testing.T) {
	enforcer, err := NewEnforcer()
	if err != nil {
		t.Fatalf("failed to create enforcer: %v", err)
	}

	entityID := "test-entity"
	roleCode := "admin"
	permissionCode := "read:users"

	err = enforcer.AddPermissionToRole(roleCode, permissionCode, entityID)
	if err != nil {
		t.Fatalf("failed to add permission to role: %v", err)
	}
}

func TestEnforcer_CheckPermission(t *testing.T) {
	enforcer, err := NewEnforcer()
	if err != nil {
		t.Fatalf("failed to create enforcer: %v", err)
	}

	entityID := "test-entity"
	userID := "user-1"
	roleCode := "admin"
	permissionCode := "read:users"

	// Add role to user
	err = enforcer.AddRoleToUser(userID, roleCode, entityID)
	if err != nil {
		t.Fatalf("failed to add role to user: %v", err)
	}

	// Add permission to role
	err = enforcer.AddPermissionToRole(roleCode, permissionCode, entityID)
	if err != nil {
		t.Fatalf("failed to add permission to role: %v", err)
	}

	// Check permission - should be allowed
	allowed, err := enforcer.CheckPermission(userID, entityID, permissionCode)
	if err != nil {
		t.Fatalf("failed to check permission: %v", err)
	}
	if !allowed {
		t.Errorf("expected permission to be allowed, got denied")
	}

	// Check different permission - should be denied
	allowed, err = enforcer.CheckPermission(userID, entityID, "write:users")
	if err != nil {
		t.Fatalf("failed to check permission: %v", err)
	}
	if allowed {
		t.Errorf("expected permission to be denied, got allowed")
	}

	// Check different entity - should be denied
	allowed, err = enforcer.CheckPermission(userID, "other-entity", permissionCode)
	if err != nil {
		t.Fatalf("failed to check permission: %v", err)
	}
	if allowed {
		t.Errorf("expected permission to be denied for different entity, got allowed")
	}
}

func TestEnforcer_RemoveRoleFromUser(t *testing.T) {
	enforcer, err := NewEnforcer()
	if err != nil {
		t.Fatalf("failed to create enforcer: %v", err)
	}

	entityID := "test-entity"
	userID := "user-1"
	roleCode := "admin"
	permissionCode := "read:users"

	// Add role to user
	err = enforcer.AddRoleToUser(userID, roleCode, entityID)
	if err != nil {
		t.Fatalf("failed to add role to user: %v", err)
	}

	// Add permission to role
	err = enforcer.AddPermissionToRole(roleCode, permissionCode, entityID)
	if err != nil {
		t.Fatalf("failed to add permission to role: %v", err)
	}

	// Verify permission is allowed
	allowed, err := enforcer.CheckPermission(userID, entityID, permissionCode)
	if err != nil {
		t.Fatalf("failed to check permission: %v", err)
	}
	if !allowed {
		t.Errorf("expected permission to be allowed before removal")
	}

	// Remove role from user
	err = enforcer.RemoveRoleFromUser(userID, roleCode, entityID)
	if err != nil {
		t.Fatalf("failed to remove role from user: %v", err)
	}

	// Verify permission is now denied
	allowed, err = enforcer.CheckPermission(userID, entityID, permissionCode)
	if err != nil {
		t.Fatalf("failed to check permission: %v", err)
	}
	if allowed {
		t.Errorf("expected permission to be denied after removal")
	}
}

func TestEnforcer_RemovePermissionFromRole(t *testing.T) {
	enforcer, err := NewEnforcer()
	if err != nil {
		t.Fatalf("failed to create enforcer: %v", err)
	}

	entityID := "test-entity"
	userID := "user-1"
	roleCode := "admin"
	permissionCode := "read:users"

	// Add role to user
	err = enforcer.AddRoleToUser(userID, roleCode, entityID)
	if err != nil {
		t.Fatalf("failed to add role to user: %v", err)
	}

	// Add permission to role
	err = enforcer.AddPermissionToRole(roleCode, permissionCode, entityID)
	if err != nil {
		t.Fatalf("failed to add permission to role: %v", err)
	}

	// Verify permission is allowed
	allowed, err := enforcer.CheckPermission(userID, entityID, permissionCode)
	if err != nil {
		t.Fatalf("failed to check permission: %v", err)
	}
	if !allowed {
		t.Errorf("expected permission to be allowed before removal")
	}

	// Remove permission from role
	err = enforcer.RemovePermissionFromRole(roleCode, permissionCode, entityID)
	if err != nil {
		t.Fatalf("failed to remove permission from role: %v", err)
	}

	// Verify permission is now denied
	allowed, err = enforcer.CheckPermission(userID, entityID, permissionCode)
	if err != nil {
		t.Fatalf("failed to check permission: %v", err)
	}
	if allowed {
		t.Errorf("expected permission to be denied after removal")
	}
}

func TestEnforcer_LoadPolicy(t *testing.T) {
	enforcer, err := NewEnforcer()
	if err != nil {
		t.Fatalf("failed to create enforcer: %v", err)
	}

	entityID := "test-entity"
	loader := &mockPolicyLoader{
		userRoles: []UserRolePolicy{
			{UserID: "user-1", RoleCode: "admin"},
			{UserID: "user-2", RoleCode: "viewer"},
		},
		rolePerms: []RolePermissionPolicy{
			{RoleCode: "admin", PermissionCode: "read:users"},
			{RoleCode: "admin", PermissionCode: "write:users"},
			{RoleCode: "viewer", PermissionCode: "read:users"},
		},
	}

	err = enforcer.LoadPolicy(context.Background(), entityID, loader)
	if err != nil {
		t.Fatalf("failed to load policy: %v", err)
	}

	// Test user-1 (admin) has read:users
	allowed, err := enforcer.CheckPermission("user-1", entityID, "read:users")
	if err != nil {
		t.Fatalf("failed to check permission: %v", err)
	}
	if !allowed {
		t.Errorf("expected user-1 to have read:users permission")
	}

	// Test user-1 (admin) has write:users
	allowed, err = enforcer.CheckPermission("user-1", entityID, "write:users")
	if err != nil {
		t.Fatalf("failed to check permission: %v", err)
	}
	if !allowed {
		t.Errorf("expected user-1 to have write:users permission")
	}

	// Test user-2 (viewer) has read:users
	allowed, err = enforcer.CheckPermission("user-2", entityID, "read:users")
	if err != nil {
		t.Fatalf("failed to check permission: %v", err)
	}
	if !allowed {
		t.Errorf("expected user-2 to have read:users permission")
	}

	// Test user-2 (viewer) does NOT have write:users
	allowed, err = enforcer.CheckPermission("user-2", entityID, "write:users")
	if err != nil {
		t.Fatalf("failed to check permission: %v", err)
	}
	if allowed {
		t.Errorf("expected user-2 to NOT have write:users permission")
	}
}

// mockPolicyLoader is a mock implementation of PolicyLoader for testing.
type mockPolicyLoader struct {
	userRoles []UserRolePolicy
	rolePerms []RolePermissionPolicy
}

func (m *mockPolicyLoader) ListAllUserRoles(ctx context.Context, entityID string) ([]UserRolePolicy, error) {
	return m.userRoles, nil
}

func (m *mockPolicyLoader) ListAllRolePermissions(ctx context.Context, entityID string) ([]RolePermissionPolicy, error) {
	return m.rolePerms, nil
}
