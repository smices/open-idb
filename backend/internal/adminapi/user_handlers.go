// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/smices/open-idb/internal/db/generated"
)

// PagedResponse is the standard envelope for list endpoints.
type PagedResponse struct {
	Items  interface{} `json:"items"`
	Total  int64       `json:"total"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
}

// UserResponse represents a user in API responses.
type UserResponse struct {
	ID              string    `json:"id"`
	EntityID        string    `json:"entity_id"`
	Username        string    `json:"username"`
	DisplayName     string    `json:"display_name"`
	Email           string    `json:"email,omitempty"`
	Phone           string    `json:"phone,omitempty"`
	AvatarURL       string    `json:"avatar_url,omitempty"`
	LifecycleStatus string    `json:"lifecycle_status"`
	UserType        string    `json:"user_type"`
	PrimarySourceID string    `json:"primary_source_id,omitempty"`
	Locale          string    `json:"locale"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// DirectoryUserResponse represents a directory user in API responses.
type DirectoryUserResponse struct {
	ID              string    `json:"id"`
	EntityID        string    `json:"entity_id"`
	SourceID        string    `json:"source_id"`
	ExternalUserID  string    `json:"external_user_id"`
	ExternalUnionID string    `json:"external_union_id,omitempty"`
	ExternalOpenID  string    `json:"external_open_id,omitempty"`
	Name            string    `json:"name"`
	Email           string    `json:"email,omitempty"`
	Phone           string    `json:"phone,omitempty"`
	AvatarURL       string    `json:"avatar_url,omitempty"`
	Status          string    `json:"status"`
	RawProfile      []byte    `json:"raw_profile,omitempty"`
	LastSyncedAt    time.Time `json:"last_synced_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ApplicationResponse represents an application in API responses.
type ApplicationResponse struct {
	ID        string    `json:"id"`
	EntityID  string    `json:"entity_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SyncJobResponse represents a sync job in API responses.
type SyncJobResponse struct {
	ID           string    `json:"id"`
	EntityID     string    `json:"entity_id"`
	SourceID     string    `json:"source_id"`
	Type         string    `json:"type"`
	Provider     string    `json:"provider"`
	Status       string    `json:"status"`
	TraceID      string    `json:"trace_id"`
	StartedAt    time.Time `json:"started_at"`
	FinishedAt   time.Time `json:"finished_at"`
	ErrorMessage string    `json:"error_message,omitempty"`
	Stats        []byte    `json:"stats,omitempty"`
}

// RoleResponse represents a role in API responses.
type RoleResponse struct {
	ID          string    `json:"id"`
	EntityID    string    `json:"entity_id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PermissionResponse represents a permission in API responses.
type PermissionResponse struct {
	ID        string    `json:"id"`
	EntityID  string    `json:"entity_id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// userService defines the data-access contract used by all admin handlers.
// *AdminService satisfies this interface.
type userService interface {
	ListUsers(ctx context.Context, entityID string, status pgtype.Text, limit, offset int32) ([]UserResponse, error)
	CountUsers(ctx context.Context, entityID string, status pgtype.Text) (int64, error)
	GetUserByID(ctx context.Context, entityID, id string) (UserResponse, error)
	UpdateUserLifecycle(ctx context.Context, entityID, id string, status string) (UserResponse, error)
	UpdateUser(ctx context.Context, entityID, id string, displayName, email, phone, locale pgtype.Text) (UserResponse, error)
	ListDirectoryUsers(ctx context.Context, entityID, sourceID string, limit, offset int32) ([]DirectoryUserResponse, error)
	CountDirectoryUsers(ctx context.Context, entityID, sourceID string) (int64, error)
	GetDirectoryUserByID(ctx context.Context, entityID, id string) (DirectoryUserResponse, error)
	ListApplications(ctx context.Context, entityID string, limit, offset int32) ([]ApplicationResponse, error)
	CountApplications(ctx context.Context, entityID string) (int64, error)
	GetApplicationByID(ctx context.Context, entityID, id string) (ApplicationResponse, error)
	CreateApplication(ctx context.Context, entityID string, name, appType string) (ApplicationResponse, error)
	UpdateApplication(ctx context.Context, entityID, id string, name, status pgtype.Text) (ApplicationResponse, error)
	DeleteApplication(ctx context.Context, entityID, id string) error
	ListAllSyncJobs(ctx context.Context, entityID string, limit, offset int32) ([]SyncJobResponse, error)
	CountAllSyncJobs(ctx context.Context, entityID string) (int64, error)
	ListRoles(ctx context.Context, entityID string, limit, offset int32) ([]RoleResponse, error)
	CountRoles(ctx context.Context, entityID string) (int64, error)
	GetRoleByID(ctx context.Context, entityID, id string) (RoleResponse, error)
	ListPermissions(ctx context.Context, entityID string, limit, offset int32) ([]PermissionResponse, error)
	CountPermissions(ctx context.Context, entityID string) (int64, error)
	GetPermissionByID(ctx context.Context, entityID, id string) (PermissionResponse, error)
}

// UserHandler handles user management endpoints.
type UserHandler struct {
	service userService
}

func NewUserHandler(service userService) UserHandler {
	return UserHandler{service: service}
}

func (h UserHandler) RegisterRoutes(r chi.Router) {
	r.Get("/admin/v1/users", h.listUsers)
	r.Get("/api/admin/v1/users", h.listUsers)
	r.Get("/admin/v1/users/{id}", h.getUser)
	r.Get("/api/admin/v1/users/{id}", h.getUser)
	r.Put("/admin/v1/users/{id}", h.updateUser)
	r.Put("/api/admin/v1/users/{id}", h.updateUser)
	r.Post("/admin/v1/users/{id}/disable", h.disableUser)
	r.Post("/api/admin/v1/users/{id}/disable", h.disableUser)
	r.Post("/admin/v1/users/{id}/enable", h.enableUser)
	r.Post("/api/admin/v1/users/{id}/enable", h.enableUser)
}

// parsePagination extracts limit and offset from query params with sensible defaults.
func parsePagination(r *http.Request) (limit int32, offset int32) {
	limit = 20
	offset = 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = int32(parsed)
			if limit > 100 {
				limit = 100
			}
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			offset = int32(parsed)
		}
	}
	return
}

// optionalText returns a valid pgtype.Text when s is non-empty, or NULL otherwise.
func optionalText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{Valid: true, String: s}
}

