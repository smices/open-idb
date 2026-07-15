// SPDX-License-Identifier: MIT

package feishu

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetUserInfoByCodeExchangesCodeAndFetchesProfile(t *testing.T) {
	var gotTokenBody map[string]string
	var gotUserInfoAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open-apis/auth/v3/app_access_token/internal":
			writeJSON(t, w, map[string]interface{}{
				"code":             0,
				"app_access_token": "app-token-123",
			})
		case "/open-apis/authen/v1/oidc/access_token":
			gotTokenBody = make(map[string]string)
			_ = json.NewDecoder(r.Body).Decode(&gotTokenBody)
			if auth := r.Header.Get("Authorization"); auth != "Bearer app-token-123" {
				t.Fatalf("oidc token auth = %q, want Bearer app-token-123", auth)
			}
			writeJSON(t, w, map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{
					"access_token": "user-token-456",
				},
			})
		case "/open-apis/authen/v1/user_info":
			gotUserInfoAuth = r.Header.Get("Authorization")
			writeJSON(t, w, map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{
					"open_id":     "ou_abc",
					"union_id":    "on_def",
					"user_id":     "emp_001",
					"name":        "张三",
					"en_name":     "Zhang San",
					"employee_no": "E-001",
					"job_title":   "Principal Engineer",
					"email":       "zhangsan@example.test",
					"mobile":      "13800000000",
					"avatar_url":  "https://example.test/avatar.png",
					"entity_key":  "tk_1",
				},
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{AppID: "app-id", AppSecret: "secret", BaseURL: server.URL}, server.Client())
	if err != nil {
		t.Fatalf("NewClient error = %v", err)
	}

	result, err := client.GetUserInfoByCode(context.Background(), "auth-code-xyz")
	if err != nil {
		t.Fatalf("GetUserInfoByCode error = %v", err)
	}

	if gotTokenBody["grant_type"] != "authorization_code" || gotTokenBody["code"] != "auth-code-xyz" {
		t.Fatalf("token body = %#v", gotTokenBody)
	}
	if gotUserInfoAuth != "Bearer user-token-456" {
		t.Fatalf("user info auth = %q", gotUserInfoAuth)
	}
	if result.OpenID != "ou_abc" {
		t.Fatalf("OpenID = %q, want ou_abc", result.OpenID)
	}
	if result.UnionID != "on_def" {
		t.Fatalf("UnionID = %q, want on_def", result.UnionID)
	}
	if result.UserID != "emp_001" {
		t.Fatalf("UserID = %q, want emp_001", result.UserID)
	}
	if result.Name != "张三" {
		t.Fatalf("Name = %q, want 张三", result.Name)
	}
	if result.EnglishName != "Zhang San" {
		t.Fatalf("EnglishName = %q, want Zhang San", result.EnglishName)
	}
	if result.EmployeeNo != "E-001" {
		t.Fatalf("EmployeeNo = %q, want E-001", result.EmployeeNo)
	}
	if result.JobTitle != "Principal Engineer" {
		t.Fatalf("JobTitle = %q, want Principal Engineer", result.JobTitle)
	}
	if result.Email != "zhangsan@example.test" {
		t.Fatalf("Email = %q", result.Email)
	}
	if result.Phone != "13800000000" {
		t.Fatalf("Phone = %q", result.Phone)
	}
	if result.Status != "active" {
		t.Fatalf("Status = %q, want active", result.Status)
	}
	if !strings.Contains(string(result.RawProfile), "张三") {
		t.Fatalf("RawProfile missing Chinese name: %s", string(result.RawProfile))
	}
}

