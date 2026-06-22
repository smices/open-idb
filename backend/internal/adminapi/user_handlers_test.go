// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// --- mock service ---

type mockUserService struct {
	listUsersFn           func(ctx context.Context, entityID string, status pgtype.Text, limit, offset int32) ([]UserResponse, error)
	countUsersFn          func(ctx context.Context, entityID string, status pgtype.Text) (int64, error)
	getUserByIDFn         func(ctx context.Context, entityID, id string) (UserResponse, error)
	updateUserLifecycleFn func(ctx context.Context, entityID, id string, status string) (UserResponse, error)
	updateUserFn          func(ctx context.Context, entityID, id string, displayName, email, phone, locale pgtype.Text) (UserResponse, error)
}

func (m *mockUserService) ListUsers(ctx context.Context, entityID string, status pgtype.Text, limit, offset int32) ([]UserResponse, error) {
	if m.listUsersFn != nil {
		return m.listUsersFn(ctx, entityID, status, limit, offset)
	}
	return nil, nil
}
func (m *mockUserService) CountUsers(ctx context.Context, entityID string, status pgtype.Text) (int64, error) {
	if m.countUsersFn != nil {
		return m.countUsersFn(ctx, entityID, status)
	}
	return 0, nil
}
func (m *mockUserService) GetUserByID(ctx context.Context, entityID, id string) (UserResponse, error) {
	if m.getUserByIDFn != nil {
		return m.getUserByIDFn(ctx, entityID, id)
	}
	return UserResponse{}, nil
}
func (m *mockUserService) UpdateUserLifecycle(ctx context.Context, entityID, id string, status string) (UserResponse, error) {
	if m.updateUserLifecycleFn != nil {
		return m.updateUserLifecycleFn(ctx, entityID, id, status)
	}
	return UserResponse{}, nil
}
func (m *mockUserService) UpdateUser(ctx context.Context, entityID, id string, displayName, email, phone, locale pgtype.Text) (UserResponse, error) {
	if m.updateUserFn != nil {
		return m.updateUserFn(ctx, entityID, id, displayName, email, phone, locale)
	}
	return UserResponse{}, nil
}
func (m *mockUserService) ListDirectoryUsers(_ context.Context, _, _ string, _, _ int32) ([]DirectoryUserResponse, error) {
	return nil, nil
}
func (m *mockUserService) CountDirectoryUsers(_ context.Context, _, _ string) (int64, error) {
	return 0, nil
}
func (m *mockUserService) GetDirectoryUserByID(_ context.Context, _, _ string) (DirectoryUserResponse, error) {
	return DirectoryUserResponse{}, nil
}
func (m *mockUserService) ListApplications(_ context.Context, _ string, _, _ int32) ([]ApplicationResponse, error) {
	return nil, nil
}
func (m *mockUserService) CountApplications(_ context.Context, _ string) (int64, error) {
	return 0, nil
}
func (m *mockUserService) GetApplicationByID(_ context.Context, _, _ string) (ApplicationResponse, error) {
	return ApplicationResponse{}, nil
}
func (m *mockUserService) CreateApplication(_ context.Context, _ string, _, _ string) (ApplicationResponse, error) {
	return ApplicationResponse{}, nil
}
func (m *mockUserService) UpdateApplication(_ context.Context, _, _ string, _, _ pgtype.Text) (ApplicationResponse, error) {
	return ApplicationResponse{}, nil
}
func (m *mockUserService) DeleteApplication(_ context.Context, _, _ string) error { return nil }
func (m *mockUserService) ListAllSyncJobs(_ context.Context, _ string, _, _ int32) ([]SyncJobResponse, error) {
	return nil, nil
}
func (m *mockUserService) CountAllSyncJobs(_ context.Context, _ string) (int64, error) { return 0, nil }
func (m *mockUserService) ListRoles(_ context.Context, _ string, _, _ int32) ([]RoleResponse, error) {
	return nil, nil
}
func (m *mockUserService) CountRoles(_ context.Context, _ string) (int64, error) { return 0, nil }
func (m *mockUserService) GetRoleByID(_ context.Context, _, _ string) (RoleResponse, error) {
	return RoleResponse{}, nil
}
func (m *mockUserService) ListPermissions(_ context.Context, _ string, _, _ int32) ([]PermissionResponse, error) {
	return nil, nil
}
func (m *mockUserService) CountPermissions(_ context.Context, _ string) (int64, error) { return 0, nil }
func (m *mockUserService) GetPermissionByID(_ context.Context, _, _ string) (PermissionResponse, error) {
	return PermissionResponse{}, nil
}

// --- helpers ---

func testUserID() string {
	id, _ := ulidValue("01HZZZZZZZ0000000000000001")
	return id
}

func testEntityID() string {
	id, _ := ulidValue("01HZZZZZZZ0000000000000099")
	return id
}

func testTime() time.Time {
	t, _ := time.Parse(time.RFC3339, "2025-01-01T00:00:00Z")
	return t
}

func sampleUserResponse() UserResponse {
	return UserResponse{
		ID:              "01HZZZZZZZ0000000000000001",
		EntityID:        "01HZZZZZZZ0000000000000099",
		Username:        "testuser",
		DisplayName:     "Test User",
		Email:           "test@example.com",
		LifecycleStatus: "active",
		UserType:        "local",
		Locale:          "en-US",
		CreatedAt:       testTime(),
		UpdatedAt:       testTime(),
	}
}