func (h UserHandler) listUsers(w http.ResponseWriter, r *http.Request) {
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
	status := optionalText(r.URL.Query().Get("status"))

	users, err := h.service.ListUsers(r.Context(), entityID, status, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "user_list_failed", err.Error())
		return
	}
	total, err := h.service.CountUsers(r.Context(), entityID, status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "user_count_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, PagedResponse{
		Items:  users,
		Total:  total,
		Limit:  int(limit),
		Offset: int(offset),
	})
}

func (h UserHandler) getUser(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, http.StatusBadRequest, "invalid_user_id", err.Error())
		return
	}
	user, err := h.service.GetUserByID(r.Context(), entityID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "user_not_found", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h UserHandler) updateUser(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, http.StatusBadRequest, "invalid_user_id", err.Error())
		return
	}
	var body struct {
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
		Phone       string `json:"phone"`
		Locale      string `json:"locale"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}
	user, err := h.service.UpdateUser(r.Context(), entityID, id,
		optionalText(body.DisplayName),
		optionalText(body.Email),
		optionalText(body.Phone),
		optionalText(body.Locale),
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "user_update_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h UserHandler) disableUser(w http.ResponseWriter, r *http.Request) {
	h.setLifecycle(w, r, "disabled")
}

func (h UserHandler) enableUser(w http.ResponseWriter, r *http.Request) {
	h.setLifecycle(w, r, "active")
}

func (h UserHandler) setLifecycle(w http.ResponseWriter, r *http.Request, status string) {
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
		writeError(w, http.StatusBadRequest, "invalid_user_id", err.Error())
		return
	}
	user, err := h.service.UpdateUserLifecycle(r.Context(), entityID, id, status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "lifecycle_update_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// --- Row conversion helpers (shared across the adminapi package) ---

func userFromRow(row generated.User) UserResponse {
	return UserResponse{
		ID:              ulidString(row.ID),
		EntityID:        ulidString(row.EntityID),
		Username:        row.Username,
		DisplayName:     row.DisplayName,
		Email:           textString(row.Email),
		Phone:           textString(row.Phone),
		AvatarURL:       textString(row.AvatarUrl),
		LifecycleStatus: row.LifecycleStatus,
		UserType:        row.UserType,
		PrimarySourceID: textString(row.PrimarySourceID),
		Locale:          textString(row.Locale),
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
	}
}

func directoryUserFromRow(row generated.DirectoryUser) DirectoryUserResponse {
	return DirectoryUserResponse{
		ID:              ulidString(row.ID),
		EntityID:        ulidString(row.EntityID),
		SourceID:        ulidString(row.SourceID),
		ExternalUserID:  row.ExternalUserID,
		ExternalUnionID: textString(row.ExternalUnionID),
		ExternalOpenID:  textString(row.ExternalOpenID),
		Name:            row.Name,
		Email:           textString(row.Email),
		Phone:           textString(row.Phone),
		AvatarURL:       textString(row.AvatarUrl),
		Status:          row.Status,
		RawProfile:      row.RawProfile,
		LastSyncedAt:    row.LastSyncedAt.Time,
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
	}
}

func applicationFromRow(row generated.Application) ApplicationResponse {
	return ApplicationResponse{
		ID:        ulidString(row.ID),
		EntityID:  ulidString(row.EntityID),
		Name:      row.Name,
		Type:      row.Type,
		Status:    row.Status,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}

func roleFromRow(row generated.Role) RoleResponse {
	return RoleResponse{
		ID:          ulidString(row.ID),
		EntityID:    ulidString(row.EntityID),
		Name:        row.Name,
		Code:        row.Code,
		Description: textString(row.Description),
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}
}

func permissionFromRow(row generated.Permission) PermissionResponse {
	return PermissionResponse{
		ID:        ulidString(row.ID),
		EntityID:  ulidString(row.EntityID),
		Code:      row.Code,
		Name:      row.Name,
		Type:      row.Type,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}

func syncJobFromRow(row generated.SyncJob) SyncJobResponse {
	return SyncJobResponse{
		ID:           ulidString(row.ID),
		EntityID:     ulidString(row.EntityID),
		SourceID:     ulidString(row.SourceID),
		Type:         row.Type,
		Provider:     row.Provider,
		Status:       row.Status,
		TraceID:      row.TraceID,
		StartedAt:    row.StartedAt.Time,
		FinishedAt:   row.FinishedAt.Time,
		ErrorMessage: textString(row.ErrorMessage),
		Stats:        row.Stats,
	}
}
