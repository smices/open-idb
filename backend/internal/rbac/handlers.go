// SPDX-License-Identifier: MIT

package rbac

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/smices/open-idb/internal/auth"
)

// Handler provides HTTP handlers for RBAC operations.
type Handler struct {
	service *Service
}

// NewHandler creates a new RBAC handler.
func NewHandler(service *Service) Handler {
	return Handler{service: service}
}

// RegisterRoutes registers RBAC routes with the router.
// Note: Role, Permission, and Resource Scope CRUD are handled by adminapi package.
// This handler only registers routes unique to RBAC runtime enforcement.
func (h Handler) RegisterRoutes(r chi.Router) {
	// Role permission and resource scope assignment (adminapi has CRUD, rbac has assignment)
	r.Get("/admin/v1/roles", h.listRoles)
	r.Get("/admin/v1/roles/{id}", h.getRole)
	r.Get("/admin/v1/roles/{id}/permissions", h.listRolePermissions)
	r.Post("/admin/v1/roles", h.createRole)
	r.Put("/admin/v1/roles/{id}", h.updateRole)
	r.Delete("/admin/v1/roles/{id}", h.deleteRole)
	r.Post("/admin/v1/roles/{id}/permissions", h.assignPermissionToRole)
	r.Delete("/admin/v1/roles/{id}/permissions/{pid}", h.removePermissionFromRole)
	r.Post("/admin/v1/roles/{id}/resource-scopes", h.assignResourceScopeToRole)

	r.Post("/admin/v1/permissions", h.createPermission)
	r.Get("/admin/v1/permissions", h.listPermissions)
	r.Get("/admin/v1/permissions/{id}", h.getPermission)
	r.Put("/admin/v1/permissions/{id}", h.updatePermission)
	r.Delete("/admin/v1/permissions/{id}", h.deletePermission)
	// Permission check (unique to rbac)
	r.Post("/admin/v1/permissions/check", h.checkPermission)

	// User roles (unique to rbac)
	r.Post("/admin/v1/users/{id}/roles", h.assignRoleToUser)
	r.Delete("/admin/v1/users/{id}/roles/{role_id}", h.removeRoleFromUser)
	r.Get("/admin/v1/users/{id}/roles", h.getUserRoles)

	// Application assignments (unique to rbac)
	r.Post("/admin/v1/applications/{id}/assignments", h.createApplicationAssignment)
	r.Get("/admin/v1/applications/{id}/assignments", h.listApplicationAssignments)
	r.Delete("/admin/v1/applications/assignments/{aid}", h.deleteApplicationAssignment)

	// Duplicate routes with /api prefix
	r.Get("/api/admin/v1/roles", h.listRoles)
	r.Get("/api/admin/v1/roles/{id}", h.getRole)
	r.Get("/api/admin/v1/roles/{id}/permissions", h.listRolePermissions)
	r.Post("/api/admin/v1/roles/{id}/permissions", h.assignPermissionToRole)
	r.Delete("/api/admin/v1/roles/{id}/permissions/{pid}", h.removePermissionFromRole)
	r.Post("/api/admin/v1/roles/{id}/resource-scopes", h.assignResourceScopeToRole)

	r.Post("/api/admin/v1/permissions/check", h.checkPermission)

	r.Post("/api/admin/v1/users/{id}/roles", h.assignRoleToUser)
	r.Delete("/api/admin/v1/users/{id}/roles/{role_id}", h.removeRoleFromUser)
	r.Get("/api/admin/v1/users/{id}/roles", h.getUserRoles)

	r.Post("/api/admin/v1/applications/{id}/assignments", h.createApplicationAssignment)
	r.Get("/api/admin/v1/applications/{id}/assignments", h.listApplicationAssignments)
	r.Delete("/api/admin/v1/applications/assignments/{aid}", h.deleteApplicationAssignment)

	// Role permission and permission CRUD (adminapi has list/detail, rbac has mutation endpoints)
	r.Post("/api/admin/v1/roles", h.createRole)
	r.Put("/api/admin/v1/roles/{id}", h.updateRole)
	r.Delete("/api/admin/v1/roles/{id}", h.deleteRole)

	r.Post("/api/admin/v1/permissions", h.createPermission)
	r.Get("/api/admin/v1/permissions", h.listPermissions)
	r.Get("/api/admin/v1/permissions/{id}", h.getPermission)
	r.Put("/api/admin/v1/permissions/{id}", h.updatePermission)
	r.Delete("/api/admin/v1/permissions/{id}", h.deletePermission)
}

