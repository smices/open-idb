// SPDX-License-Identifier: MIT

package adminapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// DirectoryHandler handles directory user browsing endpoints.
type DirectoryHandler struct {
	service userService
}

func NewDirectoryHandler(service userService) DirectoryHandler {
	return DirectoryHandler{service: service}
}

func (h DirectoryHandler) RegisterRoutes(r chi.Router) {
	r.Get("/sapi/directory-users/{id}", h.getDirectoryUser)
}

func (h DirectoryHandler) getDirectoryUser(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, http.StatusBadRequest, "invalid_directory_user_id", err.Error())
		return
	}
	user, err := h.service.GetDirectoryUserByID(r.Context(), entityID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "directory_user_not_found", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, user)
}
