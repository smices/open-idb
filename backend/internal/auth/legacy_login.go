// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	auditmodel "github.com/smices/open-idb/internal/audit/model"
	"github.com/smices/open-idb/internal/db/generated"
	"github.com/smices/open-idb/internal/id"
)

const (
	defaultLegacyFailureThreshold = 5
	defaultLegacyFailureWindow    = 10 * time.Minute
	defaultLegacySessionTTL       = 24 * time.Hour
)

const (
	legacyAuthSuccessCode = "legacy_auth_success"
	legacyAuthFailureCode = "legacy_auth_failed"
	legacyAuthLockedCode  = "legacy_auth_locked"
)

// legacyLoginQueries defines the database contract used by the legacy auth flow.
// *generated.Queries satisfies this interface.
type legacyLoginQueries interface {
	GetLegacyAppUserByUsername(ctx context.Context, arg generated.GetLegacyAppUserByUsernameParams) (generated.LegacyAppUser, error)
	VerifyLegacyAppUserCredential(ctx context.Context, arg generated.VerifyLegacyAppUserCredentialParams) (generated.LegacyAppUser, error)
	CountLegacyPasswordFailures(ctx context.Context, arg generated.CountLegacyPasswordFailuresParams) (int64, error)
	CreateLegacyPasswordEvent(ctx context.Context, arg generated.CreateLegacyPasswordEventParams) error
	TouchLegacyAppUserUsedAt(ctx context.Context, arg generated.TouchLegacyAppUserUsedAtParams) error
	GetUserByEntityAndID(ctx context.Context, arg generated.GetUserByEntityAndIDParams) (generated.User, error)
	GetApplicationByID(ctx context.Context, arg generated.GetApplicationByIDParams) (generated.Application, error)
	HasApplicationAccess(ctx context.Context, arg generated.HasApplicationAccessParams) (pgtype.Bool, error)
}

// LegacyLoginService validates legacy app username/password credentials and returns session claims.
type LegacyLoginService struct {
	queries          legacyLoginQueries
	failureThreshold int32
	failureWindow    time.Duration
	sessionTTL       time.Duration
}

// NewLegacyLoginService creates a legacy login service.
func NewLegacyLoginService(queries *generated.Queries) *LegacyLoginService {
	return &LegacyLoginService{
		queries:          queries,
		failureThreshold: defaultLegacyFailureThreshold,
		failureWindow:    defaultLegacyFailureWindow,
		sessionTTL:       defaultLegacySessionTTL,
	}
}

// SetFailurePolicy configures lockout behavior.
func (s *LegacyLoginService) SetFailurePolicy(maxFailures int32, window time.Duration) {
	if maxFailures > 0 {
		s.failureThreshold = maxFailures
	}
	if window > 0 {
		s.failureWindow = window
	}
}

func (s *LegacyLoginService) SetSessionTTL(ttl time.Duration) {
	if ttl > 0 {
		s.sessionTTL = ttl
	}
}

// LegacyLoginResult is the successful login payload.
type LegacyLoginResult struct {
	SessionValue  string
	EntityID      string
	UserID        string
	Username      string
	DisplayName   string
	ApplicationID string
}

// LegacyLoginError is a machine-readable auth error with HTTP status.
type LegacyLoginError struct {
	Code    string
	Status  int
	Message string
}

func (e LegacyLoginError) Error() string {
	return e.Message
}

