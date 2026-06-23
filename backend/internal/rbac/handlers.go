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
func (h Handler) RegisterRoutes(r chi.Router) {
	r.Get("/sapi/roles", h.listRoles)
	r.Get("/sapi/roles/{id}", h.getRole)
	r.Get("/sapi/roles/{id}/permissions", h.listRolePermissions)
	r.Post("/sapi/roles", h.createRole)
	r.Put("/sapi/roles/{id}", h.updateRole)
	r.Delete("/sapi/roles/{id}", h.deleteRole)
	r.Post("/sapi/roles/{id}/permissions", h.assignPermissionToRole)
	r.Delete("/sapi/roles/{id}/permissions/{pid}", h.removePermissionFromRole)

	r.Post("/sapi/permissions", h.createPermission)
	r.Get("/sapi/permissions", h.listPermissions)
	r.Get("/sapi/permissions/{id}", h.getPermission)
	r.Put("/sapi/permissions/{id}", h.updatePermission)
	r.Delete("/sapi/permissions/{id}", h.deletePermission)
	// Permission check (unique to rbac)
	r.Post("/sapi/permissions/check", h.checkPermission)

	// User roles (unique to rbac)
	r.Post("/sapi/users/{id}/roles", h.assignRoleToUser)
	r.Delete("/sapi/users/{id}/roles/{role_id}", h.removeRoleFromUser)
	r.Get("/sapi/users/{id}/roles", h.getUserRoles)
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

func readSession(w http.ResponseWriter, r *http.Request) (auth.Session, bool) {
	cookie, err := r.Cookie("idb_admin_session")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "admin_session_required", "idb_admin_session cookie is required")
		return auth.Session{}, false
	}
	adminSession, err := auth.ResolveAdminSession(r.Context(), cookie.Value)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_admin_session", "idb_admin_session cookie is invalid")
		return auth.Session{}, false
	}
	return auth.Session{
		ID:          adminSession.ID,
		UserID:      adminSession.AdminID,
		EntityID:    adminSession.EntityID,
		Username:    adminSession.Username,
		DisplayName: adminSession.DisplayName,
		ExpiresAt:   adminSession.ExpiresAt,
	}, true
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
