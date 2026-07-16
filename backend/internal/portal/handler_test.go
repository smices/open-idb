// SPDX-License-Identifier: MIT

package portal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/smices/open-idb/internal/auth"
	"github.com/smices/open-idb/internal/db/generated"
)

func TestServiceUsesConfiguredInternalApplicationURLForPortalEntry(t *testing.T) {
	service := service{store: fakeApplicationStore{rows: []generated.ListPortalApplicationsRow{{
		ID:       "app-expense",
		Name:     "Expense",
		Type:     "internal_app",
		EntryUrl: "https://expense.example",
	}}}}

	applications, err := service.ListApplications(context.Background(), "entity-a")
	if err != nil {
		t.Fatalf("list applications: %v", err)
	}
	if got := applications[0].EntryURL; got != "https://expense.example" {
		t.Fatalf("entry URL = %q, want configured internal_app app_url", got)
	}
}

func TestServiceOmitsNonHTTPApplicationEntryURLs(t *testing.T) {
	service := service{store: fakeApplicationStore{rows: []generated.ListPortalApplicationsRow{
		{ID: "app-javascript", Name: "Unsafe", Type: "internal_app", EntryUrl: "javascript:alert(1)"},
		{ID: "app-relative", Name: "Relative", Type: "internal_app", EntryUrl: "/internal"},
		{ID: "app-http", Name: "Safe", Type: "internal_app", EntryUrl: "https://safe.example/path"},
	}}}

	applications, err := service.ListApplications(context.Background(), "entity-a")
	if err != nil {
		t.Fatalf("list applications: %v", err)
	}
	for _, application := range applications[:2] {
		if application.EntryURL != "" {
			t.Errorf("unsafe application %q entry URL = %q, want empty", application.ID, application.EntryURL)
		}
	}
	if got := applications[2].EntryURL; got != "https://safe.example/path" {
		t.Errorf("safe application entry URL = %q", got)
	}
}

func TestListApplicationsScopesToUserSessionEntityAndReturnsSafeProjection(t *testing.T) {
	service := &fakeApplicationService{applications: []Application{
		{ID: "app-expense", Name: "Expense", Type: "internal_app", Description: "Submit expenses", LogoURL: "https://cdn.example/expense.svg", EntryURL: "https://expense.example"},
		{ID: "app-wiki", Name: "Wiki", Type: "oidc_client"},
	}}
	router := chi.NewRouter()
	newHandler(service).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/portal/applications", nil)
	req.AddCookie(userSessionCookie(t, "entity-a"))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if service.entityID != "entity-a" {
		t.Fatalf("entity id = %q, want entity-a", service.entityID)
	}

	var response struct {
		Applications []map[string]any `json:"applications"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Applications) != 2 {
		t.Fatalf("applications = %#v", response.Applications)
	}
	first := response.Applications[0]
	if first["id"] != "app-expense" || first["name"] != "Expense" || first["type"] != "internal_app" || first["entry_url"] != "https://expense.example" {
		t.Fatalf("first application = %#v", first)
	}
	for _, unsafeField := range []string{"entity_id", "status", "config", "client_secret", "client_secret_hash", "redirect_uris", "roles", "permissions", "has_access"} {
		if _, ok := first[unsafeField]; ok {
			t.Errorf("unsafe field %q present in %#v", unsafeField, first)
		}
	}
}

func TestListApplicationsDoesNotEvaluateAccessAndPreservesServiceOrdering(t *testing.T) {
	service := &fakeApplicationService{applications: []Application{
		{ID: "app-a", Name: "Alpha", Type: "internal_app"},
		{ID: "app-b", Name: "Beta", Type: "internal_app"},
	}}
	router := chi.NewRouter()
	newHandler(service).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/portal/applications", nil)
	req.AddCookie(userSessionCookie(t, "entity-a"))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if service.calls != 1 {
		t.Fatalf("service calls = %d, want 1", service.calls)
	}
	var response struct {
		Applications []Application `json:"applications"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := []string{response.Applications[0].ID, response.Applications[1].ID}; got[0] != "app-a" || got[1] != "app-b" {
		t.Fatalf("application ordering = %v", got)
	}
}

func TestListApplicationsRejectsUnauthenticatedRequest(t *testing.T) {
	router := chi.NewRouter()
	newHandler(&fakeApplicationService{}).RegisterRoutes(router)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/portal/applications", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

type fakeApplicationService struct {
	applications []Application
	entityID     string
	calls        int
}

type fakeApplicationStore struct {
	rows []generated.ListPortalApplicationsRow
	err  error
}

func (s fakeApplicationStore) ListPortalApplications(_ context.Context, _ string) ([]generated.ListPortalApplicationsRow, error) {
	return s.rows, s.err
}

func (s *fakeApplicationService) ListApplications(_ context.Context, entityID string) ([]Application, error) {
	s.entityID = entityID
	s.calls++
	return s.applications, nil
}

func userSessionCookie(t *testing.T, entityID string) *http.Cookie {
	t.Helper()
	value, err := auth.EncodeSession(auth.Session{UserID: "user-a", EntityID: entityID, Username: "user"})
	if err != nil {
		t.Fatalf("encode session: %v", err)
	}
	return &http.Cookie{Name: "idb_session", Value: value}
}
