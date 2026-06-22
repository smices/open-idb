// SPDX-License-Identifier: MIT

package rbac

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/smices/open-idb/internal/auth"
)

// Note: Role, Permission, and Resource Scope CRUD tests are in adminapi package.
// These tests cover RBAC-specific routes: permission check, user role assignment, etc.

func TestHandler_CheckPermission_MissingSession(t *testing.T) {
	handler := NewHandler(nil)

	req := httptest.NewRequest("POST", "/admin/v1/permissions/check", strings.NewReader(`{"user_id":"user-1","permission":"read:users"}`))
	rec := httptest.NewRecorder()

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestHandler_CheckPermission_MissingFields(t *testing.T) {
	// Create a valid session
	session := auth.Session{
		UserID:   "user-1",
		EntityID: "entity-1",
		Username: "testuser",
	}
	sessionValue, err := auth.EncodeSession(session)
	if err != nil {
		t.Fatalf("failed to encode session: %v", err)
	}

	handler := NewHandler(nil)

	// Create a test request with missing fields
	req := httptest.NewRequest("POST", "/admin/v1/permissions/check", strings.NewReader(`{"user_id":"user-1"}`))
	req.AddCookie(&http.Cookie{Name: "idb_session", Value: sessionValue})
	rec := httptest.NewRecorder()

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["error"] != "missing_fields" {
		t.Errorf("expected error 'missing_fields', got '%s'", response["error"])
	}
}

func TestHandler_AssignRoleToUser_MissingSession(t *testing.T) {
	handler := NewHandler(nil)

	req := httptest.NewRequest("POST", "/admin/v1/users/user-1/roles", strings.NewReader(`{"role_id":"role-1"}`))
	rec := httptest.NewRecorder()

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestHandler_APIPrefix_CheckPermission(t *testing.T) {
	// Test that /api/admin/v1/permissions/check route works
	handler := NewHandler(nil)

	req := httptest.NewRequest("POST", "/api/admin/v1/permissions/check", strings.NewReader(`{"user_id":"user-1","permission":"read:users"}`))
	rec := httptest.NewRecorder()

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	r.ServeHTTP(rec, req)

	// Should return 401 (missing session) not 404 (route not found)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d for /api prefix, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestHandler_AssignPermissionToRole_MissingSession(t *testing.T) {
	handler := NewHandler(nil)

	req := httptest.NewRequest("POST", "/admin/v1/roles/role-1/permissions", strings.NewReader(`{"permission_id":"perm-1"}`))
	rec := httptest.NewRecorder()

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestHandler_ApplicationAssignment_MissingSession(t *testing.T) {
	handler := NewHandler(nil)

	req := httptest.NewRequest("POST", "/admin/v1/applications/app-1/assignments", strings.NewReader(`{"subject_type":"user","subject_id":"user-1","effect":"allow"}`))
	rec := httptest.NewRecorder()

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}