func TestGetUserInfoByAppCodeExchangesAuthCode(t *testing.T) {
	var gotAppTokenBody map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open-apis/auth/v3/app_access_token/internal":
			writeJSON(t, w, map[string]interface{}{
				"code":             0,
				"app_access_token": "app-token",
			})
		case "/open-apis/authen/v1/access_token":
			gotAppTokenBody = make(map[string]string)
			_ = json.NewDecoder(r.Body).Decode(&gotAppTokenBody)
			writeJSON(t, w, map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{
					"access_token": "user-token",
				},
			})
		case "/open-apis/authen/v1/user_info":
			writeJSON(t, w, map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{
					"open_id":     "ou_app1",
					"name":        "李四",
					"i18n_name":   map[string]string{"en_us": "Li Si"},
					"employee_no": "E-002",
					"job_title":   "Product Manager",
					"email":       "lisi@example.test",
				},
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{AppID: "app-id", AppSecret: "secret", BaseURL: server.URL}, server.Client())
	if err != nil {
		t.Fatalf("NewClient error = %v", err)
	}

	result, err := client.GetUserInfoByAppCode(context.Background(), "app-code-123")
	if err != nil {
		t.Fatalf("GetUserInfoByAppCode error = %v", err)
	}

	if gotAppTokenBody["app_access_token"] != "app-token" {
		t.Fatalf("app_access_token = %q, want app-token", gotAppTokenBody["app_access_token"])
	}
	if gotAppTokenBody["grant_type"] != "authorization_code" {
		t.Fatalf("grant_type = %q", gotAppTokenBody["grant_type"])
	}
	if gotAppTokenBody["code"] != "app-code-123" {
		t.Fatalf("code = %q", gotAppTokenBody["code"])
	}
	if result.OpenID != "ou_app1" {
		t.Fatalf("OpenID = %q", result.OpenID)
	}
	if result.Name != "李四" {
		t.Fatalf("Name = %q, want 李四", result.Name)
	}
	if result.EnglishName != "Li Si" || result.EmployeeNo != "E-002" || result.JobTitle != "Product Manager" {
		t.Fatalf("workplace profile fields = %#v", result)
	}
	// When user_id is empty, should fall back to open_id.
	if result.UserID != "ou_app1" {
		t.Fatalf("UserID = %q, want fallback to open_id ou_app1", result.UserID)
	}
}

func TestGetUserInfoByCodeReturnsAppTokenError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]interface{}{"code": 999, "msg": "invalid app"})
	}))
	defer server.Close()

	client, err := NewClient(Config{AppID: "bad", AppSecret: "bad", BaseURL: server.URL}, server.Client())
	if err != nil {
		t.Fatalf("NewClient error = %v", err)
	}

	if _, err := client.GetUserInfoByCode(context.Background(), "code"); err == nil {
		t.Fatal("expected error for bad app credentials")
	}
}

func TestGetUserInfoByCodeReturnsOIDCTokenError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open-apis/auth/v3/app_access_token/internal":
			writeJSON(t, w, map[string]interface{}{
				"code":             0,
				"app_access_token": "app-token",
			})
		case "/open-apis/authen/v1/oidc/access_token":
			writeJSON(t, w, map[string]interface{}{
				"code": 400,
				"msg":  "invalid code",
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{AppID: "app-id", AppSecret: "secret", BaseURL: server.URL}, server.Client())
	if err != nil {
		t.Fatalf("NewClient error = %v", err)
	}

	if _, err := client.GetUserInfoByCode(context.Background(), "bad-code"); err == nil {
		t.Fatal("expected error for invalid code")
	}
}

func TestGetUserInfoByCodeHandlesDisabledUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open-apis/auth/v3/app_access_token/internal":
			writeJSON(t, w, map[string]interface{}{
				"code":             0,
				"app_access_token": "app-token",
			})
		case "/open-apis/authen/v1/oidc/access_token":
			writeJSON(t, w, map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{"access_token": "user-token"},
			})
		case "/open-apis/authen/v1/user_info":
			writeJSON(t, w, map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{
					"open_id":      "ou_disabled",
					"name":         "王五",
					"is_activated": false,
					"is_frozen":    true,
				},
			})
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{AppID: "app-id", AppSecret: "secret", BaseURL: server.URL}, server.Client())
	if err != nil {
		t.Fatalf("NewClient error = %v", err)
	}

	result, err := client.GetUserInfoByCode(context.Background(), "code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "disabled" {
		t.Fatalf("Status = %q, want disabled", result.Status)
	}
	if result.Name != "王五" {
		t.Fatalf("Name = %q, want 王五", result.Name)
	}
}

func TestGetUserInfoByCodeHandlesActiveStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open-apis/auth/v3/app_access_token/internal":
			writeJSON(t, w, map[string]interface{}{
				"code":             0,
				"app_access_token": "app-token",
			})
		case "/open-apis/authen/v1/oidc/access_token":
			writeJSON(t, w, map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{"access_token": "user-token"},
			})
		case "/open-apis/authen/v1/user_info":
			activated := true
			_ = activated
			writeJSON(t, w, map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{
					"open_id":      "ou_active",
					"name":         "Active User",
					"is_activated": true,
				},
			})
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{AppID: "app-id", AppSecret: "secret", BaseURL: server.URL}, server.Client())
	if err != nil {
		t.Fatalf("NewClient error = %v", err)
	}

	result, err := client.GetUserInfoByCode(context.Background(), "code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "active" {
		t.Fatalf("Status = %q, want active", result.Status)
	}
}