func newUserTestRouter(handler UserHandler) *chi.Mux {
	r := chi.NewRouter()
	handler.RegisterRoutes(r)
	return r
}

func adminTestSessionCookie() *http.Cookie {
	session := map[string]interface{}{
		"UserID":   "01HZZZZZZZ0000000000000002",
		"EntityID": "01HZZZZZZZ0000000000000099",
		"Username": "admin",
	}
	payload, _ := json.Marshal(session)
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return &http.Cookie{Name: "idb_session", Value: encoded}
}

// --- tests ---

func TestListUsers_ReturnsProperJSON(t *testing.T) {
	mock := &mockUserService{
		listUsersFn: func(_ context.Context, _ string, _ pgtype.Text, _, _ int32) ([]UserResponse, error) {
			return []UserResponse{sampleUserResponse()}, nil
		},
		countUsersFn: func(_ context.Context, _ string, _ pgtype.Text) (int64, error) {
			return 1, nil
		},
	}
	handler := NewUserHandler(mock)
	router := newUserTestRouter(handler)

	req := httptest.NewRequest("GET", "/admin/v1/users?limit=10&offset=0", nil)
	req.AddCookie(adminTestSessionCookie())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Items  []UserResponse `json:"items"`
		Total  int64          `json:"total"`
		Limit  int            `json:"limit"`
		Offset int            `json:"offset"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Total != 1 {
		t.Errorf("expected total 1, got %d", resp.Total)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}
	if resp.Items[0].Username != "testuser" {
		t.Errorf("expected username testuser, got %s", resp.Items[0].Username)
	}
	if resp.Items[0].LifecycleStatus != "active" {
		t.Errorf("expected lifecycle_status active, got %s", resp.Items[0].LifecycleStatus)
	}
	if resp.Limit != 10 {
		t.Errorf("expected limit 10, got %d", resp.Limit)
	}
	if resp.Offset != 0 {
		t.Errorf("expected offset 0, got %d", resp.Offset)
	}
}

func TestListUsers_MissingSession_Returns401(t *testing.T) {
	handler := NewUserHandler(&mockUserService{})
	router := newUserTestRouter(handler)

	req := httptest.NewRequest("GET", "/admin/v1/users", nil)
	// no cookie
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if resp["error"] != "session_required" {
		t.Errorf("expected error code session_required, got %s", resp["error"])
	}
}

func TestGetUserByID_Success(t *testing.T) {
	mock := &mockUserService{
		getUserByIDFn: func(_ context.Context, _ string, id string) (UserResponse, error) {
			return sampleUserResponse(), nil
		},
	}
	handler := NewUserHandler(mock)
	router := newUserTestRouter(handler)

	req := httptest.NewRequest("GET", "/admin/v1/users/01HZZZZZZZ0000000000000001", nil)
	req.AddCookie(adminTestSessionCookie())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var user UserResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &user); err != nil {
		t.Fatalf("failed to decode user response: %v", err)
	}
	if user.ID != "01HZZZZZZZ0000000000000001" {
		t.Errorf("expected id 01HZZZZZZZ0000000000000001, got %s", user.ID)
	}
	if user.Username != "testuser" {
		t.Errorf("expected username testuser, got %s", user.Username)
	}
}

func TestGetUserByID_InvalidID_Returns400(t *testing.T) {
	handler := NewUserHandler(&mockUserService{})
	router := newUserTestRouter(handler)

	req := httptest.NewRequest("GET", "/admin/v1/users/not-a-ulid", nil)
	req.AddCookie(adminTestSessionCookie())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestDisableUser_Success(t *testing.T) {
	mock := &mockUserService{
		updateUserLifecycleFn: func(_ context.Context, _, _ string, status string) (UserResponse, error) {
			if status != "disabled" {
				return UserResponse{}, nil
			}
			u := sampleUserResponse()
			u.LifecycleStatus = "disabled"
			return u, nil
		},
	}
	handler := NewUserHandler(mock)
	router := newUserTestRouter(handler)

	req := httptest.NewRequest("POST", "/admin/v1/users/01HZZZZZZZ0000000000000001/disable", nil)
	req.AddCookie(adminTestSessionCookie())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var user UserResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &user); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if user.LifecycleStatus != "disabled" {
		t.Errorf("expected lifecycle_status disabled, got %s", user.LifecycleStatus)
	}
}

func TestEnableUser_Success(t *testing.T) {
	mock := &mockUserService{
		updateUserLifecycleFn: func(_ context.Context, _, _ string, status string) (UserResponse, error) {
			u := sampleUserResponse()
			u.LifecycleStatus = status
			return u, nil
		},
	}
	handler := NewUserHandler(mock)
	router := newUserTestRouter(handler)

	req := httptest.NewRequest("POST", "/admin/v1/users/01HZZZZZZZ0000000000000001/enable", nil)
	req.AddCookie(adminTestSessionCookie())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var user UserResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &user); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if user.LifecycleStatus != "active" {
		t.Errorf("expected lifecycle_status active, got %s", user.LifecycleStatus)
	}
}
