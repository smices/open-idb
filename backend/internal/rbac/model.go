// SPDX-License-Identifier: MIT

package rbac

// RoleResponse represents a role in the system.
type RoleResponse struct {
	ID          string `json:"id"`
	EntityID    string `json:"entity_id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"created_at"`
}

// PermissionResponse represents a permission in the system.
type PermissionResponse struct {
	ID       string `json:"id"`
	EntityID string `json:"entity_id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Type     string `json:"type"`
}

// ResourceScopeResponse represents a resource scope in the system.
type ResourceScopeResponse struct {
	ID       string `json:"id"`
	EntityID string `json:"entity_id"`
	Type     string `json:"type"`
	Key      string `json:"key"`
	Name     string `json:"name"`
}

// AssignmentResponse represents an application assignment in the system.
type AssignmentResponse struct {
	ID            string `json:"id"`
	EntityID      string `json:"entity_id"`
	ApplicationID string `json:"application_id"`
	SubjectType   string `json:"subject_type"`
	SubjectID     string `json:"subject_id"`
	Effect        string `json:"effect"`
}

// ListResult represents a paginated list response.
type ListResult struct {
	Items  interface{} `json:"items"`
	Total  int64       `json:"total"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
}
