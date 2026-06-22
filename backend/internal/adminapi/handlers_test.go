// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/smices/open-idb/internal/auth"
	"github.com/smices/open-idb/internal/idp"
)

func TestTriggerFullSyncRequiresEntity(t *testing.T) {
	router := newTestRouter(&fakeSyncService{})
	req := httptest.NewRequest(http.MethodPost, "/admin/v1/identity-sources/source-1/sync/full", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestTriggerFullSyncReturnsResult(t *testing.T) {
	router := newTestRouter(&fakeSyncService{result: idp.FullSyncResult{
		JobID:               "job-1",
		DepartmentsUpserted: 1,
		UsersUpserted:       2,
		ManagedUsersCreated: 2,
		BindingsCreated:     2,
	}})
	req := httptest.NewRequest(http.MethodPost, "/admin/v1/identity-sources/source-1/sync/full", nil)
	req.Header.Set("X-IDB-Entity-ID", "entity-1")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var result idp.FullSyncResult
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.JobID != "job-1" || result.UsersUpserted != 2 {
		t.Fatalf("result = %#v", result)
	}
}

func TestTriggerIncrementalSyncReturnsResult(t *testing.T) {
	router := newTestRouter(&fakeSyncService{result: idp.FullSyncResult{
		JobID:               "job-3",
		DepartmentsUpserted: 1,
		UsersUpserted:       2,
		ManagedUsersCreated: 2,
		BindingsCreated:     2,
	}})
	req := httptest.NewRequest(http.MethodPost, "/admin/v1/identity-sources/source-1/sync/incremental", nil)
	req.Header.Set("X-IDB-Entity-ID", "entity-1")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var result idp.FullSyncResult
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.JobID != "job-3" || result.UsersUpserted != 2 {
		t.Fatalf("result = %#v", result)
	}
}

func TestAPIFullSyncCanUseSessionEntity(t *testing.T) {
	router := newTestRouter(&fakeSyncService{result: idp.FullSyncResult{
		JobID:         "job-2",
		UsersUpserted: 3,
	}})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/identity-sources/source-1/sync/full", nil)
	req.AddCookie(testSessionCookie())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var result idp.FullSyncResult
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.JobID != "job-2" || result.UsersUpserted != 3 {
		t.Fatalf("result = %#v", result)
	}
}

func TestTriggerFullSyncReturnsServiceError(t *testing.T) {
	router := newTestRouter(&fakeSyncService{err: errors.New("provider unavailable")})
	req := httptest.NewRequest(http.MethodPost, "/admin/v1/identity-sources/source-1/sync/full", nil)
	req.Header.Set("X-IDB-Entity-ID", "entity-1")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestHandleFeishuWebhookReturnsChallenge(t *testing.T) {
	router := newTestRouter(&fakeSyncService{})
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/feishu/entity-1/source-1", strings.NewReader(`{"type":"url_verification","challenge":"challenge-token"}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var response map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["challenge"] != "challenge-token" {
		t.Fatalf("response = %#v", response)
	}
}

func TestHandleDefaultFeishuWebhookReturnsChallenge(t *testing.T) {
	router := newTestRouter(&fakeSyncService{})
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/feishu", strings.NewReader(`{"type":"url_verification","challenge":"challenge-token"}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var response map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["challenge"] != "challenge-token" {
		t.Fatalf("response = %#v", response)
	}
}

func TestHandleFeishuWebhookAcceptsEventAndSchedulesIncrementalSync(t *testing.T) {
	service := &fakeSyncService{submitWebhookJobID: "webhook-job-1"}
	router := newTestRouter(service)
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/feishu/entity-1/source-1", strings.NewReader(`{
		"type":"event_callback",
		"event":{
			"event_type":"added_user",
			"object_type":"user",
			"object_id":"ou_123",
			"event_id":"evt-123"
		}
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}
	var response map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["job_id"] != "webhook-job-1" {
		t.Fatalf("response = %#v", response)
	}
	if service.submitWebhookCalls != 1 {
		t.Fatalf("submit_webhook_calls = %d, want 1", service.submitWebhookCalls)
	}
	if service.lastSubmittedEvent.EventType != "added_user" {
		t.Fatalf("event = %#v", service.lastSubmittedEvent)
	}
}

func TestHandleDefaultFeishuWebhookResolvesTarget(t *testing.T) {
	service := &fakeSyncService{
		defaultEntityID:    "entity-default",
		defaultSourceID:    "source-default",
		submitWebhookJobID: "webhook-job-1",
	}
	router := newTestRouter(service)
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/feishu", strings.NewReader(`{
		"type":"event_callback",
		"event":{
			"event_type":"added_user",
			"object_type":"user",
			"object_id":"ou_123",
			"event_id":"evt-123"
		}
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	if service.submittedEntityID != "entity-default" || service.submittedSourceID != "source-default" {
		t.Fatalf("submitted target = %s/%s", service.submittedEntityID, service.submittedSourceID)
	}
}

