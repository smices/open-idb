// SPDX-License-Identifier: MIT

package adminapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RoleHandler handles role browsing endpoints.
type RoleHandler struct {
	service userService
}

func NewRoleHandler(service userService) RoleHandler {
	return RoleHandler{service: service}
}

func (h RoleHandler) RegisterRoutes(r chi.Router) {
	r.Get("/sapi/roles", h.listRoles)
	r.Get("/sapi/roles/{id}", h.getRole)
}

func (h RoleHandler) listRoles(w http.ResponseWriter, r *http.Request) {
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

	roles, err := h.service.ListRoles(r.Context(), entityID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "role_list_failed", err.Error())
		return
	}
	total, err := h.service.CountRoles(r.Context(), entityID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "role_count_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, PagedResponse{
		Items:  roles,
		Total:  total,
		Limit:  int(limit),
		Offset: int(offset),
	})
}

func (h RoleHandler) getRole(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, http.StatusBadRequest, "invalid_role_id", err.Error())
		return
	}
	role, err := h.service.GetRoleByID(r.Context(), entityID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "role_not_found", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, role)
}
