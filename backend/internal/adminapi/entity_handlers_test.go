// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/smices/open-idb/internal/auth"
)

type mockEntityService struct {
	listEntitiesFn  func(ctx context.Context, limit, offset int32) ([]EntityResponse, error)
	countEntitiesFn func(ctx context.Context) (int64, error)
	getEntityFn     func(ctx context.Context, id string) (EntityResponse, error)
	createEntityFn  func(ctx context.Context, name, slug, defaultLocale, brandName, logoURL, loginMessage string) (EntityResponse, error)
	updateEntityFn  func(ctx context.Context, id string, name, status, defaultLocale, brandName, logoURL, loginMessage pgtype.Text) (EntityResponse, error)
}

func (m *mockEntityService) ListEntities(ctx context.Context, limit, offset int32) ([]EntityResponse, error) {
	return m.listEntitiesFn(ctx, limit, offset)
}

func (m *mockEntityService) CountEntities(ctx context.Context) (int64, error) {
	return m.countEntitiesFn(ctx)
}

func (m *mockEntityService) GetEntityByID(ctx context.Context, id string) (EntityResponse, error) {
	return m.getEntityFn(ctx, id)
}

func (m *mockEntityService) CreateEntity(ctx context.Context, name, slug, defaultLocale, brandName, logoURL, loginMessage string) (EntityResponse, error) {
	return m.createEntityFn(ctx, name, slug, defaultLocale, brandName, logoURL, loginMessage)
}

func (m *mockEntityService) UpdateEntity(ctx context.Context, id string, name, status, defaultLocale, brandName, logoURL, loginMessage pgtype.Text) (EntityResponse, error) {
	return m.updateEntityFn(ctx, id, name, status, defaultLocale, brandName, logoURL, loginMessage)
}

func TestEntityHandlerListsEntities(t *testing.T) {
	service := &mockEntityService{
		listEntitiesFn: func(_ context.Context, limit, offset int32) ([]EntityResponse, error) {
			if limit != 20 || offset != 0 {
				t.Fatalf("pagination = (%d, %d), want (20, 0)", limit, offset)
			}
			return []EntityResponse{testEntityResponse()}, nil
		},
		countEntitiesFn: func(_ context.Context) (int64, error) {
			return 1, nil
		},
	}

	r := chi.NewRouter()
	NewEntityHandler(service).RegisterRoutes(r)
	req := entityRequest(t, http.MethodGet, "/api/admin/v1/entities", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"slug":"configured_entity"`) {
		t.Fatalf("body missing configured entity: %s", rr.Body.String())
	}
}

func TestEntityHandlerCreatesEntity(t *testing.T) {
	service := &mockEntityService{
		createEntityFn: func(_ context.Context, name, slug, defaultLocale, brandName, logoURL, loginMessage string) (EntityResponse, error) {
			if name != "Configured Entity" || slug != "configured_entity" || defaultLocale != "zh-CN" || brandName != "Configured Brand" || loginMessage == "" {
				t.Fatalf("create args = (%q, %q, %q, %q, %q, %q)", name, slug, defaultLocale, brandName, logoURL, loginMessage)
			}
			return testEntityResponse(), nil
		},
	}

	r := chi.NewRouter()
	NewEntityHandler(service).RegisterRoutes(r)
	req := entityRequest(t, http.MethodPost, "/api/admin/v1/entities", strings.NewReader(`{"name":"Configured Entity","slug":"configured_entity","default_locale":"zh-CN","brand_name":"Configured Brand","login_message":"Configured login message."}`))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"name":"Configured Entity"`) {
		t.Fatalf("body missing entity name: %s", rr.Body.String())
	}
}

func entityRequest(t *testing.T, method, target string, body io.Reader) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, target, body)
	session, err := auth.EncodeSession(auth.Session{
		UserID:      "01HZZZZZZZ0000000000000001",
		EntityID:    "01HZZZZZZZ0000000000000099",
		Username:    "admin",
		DisplayName: "Administrator",
	})
	if err != nil {
		t.Fatalf("encode session: %v", err)
	}
	req.AddCookie(&http.Cookie{Name: "idb_session", Value: session})
	return req
}

func testEntityResponse() EntityResponse {
	return EntityResponse{
		ID:            "01HZZZZZZZ0000000000000099",
		Name:          "Configured Entity",
		Slug:          "configured_entity",
		Status:        "active",
		DefaultLocale: "zh-CN",
		BrandName:     "Configured Brand",
		LoginMessage:  "Configured login message.",
		CreatedAt:     time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	}
}
