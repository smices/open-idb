// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/smices/open-idb/internal/auth"
)

func TestGetFeishuConfigRequiresSession(t *testing.T) {
	router := newConfigTestRouter(&fakeConfigService{})
	req := httptest.NewRequest(http.MethodGet, "/sapi/identity-sources/feishu/config", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestGetFeishuConfigReturnsProvider(t *testing.T) {
	router := newConfigTestRouter(&fakeConfigService{provider: FeishuIdentitySourceConfig{
		Provider:        "feishu",
		DisplayName:     "Feishu",
		Status:          "active",
		OAuthConfigured: true,
		SyncEnabled:     true,
	}})
	req := httptest.NewRequest(http.MethodGet, "/sapi/identity-sources/feishu/config", nil)
	req.AddCookie(testSessionCookie())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var response FeishuIdentitySourceConfig
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Provider != "feishu" || !response.SyncEnabled {
		t.Fatalf("response = %#v", response)
	}
}

func TestGetFeishuConfigReturnsDefaultWhenMissing(t *testing.T) {
	router := newConfigTestRouter(&fakeConfigService{})
	req := httptest.NewRequest(http.MethodGet, "/sapi/identity-sources/feishu/config", nil)
	req.AddCookie(testSessionCookie())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var response FeishuIdentitySourceConfig
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Provider != "feishu" || response.Status != "disabled" {
		t.Fatalf("response = %#v", response)
	}
}

func TestUpsertFeishuConfigCallsService(t *testing.T) {
	service := &fakeConfigService{}
	router := newConfigTestRouter(service)
	req := httptest.NewRequest(http.MethodPut, "/sapi/identity-sources/feishu/config", strings.NewReader(`{"display_name":"Feishu","status":"active","oauth_configured":true,"sync_enabled":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(testSessionCookie())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if service.upsertProvider.Provider != "feishu" {
		t.Fatalf("input = %#v", service.upsertProvider)
	}
}

func TestValidateFeishuConfigAcceptsAppCredentials(t *testing.T) {
	input := UpsertFeishuIdentitySourceConfigInput{
		Provider:        "feishu",
		DisplayName:     "Feishu",
		Status:          "active",
		OAuthConfigured: true,
		SyncEnabled:     true,
		Config:          json.RawMessage(`{"app_id":"cli_xxx","app_secret":"sec_xxx"}`),
	}

	if err := validateFeishuIdentitySourceInput(input); err != nil {
		t.Fatalf("validateFeishuIdentitySourceInput() error = %v", err)
	}
}

func newConfigTestRouter(service ConfigService) http.Handler {
	r := chi.NewRouter()
	NewConfigHandler(service).RegisterRoutes(r)
	return r
}

type fakeConfigService struct {
	provider       FeishuIdentitySourceConfig
	upsertProvider UpsertFeishuIdentitySourceConfigInput
}

func (f *fakeConfigService) GetFeishuConfig(context.Context, auth.Session) (FeishuIdentitySourceConfig, error) {
	if f.provider.Provider != "" {
		return f.provider, nil
	}
	return FeishuIdentitySourceConfig{
		Provider:    "feishu",
		DisplayName: "Feishu",
		Status:      "disabled",
		Config:      json.RawMessage("{}"),
	}, nil
}

func (f *fakeConfigService) UpsertFeishuConfig(_ context.Context, _ auth.Session, input UpsertFeishuIdentitySourceConfigInput) (FeishuIdentitySourceConfig, error) {
	f.upsertProvider = input
	return FeishuIdentitySourceConfig{
		Provider:        input.Provider,
		DisplayName:     input.DisplayName,
		Status:          input.Status,
		OAuthConfigured: input.OAuthConfigured,
		SyncEnabled:     input.SyncEnabled,
	}, nil
}
