// SPDX-License-Identifier: MIT

package feishu

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/smices/open-idb/internal/idp"
)

func TestFullSyncFetchesTokenDepartmentsAndUsers(t *testing.T) {
	var tokenRequest map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open-apis/auth/v3/tenant_access_token/internal":
			if err := json.NewDecoder(r.Body).Decode(&tokenRequest); err != nil {
				t.Fatalf("decode token request: %v", err)
			}
			writeJSON(t, w, map[string]interface{}{"code": 0, "tenant_access_token": "tenant-token"})
		case "/open-apis/contact/v3/departments/0/children":
			if got := r.Header.Get("Authorization"); got != "Bearer tenant-token" {
				t.Fatalf("department auth = %q", got)
			}
			if got := r.URL.Query().Get("page_size"); got != "50" {
				t.Fatalf("department page_size = %q, want 50", got)
			}
			if got := r.URL.Query().Get("fetch_child"); got != "true" {
				t.Fatalf("department fetch_child = %q, want true", got)
			}
			if got := r.URL.Query().Get("department_id_type"); got != "department_id" {
				t.Fatalf("department_id_type = %q, want department_id", got)
			}
			writeJSON(t, w, map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{
					"items": []map[string]interface{}{
						{
							"department_id":        "od-1",
							"parent_department_id": "0",
							"name":                 "研发中心",
						},
					},
				},
			})
		case "/open-apis/contact/v3/users/find_by_department":
			if got := r.Header.Get("Authorization"); got != "Bearer tenant-token" {
				t.Fatalf("user auth = %q", got)
			}
			if got := r.URL.Query().Get("page_size"); got != "50" {
				t.Fatalf("user page_size = %q, want 50", got)
			}
			if got := r.URL.Query().Get("department_id"); got != "0" && got != "od-1" {
				t.Fatalf("user department_id = %q, want root or synced department", got)
			}
			writeJSON(t, w, map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{
					"items": []map[string]interface{}{
						{
							"user_id":    "ou_1",
							"union_id":   "on_1",
							"open_id":    "open_1",
							"name":       "张三",
							"email":      "zhangsan@example.test",
							"mobile":     "13800000000",
							"avatar_url": "https://example.test/a.png",
							"department_ids": []string{
								"0",
							},
							"status": map[string]interface{}{
								"is_activated": true,
								"is_frozen":    false,
								"is_resigned":  false,
							},
						},
					},
				},
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{AppID: "app-id", AppSecret: "secret", BaseURL: server.URL}, server.Client())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	data, err := client.FullSync(context.Background())
	if err != nil {
		t.Fatalf("FullSync() error = %v", err)
	}

	if tokenRequest["app_id"] != "app-id" || tokenRequest["app_secret"] != "secret" {
		t.Fatalf("token request = %#v", tokenRequest)
	}
	if len(data.Departments) != 1 || data.Departments[0].Name != "研发中心" {
		t.Fatalf("departments = %#v", data.Departments)
	}
	if !strings.Contains(string(data.Departments[0].RawProfile), "研发中心") {
		t.Fatalf("department raw profile = %s", string(data.Departments[0].RawProfile))
	}
	if len(data.Users) != 1 || data.Users[0].Name != "张三" || data.Users[0].Status != "active" {
		t.Fatalf("users = %#v", data.Users)
	}
	if !strings.Contains(string(data.Users[0].RawProfile), "张三") {
		t.Fatalf("user raw profile = %s", string(data.Users[0].RawProfile))
	}
	var rawUser map[string]interface{}
	if err := json.Unmarshal(data.Users[0].RawProfile, &rawUser); err != nil {
		t.Fatalf("unmarshal user raw profile: %v", err)
	}
	departmentIDs, ok := rawUser["department_ids"].([]interface{})
	if !ok || len(departmentIDs) != 1 || departmentIDs[0] != "0" {
		t.Fatalf("user department_ids = %#v, want provider value [0]", rawUser["department_ids"])
	}
	if _, ok := rawUser["department_id"]; ok {
		t.Fatalf("user raw profile should not include synthetic department_id: %#v", rawUser["department_id"])
	}
}

func TestFullSyncNormalizesOpenDepartmentParentIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open-apis/auth/v3/tenant_access_token/internal":
			writeJSON(t, w, map[string]interface{}{"code": 0, "tenant_access_token": "tenant-token"})
		case "/open-apis/contact/v3/departments/0/children":
			writeJSON(t, w, map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{
					"items": []map[string]interface{}{
						{
							"department_id":        "dep_parent",
							"open_department_id":   "od_parent",
							"parent_department_id": "0",
							"name":                 "总部",
						},
						{
							"department_id":        "dep_child",
							"open_department_id":   "od_child",
							"parent_department_id": "od_parent",
							"name":                 "平台部",
						},
					},
				},
			})
		case "/open-apis/contact/v3/users/find_by_department":
			writeJSON(t, w, map[string]interface{}{"code": 0, "data": map[string]interface{}{"items": []map[string]interface{}{}}})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{AppID: "app-id", AppSecret: "secret", BaseURL: server.URL}, server.Client())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	data, err := client.FullSync(context.Background())
	if err != nil {
		t.Fatalf("FullSync() error = %v", err)
	}
	if len(data.Departments) != 2 {
		t.Fatalf("len(departments) = %d, want 2", len(data.Departments))
	}
	if data.Departments[1].ExternalDepartmentID != "dep_child" {
		t.Fatalf("child external ID = %q, want dep_child", data.Departments[1].ExternalDepartmentID)
	}
	if data.Departments[1].ParentExternalDepartmentID != "dep_parent" {
		t.Fatalf("child parent external ID = %q, want dep_parent", data.Departments[1].ParentExternalDepartmentID)
	}
}

func TestHTTPErrorIncludesResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"code":40011,"msg":"page size is invalid"}`, http.StatusBadRequest)
	}))
	defer server.Close()

	client, err := NewClient(Config{AppID: "app-id", AppSecret: "secret", BaseURL: server.URL}, server.Client())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.FullSync(context.Background())
	if err == nil {
		t.Fatal("FullSync() error = nil, want http error")
	}
	if !strings.Contains(err.Error(), "40011") || !strings.Contains(err.Error(), "page size is invalid") {
		t.Fatalf("error = %q, want response body included", err.Error())
	}
}

func TestFullSyncReturnsProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]interface{}{"code": 999, "msg": "bad app"})
	}))
	defer server.Close()

	client, err := NewClient(Config{AppID: "app-id", AppSecret: "secret", BaseURL: server.URL}, server.Client())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if _, err := client.FullSync(context.Background()); err == nil {
		t.Fatal("FullSync() error = nil, want provider error")
	}
}

func TestIncrementalSyncFallsBackToFullSync(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open-apis/auth/v3/tenant_access_token/internal":
			writeJSON(t, w, map[string]interface{}{"code": 0, "tenant_access_token": "tenant-token"})
		case "/open-apis/contact/v3/users/ou_1":
			writeJSON(t, w, map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{
					"user": map[string]interface{}{
						"user_id":    "ou_1",
						"union_id":   "on_1",
						"open_id":    "open_1",
						"name":       "李四",
						"email":      "lisi@example.test",
						"mobile":     "13900000000",
						"avatar_url": "https://example.test/b.png",
						"status": map[string]interface{}{
							"is_activated": true,
							"is_frozen":    false,
							"is_resigned":  false,
						},
					},
				},
			})
		case "/open-apis/contact/v3/departments/od_1":
			writeJSON(t, w, map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{
					"department": map[string]interface{}{
						"department_id":        "od_1",
						"open_department_id":   "open_od_1",
						"parent_department_id": "0",
						"name":                 "平台部",
					},
				},
			})
		default:
			if r.URL.Path == "/open-apis/contact/v3/departments/od_missing" || strings.HasPrefix(r.URL.Path, "/open-apis/contact/v3/departments") {
				writeJSON(t, w, map[string]interface{}{"code": 999, "msg": "not found"})
				return
			}
			if strings.HasPrefix(r.URL.Path, "/open-apis/contact/v3/users") {
				writeJSON(t, w, map[string]interface{}{"code": 999, "msg": "not found"})
				return
			}
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{AppID: "app-id", AppSecret: "secret", BaseURL: server.URL}, server.Client())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	data, err := client.IncrementalSync(context.Background(), []idp.DirectorySyncEvent{
		{
			EventType:    "added_user",
			ObjectType:   "user",
			ObjectID:     "ou_1",
			ObjectIDType: "user_id",
		},
		{
			EventType:    "delete_user",
			ObjectType:   "user",
			ObjectID:     "ou_missing",
			ObjectIDType: "user_id",
		},
		{
			EventType:    "added_department",
			ObjectType:   "department",
			ObjectID:     "od_1",
			ObjectIDType: "department_id",
		},
		{
			EventType:    "delete_department",
			ObjectType:   "department",
			ObjectID:     "od_missing",
			ObjectIDType: "department_id",
		},
	})
	if err != nil {
		t.Fatalf("IncrementalSync() error = %v", err)
	}
	if len(data.Users) != 2 {
		t.Fatalf("unexpected users = %#v", data.Users)
	}
	if !(data.Users[0].ExternalUserID == "ou_1" || data.Users[1].ExternalUserID == "ou_1") ||
		!(data.Users[0].ExternalUserID == "ou_missing" || data.Users[1].ExternalUserID == "ou_missing") {
		t.Fatalf("unexpected users = %#v", data.Users)
	}
	userDeleted := false
	deptDeleted := false
	for _, user := range data.Users {
		if user.ExternalUserID == "ou_missing" && user.Status == "deleted" {
			userDeleted = true
		}
	}
	for _, dept := range data.Departments {
		if dept.ExternalDepartmentID == "od_missing" {
			deptDeleted = true
		}
	}
	if !userDeleted || len(data.Departments) != 2 || !deptDeleted {
		t.Fatalf("unexpected departments/users = %#v", data)
	}
}

func TestNewClientRejectsMissingCredentials(t *testing.T) {
	if _, err := NewClient(Config{AppID: "app-id"}, nil); err == nil {
		t.Fatal("NewClient() error = nil, want error")
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode json: %v", err)
	}
}
