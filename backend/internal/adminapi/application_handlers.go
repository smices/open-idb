// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// ApplicationHandler handles application CRUD endpoints.
type ApplicationHandler struct {
	service userService
}

func isValidApplicationType(appType string) bool {
	switch appType {
	case "oidc_client", "api_client", "internal_app":
		return true
	default:
		return false
	}
}

type applicationEntityResolver interface {
	ResolveOrganizationTreeEntityID(ctx context.Context, candidate string) (string, error)
}

// ApplicationWriteInput is the additive request contract used by the complete
// application editor. Config is type-specific; OIDC settings continue to use
// the existing oidc_clients table and API.
type ApplicationWriteInput struct {
	Name       string                      `json:"name"`
	Type       string                      `json:"type"`
	Status     string                      `json:"status"`
	Config     json.RawMessage             `json:"config"`
	OIDCClient *ApplicationOIDCClientInput `json:"oidc_client"`
}

type ApplicationOIDCClientInput struct {
	ClientID           string   `json:"client_id"`
	RedirectURIs       []string `json:"redirect_uris"`
	AllowedScopes      []string `json:"allowed_scopes"`
	GrantTypes         []string `json:"grant_types"`
	ResponseTypes      []string `json:"response_types"`
	PKCERequired       *bool    `json:"pkce_required"`
	WorkplaceProvider  *string  `json:"workplace_provider"`
	WorkplaceAppID     *string  `json:"workplace_app_id"`
	WorkplaceAppSecret *string  `json:"workplace_app_secret"`
}

type applicationDetailService interface {
	GetApplicationDetail(ctx context.Context, entityID, id string) (ApplicationDetailResponse, error)
	CreateApplicationDetail(ctx context.Context, entityID string, input ApplicationWriteInput) (ApplicationDetailResponse, error)
	UpdateApplicationDetail(ctx context.Context, entityID, id string, input ApplicationWriteInput) (ApplicationDetailResponse, error)
}

type applicationRequestError struct {
	message string
}

func (e *applicationRequestError) Error() string { return e.message }

func NewApplicationHandler(service userService) ApplicationHandler {
	return ApplicationHandler{service: service}
}

func (h ApplicationHandler) RegisterRoutes(r chi.Router) {
	r.Get("/sapi/applications", h.listApplications)
	r.Get("/sapi/applications/{id}", h.getApplication)
	r.Post("/sapi/applications", h.createApplication)
	r.Put("/sapi/applications/{id}", h.updateApplication)
	r.Delete("/sapi/applications/{id}", h.deleteApplication)
	r.Get("/sapi/applications/{id}/role-assignments", h.listRoleAssignments)
	r.Put("/sapi/applications/{id}/role-assignments", h.setRoleAssignments)
}

func (h ApplicationHandler) resolveEntityID(ctx context.Context, candidate string) (string, error) {
	if resolver, ok := h.service.(applicationEntityResolver); ok {
		return resolver.ResolveOrganizationTreeEntityID(ctx, candidate)
	}
	return ulidValue(candidate)
}

func (h ApplicationHandler) listApplications(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := h.resolveEntityID(r.Context(), session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	limit, offset := parsePagination(r)

	apps, err := h.service.ListApplications(r.Context(), entityID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "application_list_failed", err.Error())
		return
	}
	total, err := h.service.CountApplications(r.Context(), entityID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "application_count_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":        apps,
		"applications": apps,
		"total":        total,
		"limit":        int(limit),
		"offset":       int(offset),
	})
}

func (h ApplicationHandler) getApplication(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := h.resolveEntityID(r.Context(), session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	id, err := ulidValue(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_application_id", err.Error())
		return
	}
	var app interface{}
	if service, ok := h.service.(applicationDetailService); ok {
		app, err = service.GetApplicationDetail(r.Context(), entityID, id)
	} else {
		app, err = h.service.GetApplicationByID(r.Context(), entityID, id)
	}
	if err != nil {
		writeError(w, http.StatusNotFound, "application_not_found", err.Error())
		return
	}
	setApplicationNoStoreHeaders(w)
	writeJSON(w, http.StatusOK, app)
}

func (h ApplicationHandler) createApplication(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := h.resolveEntityID(r.Context(), session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	var body ApplicationWriteInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}
	if body.Name == "" || body.Type == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "name and type are required")
		return
	}
	if !isValidApplicationType(body.Type) {
		writeError(w, http.StatusBadRequest, "invalid_application_type", "type must be one of oidc_client, api_client, or internal_app")
		return
	}
	var app interface{}
	if service, ok := h.service.(applicationDetailService); ok {
		app, err = service.CreateApplicationDetail(r.Context(), entityID, body)
	} else {
		app, err = h.service.CreateApplication(r.Context(), entityID, body.Name, body.Type)
	}
	if err != nil {
		var requestErr *applicationRequestError
		if errors.As(err, &requestErr) {
			writeError(w, http.StatusBadRequest, "invalid_request", requestErr.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "application_create_failed", err.Error())
		return
	}
	setApplicationNoStoreHeaders(w)
	writeJSON(w, http.StatusCreated, app)
}

func (h ApplicationHandler) updateApplication(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := h.resolveEntityID(r.Context(), session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	id, err := ulidValue(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_application_id", err.Error())
		return
	}
	var body ApplicationWriteInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}
	var app interface{}
	if service, ok := h.service.(applicationDetailService); ok {
		app, err = service.UpdateApplicationDetail(r.Context(), entityID, id, body)
	} else {
		app, err = h.service.UpdateApplication(r.Context(), entityID, id,
			optionalText(body.Name),
			optionalText(body.Status),
		)
	}
	if err != nil {
		var requestErr *applicationRequestError
		if errors.As(err, &requestErr) {
			writeError(w, http.StatusBadRequest, "invalid_request", requestErr.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "application_update_failed", err.Error())
		return
	}
	setApplicationNoStoreHeaders(w)
	writeJSON(w, http.StatusOK, app)
}

func setApplicationNoStoreHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
}

func (h ApplicationHandler) deleteApplication(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := h.resolveEntityID(r.Context(), session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	id, err := ulidValue(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_application_id", err.Error())
		return
	}
	if err := h.service.DeleteApplication(r.Context(), entityID, id); err != nil {
		writeError(w, http.StatusInternalServerError, "application_delete_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h ApplicationHandler) listRoleAssignments(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := h.resolveEntityID(r.Context(), session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	id, err := ulidValue(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_application_id", err.Error())
		return
	}
	assignments, err := h.service.ListApplicationRoleAssignments(r.Context(), entityID, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "application_assignment_list_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": assignments,
		"roles": assignments,
	})
}

func (h ApplicationHandler) setRoleAssignments(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := h.resolveEntityID(r.Context(), session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	id, err := ulidValue(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_application_id", err.Error())
		return
	}
	var body struct {
		RoleIDs []string `json:"role_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}
	roleIDs := make([]string, 0, len(body.RoleIDs))
	for _, roleID := range body.RoleIDs {
		value, err := ulidValue(roleID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_role_id", err.Error())
			return
		}
		roleIDs = append(roleIDs, value)
	}
	if err := h.service.SetApplicationRoleAssignments(r.Context(), entityID, id, roleIDs); err != nil {
		writeError(w, http.StatusInternalServerError, "application_assignment_update_failed", err.Error())
		return
	}
	assignments, err := h.service.ListApplicationRoleAssignments(r.Context(), entityID, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "application_assignment_list_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": assignments,
		"roles": assignments,
	})
}
