// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/smices/open-idb/internal/auth"
	"github.com/smices/open-idb/internal/idp"
)

type SyncService interface {
	RunFullSync(ctx context.Context, input idp.FullSyncInput) (idp.FullSyncResult, error)
	RunIncrementalSync(ctx context.Context, input idp.FullSyncInput) (idp.FullSyncResult, error)
	SubmitWebhookEvent(ctx context.Context, entityID, sourceID string, event idp.DirectorySyncEvent) (string, error)
}

type DefaultFeishuWebhookTargetResolver interface {
	ResolveDefaultFeishuWebhookTarget(ctx context.Context) (entityID string, sourceID string, err error)
}

type ConsoleService interface {
	DashboardSummary(ctx context.Context, session auth.Session) (DashboardSummary, error)
	CurrentUser(ctx context.Context, session auth.Session) (CurrentUser, error)
	UpdatePassword(ctx context.Context, input UpdatePasswordInput) error
}

type DashboardSummary struct {
	Users                int64  `json:"users"`
	ActiveUsers          int64  `json:"active_users"`
	NewUsers             int64  `json:"new_users"`
	ApplicationActivity  int64  `json:"application_activity"`
	PendingAuthorization int64  `json:"pending_authorization"`
	SyncHealth           string `json:"sync_health"`
}

type CurrentUser struct {
	ID                 string   `json:"id"`
	EntityID           string   `json:"entity_id"`
	Username           string   `json:"username"`
	DisplayName        string   `json:"display_name"`
	Email              string   `json:"email,omitempty"`
	Phone              string   `json:"phone,omitempty"`
	AvatarURL          string   `json:"avatar_url,omitempty"`
	Locale             string   `json:"locale"`
	MustChangePassword bool     `json:"must_change_password"`
	WeakPassword       bool     `json:"weak_password"`
	ConsoleScope       string   `json:"console_scope"`
	Capabilities       []string `json:"capabilities"`
}

type UpdatePasswordInput struct {
	EntityID        string
	UserID          string
	Username        string
	CurrentPassword string
	NewPassword     string
	WeakPassword    bool
}

type Handler struct {
	syncService    SyncService
	consoleService ConsoleService
}

func NewHandler(syncService SyncService, consoleService ConsoleService) Handler {
	return Handler{syncService: syncService, consoleService: consoleService}
}

func (h Handler) RegisterRoutes(r chi.Router) {
	if h.syncService != nil {
		r.Post("/admin/v1/identity-sources/{source_id}/sync/full", h.triggerFullSync)
		r.Post("/api/admin/v1/identity-sources/{source_id}/sync/full", h.triggerFullSync)
		r.Post("/admin/v1/identity-sources/{source_id}/sync/incremental", h.triggerIncrementalSync)
		r.Post("/api/admin/v1/identity-sources/{source_id}/sync/incremental", h.triggerIncrementalSync)
		r.Post("/api/webhooks/feishu", h.handleFeishuWebhook)
		r.Post("/api/webhooks/feishu/{entity_id}/{source_id}", h.handleFeishuWebhook)
	}
	if h.consoleService != nil {
		r.Get("/admin/v1/dashboard/summary", h.dashboardSummary)
		r.Get("/admin/v1/me", h.currentUser)
		r.Post("/admin/v1/me/password", h.updatePassword)
		r.Get("/api/admin/v1/dashboard/summary", h.dashboardSummary)
		r.Get("/api/admin/v1/me", h.currentUser)
		r.Post("/api/admin/v1/me/password", h.updatePassword)
	}
}

func (h Handler) triggerFullSync(w http.ResponseWriter, r *http.Request) {
	h.triggerSync(w, r, idp.SyncModeFull)
}

func (h Handler) triggerIncrementalSync(w http.ResponseWriter, r *http.Request) {
	h.triggerSync(w, r, idp.SyncModeIncremental)
}

