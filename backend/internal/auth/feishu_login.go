// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	auditmodel "github.com/smices/open-idb/internal/audit/model"
	"github.com/smices/open-idb/internal/db/generated"
	"github.com/smices/open-idb/internal/id"
)

// FeishuUserProvider abstracts Feishu OAuth user info retrieval for testability.
type FeishuUserProvider interface {
	GetUserInfoByCode(ctx context.Context, code string) (FeishuUserInfo, error)
	GetUserInfoByAppCode(ctx context.Context, authCode string) (FeishuUserInfo, error)
}

// FeishuUserInfo holds user info from Feishu OAuth.
type FeishuUserInfo struct {
	UserID     string
	UnionID    string
	OpenID     string
	Name       string
	Email      string
	Phone      string
	AvatarURL  string
	Status     string
	RawProfile []byte
}

// FeishuLoginResult contains session info after successful Feishu login.
type FeishuLoginResult struct {
	SessionValue string
	EntityID     string
	UserID       string
	Username     string
	DisplayName  string
	ExpiresIn    time.Duration
}

// loginQueries is the database interface for the Feishu login flow.
// *generated.Queries satisfies this interface.
type loginQueries interface {
	UpsertDirectoryUser(ctx context.Context, arg generated.UpsertDirectoryUserParams) (generated.DirectoryUser, error)
	GetAccountBindingByProviderUID(ctx context.Context, arg generated.GetAccountBindingByProviderUIDParams) (generated.AccountBinding, error)
	CreateManagedUser(ctx context.Context, arg generated.CreateManagedUserParams) (generated.User, error)
	CreateAccountBinding(ctx context.Context, arg generated.CreateAccountBindingParams) (generated.AccountBinding, error)
	UpdateManagedUserFromDirectory(ctx context.Context, arg generated.UpdateManagedUserFromDirectoryParams) (generated.User, error)
	GetFeishuSourceByEntity(ctx context.Context, entityID string) (generated.GetFeishuSourceByEntityRow, error)
	GetEntityBySlug(ctx context.Context, slug string) (generated.BusinessEntity, error)
}

// LoginProvisionPolicy controls whether new managed users are created
// during login when no existing account binding is found.
type LoginProvisionPolicy struct {
	// AutoCreateManagedUsers allows the login flow to create a new managed
	// user when no binding exists for the external identity. If false,
	// login will fail with "no_account" for unknown identities.
	AutoCreateManagedUsers bool
}

type FeishuClientResolver interface {
	GetFeishuUserProvider(ctx context.Context, entityID string, sourceID string) (FeishuUserProvider, error)
}

// FeishuLoginService handles the complete Feishu login flow:
// find/create directory user, find/create managed user, create binding, check lifecycle, create session.
type FeishuLoginService struct {
	queries              loginQueries
	feishuClient         FeishuUserProvider
	feishuClientResolver FeishuClientResolver
	sessionTTL           time.Duration
	policy               LoginProvisionPolicy
}

// NewFeishuLoginService creates a FeishuLoginService. A zero ttl defaults to 24 hours.
// The default policy allows auto-creating managed users (matching the spec default).
func NewFeishuLoginService(queries *generated.Queries, client FeishuUserProvider, ttl time.Duration) *FeishuLoginService {
	if ttl == 0 {
		ttl = 24 * time.Hour
	}
	staticResolver := &staticFeishuClientResolver{client}
	return &FeishuLoginService{
		queries:              queries,
		feishuClient:         client,
		feishuClientResolver: staticResolver,
		sessionTTL:           ttl,
		policy:               LoginProvisionPolicy{AutoCreateManagedUsers: true},
	}
}

// SetProvisionPolicy configures the provisioning policy for the login flow.
func (s *FeishuLoginService) SetProvisionPolicy(p LoginProvisionPolicy) {
	s.policy = p
}

// SetClientResolver configures how the service resolves entity/source specific Feishu clients.
func (s *FeishuLoginService) SetClientResolver(resolver FeishuClientResolver) {
	s.feishuClientResolver = resolver
}

type staticFeishuClientResolver struct {
	client FeishuUserProvider
}

