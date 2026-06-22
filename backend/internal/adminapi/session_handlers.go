// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/smices/open-idb/internal/audit"
	"github.com/smices/open-idb/internal/db/generated"
)

// SessionResponse represents a user session in API responses.
type SessionResponse struct {
	ID          string    `json:"id"`
	EntityID    string    `json:"entity_id"`
	UserID      string    `json:"user_id"`
	DeviceID    string    `json:"device_id,omitempty"`
	IP          string    `json:"ip,omitempty"`
	UserAgent   string    `json:"user_agent,omitempty"`
	LoginMethod string    `json:"login_method"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
}

// SessionHandler handles session management endpoints.
type SessionHandler struct {
	service sessionService
}

type sessionService interface {
	ListSessionsByUser(ctx context.Context, entityID, userID string, limit int32) ([]SessionResponse, error)
	GetSessionByID(ctx context.Context, entityID, id string) (SessionResponse, error)
	RevokeSession(ctx context.Context, entityID, id string) error
}

func NewSessionHandler(service sessionService) SessionHandler {
	return SessionHandler{service: service}
}

func (h SessionHandler) RegisterRoutes(r chi.Router) {
	r.Get("/admin/v1/users/{userID}/sessions", h.listUserSessions)
	r.Get("/api/admin/v1/users/{userID}/sessions", h.listUserSessions)
	r.Post("/admin/v1/sessions/{id}/revoke", h.revokeSession)
	r.Post("/api/admin/v1/sessions/{id}/revoke", h.revokeSession)
}

func (h SessionHandler) listUserSessions(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	entityID, err := ulidValue(session.EntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", err.Error())
		return
	}
	userID, err := ulidValue(chi.URLParam(r, "userID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_user_id", err.Error())
		return
	}
	limit := int32(50)
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := parseInt32(v); err == nil && parsed > 0 {
			limit = parsed
			if limit > 200 {
				limit = 200
			}
		}
	}
	sessions, err := h.service.ListSessionsByUser(r.Context(), entityID, userID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "session_list_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": sessions,
	})
}

func (h SessionHandler) revokeSession(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, http.StatusBadRequest, "invalid_session_id", err.Error())
		return
	}
	if err := h.service.RevokeSession(r.Context(), entityID, id); err != nil {
		writeError(w, http.StatusInternalServerError, "session_revoke_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func parseInt32(s string) (int32, error) {
	var n int32
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// --- AdminService session methods ---

func (s *AdminService) ListSessionsByUser(ctx context.Context, entityID, userID string, limit int32) ([]SessionResponse, error) {
	rows, err := s.queries.ListSessionsByUser(ctx, generated.ListSessionsByUserParams{
		EntityID: entityID,
		UserID:   userID,
		Limit:    limit,
	})
	if err != nil {
		return nil, err
	}
	sessions := make([]SessionResponse, 0, len(rows))
	for _, row := range rows {
		sessions = append(sessions, sessionFromRow(row))
	}
	return sessions, nil
}

func (s *AdminService) GetSessionByID(ctx context.Context, entityID, id string) (SessionResponse, error) {
	row, err := s.queries.GetSessionByID(ctx, generated.GetSessionByIDParams{
		EntityID: entityID,
		ID:       id,
	})
	if err != nil {
		return SessionResponse{}, err
	}
	return sessionFromRow(row), nil
}

func (s *AdminService) RevokeSession(ctx context.Context, entityID, id string) error {
	if err := s.queries.RevokeSession(ctx, generated.RevokeSessionParams{
		EntityID: entityID,
		ID:       id,
	}); err != nil {
		return err
	}
	s.audit.logAction(ctx, audit.Event{
		EntityID:     ulidString(entityID),
		ActorType:    "user",
		Action:       audit.ActionLogout,
		ResourceType: "session",
		ResourceID:   ulidString(id),
	})
	return nil
}

func sessionFromRow(row generated.Session) SessionResponse {
	resp := SessionResponse{
		ID:          ulidString(row.ID),
		EntityID:    ulidString(row.EntityID),
		UserID:      ulidString(row.UserID),
		DeviceID:    row.DeviceID,
		IP:          row.Ip,
		UserAgent:   row.UserAgent,
		LoginMethod: row.LoginMethod,
		Status:      row.Status,
		CreatedAt:   row.CreatedAt.Time,
	}
	if row.ExpiresAt.Valid {
		resp.ExpiresAt = row.ExpiresAt.Time
	}
	return resp
}
