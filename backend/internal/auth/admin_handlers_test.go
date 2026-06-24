// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	auditmodel "github.com/smices/open-idb/internal/audit/model"
)

func TestAdminLoginSetsAdminSessionCookie(t *testing.T) {
	handler := NewAdminHandler(&fakeAdminAuthService{
		login: AdminLoginResult{
			AdminID:     "admin-1",
			Username:    "admin",
			DisplayName: "Administrator",
			Role:        "platform_admin",
		},
		session: AdminSession{
			ID:          "admin-session-1",
			AdminID:     "admin-1",
			Username:    "admin",
			DisplayName: "Administrator",
			Role:        "platform_admin",
			ExpiresAt:   time.Now().Add(time.Hour),
		},
	})
	router := chi.NewRouter()
	handler.RegisterRoutes(router)
	req := httptest.NewRequest(http.MethodPost, "/sapi/login/account", strings.NewReader("account=admin&password=admin123"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusFound, rec.Body.String())
	}
	if location := rec.Header().Get("Location"); location != "/admin" {
		t.Fatalf("Location = %q, want /admin", location)
	}
	cookie := rec.Result().Cookies()[0]
	if cookie.Name != "idb_admin_session" || cookie.Value != "admin-session-1" {
		t.Fatalf("cookie = %#v", cookie)
	}
}

func TestAdminMeReturnsAdminIdentity(t *testing.T) {
	service := &fakeAdminAuthService{
		current: AdminCurrentUser{
			ID:          "admin-1",
			Username:    "admin",
			DisplayName: "Administrator",
			Role:        "platform_admin",
		},
	}
	handler := NewAdminHandler(service)
	router := chi.NewRouter()
	handler.RegisterRoutes(router)
	req := httptest.NewRequest(http.MethodGet, "/sapi/me", nil)
	req = req.WithContext(WithAdminSession(req.Context(), AdminSession{AdminID: "admin-1", Username: "admin"}))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"role":"platform_admin"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestAdminManagementWritesAuditForCreate(t *testing.T) {
	audit := &fakeAdminAuditWriter{}
	service := &fakeAdminAuthService{
		created: AdminUserSummary{
			ID:          "01KVTADMINCREATE0000000001",
			EntityID:    "01KVTENTITY00000000000001",
			Username:    "ops-admin",
			DisplayName: "Ops Admin",
			Status:      "active",
			Role:        "enterprise_admin",
		},
	}
	handler := NewAdminHandler(service, audit)
	router := chi.NewRouter()
	handler.RegisterRoutes(router)
	body := strings.NewReader(`{"username":"ops-admin","display_name":"Ops Admin","role":"enterprise_admin","entity_id":"01KVTENTITY00000000000001","password":"StrongPass123!"}`)
	req := httptest.NewRequest(http.MethodPost, "/sapi/admin-users", body)
	req = req.WithContext(WithAdminSession(req.Context(), AdminSession{
		AdminID:  "01KVTACTOR000000000000001",
		Username: "admin",
		Role:     "platform_admin",
	}))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if len(audit.events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(audit.events))
	}
	event := audit.events[0]
	if event.Action != "admin.created" || event.ResourceType != "admin_user" || event.ResourceID != service.created.ID {
		t.Fatalf("unexpected audit event = %#v", event)
	}
	if event.ActorUserID != "01KVTACTOR000000000000001" || event.ActorType != "admin" {
		t.Fatalf("unexpected actor = %#v", event)
	}
}

func TestAdminCannotChangeOwnRoleOrStatus(t *testing.T) {
	target := AdminUserSummary{
		ID:          "01KVTACTOR000000000000001",
		Username:    "ops-admin",
		DisplayName: "Ops Admin",
		Status:      "active",
		Role:        "platform_admin",
	}

	if err := validateAdminUserUpdate(AdminSession{AdminID: target.ID}, target, AdminUserUpdateRequest{
		DisplayName: "Ops Admin",
		Role:        "enterprise_admin",
		EntityID:    "01KVTENTITY00000000000001",
		Status:      "active",
	}); err == nil {
		t.Fatal("expected self role change to be rejected")
	}

	if err := validateAdminUserUpdate(AdminSession{AdminID: target.ID}, target, AdminUserUpdateRequest{
		DisplayName: "Ops Admin",
		Role:        "platform_admin",
		Status:      "disabled",
	}); err == nil {
		t.Fatal("expected self status change to be rejected")
	}

	if err := validateAdminUserUpdate(AdminSession{AdminID: target.ID}, target, AdminUserUpdateRequest{
		DisplayName: "Ops Admin Updated",
		Role:        "platform_admin",
		Status:      "active",
	}); err != nil {
		t.Fatalf("display name self update rejected: %v", err)
	}
}

type fakeAdminAuthService struct {
	login   AdminLoginResult
	session AdminSession
	current AdminCurrentUser
	created AdminUserSummary
}

func (f *fakeAdminAuthService) AuthenticateAdmin(context.Context, string, string) (AdminLoginResult, error) {
	return f.login, nil
}

func (f *fakeAdminAuthService) CreateAdminSession(context.Context, AdminLoginResult, SessionMetadata) (AdminSession, error) {
	return f.session, nil
}

func (f *fakeAdminAuthService) CurrentAdmin(context.Context, AdminSession) (AdminCurrentUser, error) {
	return f.current, nil
}

func (f *fakeAdminAuthService) UpdateAdminProfile(context.Context, AdminSession, string) (AdminCurrentUser, error) {
	return f.current, nil
}

func (f *fakeAdminAuthService) UpdateAdminPassword(context.Context, AdminSession, string, string) error {
	return nil
}

func (f *fakeAdminAuthService) ListManagedAdminUsers(context.Context, AdminSession) (AdminUserListResponse, error) {
	return AdminUserListResponse{}, nil
}

func (f *fakeAdminAuthService) ListAssignableAdminRoles(context.Context, AdminSession) ([]AdminRoleOption, error) {
	return nil, nil
}

func (f *fakeAdminAuthService) CreateManagedAdminUser(context.Context, AdminSession, AdminUserCreateRequest) (AdminUserSummary, error) {
	return f.created, nil
}

func (f *fakeAdminAuthService) UpdateManagedAdminUser(context.Context, AdminSession, string, AdminUserUpdateRequest) (AdminUserSummary, error) {
	return AdminUserSummary{}, nil
}

func (f *fakeAdminAuthService) DeleteManagedAdminUser(context.Context, AdminSession, string) (AdminUserSummary, error) {
	return AdminUserSummary{}, nil
}

func (f *fakeAdminAuthService) SetManagedAdminPassword(context.Context, AdminSession, string, string) error {
	return nil
}

type fakeAdminAuditWriter struct {
	events []auditmodel.Event
}

func (f *fakeAdminAuditWriter) Write(_ context.Context, event auditmodel.Event) error {
	var copied auditmodel.Event
	raw, _ := json.Marshal(event)
	_ = json.Unmarshal(raw, &copied)
	copied.Before = event.Before
	copied.After = event.After
	f.events = append(f.events, copied)
	return nil
}
