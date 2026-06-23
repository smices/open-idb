// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
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

type fakeAdminAuthService struct {
	login   AdminLoginResult
	session AdminSession
	current AdminCurrentUser
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
