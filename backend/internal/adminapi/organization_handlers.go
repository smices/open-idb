// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/smices/open-idb/internal/db/generated"
)

// OrganizationResponse represents an organization in API responses.
type OrganizationResponse struct {
	ID        string    `json:"id"`
	EntityID  string    `json:"entity_id"`
	Name      string    `json:"name"`
	ParentID  *string   `json:"parent_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DepartmentResponse represents a department in API responses.
type DepartmentResponse struct {
	ID                   string    `json:"id"`
	EntityID             string    `json:"entity_id"`
	OrganizationID       string    `json:"organization_id"`
	Name                 string    `json:"name"`
	ParentID             *string   `json:"parent_id,omitempty"`
	SourceID             *string   `json:"source_id,omitempty"`
	ExternalDepartmentID *string   `json:"external_department_id,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// GroupResponse represents a group in API responses.
type GroupResponse struct {
	ID        string    `json:"id"`
	EntityID  string    `json:"entity_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GroupMemberResponse represents a group member in API responses.
type GroupMemberResponse struct {
	UserID          string `json:"user_id"`
	Username        string `json:"username"`
	DisplayName     string `json:"display_name"`
	Email           string `json:"email,omitempty"`
	LifecycleStatus string `json:"lifecycle_status"`
}

// organizationService defines the data-access contract for organization/department/group operations.
type organizationService interface {
	// Organizations
	ListOrganizations(ctx context.Context, entityID string, limit, offset int32) ([]OrganizationResponse, error)
	CountOrganizations(ctx context.Context, entityID string) (int64, error)
	GetOrganizationByID(ctx context.Context, entityID, id string) (OrganizationResponse, error)
	CreateOrganization(ctx context.Context, entityID string, name string, parentID string) (OrganizationResponse, error)
	UpdateOrganization(ctx context.Context, entityID, id string, name pgtype.Text, parentID string) (OrganizationResponse, error)
	DeleteOrganization(ctx context.Context, entityID, id string) error
	// Departments
	ListDepartments(ctx context.Context, entityID, orgID string, limit, offset int32) ([]DepartmentResponse, error)
	CountDepartments(ctx context.Context, entityID, orgID string) (int64, error)
	GetDepartmentByID(ctx context.Context, entityID, id string) (DepartmentResponse, error)
	CreateDepartment(ctx context.Context, entityID, orgID string, name string, parentID, sourceID string, externalDeptID pgtype.Text) (DepartmentResponse, error)
	UpdateDepartment(ctx context.Context, entityID, id string, name pgtype.Text, parentID string) (DepartmentResponse, error)
	DeleteDepartment(ctx context.Context, entityID, id string) error
	// Groups
	ListGroups(ctx context.Context, entityID string, groupType pgtype.Text, limit, offset int32) ([]GroupResponse, error)
	CountGroups(ctx context.Context, entityID string, groupType pgtype.Text) (int64, error)
	GetGroupByID(ctx context.Context, entityID, id string) (GroupResponse, error)
	CreateGroup(ctx context.Context, entityID string, name, groupType string) (GroupResponse, error)
	UpdateGroup(ctx context.Context, entityID, id string, name pgtype.Text) (GroupResponse, error)
	DeleteGroup(ctx context.Context, entityID, id string) error
	AddGroupMember(ctx context.Context, entityID, groupID, userID string) error
	RemoveGroupMember(ctx context.Context, entityID, groupID, userID string) error
	ListGroupMembers(ctx context.Context, entityID, groupID string, limit, offset int32) ([]GroupMemberResponse, error)
	CountGroupMembers(ctx context.Context, entityID, groupID string) (int64, error)
}

// OrganizationHandler handles organization CRUD endpoints.
type OrganizationHandler struct {
	service organizationService
}

func NewOrganizationHandler(service organizationService) OrganizationHandler {
	return OrganizationHandler{service: service}
}

func (h OrganizationHandler) RegisterRoutes(r chi.Router) {
	// Organizations
	r.Get("/admin/v1/organizations", h.listOrganizations)
	r.Get("/api/admin/v1/organizations", h.listOrganizations)
	r.Get("/admin/v1/organizations/{id}", h.getOrganization)
	r.Get("/api/admin/v1/organizations/{id}", h.getOrganization)
	r.Post("/admin/v1/organizations", h.createOrganization)
	r.Post("/api/admin/v1/organizations", h.createOrganization)
	r.Put("/admin/v1/organizations/{id}", h.updateOrganization)
	r.Put("/api/admin/v1/organizations/{id}", h.updateOrganization)
	r.Delete("/admin/v1/organizations/{id}", h.deleteOrganization)
	r.Delete("/api/admin/v1/organizations/{id}", h.deleteOrganization)
	// Departments
	r.Get("/admin/v1/departments", h.listDepartments)
	r.Get("/api/admin/v1/departments", h.listDepartments)
	r.Get("/admin/v1/departments/{id}", h.getDepartment)
	r.Get("/api/admin/v1/departments/{id}", h.getDepartment)
	r.Post("/admin/v1/departments", h.createDepartment)
	r.Post("/api/admin/v1/departments", h.createDepartment)
	r.Put("/admin/v1/departments/{id}", h.updateDepartment)
	r.Put("/api/admin/v1/departments/{id}", h.updateDepartment)
	r.Delete("/admin/v1/departments/{id}", h.deleteDepartment)
	r.Delete("/api/admin/v1/departments/{id}", h.deleteDepartment)
	// Groups
	r.Get("/admin/v1/groups", h.listGroups)
	r.Get("/api/admin/v1/groups", h.listGroups)
	r.Get("/admin/v1/groups/{id}", h.getGroup)
	r.Get("/api/admin/v1/groups/{id}", h.getGroup)
	r.Post("/admin/v1/groups", h.createGroup)
	r.Post("/api/admin/v1/groups", h.createGroup)
	r.Put("/admin/v1/groups/{id}", h.updateGroup)
	r.Put("/api/admin/v1/groups/{id}", h.updateGroup)
	r.Delete("/admin/v1/groups/{id}", h.deleteGroup)
	r.Delete("/api/admin/v1/groups/{id}", h.deleteGroup)
	r.Post("/admin/v1/groups/{id}/members", h.addGroupMember)
	r.Post("/api/admin/v1/groups/{id}/members", h.addGroupMember)
	r.Get("/admin/v1/groups/{id}/members", h.listGroupMembers)
	r.Get("/api/admin/v1/groups/{id}/members", h.listGroupMembers)
	r.Delete("/admin/v1/groups/{id}/members/{user_id}", h.removeGroupMember)
	r.Delete("/api/admin/v1/groups/{id}/members/{user_id}", h.removeGroupMember)
}

// --- Organizations ---

func (h OrganizationHandler) listOrganizations(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	limit, offset := parsePagination(r)
	orgs, err := h.service.ListOrganizations(r.Context(), entityID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "organization_list_failed", err.Error())
		return
	}
	total, err := h.service.CountOrganizations(r.Context(), entityID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "organization_count_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, PagedResponse{Items: orgs, Total: total, Limit: int(limit), Offset: int(offset)})
}

func (h OrganizationHandler) getOrganization(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	id, err := ulidValue(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_organization_id", err.Error())
		return
	}
	org, err := h.service.GetOrganizationByID(r.Context(), entityID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "organization_not_found", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, org)
}

func (h OrganizationHandler) createOrganization(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	var body struct {
		Name     string `json:"name"`
		ParentID string `json:"parent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}
	if body.Name == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "name is required")
		return
	}
	var parentID string
	if body.ParentID != "" {
		parentID, err = ulidValue(body.ParentID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_parent_id", err.Error())
			return
		}
	}
	org, err := h.service.CreateOrganization(r.Context(), entityID, body.Name, parentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "organization_create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, org)
}

