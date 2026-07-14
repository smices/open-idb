// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	auditmodel "github.com/smices/open-idb/internal/audit/model"
	"github.com/smices/open-idb/internal/ephemeral"
	"github.com/smices/open-idb/internal/id"
)

type AdminLoginResult struct {
	AdminID     string `json:"id"`
	EntityID    string `json:"entity_id,omitempty"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

type AdminCurrentUser struct {
	ID          string `json:"id"`
	EntityID    string `json:"entity_id,omitempty"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

type AdminUserSummary struct {
	ID          string `json:"id"`
	EntityID    string `json:"entity_id,omitempty"`
	EntityName  string `json:"entity_name,omitempty"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email,omitempty"`
	Status      string `json:"status"`
	Role        string `json:"role"`
	Protected   bool   `json:"protected"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type AdminRoleOption struct {
	Value          string `json:"value"`
	Label          string `json:"label"`
	Description    string `json:"description"`
	RequiresEntity bool   `json:"requires_entity"`
}

type AdminUserListResponse struct {
	Items []AdminUserSummary `json:"items"`
	Total int                `json:"total"`
}

type AdminUserCreateRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	EntityID    string `json:"entity_id"`
	Password    string `json:"password"`
}

type AdminUserUpdateRequest struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	EntityID    string `json:"entity_id"`
	Status      string `json:"status"`
}

type AdminAuthService interface {
	AuthenticateAdmin(ctx context.Context, username string, password string) (AdminLoginResult, error)
	CreateAdminSession(ctx context.Context, result AdminLoginResult, meta SessionMetadata) (AdminSession, error)
	CurrentAdmin(ctx context.Context, session AdminSession) (AdminCurrentUser, error)
	UpdateAdminProfile(ctx context.Context, session AdminSession, displayName string) (AdminCurrentUser, error)
	UpdateAdminPassword(ctx context.Context, session AdminSession, currentPassword string, newPassword string) error
	ListManagedAdminUsers(ctx context.Context, session AdminSession) (AdminUserListResponse, error)
	ListAssignableAdminRoles(ctx context.Context, session AdminSession) ([]AdminRoleOption, error)
	CreateManagedAdminUser(ctx context.Context, session AdminSession, request AdminUserCreateRequest) (AdminUserSummary, error)
	UpdateManagedAdminUser(ctx context.Context, session AdminSession, id string, request AdminUserUpdateRequest) (AdminUserSummary, error)
	DeleteManagedAdminUser(ctx context.Context, session AdminSession, id string) (AdminUserSummary, error)
	SetManagedAdminPassword(ctx context.Context, session AdminSession, id string, password string) error
}

type adminSessionRevoker interface {
	RevokeAdminSession(ctx context.Context, sessionID string) error
}

type AdminHandler struct {
	service    AdminAuthService
	audit      AuditEventWriter
	sessionTTL time.Duration
	ephemeral  ephemeral.Store
}

func NewAdminHandler(service AdminAuthService, writers ...AuditEventWriter) AdminHandler {
	h := AdminHandler{service: service, sessionTTL: 24 * time.Hour}
	if len(writers) > 0 {
		h.audit = writers[0]
	}
	return h
}

func (h *AdminHandler) SetSessionTTL(ttl time.Duration) {
	if ttl > 0 {
		h.sessionTTL = ttl
	}
}

func (h *AdminHandler) SetEphemeralStore(store ephemeral.Store) {
	h.ephemeral = store
}

func (h AdminHandler) RegisterRoutes(r chi.Router) {
	r.Post("/sapi/login/account", h.loginAccount)
	r.Post("/sapi/logout", h.logout)
	r.Get("/sapi/me", h.currentAdmin)
	r.Patch("/sapi/me", h.updateAdminProfile)
	r.Post("/sapi/me/password", h.updateAdminPassword)
	r.Get("/sapi/admin-users", h.listAdminUsers)
	r.Post("/sapi/admin-users", h.createAdminUser)
	r.Get("/sapi/admin-users/roles", h.listAdminRoles)
	r.Put("/sapi/admin-users/{id}", h.updateAdminUser)
	r.Delete("/sapi/admin-users/{id}", h.deleteAdminUser)
	r.Post("/sapi/admin-users/{id}/password", h.setAdminUserPassword)
}

func (h AdminHandler) loginAccount(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeError(w, http.StatusServiceUnavailable, "admin_auth_unavailable", "admin authentication is unavailable")
		return
	}
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_login_request", "invalid form body")
		return
	}
	account := r.PostForm.Get("account")
	password := r.PostForm.Get("password")
	limitKey := ephemeral.Key("rate:admin_login", account, r.RemoteAddr)
	limit, limitErr := ephemeral.CheckLimit(r.Context(), h.ephemeral, limitKey, 10, 15*time.Minute)
	if limitErr != nil {
		writeError(w, http.StatusServiceUnavailable, "rate_limit_unavailable", "login rate limit is unavailable")
		return
	}
	if !limit.Allowed {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "too many login attempts")
		return
	}
	result, err := h.service.AuthenticateAdmin(r.Context(), account, password)
	if err != nil {
		if acceptsHTML(r) {
			http.Redirect(w, r, "/admin/login?login_error=invalid_credentials", http.StatusSeeOther)
			return
		}
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid account or password")
		return
	}
	session, err := h.service.CreateAdminSession(r.Context(), result, SessionMetadata{
		LoginMethod: "password",
		IP:          r.RemoteAddr,
		UserAgent:   r.UserAgent(),
		TTL:         h.sessionTTL,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "session_create_failed", "could not create admin session")
		return
	}
	setAdminSessionCookie(w, r, session.ID, session.ExpiresAt)
	http.Redirect(w, r, safeReturnToWithDefault(r.PostForm.Get("return_to"), "/admin"), http.StatusFound)
}

func (h AdminHandler) logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(adminSessionCookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		clearAdminSessionCookie(w, r)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	revoker, ok := h.service.(adminSessionRevoker)
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "logout_unavailable", "admin session revocation is unavailable")
		return
	}
	if err := revoker.RevokeAdminSession(r.Context(), cookie.Value); err != nil {
		writeError(w, http.StatusInternalServerError, "logout_failed", "admin session could not be revoked")
		return
	}
	clearAdminSessionCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (h AdminHandler) currentAdmin(w http.ResponseWriter, r *http.Request) {
	session, ok := h.readAdminSession(w, r)
	if !ok {
		return
	}
	user, err := h.service.CurrentAdmin(r.Context(), session)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_admin_session", "idb_admin_session cookie is invalid")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(user)
}

func (h AdminHandler) updateAdminProfile(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeError(w, http.StatusServiceUnavailable, "admin_auth_unavailable", "admin authentication is unavailable")
		return
	}
	session, ok := h.readAdminSession(w, r)
	if !ok {
		return
	}
	var request struct {
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_profile_request", "invalid json body")
		return
	}
	user, err := h.service.UpdateAdminProfile(r.Context(), session, request.DisplayName)
	if err != nil {
		writeError(w, http.StatusBadRequest, "profile_update_failed", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(user)
}

func (h AdminHandler) updateAdminPassword(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeError(w, http.StatusServiceUnavailable, "admin_auth_unavailable", "admin authentication is unavailable")
		return
	}
	session, ok := h.readAdminSession(w, r)
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
	if isWeakAdminPassword(request.NewPassword) {
		writeError(w, http.StatusBadRequest, "weak_password", "new password does not meet minimum strength requirements")
		return
	}
	if err := h.service.UpdateAdminPassword(r.Context(), session, request.CurrentPassword, request.NewPassword); err != nil {
		writeError(w, http.StatusUnauthorized, "password_update_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h AdminHandler) listAdminUsers(w http.ResponseWriter, r *http.Request) {
	session, ok := h.readAdminSession(w, r)
	if !ok {
		return
	}
	response, err := h.service.ListManagedAdminUsers(r.Context(), session)
	if err != nil {
		h.writeAdminManagementError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (h AdminHandler) listAdminRoles(w http.ResponseWriter, r *http.Request) {
	session, ok := h.readAdminSession(w, r)
	if !ok {
		return
	}
	roles, err := h.service.ListAssignableAdminRoles(r.Context(), session)
	if err != nil {
		h.writeAdminManagementError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(roles)
}

func (h AdminHandler) createAdminUser(w http.ResponseWriter, r *http.Request) {
	session, ok := h.readAdminSession(w, r)
	if !ok {
		return
	}
	var request AdminUserCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_admin_user_request", "invalid json body")
		return
	}
	if isWeakAdminPassword(request.Password) {
		writeError(w, http.StatusBadRequest, "weak_password", "password does not meet minimum strength requirements")
		return
	}
	user, err := h.service.CreateManagedAdminUser(r.Context(), session, request)
	if err != nil {
		h.writeAdminManagementError(w, err)
		return
	}
	if err := h.writeManagedAdminAudit(r, session, auditmodel.Event{
		EntityID:     user.EntityID,
		Action:       "admin.created",
		ResourceType: "admin_user",
		ResourceID:   user.ID,
		After:        user,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "audit_write_failed", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(user)
}

func (h AdminHandler) updateAdminUser(w http.ResponseWriter, r *http.Request) {
	session, ok := h.readAdminSession(w, r)
	if !ok {
		return
	}
	var request AdminUserUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_admin_user_request", "invalid json body")
		return
	}
	user, err := h.service.UpdateManagedAdminUser(r.Context(), session, chi.URLParam(r, "id"), request)
	if err != nil {
		h.writeAdminManagementError(w, err)
		return
	}
	if err := h.writeManagedAdminAudit(r, session, auditmodel.Event{
		EntityID:     user.EntityID,
		Action:       "admin.updated",
		ResourceType: "admin_user",
		ResourceID:   user.ID,
		After:        user,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "audit_write_failed", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(user)
}

func (h AdminHandler) deleteAdminUser(w http.ResponseWriter, r *http.Request) {
	session, ok := h.readAdminSession(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	deleted, err := h.service.DeleteManagedAdminUser(r.Context(), session, id)
	if err != nil {
		h.writeAdminManagementError(w, err)
		return
	}
	if err := h.writeManagedAdminAudit(r, session, auditmodel.Event{
		EntityID:     session.EntityID,
		Action:       "admin.deleted",
		ResourceType: "admin_user",
		ResourceID:   id,
		Before:       deleted,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "audit_write_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h AdminHandler) setAdminUserPassword(w http.ResponseWriter, r *http.Request) {
	session, ok := h.readAdminSession(w, r)
	if !ok {
		return
	}
	var request struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_password_request", "invalid json body")
		return
	}
	if isWeakAdminPassword(request.Password) {
		writeError(w, http.StatusBadRequest, "weak_password", "password does not meet minimum strength requirements")
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.service.SetManagedAdminPassword(r.Context(), session, id, request.Password); err != nil {
		h.writeAdminManagementError(w, err)
		return
	}
	if err := h.writeManagedAdminAudit(r, session, auditmodel.Event{
		EntityID:     session.EntityID,
		Action:       "admin.password_updated",
		ResourceType: "admin_user",
		ResourceID:   id,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "audit_write_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h AdminHandler) writeManagedAdminAudit(r *http.Request, session AdminSession, event auditmodel.Event) error {
	if h.audit == nil {
		return nil
	}
	if event.EntityID == "" {
		event.EntityID = session.EntityID
	}
	if event.EntityID == "" {
		return nil
	}
	event.ActorUserID = session.AdminID
	event.ActorType = "admin"
	event.IP = r.RemoteAddr
	event.UserAgent = r.UserAgent()
	if event.TraceID == "" {
		event.TraceID = id.NewULID()
	}
	return h.audit.Write(r.Context(), event)
}

func (h AdminHandler) writeAdminManagementError(w http.ResponseWriter, err error) {
	if adminErr, ok := err.(AdminManagementError); ok {
		writeError(w, adminErr.Status, adminErr.Code, adminErr.Message)
		return
	}
	writeError(w, http.StatusBadRequest, "admin_user_operation_failed", err.Error())
}

func (h AdminHandler) readAdminSession(w http.ResponseWriter, r *http.Request) (AdminSession, bool) {
	session, ok := AdminSessionFromContext(r.Context())
	if ok {
		return session, true
	}
	cookie, err := r.Cookie(adminSessionCookieName)
	if err != nil || cookie.Value == "" {
		writeError(w, http.StatusUnauthorized, "admin_session_required", "idb_admin_session cookie is required")
		return AdminSession{}, false
	}
	resolved, err := ResolveAdminSession(r.Context(), cookie.Value)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_admin_session", "idb_admin_session cookie is invalid")
		return AdminSession{}, false
	}
	return resolved, true
}

func isWeakAdminPassword(password string) bool {
	if len(password) < 6 {
		return true
	}
	hasLetter := strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	hasDigit := strings.ContainsAny(password, "0123456789")
	return !(hasLetter && hasDigit)
}

func safeReturnToWithDefault(value string, fallback string) string {
	resolved := safeReturnTo(value)
	if value == "" && fallback != "" {
		return fallback
	}
	return resolved
}
