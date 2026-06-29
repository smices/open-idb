// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/smices/open-idb/internal/sso"
)

type mockOIDCDirectoryService struct {
	rootFn     func(ctx context.Context, entityID string, limit, offset int32) (OrganizationTreeRootResponse, error)
	childrenFn func(ctx context.Context, entityID string, kind OrganizationTreeNodeKind, parentID string, limit, offset int32) ([]OrganizationTreeNode, error)
	searchFn   func(ctx context.Context, entityID, query string, limit, offset int32) (OrganizationTreeSearchResponse, error)
}

func (m mockOIDCDirectoryService) ResolveOrganizationTreeEntityID(_ context.Context, candidate string) (string, error) {
	return candidate, nil
}

func (m mockOIDCDirectoryService) GetOrganizationTreeRoot(ctx context.Context, entityID string, limit, offset int32) (OrganizationTreeRootResponse, error) {
	if m.rootFn != nil {
		return m.rootFn(ctx, entityID, limit, offset)
	}
	return OrganizationTreeRootResponse{}, nil
}

func (m mockOIDCDirectoryService) ListOrganizationTreeChildren(ctx context.Context, entityID string, kind OrganizationTreeNodeKind, parentID string, limit, offset int32) ([]OrganizationTreeNode, error) {
	if m.childrenFn != nil {
		return m.childrenFn(ctx, entityID, kind, parentID, limit, offset)
	}
	return nil, nil
}

func (m mockOIDCDirectoryService) SearchOrganizationTree(ctx context.Context, entityID, query string, limit, offset int32) (OrganizationTreeSearchResponse, error) {
	if m.searchFn != nil {
		return m.searchFn(ctx, entityID, query, limit, offset)
	}
	return OrganizationTreeSearchResponse{}, nil
}

type mockOIDCTokenService struct {
	token sso.SSOTokenLookup
	err   error
}

func (m mockOIDCTokenService) IntrospectToken(_ context.Context, _, _ string) (sso.SSOTokenLookup, error) {
	return m.token, m.err
}

func TestOIDCDirectorySearchRequiresDirectoryReadScope(t *testing.T) {
	handler := NewOIDCDirectoryHandler(
		mockOIDCDirectoryService{},
		mockOIDCTokenService{token: sso.SSOTokenLookup{
			EntityID:  testEntityID(),
			UserID:    testUserID(),
			ClientID:  "client_1",
			TokenType: "access",
			Scopes:    []string{"openid", "profile"},
		}},
	)
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/directory/organization-tree/search?q=jacky", nil)
	req.Header.Set("X-IDB-Entity-ID", testEntityID())
	req.Header.Set("Authorization", "Bearer token")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestOIDCDirectorySearchUsesBearerTokenEntity(t *testing.T) {
	wantEntityID := testEntityID()
	handler := NewOIDCDirectoryHandler(
		mockOIDCDirectoryService{
			searchFn: func(_ context.Context, entityID, query string, limit, offset int32) (OrganizationTreeSearchResponse, error) {
				if entityID != wantEntityID {
					t.Fatalf("entityID = %q, want %q", entityID, wantEntityID)
				}
				if query != "jacky" {
					t.Fatalf("query = %q, want jacky", query)
				}
				return OrganizationTreeSearchResponse{
					Items: []OrganizationTreeNode{{
						ID:          testUserID(),
						Kind:        organizationTreeKindUser,
						Name:        "朱辉",
						EnglishName: "Jacky",
						Email:       "jacky@example.test",
						Status:      "active",
					}},
					Total:  1,
					Limit:  20,
					Offset: 0,
				}, nil
			},
		},
		mockOIDCTokenService{token: sso.SSOTokenLookup{
			EntityID:  wantEntityID,
			UserID:    testUserID(),
			ClientID:  "client_1",
			TokenType: "access",
			Scopes:    []string{"openid", directoryReadScope},
		}},
	)
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/directory/organization-tree/search?q=jacky", nil)
	req.Header.Set("X-IDB-Entity-ID", wantEntityID)
	req.Header.Set("Authorization", "Bearer token")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var response OrganizationTreeSearchResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Items) != 1 || response.Items[0].EnglishName != "Jacky" {
		t.Fatalf("unexpected response: %+v", response)
	}
}
