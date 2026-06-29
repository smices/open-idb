// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/smices/open-idb/internal/auth"
	"github.com/smices/open-idb/internal/db/generated"
)

type PlatformBrandingResponse struct {
	PlatformName string    `json:"platform_name"`
	LogoURL      string    `json:"logo_url"`
	FaviconURL   string    `json:"favicon_url"`
	TitleSuffix  string    `json:"title_suffix"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type platformService interface {
	GetPlatformBranding(ctx context.Context) (PlatformBrandingResponse, error)
	UpdatePlatformBranding(ctx context.Context, platformName, logoURL, faviconURL, titleSuffix string) (PlatformBrandingResponse, error)
}

type PlatformHandler struct {
	service platformService
}

func NewPlatformHandler(service platformService) PlatformHandler {
	return PlatformHandler{service: service}
}

func (h PlatformHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/platform/branding", h.getBranding)
	r.Get("/sapi/platform/branding", h.getBrandingForAdmin)
	r.Put("/sapi/platform/branding", h.updateBranding)
}

func (h PlatformHandler) getBranding(w http.ResponseWriter, r *http.Request) {
	branding, err := h.service.GetPlatformBranding(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "platform_branding_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, branding)
}

func (h PlatformHandler) getBrandingForAdmin(w http.ResponseWriter, r *http.Request) {
	if _, ok := readSession(w, r); !ok {
		return
	}
	h.getBranding(w, r)
}

func (h PlatformHandler) updateBranding(w http.ResponseWriter, r *http.Request) {
	adminSession, ok := readAdminSessionForPlatform(w, r)
	if !ok {
		return
	}
	if adminSession.Role != "platform_admin" {
		writeError(w, http.StatusForbidden, "platform_admin_required", "platform administrator role is required")
		return
	}
	var body struct {
		PlatformName string `json:"platform_name"`
		LogoURL      string `json:"logo_url"`
		FaviconURL   string `json:"favicon_url"`
		TitleSuffix  string `json:"title_suffix"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}
	if strings.TrimSpace(body.PlatformName) == "" {
		writeError(w, http.StatusBadRequest, "invalid_platform_name", "platform_name is required")
		return
	}
	branding, err := h.service.UpdatePlatformBranding(r.Context(), body.PlatformName, body.LogoURL, body.FaviconURL, body.TitleSuffix)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "platform_branding_update_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, branding)
}

func platformBrandingFromRow(row generated.PlatformSetting) PlatformBrandingResponse {
	return PlatformBrandingResponse{
		PlatformName: row.PlatformName,
		LogoURL:      row.LogoUrl,
		FaviconURL:   row.FaviconUrl,
		TitleSuffix:  row.TitleSuffix,
		UpdatedAt:    row.UpdatedAt.Time,
	}
}

func readAdminSessionForPlatform(w http.ResponseWriter, r *http.Request) (auth.AdminSession, bool) {
	cookie, err := r.Cookie("idb_admin_session")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "admin_session_required", "idb_admin_session cookie is required")
		return auth.AdminSession{}, false
	}
	session, err := auth.ResolveAdminSession(r.Context(), cookie.Value)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_admin_session", "idb_admin_session cookie is invalid")
		return auth.AdminSession{}, false
	}
	return session, true
}