func (h OrganizationHandler) updateOrganization(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	id, err := ulidValue(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_organization_id", err.Error())
		return
	}
	var body struct {
		Name     string `json:"name"`
		ParentID string `json:"parent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}
	var parentID string
	if body.ParentID != "" {
		parentID, err = ulidValue(body.ParentID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_parent_id", err.Error())
			return
		}
	}
	org, err := h.service.UpdateOrganization(r.Context(), entityID, id, optionalText(body.Name), parentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "organization_update_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, org)
}

func (h OrganizationHandler) deleteOrganization(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	id, err := ulidValue(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_organization_id", err.Error())
		return
	}
	if err := h.service.DeleteOrganization(r.Context(), entityID, id); err != nil {
		writeError(w, http.StatusInternalServerError, "organization_delete_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Departments ---

func (h OrganizationHandler) listDepartments(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	limit, offset := parsePagination(r)
	var orgID string
	if orgIDStr := r.URL.Query().Get("organization_id"); orgIDStr != "" {
		orgID, err = ulidValue(orgIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_organization_id", err.Error())
			return
		}
	}
	depts, err := h.service.ListDepartments(r.Context(), entityID, orgID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "department_list_failed", err.Error())
		return
	}
	total, err := h.service.CountDepartments(r.Context(), entityID, orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "department_count_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, PagedResponse{Items: depts, Total: total, Limit: int(limit), Offset: int(offset)})
}

func (h OrganizationHandler) getDepartment(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	id, err := ulidValue(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_department_id", err.Error())
		return
	}
	dept, err := h.service.GetDepartmentByID(r.Context(), entityID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "department_not_found", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, dept)
}

func (h OrganizationHandler) createDepartment(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	var body struct {
		OrganizationID       string `json:"organization_id"`
		Name                 string `json:"name"`
		ParentID             string `json:"parent_id"`
		SourceID             string `json:"source_id"`
		ExternalDepartmentID string `json:"external_department_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}
	if body.Name == "" || body.OrganizationID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "name and organization_id are required")
		return
	}
	orgID, err := ulidValue(body.OrganizationID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_organization_id", err.Error())
		return
	}
	var parentID string
	if body.ParentID != "" {
		parentID, err = ulidValue(body.ParentID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_parent_id", err.Error())
			return
		}
	}
	var sourceID string
	if body.SourceID != "" {
		sourceID, err = ulidValue(body.SourceID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_source_id", err.Error())
			return
		}
	}
	dept, err := h.service.CreateDepartment(r.Context(), entityID, orgID, body.Name, parentID, sourceID, optionalText(body.ExternalDepartmentID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "department_create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, dept)
}

func (h OrganizationHandler) updateDepartment(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	id, err := ulidValue(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_department_id", err.Error())
		return
	}
	var body struct {
		Name     string `json:"name"`
		ParentID string `json:"parent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}
	var parentID string
	if body.ParentID != "" {
		parentID, err = ulidValue(body.ParentID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_parent_id", err.Error())
			return
		}
	}
	dept, err := h.service.UpdateDepartment(r.Context(), entityID, id, optionalText(body.Name), parentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "department_update_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, dept)
}

func (h OrganizationHandler) deleteDepartment(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	id, err := ulidValue(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_department_id", err.Error())
		return
	}
	if err := h.service.DeleteDepartment(r.Context(), entityID, id); err != nil {
		writeError(w, http.StatusInternalServerError, "department_delete_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Groups ---

func (h OrganizationHandler) listGroups(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	limit, offset := parsePagination(r)
	var groupType pgtype.Text
	if t := r.URL.Query().Get("type"); t != "" {
		groupType = pgtype.Text{String: t, Valid: true}
	}
	groups, err := h.service.ListGroups(r.Context(), entityID, groupType, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "group_list_failed", err.Error())
		return
	}
	total, err := h.service.CountGroups(r.Context(), entityID, groupType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "group_count_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, PagedResponse{Items: groups, Total: total, Limit: int(limit), Offset: int(offset)})
}

func (h OrganizationHandler) getGroup(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	id, err := ulidValue(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_group_id", err.Error())
		return
	}
	group, err := h.service.GetGroupByID(r.Context(), entityID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "group_not_found", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, group)
}

func (h OrganizationHandler) createGroup(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	var body struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}
	if body.Name == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "name is required")
		return
	}
	if body.Type == "" {
		body.Type = "manual"
	}
	group, err := h.service.CreateGroup(r.Context(), entityID, body.Name, body.Type)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "group_create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, group)
}

func (h OrganizationHandler) updateGroup(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	id, err := ulidValue(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_group_id", err.Error())
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}
	group, err := h.service.UpdateGroup(r.Context(), entityID, id, optionalText(body.Name))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "group_update_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, group)
}

func (h OrganizationHandler) deleteGroup(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	id, err := ulidValue(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_group_id", err.Error())
		return
	}
	if err := h.service.DeleteGroup(r.Context(), entityID, id); err != nil {
		writeError(w, http.StatusInternalServerError, "group_delete_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h OrganizationHandler) addGroupMember(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	groupID, err := ulidValue(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_group_id", err.Error())
		return
	}
	var body struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}
	if body.UserID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "user_id is required")
		return
	}
	userID, err := ulidValue(body.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_user_id", err.Error())
		return
	}
	if err := h.service.AddGroupMember(r.Context(), entityID, groupID, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "group_member_add_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h OrganizationHandler) removeGroupMember(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	groupID, err := ulidValue(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_group_id", err.Error())
		return
	}
	userID, err := ulidValue(chi.URLParam(r, "user_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_user_id", err.Error())
		return
	}
	if err := h.service.RemoveGroupMember(r.Context(), entityID, groupID, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "group_member_remove_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h OrganizationHandler) listGroupMembers(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	groupID, err := ulidValue(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_group_id", err.Error())
		return
	}

	limit, offset := parsePagination(r)

	members, err := h.service.ListGroupMembers(r.Context(), entityID, groupID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "group_members_list_failed", err.Error())
		return
	}
	total, err := h.service.CountGroupMembers(r.Context(), entityID, groupID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "group_members_count_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, PagedResponse{Items: members, Total: total, Limit: int(limit), Offset: int(offset)})
}

// --- Row converters ---

func organizationFromRow(row generated.Organization) OrganizationResponse {
	return OrganizationResponse{
		ID:        ulidString(row.ID),
		EntityID:  ulidString(row.EntityID),
		Name:      row.Name,
		ParentID:  optionalULIDString(row.ParentID),
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}

func departmentFromRow(row generated.Department) DepartmentResponse {
	return DepartmentResponse{
		ID:                   ulidString(row.ID),
		EntityID:             ulidString(row.EntityID),
		OrganizationID:       ulidString(row.OrganizationID),
		Name:                 row.Name,
		ParentID:             optionalULIDString(row.ParentID),
		SourceID:             optionalULIDString(row.SourceID),
		ExternalDepartmentID: optionalTextString(row.ExternalDepartmentID),
		CreatedAt:            row.CreatedAt.Time,
		UpdatedAt:            row.UpdatedAt.Time,
	}
}

func groupFromRow(row generated.Group) GroupResponse {
	return GroupResponse{
		ID:        ulidString(row.ID),
		EntityID:  ulidString(row.EntityID),
		Name:      row.Name,
		Type:      row.Type,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}

func optionalULIDString(v pgtype.Text) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

func optionalTextString(v pgtype.Text) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}