// AuthenticateLegacy verifies legacy credentials and issues a browser session cookie value.
func (s *LegacyLoginService) AuthenticateLegacy(ctx context.Context, entityID string, applicationID string, username string, password string, clientIP string, userAgent string, traceID string) (LegacyLoginResult, error) {
	var empty LegacyLoginResult
	if s == nil || s.queries == nil {
		return empty, LegacyLoginError{Code: legacyAuthFailureCode, Status: http.StatusInternalServerError, Message: "auth service unavailable"}
	}

	if entityID == "" || applicationID == "" || username == "" || password == "" {
		return empty, LegacyLoginError{Code: legacyAuthFailureCode, Status: http.StatusBadRequest, Message: "entity_id, application_id, username and password are required"}
	}

	entityULID, err := parseULID(entityID)
	if err != nil {
		return empty, LegacyLoginError{Code: legacyAuthFailureCode, Status: http.StatusBadRequest, Message: "invalid entity_id"}
	}
	applicationULID, err := parseULID(applicationID)
	if err != nil {
		return empty, LegacyLoginError{Code: legacyAuthFailureCode, Status: http.StatusBadRequest, Message: "invalid application_id"}
	}

	windowStart := time.Now().Add(-s.failureWindow)
	failureCount, err := s.queries.CountLegacyPasswordFailures(ctx, generated.CountLegacyPasswordFailuresParams{
		EntityID:      entityULID,
		ApplicationID: applicationULID,
		Username:      textValue(username),
		OccurredAt:    pgTimestamp(windowStart),
	})
	if err != nil {
		return empty, LegacyLoginError{Code: legacyAuthFailureCode, Status: http.StatusInternalServerError, Message: err.Error()}
	}
	if failureCount >= int64(s.failureThreshold) {
		_ = s.recordLegacyPasswordEvent(ctx, entityULID, applicationULID, "", username, "locked", clientIP, userAgent, traceID, "too many failures")
		return empty, LegacyLoginError{Code: legacyAuthLockedCode, Status: http.StatusLocked, Message: "account temporarily locked"}
	}

	mapping, err := s.queries.GetLegacyAppUserByUsername(ctx, generated.GetLegacyAppUserByUsernameParams{
		EntityID:      entityULID,
		ApplicationID: applicationULID,
		Username:      username,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return s.failedWithEvent(ctx, entityULID, applicationULID, "", username, clientIP, userAgent, traceID, "mapping_not_found", nil, http.StatusUnauthorized, "invalid credentials")
		}
		return s.failedWithEvent(ctx, entityULID, applicationULID, "", username, clientIP, userAgent, traceID, "mapping_lookup_failed", err, http.StatusInternalServerError, "mapping query failed")
	}

	if !mapping.IsActive {
		return s.failedWithEvent(ctx, entityULID, applicationULID, mapping.UserID, username, clientIP, userAgent, traceID, "account_disabled", nil, http.StatusUnauthorized, "invalid credentials")
	}
	if mapping.AuthScheme != "local" {
		return s.failedWithEvent(ctx, entityULID, applicationULID, mapping.UserID, username, clientIP, userAgent, traceID, "unsupported_auth_scheme", nil, http.StatusNotImplemented, "unsupported authentication scheme")
	}

	verified, err := s.queries.VerifyLegacyAppUserCredential(ctx, generated.VerifyLegacyAppUserCredentialParams{
		EntityID:      entityULID,
		ApplicationID: applicationULID,
		Username:      username,
		Crypt:         password,
	})
	if err != nil {
		newFailureCount := failureCount + 1
		_ = s.recordLegacyPasswordEvent(ctx, entityULID, applicationULID, mapping.UserID, username, "failed", clientIP, userAgent, traceID, "invalid password")
		if newFailureCount >= int64(s.failureThreshold) {
			_ = s.recordLegacyPasswordEvent(ctx, entityULID, applicationULID, mapping.UserID, username, "locked", clientIP, userAgent, traceID, "too many failures")
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return empty, LegacyLoginError{Code: legacyAuthFailureCode, Status: http.StatusUnauthorized, Message: "invalid credentials"}
		}
		return empty, LegacyLoginError{Code: legacyAuthFailureCode, Status: http.StatusInternalServerError, Message: err.Error()}
	}

	app, err := s.queries.GetApplicationByID(ctx, generated.GetApplicationByIDParams{
		EntityID: entityULID,
		ID:       applicationULID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return s.failedWithEvent(ctx, entityULID, applicationULID, verified.UserID, username, clientIP, userAgent, traceID, "application_not_found", nil, http.StatusUnauthorized, "invalid credentials")
		}
		return s.failedWithEvent(ctx, entityULID, applicationULID, verified.UserID, username, clientIP, userAgent, traceID, "application_lookup_failed", err, http.StatusInternalServerError, "application lookup failed")
	}
	if app.Status != "active" {
		return s.failedWithEvent(ctx, entityULID, applicationULID, verified.UserID, username, clientIP, userAgent, traceID, "application_inactive", nil, http.StatusUnauthorized, "invalid credentials")
	}

	managedUser, err := s.queries.GetUserByEntityAndID(ctx, generated.GetUserByEntityAndIDParams{
		EntityID: entityULID,
		ID:       verified.UserID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return s.failedWithEvent(ctx, entityULID, applicationULID, verified.UserID, username, clientIP, userAgent, traceID, "user_not_found", nil, http.StatusUnauthorized, "invalid credentials")
		}
		return s.failedWithEvent(ctx, entityULID, applicationULID, verified.UserID, username, clientIP, userAgent, traceID, "managed_user_lookup_failed", err, http.StatusInternalServerError, "managed user lookup failed")
	}
	if managedUser.LifecycleStatus != "active" {
		return s.failedWithEvent(ctx, entityULID, applicationULID, verified.UserID, username, clientIP, userAgent, traceID, "managed_user_inactive", nil, http.StatusUnauthorized, "invalid credentials")
	}

	access, err := s.queries.HasApplicationAccess(ctx, generated.HasApplicationAccessParams{
		EntityID:      entityULID,
		ApplicationID: applicationULID,
		SubjectID:     verified.UserID,
	})
	if err != nil {
		return s.failedWithEvent(ctx, entityULID, applicationULID, verified.UserID, username, clientIP, userAgent, traceID, "access_check_failed", err, http.StatusInternalServerError, "access policy check failed")
	}
	if !access.Bool {
		return s.failedWithEvent(ctx, entityULID, applicationULID, verified.UserID, username, clientIP, userAgent, traceID, "access_denied", nil, http.StatusForbidden, "access denied")
	}

	_ = s.queries.TouchLegacyAppUserUsedAt(ctx, generated.TouchLegacyAppUserUsedAtParams{
		EntityID: entityULID,
		ID:       verified.ID,
	})

	session, err := createSessionValue(ctx, s.queries, Session{
		UserID:      ulidString(verified.UserID),
		EntityID:    entityID,
		Username:    verified.Username,
		DisplayName: managedUser.DisplayName,
	}, SessionMetadata{
		LoginMethod: "password",
		IP:          clientIP,
		UserAgent:   userAgent,
		TTL:         s.sessionTTL,
	})
	if err != nil {
		return empty, LegacyLoginError{Code: legacyAuthFailureCode, Status: http.StatusInternalServerError, Message: "could not create login session"}
	}

	_ = s.recordLegacyPasswordEvent(ctx, entityULID, applicationULID, verified.UserID, username, "success", clientIP, userAgent, traceID, "login success")

	return LegacyLoginResult{
		SessionValue:  session.ID,
		EntityID:      entityID,
		UserID:        ulidString(verified.UserID),
		Username:      verified.Username,
		DisplayName:   managedUser.DisplayName,
		ApplicationID: applicationID,
	}, nil
}