func (r *staticFeishuClientResolver) GetFeishuUserProvider(context.Context, string, string) (FeishuUserProvider, error) {
	return r.client, nil
}

// LookupSourceID finds the Feishu identity source for a entity and returns its ID as a string.
func (s *FeishuLoginService) LookupSourceID(ctx context.Context, entityID string) (string, error) {
	entityULID, err := resolveEntityRef(ctx, s.queries, entityID)
	if err != nil {
		return "", fmt.Errorf("invalid entity_id: %w", err)
	}
	source, err := s.queries.GetFeishuSourceByEntity(ctx, entityULID)
	if err != nil {
		return "", err
	}
	return ulidString(source.ID), nil
}

// LoginViaOAuth handles the redirect-based OAuth flow (after callback).
func (s *FeishuLoginService) LoginViaOAuth(ctx context.Context, entityID string, sourceID string, code string) (FeishuLoginResult, error) {
	client, err := s.resolveClient(ctx, entityID, sourceID)
	if err != nil {
		return FeishuLoginResult{}, err
	}
	info, err := client.GetUserInfoByCode(ctx, code)
	if err != nil {
		return FeishuLoginResult{}, fmt.Errorf("feishu oauth: %w", err)
	}
	return s.completeLogin(ctx, entityID, sourceID, info)
}

// LoginViaAppCode handles the embedded app flow.
func (s *FeishuLoginService) LoginViaAppCode(ctx context.Context, entityID string, sourceID string, authCode string) (FeishuLoginResult, error) {
	client, err := s.resolveClient(ctx, entityID, sourceID)
	if err != nil {
		return FeishuLoginResult{}, err
	}
	info, err := client.GetUserInfoByAppCode(ctx, authCode)
	if err != nil {
		return FeishuLoginResult{}, fmt.Errorf("feishu app code: %w", err)
	}
	return s.completeLogin(ctx, entityID, sourceID, info)
}

func (s *FeishuLoginService) resolveClient(ctx context.Context, entityID string, sourceID string) (FeishuUserProvider, error) {
	if s.feishuClientResolver != nil {
		return s.feishuClientResolver.GetFeishuUserProvider(ctx, entityID, sourceID)
	}
	if s.feishuClient == nil {
		return nil, fmt.Errorf("feishu client is not configured")
	}
	return s.feishuClient, nil
}

