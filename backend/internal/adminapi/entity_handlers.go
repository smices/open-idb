// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type EntityResponse struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	Status        string    `json:"status"`
	DefaultLocale string    `json:"default_locale"`
	BrandName     string    `json:"brand_name"`
	LogoURL       string    `json:"logo_url"`
	LoginMessage  string    `json:"login_message"`
	CreatedAt     time.Time `json:"created_at"`
}

type entityService interface {
	ListEntities(ctx context.Context, limit, offset int32) ([]EntityResponse, error)
	CountEntities(ctx context.Context) (int64, error)
	GetEntityByID(ctx context.Context, id string) (EntityResponse, error)
	CreateEntity(ctx context.Context, name, slug, defaultLocale, brandName, logoURL, loginMessage string) (EntityResponse, error)
	UpdateEntity(ctx context.Context, id string, name, status, defaultLocale, brandName, logoURL, loginMessage pgtype.Text) (EntityResponse, error)
}

type EntityHandler struct {
	service entityService
}

func NewEntityHandler(service entityService) EntityHandler {
	return EntityHandler{service: service}
}

func (h EntityHandler) RegisterRoutes(r chi.Router) {
	r.Get("/sapi/entities", h.listEntities)
	r.Get("/sapi/entities/{id}", h.getEntity)
	r.Post("/sapi/entities", h.createEntity)
	r.Put("/sapi/entities/{id}", h.updateEntity)
}

func (h EntityHandler) listEntities(w http.ResponseWriter, r *http.Request) {
	if _, ok := readSession(w, r); !ok {
		return
	}
	limit, offset := parsePagination(r)
	entities, err := h.service.ListEntities(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "entity_list_failed", err.Error())
		return
	}
	total, err := h.service.CountEntities(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "entity_count_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, PagedResponse{
		Items:  entities,
		Total:  total,
		Limit:  int(limit),
		Offset: int(offset),
	})
}

func (h EntityHandler) getEntity(w http.ResponseWriter, r *http.Request) {
	if _, ok := readSession(w, r); !ok {
		return
	}
	id, err := ulidValue(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	entity, err := h.service.GetEntityByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "entity_not_found", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, entity)
}

func (h EntityHandler) createEntity(w http.ResponseWriter, r *http.Request) {
	if _, ok := readSession(w, r); !ok {
		return
	}
	var body struct {
		Name          string `json:"name"`
		Slug          string `json:"slug"`
		DefaultLocale string `json:"default_locale"`
		BrandName     string `json:"brand_name"`
		LogoURL       string `json:"logo_url"`
		LoginMessage  string `json:"login_message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}
	if body.Name == "" || body.Slug == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "name and slug are required")
		return
	}
	if body.DefaultLocale == "" {
		body.DefaultLocale = "en-US"
	}
	entity, err := h.service.CreateEntity(r.Context(), body.Name, body.Slug, body.DefaultLocale, body.BrandName, body.LogoURL, body.LoginMessage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "entity_create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, entity)
}

func (h EntityHandler) updateEntity(w http.ResponseWriter, r *http.Request) {
	if _, ok := readSession(w, r); !ok {
		return
	}
	id, err := ulidValue(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	var body struct {
		Name          string `json:"name"`
		Status        string `json:"status"`
		DefaultLocale string `json:"default_locale"`
		BrandName     string `json:"brand_name"`
		LogoURL       string `json:"logo_url"`
		LoginMessage  string `json:"login_message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}
	entity, err := h.service.UpdateEntity(
		r.Context(),
		id,
		optionalText(body.Name),
		optionalText(body.Status),
		optionalText(body.DefaultLocale),
		optionalText(body.BrandName),
		optionalText(body.LogoURL),
		optionalText(body.LoginMessage),
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "entity_update_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, entity)
}

func entityFromRow(id string, name, slug, status, defaultLocale, brandName, logoURL, loginMessage string, createdAt pgtype.Timestamptz) EntityResponse {
	return EntityResponse{
		ID:            ulidString(id),
		Name:          name,
		Slug:          slug,
		Status:        status,
		DefaultLocale: defaultLocale,
		BrandName:     brandName,
		LogoURL:       logoURL,
		LoginMessage:  loginMessage,
		CreatedAt:     createdAt.Time,
	}
}