func (s *LegacyLoginService) failedWithEvent(
	ctx context.Context,
	entityULID string,
	applicationULID string,
	userID string,
	username string,
	clientIP string,
	userAgent string,
	traceID string,
	reason string,
	err error,
	status int,
	message string,
) (LegacyLoginResult, error) {
	event := "failed"
	code := legacyAuthFailureCode
	if reason == "access_denied" {
		event = "access_denied"
		code = "access_denied"
	}
	_ = s.recordLegacyPasswordEvent(ctx, entityULID, applicationULID, userID, username, event, clientIP, userAgent, traceID, reason)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return LegacyLoginResult{}, LegacyLoginError{Code: legacyAuthFailureCode, Status: http.StatusInternalServerError, Message: err.Error()}
	}
	return LegacyLoginResult{}, LegacyLoginError{Code: code, Status: status, Message: message}
}

func (s *LegacyLoginService) recordLegacyPasswordEvent(
	ctx context.Context,
	entityULID string,
	applicationULID string,
	userID string,
	username string,
	event string,
	clientIP string,
	userAgent string,
	traceID string,
	reason string,
) error {
	if s == nil || s.queries == nil {
		return nil
	}

	return s.queries.CreateLegacyPasswordEvent(ctx, generated.CreateLegacyPasswordEventParams{
		EntityID:      entityULID,
		ApplicationID: applicationULID,
		UserID:        textValue(userID),
		Username:      textValue(username),
		Event:         event,
		ClientIp:      textValue(clientIP),
		UserAgent:     textValue(userAgent),
		TraceID:       textValue(traceID),
		Reason:        textValue(reason),
	})
}