// completeLogin is the shared core logic for both login paths.
func (s *FeishuLoginService) completeLogin(ctx context.Context, entityID string, sourceID string, info FeishuUserInfo) (FeishuLoginResult, error) {
	entityULID, err := parseULID(entityID)
	if err != nil {
		return FeishuLoginResult{}, fmt.Errorf("invalid entity_id: %w", err)
	}
	sourceULID, err := parseULID(sourceID)
	if err != nil {
		return FeishuLoginResult{}, fmt.Errorf("invalid source_id: %w", err)
	}

	// Step 1: Find or create directory_user by (entity_id, source_id, external_user_id).
	directoryUser, err := s.queries.UpsertDirectoryUser(ctx, generated.UpsertDirectoryUserParams{
		EntityID:        entityULID,
		SourceID:        sourceULID,
		ExternalUserID:  info.UserID,
		ExternalUnionID: pgText(info.UnionID),
		ExternalOpenID:  pgText(info.OpenID),
		Name:            info.Name,
		Email:           pgText(info.Email),
		Phone:           pgText(info.Phone),
		AvatarUrl:       pgText(info.AvatarURL),
		Status:          normalizeStatus(info.Status),
		RawProfile:      info.RawProfile,
	})
	if err != nil {
		return FeishuLoginResult{}, fmt.Errorf("upsert directory user: %w", err)
	}

	// Step 2: Check for existing account_binding by (entity_id, source_id, provider_uid).
	providerUID := info.UserID
	if providerUID == "" {
		providerUID = info.OpenID
	}

	binding, err := s.queries.GetAccountBindingByProviderUID(ctx, generated.GetAccountBindingByProviderUIDParams{
		EntityID:    entityULID,
		SourceID:    sourceULID,
		ProviderUid: providerUID,
	})
	if err == nil {
		// Step 3a: Binding exists — update managed user from directory data.
		updated, err := s.queries.UpdateManagedUserFromDirectory(ctx, generated.UpdateManagedUserFromDirectoryParams{
			EntityID:        entityULID,
			ID:              binding.UserID,
			PrimarySourceID: pgText(sourceULID),
			DisplayName:     info.Name,
			Email:           pgText(info.Email),
			Phone:           pgText(info.Phone),
			AvatarUrl:       pgText(info.AvatarURL),
			LifecycleStatus: lifecycleStatus(info.Status),
		})
		if err != nil {
			return FeishuLoginResult{}, fmt.Errorf("update managed user: %w", err)
		}
		return s.buildResult(updated)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return FeishuLoginResult{}, fmt.Errorf("lookup binding: %w", err)
	}

	// Step 3b: No binding — check policy before creating managed user.
	if !s.policy.AutoCreateManagedUsers {
		return FeishuLoginResult{}, fmt.Errorf("no_account")
	}

	username := info.Email
	if username == "" {
		username = info.UserID
	}
	managedUser, err := s.queries.CreateManagedUser(ctx, generated.CreateManagedUserParams{
		EntityID:        entityULID,
		Username:        username,
		DisplayName:     info.Name,
		Email:           pgText(info.Email),
		Phone:           pgText(info.Phone),
		AvatarUrl:       pgText(info.AvatarURL),
		LifecycleStatus: lifecycleStatus(info.Status),
		UserType:        "employee",
		PrimarySourceID: pgText(sourceULID),
		Locale:          pgtype.Text{String: "en-US", Valid: true},
	})
	if err != nil {
		return FeishuLoginResult{}, fmt.Errorf("create managed user: %w", err)
	}

	if _, err := s.queries.CreateAccountBinding(ctx, generated.CreateAccountBindingParams{
		EntityID:        entityULID,
		UserID:          managedUser.ID,
		SourceID:        sourceULID,
		DirectoryUserID: directoryUser.ID,
		ProviderUid:     providerUID,
		ProviderUnionID: pgText(info.UnionID),
		IsPrimary:       true,
	}); err != nil {
		return FeishuLoginResult{}, fmt.Errorf("create binding: %w", err)
	}

	return s.buildResult(managedUser)
}

// buildResult checks lifecycle status and creates a session.
func (s *FeishuLoginService) buildResult(user generated.User) (FeishuLoginResult, error) {
	if user.LifecycleStatus == "disabled" || user.LifecycleStatus == "locked" {
		return FeishuLoginResult{}, fmt.Errorf("user_disabled")
	}

	sessionValue, err := EncodeSession(Session{
		UserID:      ulidString(user.ID),
		EntityID:    ulidString(user.EntityID),
		Username:    user.Username,
		DisplayName: user.DisplayName,
	})
	if err != nil {
		return FeishuLoginResult{}, fmt.Errorf("encode session: %w", err)
	}

	return FeishuLoginResult{
		SessionValue: sessionValue,
		EntityID:     ulidString(user.EntityID),
		UserID:       ulidString(user.ID),
		Username:     user.Username,
		DisplayName:  user.DisplayName,
		ExpiresIn:    s.sessionTTL,
	}, nil
}

// --- HTTP Handler ---

// FeishuLoginHandler exposes Feishu login HTTP endpoints.
type FeishuLoginHandler struct {
	loginService    *FeishuLoginService
	providerService *LoginProviderService
	feishuAppID     string
	redirectURI     string
	webBaseURL      string
	audit           AuditEventWriter
}

// NewFeishuLoginHandler creates a FeishuLoginHandler. An optional
// AuditEventWriter may be provided to enable audit logging of Feishu
// login events. Existing callers that do not pass a writer are unaffected.
func NewFeishuLoginHandler(login *FeishuLoginService, providers *LoginProviderService, appID string, redirectURI string, writers ...AuditEventWriter) FeishuLoginHandler {
	h := FeishuLoginHandler{
		loginService:    login,
		providerService: providers,
		feishuAppID:     appID,
		redirectURI:     redirectURI,
	}
	if len(writers) > 0 {
		h.audit = writers[0]
	}
	return h
}

