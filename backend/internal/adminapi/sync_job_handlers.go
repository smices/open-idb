// SPDX-License-Identifier: MIT

package adminapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// SyncJobHandler handles sync job browsing endpoints.
type SyncJobHandler struct {
	service userService
}

func NewSyncJobHandler(service userService) SyncJobHandler {
	return SyncJobHandler{service: service}
}

func (h SyncJobHandler) RegisterRoutes(r chi.Router) {
	r.Get("/sapi/sync-jobs", h.listSyncJobs)
}

func (h SyncJobHandler) listSyncJobs(w http.ResponseWriter, r *http.Request) {
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

	jobs, err := h.service.ListAllSyncJobs(r.Context(), entityID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "sync_job_list_failed", err.Error())
		return
	}
	total, err := h.service.CountAllSyncJobs(r.Context(), entityID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "sync_job_count_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, PagedResponse{
		Items:  jobs,
		Total:  total,
		Limit:  int(limit),
		Offset: int(offset),
	})
}