// LegacyLoginHandler exposes the legacy credential login endpoint.
type LegacyLoginHandler struct {
	service *LegacyLoginService
	audit   AuditEventWriter
}

// NewLegacyLoginHandler creates a legacy login handler. An optional AuditEventWriter may
// be provided to record auth success/failure events.
func NewLegacyLoginHandler(service *LegacyLoginService, writers ...AuditEventWriter) LegacyLoginHandler {
	h := LegacyLoginHandler{service: service}
	if len(writers) > 0 {
		h.audit = writers[0]
	}
	return h
}

// RegisterRoutes adds legacy login API routes.
func (h LegacyLoginHandler) RegisterRoutes(r chi.Router) {
	r.Post("/api/login/legacy", h.loginLegacy)
}

func (h LegacyLoginHandler) loginLegacy(w http.ResponseWriter, r *http.Request) {
	var body struct {
		EntityID      string `json:"entity_id"`
		ApplicationID string `json:"application_id"`
		Username      string `json:"username"`
		Password      string `json:"password"`
	}
	if h.service == nil {
		writeError(w, http.StatusInternalServerError, legacyAuthFailureCode, "legacy authentication unavailable")
		return
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "invalid json body")
		return
	}

	traceID := id.NewULID()
	result, err := h.service.AuthenticateLegacy(
		r.Context(),
		body.EntityID,
		body.ApplicationID,
		body.Username,
		body.Password,
		r.RemoteAddr,
		r.UserAgent(),
		traceID,
	)
	if err != nil {
		var legacyErr LegacyLoginError
		if !errors.As(err, &legacyErr) {
			writeError(w, http.StatusInternalServerError, legacyAuthFailureCode, "legacy authentication failed")
			return
		}
		if legacyErr.Code == "" {
			legacyErr.Code = legacyAuthFailureCode
		}

		h.writeLegacyAudit(r, body.EntityID, legacyErr, traceID, body.ApplicationID, body.Username)
		writeError(w, legacyErr.Status, legacyErr.Code, legacyErr.Message)
		return
	}

	h.writeLegacyAuditSuccess(r, result.UserID, result.EntityID, result.ApplicationID, traceID, body.Username)
	http.SetCookie(w, &http.Cookie{
		Name:     "idb_session",
		Value:    result.SessionValue,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.defaultSessionTTL().Seconds()),
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":           legacyAuthSuccessCode,
		"user_id":        result.UserID,
		"entity_id":      result.EntityID,
		"application_id": result.ApplicationID,
		"username":       result.Username,
		"display_name":   result.DisplayName,
		"session":        result.SessionValue,
	})
}

func (h LegacyLoginHandler) defaultSessionTTL() time.Duration {
	if h.service == nil || h.service.sessionTTL == 0 {
		return defaultLegacySessionTTL
	}
	return h.service.sessionTTL
}

func (h LegacyLoginHandler) writeLegacyAudit(r *http.Request, entityID string, loginErr LegacyLoginError, traceID string, appID string, username string) {
	if h.audit == nil {
		return
	}

	after := map[string]string{
		"login_method":   "legacy",
		"application_id": appID,
		"username":       username,
	}
	if loginErr.Code != "" {
		after["result"] = loginErr.Code
	}
	_ = h.audit.Write(r.Context(), auditmodel.Event{
		EntityID:  entityID,
		ActorType: "user",
		Action:    auditmodel.ActionLoginFailed,
		IP:        r.RemoteAddr,
		UserAgent: r.UserAgent(),
		TraceID:   traceID,
		After:     after,
	})
}

func (h LegacyLoginHandler) writeLegacyAuditSuccess(r *http.Request, userID string, entityID string, appID string, traceID string, username string) {
	if h.audit == nil {
		return
	}

	_ = h.audit.Write(r.Context(), auditmodel.Event{
		EntityID:     entityID,
		ActorUserID:  userID,
		ActorType:    "user",
		Action:       auditmodel.ActionLoginSuccess,
		ResourceType: "user",
		ResourceID:   userID,
		IP:           r.RemoteAddr,
		UserAgent:    r.UserAgent(),
		TraceID:      traceID,
		After: map[string]string{
			"login_method":   "legacy",
			"application_id": appID,
			"username":       username,
		},
	})
}

func pgTimestamp(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func textValue(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}
