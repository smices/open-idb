// SPDX-License-Identifier: MIT

package audit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/smices/open-idb/internal/auth"
)

// fakeAuditService implements AuditService for handler tests.
type fakeAuditService struct {
	result       ListResult
	opts         ListOptions
	err          error
	deleteEntity string
	deleteID     string
	deleteResult int64
	clearEntity  string
	clearResult  int64
}

func (f *fakeAuditService) List(_ context.Context, _ string, opts ListOptions) (ListResult, error) {
	f.opts = opts
	if f.err != nil {
		return ListResult{}, f.err
	}
	return f.result, nil
}

func (f *fakeAuditService) Delete(_ context.Context, entityID, id string) (int64, error) {
	f.deleteEntity = entityID
	f.deleteID = id
	return f.deleteResult, f.err
}

func (f *fakeAuditService) Clear(_ context.Context, entityID string) (int64, error) {
	f.clearEntity = entityID
	return f.clearResult, f.err
}

func newTestRouter(svc AuditService) http.Handler {
	r := chi.NewRouter()
	NewHandler(svc).RegisterRoutes(r)
	return r
}

func testSessionCookie() *http.Cookie {
	payload, _ := auth.EncodeAdminSession(auth.AdminSession{
		AdminID:     "admin-1",
		EntityID:    "01HZZZZZZZ0000000000000100",
		Username:    "admin",
		DisplayName: "Admin",
		Role:        "enterprise_admin",
	})
	return &http.Cookie{
		Name:  "idb_admin_session",
		Value: payload,
	}
}