func (h Handler) createRole(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	var request struct {
		Name        string `json:"name"`
		Code        string `json:"code"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}

	if request.Name == "" || request.Code == "" {
		writeError(w, http.StatusBadRequest, "missing_fields", "name and code are required")
		return
	}

	role, err := h.service.CreateRole(r.Context(), session.EntityID, request.Name, request.Code, request.Description)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create_role_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, role)
}

func (h Handler) listRoles(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	limit := int32(20)
	offset := int32(0)

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := parseInt32(l); err == nil {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := parseInt32(o); err == nil {
			offset = parsed
		}
	}

	result, err := h.service.ListRoles(r.Context(), session.EntityID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list_roles_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h Handler) getRole(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	id := chi.URLParam(r, "id")
	role, err := h.service.GetRole(r.Context(), session.EntityID, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "get_role_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, role)
}

func (h Handler) listRolePermissions(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	roleID := chi.URLParam(r, "id")
	perms, err := h.service.ListRolePermissions(r.Context(), session.EntityID, roleID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list_role_permissions_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, perms)
}

func (h Handler) updateRole(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	id := chi.URLParam(r, "id")
	var request struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}

	role, err := h.service.UpdateRole(r.Context(), session.EntityID, id, request.Name, request.Description)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "update_role_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, role)
}

func (h Handler) deleteRole(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	id := chi.URLParam(r, "id")
	err := h.service.DeleteRole(r.Context(), session.EntityID, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "delete_role_failed", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) assignPermissionToRole(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	roleID := chi.URLParam(r, "id")
	var request struct {
		PermissionID string `json:"permission_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}

	if request.PermissionID == "" {
		writeError(w, http.StatusBadRequest, "missing_fields", "permission_id is required")
		return
	}

	err := h.service.AssignPermissionToRole(r.Context(), session.EntityID, roleID, request.PermissionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "assign_permission_failed", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) removePermissionFromRole(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	roleID := chi.URLParam(r, "id")
	permissionID := chi.URLParam(r, "pid")

	err := h.service.RemovePermissionFromRole(r.Context(), session.EntityID, roleID, permissionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "remove_permission_failed", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) assignResourceScopeToRole(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	roleID := chi.URLParam(r, "id")
	var request struct {
		ResourceScopeID string `json:"resource_scope_id"`
		Effect          string `json:"effect"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}

	if request.ResourceScopeID == "" || request.Effect == "" {
		writeError(w, http.StatusBadRequest, "missing_fields", "resource_scope_id and effect are required")
		return
	}

	err := h.service.AssignResourceScopeToRole(r.Context(), session.EntityID, roleID, request.ResourceScopeID, request.Effect)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "assign_resource_scope_failed", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) createPermission(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	var request struct {
		Code string `json:"code"`
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}

	if request.Code == "" || request.Name == "" || request.Type == "" {
		writeError(w, http.StatusBadRequest, "missing_fields", "code, name, and type are required")
		return
	}

	perm, err := h.service.CreatePermission(r.Context(), session.EntityID, request.Code, request.Name, request.Type)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create_permission_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, perm)
}

func (h Handler) listPermissions(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	limit := int32(20)
	offset := int32(0)

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := parseInt32(l); err == nil {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := parseInt32(o); err == nil {
			offset = parsed
		}
	}

	result, err := h.service.ListPermissions(r.Context(), session.EntityID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list_permissions_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h Handler) getPermission(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	id := chi.URLParam(r, "id")
	perm, err := h.service.GetPermission(r.Context(), session.EntityID, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "get_permission_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, perm)
}

func (h Handler) updatePermission(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	id := chi.URLParam(r, "id")
	var request struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}

	perm, err := h.service.UpdatePermission(r.Context(), session.EntityID, id, request.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "update_permission_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, perm)
}

func (h Handler) deletePermission(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	id := chi.URLParam(r, "id")
	err := h.service.DeletePermission(r.Context(), session.EntityID, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "delete_permission_failed", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) checkPermission(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	var request struct {
		UserID     string `json:"user_id"`
		Permission string `json:"permission"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}

	if request.UserID == "" || request.Permission == "" {
		writeError(w, http.StatusBadRequest, "missing_fields", "user_id and permission are required")
		return
	}

	allowed, err := h.service.CheckAccess(r.Context(), session.EntityID, request.UserID, request.Permission)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "check_permission_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"allowed": allowed})
}

func (h Handler) assignRoleToUser(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	userID := chi.URLParam(r, "id")
	var request struct {
		RoleID string `json:"role_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}

	if request.RoleID == "" {
		writeError(w, http.StatusBadRequest, "missing_fields", "role_id is required")
		return
	}

	err := h.service.AssignRoleToUser(r.Context(), session.EntityID, userID, request.RoleID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "assign_role_failed", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) removeRoleFromUser(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	userID := chi.URLParam(r, "id")
	roleID := chi.URLParam(r, "role_id")

	err := h.service.RemoveRoleFromUser(r.Context(), session.EntityID, userID, roleID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "remove_role_failed", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) getUserRoles(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	userID := chi.URLParam(r, "id")
	roles, err := h.service.GetUserRoles(r.Context(), session.EntityID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "get_user_roles_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, roles)
}

func (h Handler) listResourceScopes(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	scopeType := r.URL.Query().Get("type")
	limit := int32(20)
	offset := int32(0)

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := parseInt32(l); err == nil {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := parseInt32(o); err == nil {
			offset = parsed
		}
	}

	result, err := h.service.ListResourceScopes(r.Context(), session.EntityID, scopeType, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list_resource_scopes_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h Handler) createResourceScope(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	var request struct {
		Type string `json:"type"`
		Key  string `json:"key"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}

	if request.Type == "" || request.Key == "" || request.Name == "" {
		writeError(w, http.StatusBadRequest, "missing_fields", "type, key, and name are required")
		return
	}

	scope, err := h.service.CreateResourceScope(r.Context(), session.EntityID, request.Type, request.Key, request.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create_resource_scope_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, scope)
}

func (h Handler) createApplicationAssignment(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	appID := chi.URLParam(r, "id")
	var request struct {
		SubjectType string `json:"subject_type"`
		SubjectID   string `json:"subject_id"`
		Effect      string `json:"effect"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}

	if request.SubjectType == "" || request.SubjectID == "" || request.Effect == "" {
		writeError(w, http.StatusBadRequest, "missing_fields", "subject_type, subject_id, and effect are required")
		return
	}

	assignment, err := h.service.CreateApplicationAssignment(r.Context(), session.EntityID, appID, request.SubjectType, request.SubjectID, request.Effect)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create_assignment_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, assignment)
}

func (h Handler) listApplicationAssignments(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	appID := chi.URLParam(r, "id")
	limit := int32(20)
	offset := int32(0)

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := parseInt32(l); err == nil {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := parseInt32(o); err == nil {
			offset = parsed
		}
	}

	result, err := h.service.ListApplicationAssignments(r.Context(), session.EntityID, appID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list_assignments_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h Handler) deleteApplicationAssignment(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	aid := chi.URLParam(r, "aid")
	err := h.service.DeleteApplicationAssignment(r.Context(), session.EntityID, aid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "delete_assignment_failed", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// readSession reads and validates the idb_session cookie.
func readSession(w http.ResponseWriter, r *http.Request) (auth.Session, bool) {
	cookie, err := r.Cookie("idb_session")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "session_required", "idb_session cookie is required")
		return auth.Session{}, false
	}
	session, err := auth.ResolveSession(r.Context(), cookie.Value)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_session", "idb_session cookie is invalid")
		return auth.Session{}, false
	}
	return session, true
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// writeError writes an error response.
func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, map[string]string{
		"error":             code,
		"error_description": message,
	})
}

// parseInt32 parses a string to int32.
func parseInt32(s string) (int32, error) {
	var n int32
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// context is imported but not used directly in handlers (passed via r.Context())
var _ context.Context
