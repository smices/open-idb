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
