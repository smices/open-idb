// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestLoginAccountSetsSessionAndRedirects(t *testing.T) {
	router := chi.NewRouter()
	NewHandler(fakeLoginService{result: LoginResult{
		UserID:             "user-1",
		EntityID:           "entity-1",
		Username:           "admin",
		DisplayName:        "Administrator",
		MustChangePassword: true,
		WeakPassword:       true,
	}}).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/login/account", strings.NewReader("account=admin&password=admin123"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	if location := rec.Header().Get("Location"); location != "/dashboard" {
		t.Fatalf("Location = %q, want /dashboard", location)
	}
	if cookie := rec.Result().Cookies()[0]; cookie.Name != "idb_session" || cookie.Value == "" {
		t.Fatalf("cookie = %#v", cookie)
	}
}

func TestLoginAccountRejectsInvalidCredentials(t *testing.T) {
	router := chi.NewRouter()
	NewHandler(fakeLoginService{err: errInvalidLogin{}}).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/login/account", strings.NewReader("account=admin&password=bad"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestLoginAccountRedirectsBrowserFormErrorsToLogin(t *testing.T) {
	router := chi.NewRouter()
	NewHandler(fakeLoginService{err: errInvalidLogin{}}).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/login/account", strings.NewReader("account=admin&password=bad"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if location := rec.Header().Get("Location"); location != "/?login_error=invalid_credentials" {
		t.Fatalf("Location = %q", location)
	}
}

func TestSafeReturnToRejectsExternalURL(t *testing.T) {
	got := safeReturnTo("https://attacker.example/oauth2/authorize?client_id=demo-app&idp=feishu")

	if got != "/dashboard" {
		t.Fatalf("safeReturnTo() = %q, want /dashboard", got)
	}
}

func TestLoginContextResolvesOIDCApplicationReturnTo(t *testing.T) {
	router := chi.NewRouter()
	NewHandler(fakeLoginService{
		appClientID: "demo-app",
		appContext: LoginContext{
			Mode: LoginModeApp,
			Entity: &LoginContextEntity{
				ID:        "entity-1",
				Slug:      "configured_entity",
				Name:      "Configured Entity",
				BrandName: "Configured Brand",
			},
			Application: &LoginContextApplication{
				ID:   "app-1",
				Name: "Demo App",
			},
			Methods:              []string{"password", "feishu"},
			AllowEntitySelection: false,
			Reason:               "application",
		},
	}).RegisterRoutes(router)

	returnTo := "/api/oauth2/authorize?response_type=code&client_id=demo-app&redirect_uri=https%3A%2F%2Fdemo-app.local.test%2Fauth%2Foidc%2Fcallback"
	req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/auth/context?path=/login&return_to="+url.QueryEscape(returnTo), nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var ctx LoginContext
	if err := json.NewDecoder(rec.Body).Decode(&ctx); err != nil {
		t.Fatalf("decode context: %v", err)
	}
	if ctx.Mode != LoginModeApp {
		t.Fatalf("mode = %q, want %q", ctx.Mode, LoginModeApp)
	}
	if ctx.Entity == nil || ctx.Entity.ID != "entity-1" {
		t.Fatalf("entity = %#v, want entity-1", ctx.Entity)
	}
	if ctx.Application == nil || ctx.Application.Name != "Demo App" {
		t.Fatalf("application = %#v, want Demo App", ctx.Application)
	}
	if !stringSliceContains(ctx.Methods, "feishu") {
		t.Fatalf("methods = %#v, want feishu", ctx.Methods)
	}
	if ctx.ReturnTo != returnTo {
		t.Fatalf("return_to = %q, want %q", ctx.ReturnTo, returnTo)
	}
}

func TestLoginContextBuildsFeishuAutoRedirectForOIDCReturnTo(t *testing.T) {
	router := chi.NewRouter()
	NewHandler(fakeLoginService{
		appClientID: "demo-app",
		appContext: LoginContext{
			Mode: LoginModeApp,
			Entity: &LoginContextEntity{
				ID:   "entity-1",
				Slug: "configured_entity",
				Name: "Configured Entity",
			},
			Application: &LoginContextApplication{
				ID:   "app-1",
				Name: "Demo App",
			},
			Methods:              []string{"password", "feishu"},
			AllowEntitySelection: false,
			Reason:               "application",
		},
	}).RegisterRoutes(router)

	returnTo := "/api/oauth2/authorize?response_type=code&client_id=demo-app&redirect_uri=https%3A%2F%2Fdemo-app.local.test%2Fauth%2Foidc%2Fcallback&idp=feishu"
	req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/auth/context?path=/login&return_to="+url.QueryEscape(returnTo), nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var ctx LoginContext
	if err := json.NewDecoder(rec.Body).Decode(&ctx); err != nil {
		t.Fatalf("decode context: %v", err)
	}
	if ctx.PreferredProvider != "feishu" {
		t.Fatalf("preferred_provider = %q, want feishu", ctx.PreferredProvider)
	}
	if !strings.HasPrefix(ctx.AutoRedirectURL, "/api/auth/feishu/login?") {
		t.Fatalf("auto_redirect_url = %q, want Feishu login URL", ctx.AutoRedirectURL)
	}
	if !strings.Contains(ctx.AutoRedirectURL, "entity_id=entity-1") {
		t.Fatalf("auto_redirect_url = %q, want entity_id", ctx.AutoRedirectURL)
	}
	if !strings.Contains(ctx.AutoRedirectURL, "return_to=") {
		t.Fatalf("auto_redirect_url = %q, want return_to", ctx.AutoRedirectURL)
	}
}

type fakeLoginService struct {
	result      LoginResult
	err         error
	entity      LoginContextEntity
	appClientID string
	appContext  LoginContext
	defaultCtx  LoginContext
}

func (f fakeLoginService) AuthenticateLocal(context.Context, string, string) (LoginResult, error) {
	return f.result, f.err
}

func (f fakeLoginService) AuthenticateLocalWithEntity(context.Context, string, string, string) (LoginResult, error) {
	return f.result, f.err
}

func (f fakeLoginService) GetLoginContextEntityBySlug(_ context.Context, slug string) (LoginContextEntity, error) {
	if f.entity.Slug == slug {
		return f.entity, nil
	}
	return LoginContextEntity{}, errInvalidLogin{}
}

func (f fakeLoginService) GetLoginContextApplicationByClientID(_ context.Context, clientID string) (LoginContext, error) {
	if f.appClientID == clientID {
		return f.appContext, nil
	}
	return LoginContext{}, errInvalidLogin{}
}

func (f fakeLoginService) GetDefaultLoginContext(context.Context) (LoginContext, error) {
	if f.defaultCtx.Entity != nil {
		return f.defaultCtx, nil
	}
	return LoginContext{}, errInvalidLogin{}
}

type errInvalidLogin struct{}

func (errInvalidLogin) Error() string {
	return "invalid login"
}

func stringSliceContains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