// SetWebBaseURL configures the browser redirect target after a successful
// OAuth callback. The value must be an absolute URL; invalid values are ignored.
func (h *FeishuLoginHandler) SetWebBaseURL(value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		h.webBaseURL = ""
		return
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		h.webBaseURL = ""
		return
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	h.webBaseURL = strings.TrimRight(parsed.String(), "/")
}

// RegisterRoutes adds Feishu login routes to the router.
func (h FeishuLoginHandler) RegisterRoutes(r chi.Router) {
	r.Get("/auth/feishu/login", h.loginRedirect)
	r.Get("/auth/feishu/callback", h.loginCallback)
	r.Post("/auth/feishu/exchange", h.loginExchange)
	r.Get("/api/auth/feishu/login", h.loginRedirect)
	r.Get("/api/auth/feishu/callback", h.loginCallback)
	r.Post("/api/auth/feishu/exchange", h.loginExchange)
	r.Get("/api/admin/v1/auth/providers", h.listProviders)
}

// oauthState is encoded into the OAuth state parameter for the redirect flow.
type oauthState struct {
	EntityID string `json:"entity_id"`
	SourceID string `json:"source_id"`
	ReturnTo string `json:"return_to"`
}

func (h FeishuLoginHandler) loginRedirect(w http.ResponseWriter, r *http.Request) {
	entityID := entityFromRequest(r)
	if entityID == "" {
		writeError(w, http.StatusBadRequest, "missing_entity", "entity_id is required")
		return
	}

	resolvedEntityID, err := resolveEntityRef(r.Context(), h.providerService.queries, entityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_entity", err.Error())
		return
	}

	feishuConfig, err := h.providerService.ResolveFeishuConfig(r.Context(), resolvedEntityID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_feishu_config", err.Error())
		return
	}
	if strings.TrimSpace(feishuConfig.AppID) == "" {
		writeError(w, http.StatusBadRequest, "no_feishu_app_id", "feishu app_id is required")
		return
	}

	sourceID, err := h.loginService.LookupSourceID(r.Context(), resolvedEntityID)
	if err != nil {
		writeError(w, http.StatusNotFound, "no_feishu_source", "no feishu integration configured")
		return
	}

	state := oauthState{
		EntityID: resolvedEntityID,
		SourceID: sourceID,
		ReturnTo: r.URL.Query().Get("return_to"),
	}
	stateBytes, _ := json.Marshal(state)
	stateEncoded := base64.RawURLEncoding.EncodeToString(stateBytes)

	params := url.Values{}
	params.Set("app_id", feishuConfig.AppID)
	params.Set("redirect_uri", h.redirectURI)
	params.Set("state", stateEncoded)
	params.Set("response_type", "code")
	http.Redirect(w, r, "https://open.feishu.cn/open-apis/authen/v1/authorize?"+params.Encode(), http.StatusFound)
}