func (h Handler) triggerSync(w http.ResponseWriter, r *http.Request, syncType idp.SyncMode) {
	entityID, ok := entityIDForRequest(w, r)
	if !ok {
		return
	}

	var (
		result idp.FullSyncResult
		err    error
	)
	input := idp.FullSyncInput{
		EntityID: entityID,
		SourceID: chi.URLParam(r, "source_id"),
		Provider: "",
		SyncType: syncType,
	}

	switch syncType {
	case idp.SyncModeIncremental:
		result, err = h.syncService.RunIncrementalSync(r.Context(), input)
	default:
		result, err = h.syncService.RunFullSync(r.Context(), input)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "sync_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h Handler) handleFeishuWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_webhook_body", "invalid request body")
		return
	}

	var envelope struct {
		Type      string `json:"type"`
		Challenge string `json:"challenge"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_webhook_body", "invalid json body")
		return
	}
	if strings.EqualFold(envelope.Type, "url_verification") {
		writeJSON(w, http.StatusOK, map[string]string{
			"challenge": envelope.Challenge,
		})
		return
	}
	if h.syncService == nil {
		writeError(w, http.StatusServiceUnavailable, "sync_service_unavailable", "sync service is not configured")
		return
	}

	event, err := parseFeishuWebhookEvent(body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_webhook_event", err.Error())
		return
	}
	if !event.IsKnown() {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
		return
	}

	entityID, sourceID, ok := h.feishuWebhookTarget(w, r)
	if !ok {
		return
	}
	jobID, err := h.syncService.SubmitWebhookEvent(r.Context(), entityID, sourceID, event)
	if err != nil {
		writeError(w, http.StatusBadRequest, "submit_webhook_failed", err.Error())
		return
	}

	// Trigger asynchronous incremental sync immediately; webhook provider only requires fast ack.
	go func() {
		_, _ = h.syncService.RunIncrementalSync(context.Background(), idp.FullSyncInput{
			EntityID: entityID,
			SourceID: sourceID,
		})
	}()
	writeJSON(w, http.StatusAccepted, map[string]string{
		"status": "accepted",
		"job_id": jobID,
	})
}

func (h Handler) feishuWebhookTarget(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	entityID := chi.URLParam(r, "entity_id")
	sourceID := chi.URLParam(r, "source_id")
	if entityID != "" && sourceID != "" {
		return entityID, sourceID, true
	}
	resolver, ok := h.syncService.(DefaultFeishuWebhookTargetResolver)
	if !ok {
		writeError(w, http.StatusBadRequest, "default_feishu_webhook_unavailable", "default feishu webhook target is not configured")
		return "", "", false
	}
	entityID, sourceID, err := resolver.ResolveDefaultFeishuWebhookTarget(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "default_feishu_webhook_unavailable", err.Error())
		return "", "", false
	}
	return entityID, sourceID, true
}

func entityIDForRequest(w http.ResponseWriter, r *http.Request) (string, bool) {
	if entityID := r.Header.Get("X-IDB-Entity-ID"); entityID != "" {
		return entityID, true
	}
	session, ok := readSession(w, r)
	if !ok {
		return "", false
	}
	return session.EntityID, true
}

func (h Handler) dashboardSummary(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	summary, err := h.consoleService.DashboardSummary(r.Context(), session)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "dashboard_summary_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h Handler) currentUser(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	user, err := h.consoleService.CurrentUser(r.Context(), session)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "current_user_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h Handler) updatePassword(w http.ResponseWriter, r *http.Request) {
	session, ok := readSession(w, r)
	if !ok {
		return
	}
	var request struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_password_request", "invalid json body")
		return
	}
	weakPassword := isWeakPassword(request.NewPassword)
	if weakPassword {
		writeError(w, http.StatusBadRequest, "weak_password", "new password does not meet minimum strength requirements")
		return
	}
	err := h.consoleService.UpdatePassword(r.Context(), UpdatePasswordInput{
		EntityID:        session.EntityID,
		UserID:          session.UserID,
		Username:        session.Username,
		CurrentPassword: request.CurrentPassword,
		NewPassword:     request.NewPassword,
		WeakPassword:    weakPassword,
	})
	if err != nil {
		writeError(w, http.StatusUnauthorized, "password_update_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func readSession(w http.ResponseWriter, r *http.Request) (auth.Session, bool) {
	cookie, err := r.Cookie("idb_session")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "session_required", "idb_session cookie is required")
		return auth.Session{}, false
	}
	session, err := auth.ResolveSession(r.Context(), cookie.Value)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_session", "idb_session cookie is invalid")
		return auth.Session{}, false
	}
	return session, true
}

func isWeakPassword(password string) bool {
	if len(password) < 12 {
		return true
	}
	hasLower := strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz")
	hasUpper := strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	hasDigit := strings.ContainsAny(password, "0123456789")
	hasSymbol := strings.ContainsAny(password, "!@#$%^&*()-_=+[]{};:,.<>/?")
	return !(hasLower && hasUpper && hasDigit && hasSymbol)
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, map[string]string{
		"error":             code,
		"error_description": message,
	})
}

func parseFeishuWebhookEvent(payload []byte) (idp.DirectorySyncEvent, error) {
	var envelope struct {
		Type         string          `json:"type"`
		EventType    string          `json:"event_type"`
		EventID      string          `json:"event_id"`
		ObjectType   string          `json:"object_type"`
		ObjectID     string          `json:"object_id"`
		ObjectIDType string          `json:"object_id_type"`
		EventPayload json.RawMessage `json:"event"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return idp.DirectorySyncEvent{}, err
	}

	var nested struct {
		EventType    string `json:"event_type"`
		Type         string `json:"type"`
		ObjectType   string `json:"object_type"`
		ObjectID     string `json:"object_id"`
		ObjectIDType string `json:"object_id_type"`
		UserID       string `json:"user_id"`
		OpenID       string `json:"open_id"`
		UnionID      string `json:"union_id"`
		DepartmentID string `json:"department_id"`
		OpenDeptID   string `json:"open_department_id"`
		EventID      string `json:"event_id"`
	}

	if len(envelope.EventPayload) > 0 {
		if err := json.Unmarshal(envelope.EventPayload, &nested); err != nil {
			return idp.DirectorySyncEvent{}, err
		}
	}

	eventType := firstNonEmptyString(envelope.EventType, nested.EventType, nested.Type)
	eventPayload := map[string]interface{}{}
	if len(envelope.EventPayload) > 0 {
		_ = json.Unmarshal(envelope.EventPayload, &eventPayload)
	}
	eventObjectType := strings.ToLower(strings.TrimSpace(firstNonEmptyString(
		envelope.ObjectType,
		nested.ObjectType,
		stringValue(eventPayload, "object_type"),
	)))
	objectIDType := firstNonEmptyString(
		nested.ObjectIDType,
		envelope.ObjectIDType,
	)
	objectID := firstNonEmptyString(
		envelope.ObjectID,
		nested.ObjectID,
		nested.UserID,
		nested.OpenID,
		nested.UnionID,
		nested.DepartmentID,
		nested.OpenDeptID,
		stringValue(eventPayload, "user_id"),
		stringValue(eventPayload, "open_id"),
		stringValue(eventPayload, "union_id"),
		stringValue(eventPayload, "department_id"),
		stringValue(eventPayload, "open_department_id"),
	)

	if eventObjectType == "" {
		if nested.UserID != "" || nested.OpenID != "" || nested.UnionID != "" {
			eventObjectType = "user"
		}
		if nested.DepartmentID != "" || nested.OpenDeptID != "" {
			eventObjectType = "department"
		}
	}
	if eventObjectType == "" {
		if stringValue(eventPayload, "user_id") != "" ||
			stringValue(eventPayload, "open_id") != "" ||
			stringValue(eventPayload, "union_id") != "" {
			eventObjectType = "user"
		}
		if stringValue(eventPayload, "department_id") != "" ||
			stringValue(eventPayload, "open_department_id") != "" ||
			stringValue(eventPayload, "department_type") != "" {
			eventObjectType = "department"
		}
	}

	if eventObjectType == "" {
		return idp.DirectorySyncEvent{}, jsonError("unsupported webhook event")
	}
	if objectID == "" {
		objectFromPayload := firstNonEmptyString(
			extractObjectID(eventPayload, eventObjectType),
			stringValue(eventPayload, eventObjectType+"_id"),
		)
		objectID = objectFromPayload
	}
	if objectID == "" {
		return idp.DirectorySyncEvent{}, jsonError("webhook event missing object id")
	}

	if objectIDType == "" && eventObjectType == "user" && nested.UserID != "" {
		objectIDType = "user_id"
	}
	if objectIDType == "" && eventObjectType == "department" && nested.DepartmentID != "" {
		objectIDType = "department_id"
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return idp.DirectorySyncEvent{}, err
	}

	return idp.DirectorySyncEvent{
		EventType:    strings.ToLower(strings.TrimSpace(eventType)),
		ObjectType:   eventObjectType,
		ObjectID:     objectID,
		ObjectIDType: objectIDType,
		EventID:      firstNonEmptyString(envelope.EventID, nested.EventID),
		Raw:          raw,
	}, nil
}

func stringValue(values map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if raw, ok := values[key]; ok {
			if value, ok := raw.(string); ok {
				value = strings.TrimSpace(value)
				if value != "" {
					return value
				}
			}
		}
	}
	return ""
}

func extractObjectID(payload map[string]interface{}, objectType string) string {
	objectNode, ok := payload["object"].(map[string]interface{})
	if !ok {
		return ""
	}
	switch objectType {
	case "user":
		return firstNonEmptyString(
			stringValue(objectNode, "user_id"),
			stringValue(objectNode, "open_id"),
			stringValue(objectNode, "union_id"),
			stringValue(objectNode, "id"),
		)
	case "department":
		return firstNonEmptyString(
			stringValue(objectNode, "department_id"),
			stringValue(objectNode, "open_department_id"),
			stringValue(objectNode, "id"),
		)
	default:
		return ""
	}
}

func jsonError(message string) error {
	return &adminJSONError{msg: message}
}

type adminJSONError struct {
	msg string
}

func (e *adminJSONError) Error() string {
	return e.msg
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
