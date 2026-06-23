// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// ApplicationHandler handles application CRUD endpoints.
type ApplicationHandler struct {
	service userService
}

type applicationEntityResolver interface {
	ResolveOrganizationTreeEntityID(ctx context.Context, candidate string) (string, error)
}

func NewApplicationHandler(service userService) ApplicationHandler {
	return ApplicationHandler{service: service}
}

func (h ApplicationHandler) RegisterRoutes(r chi.Router) {
	r.Get("/sapi/applications", h.listApplications)
	r.Get("/sapi/applications/{id}", h.getApplication)
	r.Post("/sapi/applications", h.createApplication)
	r.Put("/sapi/applications/{id}", h.updateApplication)
	r.Delete("/sapi/applications/{id}", h.deleteApplication)
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
	app, err := h.service.GetApplicationByID(r.Context(), entityID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "application_not_found", err.Error())
		return
	}
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
	var body struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}
	if body.Name == "" || body.Type == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "name and type are required")
		return
	}
	app, err := h.service.CreateApplication(r.Context(), entityID, body.Name, body.Type)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "application_create_failed", err.Error())
		return
	}
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
	var body struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}
	app, err := h.service.UpdateApplication(r.Context(), entityID, id,
		optionalText(body.Name),
		optionalText(body.Status),
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "application_update_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, app)
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