func (h FeishuLoginHandler) loginCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	stateParam := r.URL.Query().Get("state")
	if code == "" || stateParam == "" {
		h.writeAudit(r, auditmodel.Event{
			Action:    auditmodel.ActionLoginFailed,
			ActorType: "user",
			IP:        r.RemoteAddr,
			UserAgent: r.UserAgent(),
			TraceID:   id.NewULID(),
			After:     map[string]string{"login_method": "feishu", "reason": "missing_params"},
		})
		if acceptsHTML(r) {
			http.Redirect(w, r, "/?login_error=missing_params", http.StatusSeeOther)
			return
		}
		writeError(w, http.StatusBadRequest, "missing_params", "code and state are required")
		return
	}

	stateBytes, err := base64.RawURLEncoding.DecodeString(stateParam)
	if err != nil {
		h.writeAudit(r, auditmodel.Event{
			Action:    auditmodel.ActionLoginFailed,
			ActorType: "user",
			IP:        r.RemoteAddr,
			UserAgent: r.UserAgent(),
			TraceID:   id.NewULID(),
			After:     map[string]string{"login_method": "feishu", "reason": "invalid_state"},
		})
		if acceptsHTML(r) {
			http.Redirect(w, r, "/?login_error=invalid_state", http.StatusSeeOther)
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_state", "state parameter is malformed")
		return
	}
	var state oauthState
	if err := json.Unmarshal(stateBytes, &state); err != nil {
		h.writeAudit(r, auditmodel.Event{
			Action:    auditmodel.ActionLoginFailed,
			ActorType: "user",
			IP:        r.RemoteAddr,
			UserAgent: r.UserAgent(),
			TraceID:   id.NewULID(),
			After:     map[string]string{"login_method": "feishu", "reason": "invalid_state"},
		})
		if acceptsHTML(r) {
			http.Redirect(w, r, "/?login_error=invalid_state", http.StatusSeeOther)
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_state", "state parameter is malformed")
		return
	}

	// Resolve source_id if not present in state.
	sourceID := state.SourceID
	if sourceID == "" {
		sourceID, err = h.loginService.LookupSourceID(r.Context(), state.EntityID)
		if err != nil {
			h.writeAudit(r, auditmodel.Event{
				EntityID:  state.EntityID,
				Action:    auditmodel.ActionLoginFailed,
				ActorType: "user",
				IP:        r.RemoteAddr,
				UserAgent: r.UserAgent(),
				TraceID:   id.NewULID(),
				After:     map[string]string{"login_method": "feishu", "reason": "no_feishu_source"},
			})
			if acceptsHTML(r) {
				http.Redirect(w, r, "/?login_error=no_feishu_source", http.StatusSeeOther)
				return
			}
			writeError(w, http.StatusNotFound, "no_feishu_source", "no feishu integration configured")
			return
		}
	}

	traceID := id.NewULID()
	result, err := h.loginService.LoginViaOAuth(r.Context(), state.EntityID, sourceID, code)
	if err != nil {
		h.writeAudit(r, auditmodel.Event{
			EntityID:  state.EntityID,
			Action:    auditmodel.ActionLoginFailed,
			ActorType: "user",
			IP:        r.RemoteAddr,
			UserAgent: r.UserAgent(),
			TraceID:   traceID,
			After:     map[string]string{"login_method": "feishu", "reason": err.Error()},
		})
		if acceptsHTML(r) {
			http.Redirect(w, r, "/?login_error=feishu_login_failed", http.StatusSeeOther)
			return
		}
		if err.Error() == "user_disabled" {
			writeError(w, http.StatusForbidden, "user_disabled", "user account is disabled or locked")
			return
		}
		writeError(w, http.StatusUnauthorized, "feishu_login_failed", err.Error())
		return
	}

	h.writeAudit(r, auditmodel.Event{
		EntityID:     result.EntityID,
		ActorUserID:  result.UserID,
		ActorType:    "user",
		Action:       auditmodel.ActionLoginSuccess,
		ResourceType: "user",
		ResourceID:   result.UserID,
		IP:           r.RemoteAddr,
		UserAgent:    r.UserAgent(),
		TraceID:      traceID,
		After:        map[string]string{"login_method": "feishu", "username": result.Username},
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "idb_session",
		Value:    result.SessionValue,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(result.ExpiresIn.Seconds()),
	})
	returnTo := safeReturnTo(state.ReturnTo)
	if acceptsHTML(r) {
		http.Redirect(w, r, h.browserReturnTo(returnTo), http.StatusFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"session": result.SessionValue,
		"user_id": result.UserID,
	})
}

func (h FeishuLoginHandler) browserReturnTo(returnTo string) string {
	returnTo = safeReturnTo(returnTo)
	if h.webBaseURL == "" {
		return returnTo
	}
	return h.webBaseURL + returnTo
}

func (h FeishuLoginHandler) loginExchange(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AuthCode string `json:"auth_code"`
		EntityID string `json:"entity_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.AuthCode == "" {
		writeError(w, http.StatusBadRequest, "invalid_body", "auth_code is required")
		return
	}

	entityID := body.EntityID
	if entityID == "" {
		entityID = entityFromRequest(r)
	}
	if entityID == "" {
		writeError(w, http.StatusBadRequest, "missing_entity", "entity_id is required")
		return
	}

	sourceID, err := h.loginService.LookupSourceID(r.Context(), entityID)
	if err != nil {
		h.writeAudit(r, auditmodel.Event{
			EntityID:  entityID,
			Action:    auditmodel.ActionLoginFailed,
			ActorType: "user",
			IP:        r.RemoteAddr,
			UserAgent: r.UserAgent(),
			TraceID:   id.NewULID(),
			After:     map[string]string{"login_method": "feishu", "reason": "no_feishu_source"},
		})
		writeError(w, http.StatusNotFound, "no_feishu_source", "no feishu integration configured")
		return
	}

	traceID := id.NewULID()
	result, err := h.loginService.LoginViaAppCode(r.Context(), entityID, sourceID, body.AuthCode)
	if err != nil {
		h.writeAudit(r, auditmodel.Event{
			EntityID:  entityID,
			Action:    auditmodel.ActionLoginFailed,
			ActorType: "user",
			IP:        r.RemoteAddr,
			UserAgent: r.UserAgent(),
			TraceID:   traceID,
			After:     map[string]string{"login_method": "feishu", "reason": err.Error()},
		})
		if err.Error() == "user_disabled" {
			writeError(w, http.StatusForbidden, "user_disabled", "user account is disabled or locked")
			return
		}
		writeError(w, http.StatusUnauthorized, "exchange_failed", err.Error())
		return
	}

	h.writeAudit(r, auditmodel.Event{
		EntityID:     result.EntityID,
		ActorUserID:  result.UserID,
		ActorType:    "user",
		Action:       auditmodel.ActionLoginSuccess,
		ResourceType: "user",
		ResourceID:   result.UserID,
		IP:           r.RemoteAddr,
		UserAgent:    r.UserAgent(),
		TraceID:      traceID,
		After:        map[string]string{"login_method": "feishu", "username": result.Username},
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "idb_session",
		Value:    result.SessionValue,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(result.ExpiresIn.Seconds()),
	})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"session":      result.SessionValue,
		"entity_id":    result.EntityID,
		"user_id":      result.UserID,
		"username":     result.Username,
		"display_name": result.DisplayName,
	})
}

// writeAudit records an audit event if an audit writer is configured.
// Errors are silently ignored so they do not disrupt the login flow.
func (h FeishuLoginHandler) writeAudit(r *http.Request, event auditmodel.Event) {
	if h.audit == nil {
		return
	}
	_ = h.audit.Write(r.Context(), event)
}

func (h FeishuLoginHandler) listProviders(w http.ResponseWriter, r *http.Request) {
	entityID := entityFromRequest(r)
	if entityID == "" {
		writeError(w, http.StatusBadRequest, "missing_entity", "entity_id is required")
		return
	}

	providers, err := h.providerService.ListProviders(r.Context(), entityID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "providers_error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(providers)
}

// entityFromRequest extracts entity ID from session cookie, X-IDB-Entity-ID header, or query param.
func entityFromRequest(r *http.Request) string {
	if cookie, err := r.Cookie("idb_session"); err == nil && cookie.Value != "" {
		if session, err := DecodeSession(cookie.Value); err == nil && session.EntityID != "" {
			return session.EntityID
		}
	}
	if header := r.Header.Get("X-IDB-Entity-ID"); header != "" {
		return header
	}
	return r.URL.Query().Get("entity_id")
}

// --- Package-level helpers ---

func parseULID(value string) (string, error) {
	if err := id.ValidateULID(value); err != nil {
		return "", err
	}
	return value, nil
}

func pgText(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: value != ""}
}

func normalizeStatus(status string) string {
	switch status {
	case "active", "disabled", "deleted", "unknown":
		return status
	default:
		return "unknown"
	}
}

func lifecycleStatus(status string) string {
	switch normalizeStatus(status) {
	case "active":
		return "active"
	case "disabled", "deleted":
		return "disabled"
	default:
		return "locked"
	}
}
