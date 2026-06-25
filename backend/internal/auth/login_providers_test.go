// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/smices/open-idb/internal/db/generated"
)

// --- Mock ---

type mockProviderQueries struct {
	listFn         func(context.Context, string) ([]generated.ListLoginProvidersRow, error)
	entityBySlugFn func(context.Context, string) (generated.BusinessEntity, error)
}

func (m *mockProviderQueries) ListLoginProviders(ctx context.Context, entityID string) ([]generated.ListLoginProvidersRow, error) {
	if m.listFn != nil {
		return m.listFn(ctx, entityID)
	}
	return nil, nil
}

func (m *mockProviderQueries) GetEntityBySlug(ctx context.Context, slug string) (generated.BusinessEntity, error) {
	if m.entityBySlugFn != nil {
		return m.entityBySlugFn(ctx, slug)
	}
	return generated.BusinessEntity{}, fmt.Errorf("entity slug %q not found", slug)
}

// --- Tests ---

func TestListProvidersReturnsActiveConfiguredProviders(t *testing.T) {
	queries := &mockProviderQueries{
		listFn: func(_ context.Context, _ string) ([]generated.ListLoginProvidersRow, error) {
			return []generated.ListLoginProvidersRow{
				{Provider: "dingtalk", DisplayName: "钉钉", Status: "active", OauthConfigured: true},
				{Provider: "feishu", DisplayName: "飞书", Status: "active", OauthConfigured: true},
			}, nil
		},
	}
	svc := &LoginProviderService{
		queries:           queries,
		feishuAppID:       "test-app-id",
		feishuRedirectURI: "https://example.test/auth/feishu/callback",
	}

	providers, err := svc.ListProviders(context.Background(), "01HZZZZZZZ0000000000000001")
	if err != nil {
		t.Fatalf("ListProviders error = %v", err)
	}
	if len(providers) != 2 {
		t.Fatalf("len = %d, want 2", len(providers))
	}

	// First provider: dingtalk (no OAuth URL for non-feishu providers).
	if providers[0].Provider != "dingtalk" {
		t.Fatalf("provider[0] = %q", providers[0].Provider)
	}
	if providers[0].DisplayName != "钉钉" {
		t.Fatalf("display_name[0] = %q", providers[0].DisplayName)
	}
	if providers[0].OAuthURL != "" {
		t.Fatalf("dingtalk should not have oauth_url, got %q", providers[0].OAuthURL)
	}
	if providers[0].AppID != "" || providers[0].WorkplaceExchangeURL != "" {
		t.Fatalf("dingtalk should not expose workplace fields: %#v", providers[0])
	}

	// Second provider: feishu (should have OAuth URL).
	if providers[1].Provider != "feishu" {
		t.Fatalf("provider[1] = %q", providers[1].Provider)
	}
	if providers[1].DisplayName != "飞书" {
		t.Fatalf("display_name[1] = %q", providers[1].DisplayName)
	}
	if providers[1].OAuthURL == "" {
		t.Fatal("feishu provider missing oauth_url")
	}
	if !strings.Contains(providers[1].OAuthURL, "test-app-id") {
		t.Fatalf("oauth_url missing app_id: %q", providers[1].OAuthURL)
	}
	if !strings.Contains(providers[1].OAuthURL, "open.feishu.cn") {
		t.Fatalf("oauth_url missing feishu domain: %q", providers[1].OAuthURL)
	}
	if !strings.Contains(providers[1].OAuthURL, "redirect_uri") {
		t.Fatalf("oauth_url missing redirect_uri: %q", providers[1].OAuthURL)
	}
	if providers[1].AppID != "test-app-id" {
		t.Fatalf("app_id = %q, want test-app-id", providers[1].AppID)
	}
	if providers[1].WorkplaceExchangeURL != "/api/auth/feishu/exchange" {
		t.Fatalf("workplace_exchange_url = %q", providers[1].WorkplaceExchangeURL)
	}
}

func TestListProvidersEmptyWhenNoneConfigured(t *testing.T) {
	queries := &mockProviderQueries{
		listFn: func(_ context.Context, _ string) ([]generated.ListLoginProvidersRow, error) {
			return []generated.ListLoginProvidersRow{}, nil
		},
	}
	svc := &LoginProviderService{
		queries:     queries,
		feishuAppID: "test-app-id",
	}

	providers, err := svc.ListProviders(context.Background(), "01HZZZZZZZ0000000000000001")
	if err != nil {
		t.Fatalf("ListProviders error = %v", err)
	}
	if len(providers) != 0 {
		t.Fatalf("len = %d, want 0", len(providers))
	}
}

