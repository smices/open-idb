// SPDX-License-Identifier: MIT

package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestLoginContextEntityAdminRoute(t *testing.T) {
	router := chi.NewRouter()
	NewHandler(fakeLoginService{
		entity: LoginContextEntity{
			ID:           "01HZZZZZZZ0000000000000001",
			Slug:         "configured_entity",
			Name:         "Configured Entity",
			BrandName:    "Configured Brand",
			LogoURL:      "https://example.test/logo.png",
			LoginMessage: "Configured login message.",
		},
	}).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/sapi/auth/context?path=/admin/login&return_to=/admin", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var ctx LoginContext
	if err := json.NewDecoder(rec.Body).Decode(&ctx); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if ctx.Mode != LoginModeEntityAdmin {
		t.Fatalf("mode = %q, want %q", ctx.Mode, LoginModeEntityAdmin)
	}
	if ctx.AllowEntitySelection {
		t.Fatal("entity admin route must not allow entity selection")
	}
}

func TestLoginContextDoesNotExposeSystemAdminRoute(t *testing.T) {
	router := chi.NewRouter()
	NewHandler(fakeLoginService{}).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/sapi/auth/context?path=/system/login", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var ctx LoginContext
	if err := json.NewDecoder(rec.Body).Decode(&ctx); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if ctx.Mode != LoginModeUser {
		t.Fatalf("mode = %q, want %q", ctx.Mode, LoginModeUser)
	}
	if ctx.Entity != nil {
		t.Fatalf("entity = %#v, want nil", ctx.Entity)
	}
}

func TestLoginContextDirectLoginUsesDefaultEnterpriseContext(t *testing.T) {
	router := chi.NewRouter()
	NewHandler(fakeLoginService{
		defaultCtx: LoginContext{
			Mode: LoginModeUser,
			Entity: &LoginContextEntity{
				ID:        "01HZZZZZZZ0000000000000002",
				Slug:      "configured_entity",
				Name:      "Configured Entity",
				BrandName: "Configured Brand",
			},
			Methods:              []string{"password", "feishu"},
			AllowEntitySelection: false,
			Reason:               "default_entity",
		},
	}).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/context?path=/login", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var ctx LoginContext
	if err := json.NewDecoder(rec.Body).Decode(&ctx); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if ctx.Entity == nil || ctx.Entity.Slug != "configured_entity" || ctx.Entity.BrandName != "Configured Brand" {
		t.Fatalf("entity = %#v, want configured default entity", ctx.Entity)
	}
	if !stringSliceContains(ctx.Methods, "feishu") {
		t.Fatalf("methods = %#v, want feishu", ctx.Methods)
	}
	if ctx.Reason != "default_entity" {
		t.Fatalf("reason = %q, want default_entity", ctx.Reason)
	}
}

func TestPreferredProviderAcceptsLarkAlias(t *testing.T) {
	returnTo := "/oauth2/authorize?response_type=code&client_id=my-app&idp=lark"

	got := preferredProviderFromReturnTo(returnTo)

	if got != "feishu" {
		t.Fatalf("preferred provider = %q, want feishu", got)
	}
}

func TestPreferredProviderRejectsUnsafeName(t *testing.T) {
	returnTo := "/oauth2/authorize?response_type=code&client_id=my-app&idp=../feishu"

	got := preferredProviderFromReturnTo(returnTo)

	if got != "" {
		t.Fatalf("preferred provider = %q, want empty", got)
	}
}

func TestApplyPreferredProviderRequiresAllowedMethod(t *testing.T) {
	ctx := LoginContext{
		Mode:    LoginModeUser,
		Entity:  &LoginContextEntity{ID: "entity-1"},
		Methods: []string{"password"},
	}

	ctx.applyPreferredProvider("feishu")
	ctx.applyAutoRedirect()

	if ctx.PreferredProvider != "" {
		t.Fatalf("preferred provider = %q, want empty", ctx.PreferredProvider)
	}
	if ctx.AutoRedirectURL != "" {
		t.Fatalf("auto redirect URL = %q, want empty", ctx.AutoRedirectURL)
	}
}
