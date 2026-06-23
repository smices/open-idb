// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// internalService defines the data-access contract for internal v1 API handlers.
// *InternalService satisfies this interface.
type internalService interface {
	IntrospectToken(ctx context.Context, entityID string, tokenHash string) (IntrospectResponse, error)
	CheckPermission(ctx context.Context, entityID string, input CheckPermissionInput) (CheckPermissionResult, error)
	GetUserAccess(ctx context.Context, entityID string, userID string) (UserAccessSummary, error)
	CreateAuditEvent(ctx context.Context, entityID string, input AuditEventInput) (AuditEventResult, error)
}

// --- Response and input types ---

// IntrospectResponse is the response for POST /sapi/internal/introspect.
type IntrospectResponse struct {
	Active    bool     `json:"active"`
	UserID    string   `json:"user_id,omitempty"`
	ClientID  string   `json:"client_id,omitempty"`
	Scopes    []string `json:"scopes,omitempty"`
	ExpiresAt string   `json:"expires_at,omitempty"`
	TokenType string   `json:"token_type,omitempty"`
}

// CheckPermissionInput is the request body for POST /sapi/internal/permissions/check.
type CheckPermissionInput struct {
	UserID         string `json:"user_id"`
	PermissionCode string `json:"permission_code"`
	ResourceType   string `json:"resource_type,omitempty"`
	ResourceKey    string `json:"resource_key,omitempty"`
}

// CheckPermissionResult is the response for POST /sapi/internal/permissions/check.
type CheckPermissionResult struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

// UserAccessSummary is the response for GET /sapi/internal/users/{id}/access.
type UserAccessSummary struct {
	UserID       string                  `json:"user_id"`
	EntityID     string                  `json:"entity_id"`
	Status       string                  `json:"lifecycle_status"`
	Applications []UserApplicationAccess `json:"applications"`
}

// UserApplicationAccess describes one application the user can reach.
type UserApplicationAccess struct {
	ApplicationID   string           `json:"application_id"`
	ApplicationName string           `json:"application_name"`
	ApplicationType string           `json:"application_type"`
	HasAccess       bool             `json:"has_access"`
	Roles           []RoleAccessInfo `json:"roles"`
}

// RoleAccessInfo describes a role with its permissions and resource scopes.
type RoleAccessInfo struct {
	RoleID         string              `json:"role_id"`
	RoleCode       string              `json:"role_code"`
	Permissions    []string            `json:"permissions"`
	ResourceScopes []ResourceScopeInfo `json:"resource_scopes"`
}

// ResourceScopeInfo describes a resource scope entry.
type ResourceScopeInfo struct {
	Type   string `json:"type"`
	Key    string `json:"key"`
	Effect string `json:"effect"`
}

// AuditEventInput is the request body for POST /sapi/internal/audit-events.
type AuditEventInput struct {
	ActorUserID  string          `json:"actor_user_id,omitempty"`
	ActorType    string          `json:"actor_type"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resource_type"`
	ResourceID   string          `json:"resource_id"`
	BeforeState  json.RawMessage `json:"before_state,omitempty"`
	AfterState   json.RawMessage `json:"after_state,omitempty"`
	Ip           string          `json:"ip,omitempty"`
	UserAgent    string          `json:"user_agent,omitempty"`
	TraceID      string          `json:"trace_id,omitempty"`
}

// AuditEventResult is the response for POST /sapi/internal/audit-events.
type AuditEventResult struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
}

// --- Handler ---

// InternalHandler handles internal v1 API endpoints used for service-to-service
// authorization. All requests require the X-IDB-Entity-ID header.
type InternalHandler struct {
	service internalService
}

// NewInternalHandler creates a new InternalHandler.
func NewInternalHandler(service internalService) InternalHandler {
	return InternalHandler{service: service}
}

// RegisterRoutes registers internal API routes with the router.
func (h InternalHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/me/access", h.getCurrentUserAccess)
	r.Post("/sapi/internal/introspect", h.introspectToken)
	r.Post("/sapi/internal/permissions/check", h.checkPermission)
	r.Get("/sapi/internal/users/{id}/access", h.getUserAccess)
	r.Post("/sapi/internal/audit-events", h.createAuditEvent)
}

func (h InternalHandler) getCurrentUserAccess(w http.ResponseWriter, r *http.Request) {
	session, ok := readUserSession(w, r)
	if !ok {
		return
	}
	result, err := h.service.GetUserAccess(r.Context(), session.EntityID, session.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "get_user_access_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// introspectToken handles POST /sapi/internal/introspect.
// It accepts a token_hash and returns whether the token is active along with
// its associated metadata.
func (h InternalHandler) introspectToken(w http.ResponseWriter, r *http.Request) {
	entityID, ok := requireEntityID(w, r)
	if !ok {
		return
	}

	var request struct {
		TokenHash string `json:"token_hash"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}
	if request.TokenHash == "" {
		writeError(w, http.StatusBadRequest, "missing_fields", "token_hash is required")
		return
	}

	result, err := h.service.IntrospectToken(r.Context(), entityID, request.TokenHash)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "introspect_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// checkPermission handles POST /sapi/internal/permissions/check.
// It verifies whether a user holds a specific permission, optionally scoped
// to a resource type and key.
func (h InternalHandler) checkPermission(w http.ResponseWriter, r *http.Request) {
	entityID, ok := requireEntityID(w, r)
	if !ok {
		return
	}

	var request CheckPermissionInput
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}
	if request.UserID == "" || request.PermissionCode == "" {
		writeError(w, http.StatusBadRequest, "missing_fields", "user_id and permission_code are required")
		return
	}

	result, err := h.service.CheckPermission(r.Context(), entityID, request)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "check_permission_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// getUserAccess handles GET /sapi/internal/users/{id}/access.
// It returns a full access summary for the specified user including their
// accessible applications, roles, permissions, and resource scopes.
func (h InternalHandler) getUserAccess(w http.ResponseWriter, r *http.Request) {
	entityID, ok := requireEntityID(w, r)
	if !ok {
		return
	}

	userIDStr := chi.URLParam(r, "id")
	userID, err := ulidValue(userIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_user_id", "invalid user id format")
		return
	}

	result, err := h.service.GetUserAccess(r.Context(), entityID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "get_user_access_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// createAuditEvent handles POST /sapi/internal/audit-events.
// It ingests an audit event from another service and writes it to the audit_logs table.
func (h InternalHandler) createAuditEvent(w http.ResponseWriter, r *http.Request) {
	entityID, ok := requireEntityID(w, r)
	if !ok {
		return
	}

	var request AuditEventInput
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid json body")
		return
	}
	if request.Action == "" || request.ActorType == "" {
		writeError(w, http.StatusBadRequest, "missing_fields", "action and actor_type are required")
		return
	}

	result, err := h.service.CreateAuditEvent(r.Context(), entityID, request)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create_audit_event_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

// requireEntityID extracts and validates the X-IDB-Entity-ID header.
// Internal APIs require this header (no session cookie fallback).
func requireEntityID(w http.ResponseWriter, r *http.Request) (string, bool) {
	headerValue := r.Header.Get("X-IDB-Entity-ID")
	if headerValue == "" {
		writeError(w, http.StatusUnauthorized, "entity_required", "X-IDB-Entity-ID header is required")
		return "", false
	}
	entityID, err := ulidValue(headerValue)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity_id", "X-IDB-Entity-ID is not a valid ULID")
		return "", false
	}
	return entityID, true
}