func TestListProvidersFeishuWithoutAppIDHasNoURL(t *testing.T) {
	queries := &mockProviderQueries{
		listFn: func(_ context.Context, _ string) ([]generated.ListLoginProvidersRow, error) {
			return []generated.ListLoginProvidersRow{
				{Provider: "feishu", DisplayName: "飞书", Status: "active", OauthConfigured: true},
			}, nil
		},
	}
	svc := &LoginProviderService{
		queries:     queries,
		feishuAppID: "", // no app ID configured
	}

	providers, err := svc.ListProviders(context.Background(), "01HZZZZZZZ0000000000000001")
	if err != nil {
		t.Fatalf("ListProviders error = %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("len = %d, want 1", len(providers))
	}
	if providers[0].OAuthURL != "" {
		t.Fatalf("feishu without app_id should not have oauth_url, got %q", providers[0].OAuthURL)
	}
	if providers[0].AppID != "" || providers[0].WorkplaceExchangeURL != "" {
		t.Fatalf("feishu without app_id should not expose workplace fields: %#v", providers[0])
	}
}

func TestListProvidersInvalidEntityID(t *testing.T) {
	svc := &LoginProviderService{
		queries:     &mockProviderQueries{},
		feishuAppID: "test-app-id",
	}

	_, err := svc.ListProviders(context.Background(), "not-a-ulid")
	if err == nil {
		t.Fatal("expected error for invalid entity ID")
	}
}

func TestListProvidersAcceptsEntitySlug(t *testing.T) {
	var gotEntityID string
	queries := &mockProviderQueries{
		entityBySlugFn: func(_ context.Context, slug string) (generated.BusinessEntity, error) {
			if slug != "configured_entity" {
				t.Fatalf("slug = %q, want configured_entity", slug)
			}
			return generated.BusinessEntity{ID: "01HZZZZZZZ0000000000000001", Slug: "configured_entity"}, nil
		},
		listFn: func(_ context.Context, entityID string) ([]generated.ListLoginProvidersRow, error) {
			gotEntityID = entityID
			return []generated.ListLoginProvidersRow{}, nil
		},
	}
	svc := &LoginProviderService{queries: queries}

	_, err := svc.ListProviders(context.Background(), "configured_entity")
	if err != nil {
		t.Fatalf("ListProviders error = %v", err)
	}
	if gotEntityID != "01HZZZZZZZ0000000000000001" {
		t.Fatalf("entityID = %q, want resolved ULID", gotEntityID)
	}
}

func TestListProvidersHandlerEndpoint(t *testing.T) {
	queries := &mockProviderQueries{
		listFn: func(_ context.Context, _ string) ([]generated.ListLoginProvidersRow, error) {
			return []generated.ListLoginProvidersRow{
				{Provider: "feishu", DisplayName: "飞书", Status: "active", OauthConfigured: true},
			}, nil
		},
	}
	loginSvc := &FeishuLoginService{
		queries:      &mockLoginQueries{},
		feishuClient: &mockFeishuProvider{},
		sessionTTL:   0,
	}
	providerSvc := &LoginProviderService{
		queries:           queries,
		feishuAppID:       "my-app",
		feishuRedirectURI: "https://example.test/callback",
	}
	handler := NewFeishuLoginHandler(loginSvc, providerSvc, "my-app", "https://example.test/callback")

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/providers", nil)
	req.Header.Set("X-IDB-Entity-ID", "01HZZZZZZZ0000000000000001")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var providers []LoginProvider
	if err := json.NewDecoder(rec.Body).Decode(&providers); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("len = %d, want 1", len(providers))
	}
	if providers[0].Provider != "feishu" {
		t.Fatalf("provider = %q", providers[0].Provider)
	}
	if providers[0].OAuthURL == "" {
		t.Fatal("missing oauth_url")
	}
	if providers[0].AppID != "my-app" {
		t.Fatalf("app_id = %q, want my-app", providers[0].AppID)
	}
	if providers[0].WorkplaceExchangeURL != "/api/auth/feishu/exchange" {
		t.Fatalf("workplace_exchange_url = %q", providers[0].WorkplaceExchangeURL)
	}
}

func TestListProvidersHandlerMissingEntity(t *testing.T) {
	loginSvc := &FeishuLoginService{
		queries:      &mockLoginQueries{},
		feishuClient: &mockFeishuProvider{},
	}
	providerSvc := &LoginProviderService{
		queries: &mockProviderQueries{},
	}
	handler := NewFeishuLoginHandler(loginSvc, providerSvc, "", "")

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/providers", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
