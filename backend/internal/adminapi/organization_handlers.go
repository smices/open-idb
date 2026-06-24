// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type organizationService interface {
	ResolveOrganizationTreeEntityID(ctx context.Context, candidate string) (string, error)
	GetOrganizationTreeRoot(ctx context.Context, entityID string, limit, offset int32) (OrganizationTreeRootResponse, error)
	ListOrganizationTreeChildren(ctx context.Context, entityID string, kind OrganizationTreeNodeKind, parentID string, limit, offset int32) ([]OrganizationTreeNode, error)
	SearchOrganizationTree(ctx context.Context, entityID, query string, limit, offset int32) (OrganizationTreeSearchResponse, error)
}

type OrganizationHandler struct {
	service organizationService
}

func NewOrganizationHandler(service organizationService) OrganizationHandler {
	return OrganizationHandler{service: service}
}

func (h OrganizationHandler) RegisterRoutes(r chi.Router) {
	r.Get("/sapi/organization-tree/root", h.getOrganizationTreeRoot)
	r.Get("/sapi/organization-tree/children", h.listOrganizationTreeChildren)
	r.Get("/sapi/organization-tree/search", h.searchOrganizationTree)
}

func (h OrganizationHandler) getOrganizationTreeRoot(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := h.service.ResolveOrganizationTreeEntityID(r.Context(), session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	limit, offset := parsePagination(r)
	root, err := h.service.GetOrganizationTreeRoot(r.Context(), entityID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "organization_tree_root_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, root)
}

func (h OrganizationHandler) listOrganizationTreeChildren(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := h.service.ResolveOrganizationTreeEntityID(r.Context(), session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	parentID, err := ulidValue(r.URL.Query().Get("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_parent_id", err.Error())
		return
	}
	kind := OrganizationTreeNodeKind(r.URL.Query().Get("kind"))
	limit, offset := parsePagination(r)
	children, err := h.service.ListOrganizationTreeChildren(r.Context(), entityID, kind, parentID, limit, offset)
	if err != nil {
		writeError(w, http.StatusBadRequest, "organization_tree_children_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, PagedResponse{Items: children, Total: int64(len(children)), Limit: int(limit), Offset: int(offset)})
}

func (h OrganizationHandler) searchOrganizationTree(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := h.service.ResolveOrganizationTreeEntityID(r.Context(), session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	query := r.URL.Query().Get("q")
	limit, offset := parsePagination(r)
	results, err := h.service.SearchOrganizationTree(r.Context(), entityID, query, limit, offset)
	if err != nil {
		writeError(w, http.StatusBadRequest, "organization_tree_search_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, results)
}
