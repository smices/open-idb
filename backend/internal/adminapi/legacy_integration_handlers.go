// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// legacyAppUserService defines operations for legacy app user mapping management.
// *AdminService satisfies this interface when methods are fully implemented.
type legacyAppUserService interface {
	ListLegacyAppUsers(ctx context.Context, entityID, appID string, limit, offset int32) ([]LegacyAppUserResponse, error)
	CountLegacyAppUsers(ctx context.Context, entityID, appID string) (int64, error)
	GetLegacyAppUser(ctx context.Context, entityID, appID string, username string) (LegacyAppUserResponse, error)
	CreateLegacyAppUser(ctx context.Context, entityID, appID string, input LegacyAppUserCreateInput) (LegacyAppUserResponse, error)
	UpdateLegacyAppUser(ctx context.Context, entityID, appID string, username string, input LegacyAppUserUpdateInput) (LegacyAppUserResponse, error)
	SetLegacyAppUserStatus(ctx context.Context, entityID, appID string, username string, isActive bool) (LegacyAppUserResponse, error)
	DeleteLegacyAppUser(ctx context.Context, entityID, appID string, username string) error
}

// LegacyAppUserHandler handles admin endpoints for legacy app user mappings.
type LegacyAppUserHandler struct {
	service legacyAppUserService
}

func NewLegacyAppUserHandler(service legacyAppUserService) LegacyAppUserHandler {
	return LegacyAppUserHandler{service: service}
}

func (h LegacyAppUserHandler) RegisterRoutes(r chi.Router) {
	r.Get("/admin/v1/applications/{application_id}/legacy-users", h.listLegacyUsers)
	r.Get("/api/admin/v1/applications/{application_id}/legacy-users", h.listLegacyUsers)
	r.Get("/admin/v1/applications/{application_id}/legacy-users/{username}", h.getLegacyUser)
	r.Get("/api/admin/v1/applications/{application_id}/legacy-users/{username}", h.getLegacyUser)
	r.Post("/admin/v1/applications/{application_id}/legacy-users", h.createLegacyUser)
	r.Post("/api/admin/v1/applications/{application_id}/legacy-users", h.createLegacyUser)
	r.Put("/admin/v1/applications/{application_id}/legacy-users/{username}", h.updateLegacyUser)
	r.Put("/api/admin/v1/applications/{application_id}/legacy-users/{username}", h.updateLegacyUser)
	r.Delete("/admin/v1/applications/{application_id}/legacy-users/{username}", h.deleteLegacyUser)
	r.Delete("/api/admin/v1/applications/{application_id}/legacy-users/{username}", h.deleteLegacyUser)
	r.Post("/admin/v1/applications/{application_id}/legacy-users/{username}/enable", h.enableLegacyUser)
	r.Post("/api/admin/v1/applications/{application_id}/legacy-users/{username}/enable", h.enableLegacyUser)
	r.Post("/admin/v1/applications/{application_id}/legacy-users/{username}/disable", h.disableLegacyUser)
	r.Post("/api/admin/v1/applications/{application_id}/legacy-users/{username}/disable", h.disableLegacyUser)
}

func (h LegacyAppUserHandler) listLegacyUsers(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	applicationID, err := ulidValue(chi.URLParam(r, "application_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_application_id", err.Error())
		return
	}

	limit, offset := parsePagination(r)
	items, err := h.service.ListLegacyAppUsers(r.Context(), entityID, applicationID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "legacy_app_user_list_failed", err.Error())
		return
	}
	total, err := h.service.CountLegacyAppUsers(r.Context(), entityID, applicationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "legacy_app_user_count_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, PagedResponse{
		Items:  items,
		Total:  total,
		Limit:  int(limit),
		Offset: int(offset),
	})
}

func (h LegacyAppUserHandler) getLegacyUser(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	applicationID, err := ulidValue(chi.URLParam(r, "application_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_application_id", err.Error())
		return
	}
	username := strings.TrimSpace(chi.URLParam(r, "username"))
	if username == "" {
		writeError(w, http.StatusBadRequest, "invalid_username", "username is required")
		return
	}

	user, err := h.service.GetLegacyAppUser(r.Context(), entityID, applicationID, username)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "legacy_app_user_not_found", err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusNotFound, "legacy_app_user_not_found", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h LegacyAppUserHandler) createLegacyUser(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	applicationID, err := ulidValue(chi.URLParam(r, "application_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_application_id", err.Error())
		return
	}

	var req struct {
		Username             string `json:"username"`
		UserID               string `json:"user_id"`
		Password             string `json:"password"`
		LegacyUserIdentifier string `json:"legacy_user_identifier"`
		IsActive             *bool  `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}
	username := strings.TrimSpace(req.Username)
	userID := strings.TrimSpace(req.UserID)
	if username == "" || userID == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "username, user_id and password are required")
		return
	}

	created, err := h.service.CreateLegacyAppUser(r.Context(), entityID, applicationID, LegacyAppUserCreateInput{
		Username:             username,
		UserID:               userID,
		Password:             req.Password,
		LegacyUserIdentifier: req.LegacyUserIdentifier,
		IsActive:             req.IsActive,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "legacy_app_user_create_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

func (h LegacyAppUserHandler) updateLegacyUser(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	applicationID, err := ulidValue(chi.URLParam(r, "application_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_application_id", err.Error())
		return
	}
	username := strings.TrimSpace(chi.URLParam(r, "username"))
	if username == "" {
		writeError(w, http.StatusBadRequest, "invalid_username", "username is required")
		return
	}

	var req struct {
		UserID               *string `json:"user_id"`
		Password             *string `json:"password"`
		LegacyUserIdentifier *string `json:"legacy_user_identifier"`
		IsActive             *bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}

	updated, err := h.service.UpdateLegacyAppUser(r.Context(), entityID, applicationID, username, LegacyAppUserUpdateInput{
		UserID:               req.UserID,
		Password:             req.Password,
		LegacyUserIdentifier: req.LegacyUserIdentifier,
		IsActive:             req.IsActive,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "legacy_app_user_not_found", err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "legacy_app_user_update_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h LegacyAppUserHandler) enableLegacyUser(w http.ResponseWriter, r *http.Request) {
	h.updateLegacyUserStatus(w, r, true)
}

func (h LegacyAppUserHandler) disableLegacyUser(w http.ResponseWriter, r *http.Request) {
	h.updateLegacyUserStatus(w, r, false)
}

func (h LegacyAppUserHandler) updateLegacyUserStatus(w http.ResponseWriter, r *http.Request, isActive bool) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	applicationID, err := ulidValue(chi.URLParam(r, "application_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_application_id", err.Error())
		return
	}
	username := strings.TrimSpace(chi.URLParam(r, "username"))
	if username == "" {
		writeError(w, http.StatusBadRequest, "invalid_username", "username is required")
		return
	}

	updated, err := h.service.SetLegacyAppUserStatus(r.Context(), entityID, applicationID, username, isActive)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "legacy_app_user_not_found", err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "legacy_app_user_set_status_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h LegacyAppUserHandler) deleteLegacyUser(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}

	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	applicationID, err := ulidValue(chi.URLParam(r, "application_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_application_id", err.Error())
		return
	}
	username := strings.TrimSpace(chi.URLParam(r, "username"))
	if username == "" {
		writeError(w, http.StatusBadRequest, "invalid_username", "username is required")
		return
	}

	if err := h.service.DeleteLegacyAppUser(r.Context(), entityID, applicationID, username); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "legacy_app_user_not_found", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "legacy_app_user_delete_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
