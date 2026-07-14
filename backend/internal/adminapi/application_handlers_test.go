// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

type applicationTestService struct {
	*mockUserService
	createApplicationFn func(context.Context, string, string, string) (ApplicationResponse, error)
}

type applicationDetailTestService struct {
	*applicationTestService
	getDetailFn    func(context.Context, string, string) (ApplicationDetailResponse, error)
	createDetailFn func(context.Context, string, ApplicationWriteInput) (ApplicationDetailResponse, error)
	updateDetailFn func(context.Context, string, string, ApplicationWriteInput) (ApplicationDetailResponse, error)
}

func (s *applicationDetailTestService) GetApplicationDetail(ctx context.Context, entityID, id string) (ApplicationDetailResponse, error) {
	return s.getDetailFn(ctx, entityID, id)
}

func (s *applicationDetailTestService) CreateApplicationDetail(ctx context.Context, entityID string, input ApplicationWriteInput) (ApplicationDetailResponse, error) {
	return s.createDetailFn(ctx, entityID, input)
}

func (s *applicationDetailTestService) UpdateApplicationDetail(ctx context.Context, entityID, id string, input ApplicationWriteInput) (ApplicationDetailResponse, error) {
	return s.updateDetailFn(ctx, entityID, id, input)
}

func (s *applicationTestService) CreateApplication(ctx context.Context, entityID, name, appType string) (ApplicationResponse, error) {
	if s.createApplicationFn == nil {
		return ApplicationResponse{}, fmt.Errorf("unexpected CreateApplication call")
	}
	return s.createApplicationFn(ctx, entityID, name, appType)
}

func newApplicationTestRouter(service userService) *chi.Mux {
	router := chi.NewRouter()
	NewApplicationHandler(service).RegisterRoutes(router)
	return router
}

func TestApplicationHandlerCreateAcceptsContractTypes(t *testing.T) {
	for _, appType := range []string{"oidc_client", "api_client", "internal_app"} {
		t.Run(appType, func(t *testing.T) {
			service := &applicationTestService{mockUserService: &mockUserService{}}
			service.createApplicationFn = func(_ context.Context, entityID, name, gotType string) (ApplicationResponse, error) {
				if entityID != "01HZZZZZZZ0000000000000099" || name != "Contract App" || gotType != appType {
					t.Fatalf("CreateApplication args = (%q, %q, %q)", entityID, name, gotType)
				}
				return ApplicationResponse{Name: name, Type: gotType, Status: "active"}, nil
			}

			req := httptest.NewRequest(http.MethodPost, "/sapi/applications", strings.NewReader(fmt.Sprintf(`{"name":"Contract App","type":%q}`, appType)))
			req.AddCookie(adminTestSessionCookie())
			rr := httptest.NewRecorder()

			newApplicationTestRouter(service).ServeHTTP(rr, req)

			if rr.Code != http.StatusCreated {
				t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestApplicationHandlerCreateRejectsUnsupportedType(t *testing.T) {
	called := false
	service := &applicationTestService{mockUserService: &mockUserService{}}
	service.createApplicationFn = func(_ context.Context, _, _, _ string) (ApplicationResponse, error) {
		called = true
		return ApplicationResponse{}, nil
	}
	req := httptest.NewRequest(http.MethodPost, "/sapi/applications", strings.NewReader(`{"name":"Legacy App","type":"oidc"}`))
	req.AddCookie(adminTestSessionCookie())
	rr := httptest.NewRecorder()

	newApplicationTestRouter(service).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if called {
		t.Fatal("CreateApplication was called for an unsupported type")
	}
	if !strings.Contains(rr.Body.String(), `"error":"invalid_application_type"`) {
		t.Fatalf("body = %s", rr.Body.String())
	}
}

func TestApplicationHandlerCreatesCompleteOIDCApplicationInOneServiceCall(t *testing.T) {
	legacyCalled := false
	base := &applicationTestService{mockUserService: &mockUserService{}}
	base.createApplicationFn = func(context.Context, string, string, string) (ApplicationResponse, error) {
		legacyCalled = true
		return ApplicationResponse{}, nil
	}
	service := &applicationDetailTestService{applicationTestService: base}
	service.createDetailFn = func(_ context.Context, entityID string, input ApplicationWriteInput) (ApplicationDetailResponse, error) {
		if entityID != "01HZZZZZZZ0000000000000099" {
			t.Fatalf("entityID = %q", entityID)
		}
		if input.Type != "oidc_client" || input.OIDCClient == nil {
			t.Fatalf("input = %#v", input)
		}
		if len(input.OIDCClient.RedirectURIs) != 1 || input.OIDCClient.RedirectURIs[0] != "https://client.example/callback" {
			t.Fatalf("redirect URIs = %#v", input.OIDCClient.RedirectURIs)
		}
		return ApplicationDetailResponse{
			ApplicationResponse: ApplicationResponse{ID: "01HZZZZZZZ0000000000000010", Name: input.Name, Type: input.Type, Status: "active"},
			OIDCClient:          &OIDCClientResponse{ClientID: "client-id", ClientSecret: "readable-secret"},
		}, nil
	}

	req := httptest.NewRequest(http.MethodPost, "/sapi/applications", strings.NewReader(`{
		"name":"Complete OIDC App",
		"type":"oidc_client",
		"status":"active",
		"oidc_client":{"redirect_uris":["https://client.example/callback"],"pkce_required":true}
	}`))
	req.AddCookie(adminTestSessionCookie())
	rr := httptest.NewRecorder()

	newApplicationTestRouter(service).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if rr.Header().Get("Cache-Control") != "no-store" || rr.Header().Get("Pragma") != "no-cache" {
		t.Fatalf("application create cache headers = %#v", rr.Header())
	}
	if legacyCalled {
		t.Fatal("legacy CreateApplication was called instead of the complete mutation")
	}
	if !strings.Contains(rr.Body.String(), `"client_secret":"readable-secret"`) {
		t.Fatalf("response does not contain the readable client secret: %s", rr.Body.String())
	}
}

func TestApplicationHandlerUpdatePassesEditableOIDCCallbacks(t *testing.T) {
	service := &applicationDetailTestService{applicationTestService: &applicationTestService{mockUserService: &mockUserService{}}}
	service.updateDetailFn = func(_ context.Context, _, _ string, input ApplicationWriteInput) (ApplicationDetailResponse, error) {
		if input.OIDCClient == nil || len(input.OIDCClient.RedirectURIs) != 1 || input.OIDCClient.RedirectURIs[0] != "https://new.example/callback" {
			t.Fatalf("input = %#v", input)
		}
		return ApplicationDetailResponse{ApplicationResponse: ApplicationResponse{Name: input.Name, Type: "oidc_client", Status: input.Status}}, nil
	}

	req := httptest.NewRequest(http.MethodPut, "/sapi/applications/01HZZZZZZZ0000000000000010", strings.NewReader(`{
		"name":"Updated OIDC App",
		"type":"oidc_client",
		"status":"active",
		"oidc_client":{"redirect_uris":["https://new.example/callback"]}
	}`))
	req.AddCookie(adminTestSessionCookie())
	rr := httptest.NewRecorder()

	newApplicationTestRouter(service).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if rr.Header().Get("Cache-Control") != "no-store" || rr.Header().Get("Pragma") != "no-cache" {
		t.Fatalf("application update cache headers = %#v", rr.Header())
	}
}
