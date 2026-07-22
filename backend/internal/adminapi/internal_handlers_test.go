// SPDX-License-Identifier: MIT

package adminapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestInternalServiceRoutesAreNotExposedThroughAdminAPI(t *testing.T) {
	router := chi.NewRouter()
	NewInternalHandler(nil).RegisterRoutes(router)

	for _, path := range []string{
		"/sapi/internal/introspect",
		"/sapi/internal/permissions/check",
		"/sapi/internal/users/01HZZZZZZZ0000000000000010/access",
		"/sapi/internal/audit-events",
	} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
			}
		})
	}
}
