// SPDX-License-Identifier: MIT

package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
)

type HealthResponse struct {
	Status string `json:"status"`
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	writeHealth(w, http.StatusOK, "ok")
}

// ReadinessHandler verifies dependencies required to serve authenticated and
// OIDC traffic. A nil check preserves the lightweight router's test/default
// behaviour while production injects a PostgreSQL check.
func ReadinessHandler(check func(context.Context) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if check != nil {
			if err := check(r.Context()); err != nil {
				writeHealth(w, http.StatusServiceUnavailable, "unavailable")
				return
			}
		}
		writeHealth(w, http.StatusOK, "ok")
	}
}

func writeHealth(w http.ResponseWriter, status int, value string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(HealthResponse{Status: value})
}