func TestListAuditLogsRequiresSession(t *testing.T) {
	router := newTestRouter(&fakeAuditService{})
	req := httptest.NewRequest(http.MethodGet, "/sapi/audit-logs", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != "admin_session_required" {
		t.Errorf("error = %q, want %q", body["error"], "admin_session_required")
	}
}

func TestListAuditLogsInvalidSessionReturns401(t *testing.T) {
	router := newTestRouter(&fakeAuditService{})
	req := httptest.NewRequest(http.MethodGet, "/sapi/audit-logs", nil)
	req.AddCookie(&http.Cookie{Name: "idb_admin_session", Value: "garbage-not-base64-json"})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != "invalid_admin_session" {
		t.Errorf("error = %q, want %q", body["error"], "invalid_admin_session")
	}
}

func TestListAuditLogsReturnsProperJSON(t *testing.T) {
	fixedTime := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	svc := &fakeAuditService{
		result: ListResult{
			Total: 2,
			Items: []AuditLogEntry{
				{
					ID:           "id-1",
					EntityID:     "entity-1",
					ActorUserID:  "user-1",
					ActorType:    "user",
					Action:       ActionLoginSuccess,
					ResourceType: "session",
					ResourceID:   "sess-1",
					Before:       nil,
					After:        json.RawMessage(`{"status":"active"}`),
					IP:           "10.0.0.1",
					UserAgent:    "test-agent",
					TraceID:      "trace-1",
					CreatedAt:    fixedTime,
				},
				{
					ID:           "id-2",
					EntityID:     "entity-1",
					ActorType:    "system",
					Action:       ActionSyncStarted,
					ResourceType: "sync_job",
					ResourceID:   "job-1",
					CreatedAt:    fixedTime,
				},
			},
		},
	}
	router := newTestRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/sapi/audit-logs", nil)
	req.AddCookie(testSessionCookie())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var response struct {
		Items  []json.RawMessage `json:"items"`
		Total  int64             `json:"total"`
		Limit  int               `json:"limit"`
		Offset int               `json:"offset"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Total != 2 {
		t.Errorf("total = %d, want 2", response.Total)
	}
	if response.Limit != 50 {
		t.Errorf("limit = %d, want 50 (default)", response.Limit)
	}
	if response.Offset != 0 {
		t.Errorf("offset = %d, want 0 (default)", response.Offset)
	}
	if len(response.Items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(response.Items))
	}

	// Verify first item decodes correctly
	var first AuditLogEntry
	if err := json.Unmarshal(response.Items[0], &first); err != nil {
		t.Fatalf("unmarshal first item: %v", err)
	}
	if first.ID != "id-1" {
		t.Errorf("first.ID = %q, want %q", first.ID, "id-1")
	}
	if first.Action != ActionLoginSuccess {
		t.Errorf("first.Action = %q, want %q", first.Action, ActionLoginSuccess)
	}
}

func TestListAuditLogsParsesQueryParams(t *testing.T) {
	svc := &fakeAuditService{result: ListResult{Items: []AuditLogEntry{}, Total: 0}}
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet,
		"/sapi/audit-logs?action=auth.login.success&resource_type=session&actor_type=user&limit=25&offset=10",
		nil)
	req.AddCookie(testSessionCookie())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	opts := svc.opts
	if opts.Action != ActionLoginSuccess {
		t.Errorf("Action = %q, want %q", opts.Action, ActionLoginSuccess)
	}
	if opts.ResourceType != "session" {
		t.Errorf("ResourceType = %q, want %q", opts.ResourceType, "session")
	}
	if opts.ActorType != "user" {
		t.Errorf("ActorType = %q, want %q", opts.ActorType, "user")
	}
	if opts.Limit != 25 {
		t.Errorf("Limit = %d, want 25", opts.Limit)
	}
	if opts.Offset != 10 {
		t.Errorf("Offset = %d, want 10", opts.Offset)
	}
}

func TestListAuditLogsDefaultPagination(t *testing.T) {
	svc := &fakeAuditService{result: ListResult{Items: []AuditLogEntry{}, Total: 0}}
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/sapi/audit-logs", nil)
	req.AddCookie(testSessionCookie())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if svc.opts.Limit != 50 {
		t.Errorf("default Limit = %d, want 50", svc.opts.Limit)
	}
	if svc.opts.Offset != 0 {
		t.Errorf("default Offset = %d, want 0", svc.opts.Offset)
	}
}

func TestDeleteAuditLog_Success(t *testing.T) {
	svc := &fakeAuditService{deleteResult: 1}
	router := newTestRouter(svc)
	req := httptest.NewRequest(http.MethodDelete, "/sapi/audit-logs/01HZZZZZZZ0000000000000200", nil)
	req.AddCookie(testSessionCookie())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if svc.deleteID != "01HZZZZZZZ0000000000000200" {
		t.Fatalf("deleted id = %q", svc.deleteID)
	}
	if svc.deleteEntity != "01HZZZZZZZ0000000000000100" {
		t.Fatalf("deleted entity = %q", svc.deleteEntity)
	}
}

func TestDeleteAuditLog_NotFound(t *testing.T) {
	router := newTestRouter(&fakeAuditService{})
	req := httptest.NewRequest(http.MethodDelete, "/sapi/audit-logs/01HZZZZZZZ0000000000000200", nil)
	req.AddCookie(testSessionCookie())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestClearAuditLogs_ReturnsDeletedCount(t *testing.T) {
	svc := &fakeAuditService{clearResult: 7}
	router := newTestRouter(svc)
	req := httptest.NewRequest(http.MethodDelete, "/sapi/audit-logs", nil)
	req.AddCookie(testSessionCookie())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body struct {
		Deleted int64 `json:"deleted"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Deleted != 7 {
		t.Fatalf("deleted = %d, want 7", body.Deleted)
	}
	if svc.clearEntity != "01HZZZZZZZ0000000000000100" {
		t.Fatalf("cleared entity = %q", svc.clearEntity)
	}
}

func TestListAuditLogsInvalidPaginationFallsBackToDefaults(t *testing.T) {
	svc := &fakeAuditService{result: ListResult{Items: []AuditLogEntry{}, Total: 0}}
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/sapi/audit-logs?limit=abc&offset=-5", nil)
	req.AddCookie(testSessionCookie())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if svc.opts.Limit != 50 {
		t.Errorf("Limit = %d, want 50 (fallback)", svc.opts.Limit)
	}
	if svc.opts.Offset != 0 {
		t.Errorf("Offset = %d, want 0 (fallback for negative)", svc.opts.Offset)
	}
}

func TestListAuditLogsAPIPathAlsoWorks(t *testing.T) {
	svc := &fakeAuditService{result: ListResult{Items: []AuditLogEntry{}, Total: 5}}
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/sapi/audit-logs", nil)
	req.AddCookie(testSessionCookie())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response struct {
		Total int64 `json:"total"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Total != 5 {
		t.Errorf("total = %d, want 5", response.Total)
	}
}
