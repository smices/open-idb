// SPDX-License-Identifier: MIT

package adminapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// PermissionHandler handles permission browsing endpoints.
type PermissionHandler struct {
	service userService
}

func NewPermissionHandler(service userService) PermissionHandler {
	return PermissionHandler{service: service}
}

func (h PermissionHandler) RegisterRoutes(r chi.Router) {
	r.Get("/sapi/permissions", h.listPermissions)
	r.Get("/sapi/permissions/{id}", h.getPermission)
}

func (h PermissionHandler) listPermissions(w http.ResponseWriter, r *http.Request) {
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

	perms, err := h.service.ListPermissions(r.Context(), entityID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "permission_list_failed", err.Error())
		return
	}
	total, err := h.service.CountPermissions(r.Context(), entityID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "permission_count_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, PagedResponse{
		Items:  perms,
		Total:  total,
		Limit:  int(limit),
		Offset: int(offset),
	})
}

func (h PermissionHandler) getPermission(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, http.StatusBadRequest, "invalid_permission_id", err.Error())
		return
	}
	perm, err := h.service.GetPermissionByID(r.Context(), entityID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "permission_not_found", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, perm)
}