func TestHandleFeishuWebhookRejectsInvalidEvent(t *testing.T) {
	service := &fakeSyncService{submitWebhookJobID: "webhook-job-1"}
	router := newTestRouter(service)
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/feishu/entity-1/source-1", strings.NewReader(`{
		"type":"event_callback",
		"event":{
			"event_type":"updated_role",
			"event_id":"evt-123"
		}
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if service.submitWebhookCalls != 0 {
		t.Fatalf("submit_webhook_calls = %d, want 0", service.submitWebhookCalls)
	}
}

func TestDashboardSummaryRequiresSession(t *testing.T) {
	router := newConsoleTestRouter(&fakeConsoleService{})
	req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/dashboard/summary", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestDashboardSummaryReturnsConsoleMetrics(t *testing.T) {
	router := newConsoleTestRouter(&fakeConsoleService{summary: DashboardSummary{
		Users:                10,
		ActiveUsers:          8,
		NewUsers:             2,
		ApplicationActivity:  5,
		PendingAuthorization: 1,
		SyncHealth:           "ready",
	}})
	req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/dashboard/summary", nil)
	req.AddCookie(testSessionCookie())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var response DashboardSummary
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Users != 10 || response.PendingAuthorization != 1 || response.SyncHealth != "ready" {
		t.Fatalf("response = %#v", response)
	}
}

func TestCurrentUserReturnsSessionBackedProfile(t *testing.T) {
	router := newConsoleTestRouter(&fakeConsoleService{user: CurrentUser{
		ID:                 "user-1",
		EntityID:           "entity-1",
		Username:           "admin",
		DisplayName:        "Administrator",
		Locale:             "en-US",
		MustChangePassword: true,
		WeakPassword:       true,
	}})
	req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/me", nil)
	req.AddCookie(testSessionCookie())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var response CurrentUser
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Username != "admin" || !response.MustChangePassword || !response.WeakPassword {
		t.Fatalf("response = %#v", response)
	}
}

func TestUpdatePasswordRejectsWeakPassword(t *testing.T) {
	router := newConsoleTestRouter(&fakeConsoleService{})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/me/password", strings.NewReader(`{"current_password":"admin123","new_password":"123456"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(testSessionCookie())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUpdatePasswordCallsConsoleService(t *testing.T) {
	console := &fakeConsoleService{}
	router := newConsoleTestRouter(console)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/me/password", strings.NewReader(`{"current_password":"admin123","new_password":"StrongerPass123!"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(testSessionCookie())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if console.passwordInput.UserID != "user-1" || console.passwordInput.NewPassword != "StrongerPass123!" {
		t.Fatalf("password input = %#v", console.passwordInput)
	}
}

func newTestRouter(service SyncService) http.Handler {
	r := chi.NewRouter()
	NewHandler(service, nil).RegisterRoutes(r)
	return r
}

func newConsoleTestRouter(service ConsoleService) http.Handler {
	r := chi.NewRouter()
	NewHandler(nil, service).RegisterRoutes(r)
	return r
}

type fakeSyncService struct {
	result              idp.FullSyncResult
	err                 error
	submitWebhookJobID  string
	submitWebhookErr    error
	defaultEntityID     string
	defaultSourceID     string
	defaultTargetErr    error
	submitWebhookCalls  int
	runIncrementalCalls int
	submittedEntityID   string
	submittedSourceID   string
	lastSubmittedEvent  idp.DirectorySyncEvent
}

func (f *fakeSyncService) RunFullSync(context.Context, idp.FullSyncInput) (idp.FullSyncResult, error) {
	return f.result, f.err
}

func (f *fakeSyncService) RunIncrementalSync(context.Context, idp.FullSyncInput) (idp.FullSyncResult, error) {
	f.runIncrementalCalls++
	return f.result, f.err
}

func (f *fakeSyncService) SubmitWebhookEvent(_ context.Context, entityID, sourceID string, event idp.DirectorySyncEvent) (string, error) {
	f.submitWebhookCalls++
	f.submittedEntityID = entityID
	f.submittedSourceID = sourceID
	f.lastSubmittedEvent = event
	return f.submitWebhookJobID, f.submitWebhookErr
}

func (f *fakeSyncService) ResolveDefaultFeishuWebhookTarget(context.Context) (string, string, error) {
	return f.defaultEntityID, f.defaultSourceID, f.defaultTargetErr
}

type fakeConsoleService struct {
	summary       DashboardSummary
	user          CurrentUser
	err           error
	passwordInput UpdatePasswordInput
}

func (f *fakeConsoleService) DashboardSummary(context.Context, auth.Session) (DashboardSummary, error) {
	return f.summary, f.err
}

func (f *fakeConsoleService) CurrentUser(context.Context, auth.Session) (CurrentUser, error) {
	return f.user, f.err
}

func (f *fakeConsoleService) UpdatePassword(_ context.Context, input UpdatePasswordInput) error {
	f.passwordInput = input
	return f.err
}

func testSessionCookie() *http.Cookie {
	payload := `{"UserID":"user-1","EntityID":"entity-1","Username":"admin","DisplayName":"Administrator","MustChangePassword":true,"WeakPassword":true}`
	return &http.Cookie{
		Name:  "idb_session",
		Value: base64.RawURLEncoding.EncodeToString([]byte(payload)),
	}
}
