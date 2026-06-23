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

func TestListIMProviderConfigsRequiresSession(t *testing.T) {
	router := newConfigTestRouter(&fakeConfigService{})
	req := httptest.NewRequest(http.MethodGet, "/sapi/integrations/im", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestListIMProviderConfigsReturnsProviders(t *testing.T) {
	router := newConfigTestRouter(&fakeConfigService{providers: []IMProviderConfig{{
		Provider:        "feishu",
		DisplayName:     "Feishu",
		Status:          "active",
		OAuthConfigured: true,
		SyncEnabled:     true,
	}}})
	req := httptest.NewRequest(http.MethodGet, "/sapi/integrations/im", nil)
	req.AddCookie(testSessionCookie())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var response []IMProviderConfig
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response) != 1 || response[0].Provider != "feishu" || !response[0].SyncEnabled {
		t.Fatalf("response = %#v", response)
	}
}

func TestUpsertIMProviderConfigCallsService(t *testing.T) {
	service := &fakeConfigService{}
	router := newConfigTestRouter(service)
	req := httptest.NewRequest(http.MethodPut, "/sapi/integrations/im/feishu", strings.NewReader(`{"display_name":"Feishu","status":"active","oauth_configured":true,"sync_enabled":true}`))
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
	input := UpsertIMProviderConfigInput{
		Provider:        "feishu",
		DisplayName:     "Feishu",
		Status:          "active",
		OAuthConfigured: true,
		SyncEnabled:     true,
		Config:          json.RawMessage(`{"app_id":"cli_xxx","app_secret":"sec_xxx"}`),
	}

	if err := validateIMProviderInput(input); err != nil {
		t.Fatalf("validateIMProviderInput() error = %v", err)
	}
}

func newConfigTestRouter(service ConfigService) http.Handler {
	r := chi.NewRouter()
	NewConfigHandler(service).RegisterRoutes(r)
	return r
}

type fakeConfigService struct {
	providers      []IMProviderConfig
	upsertProvider UpsertIMProviderConfigInput
}

func (f *fakeConfigService) ListIMProviderConfigs(context.Context, auth.Session) ([]IMProviderConfig, error) {
	return f.providers, nil
}

func (f *fakeConfigService) UpsertIMProviderConfig(_ context.Context, _ auth.Session, input UpsertIMProviderConfigInput) (IMProviderConfig, error) {
	f.upsertProvider = input
	return IMProviderConfig{
		Provider:        input.Provider,
		DisplayName:     input.DisplayName,
		Status:          input.Status,
		OAuthConfigured: input.OAuthConfigured,
		SyncEnabled:     input.SyncEnabled,
	}, nil
}
