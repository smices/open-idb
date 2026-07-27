// SPDX-License-Identifier: MIT

package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDatabaseMetricsExposePoolCapacityAndAcquireCounters(t *testing.T) {
	cfg, err := pgxpool.ParseConfig("postgres://user:pass@127.0.0.1:1/db?pool_max_conns=5&pool_min_conns=0")
	if err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	NewRouter(WithPoolStats(pool)).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	for _, metric := range []string{
		"open_idb_db_pool_max_connections 5",
		"open_idb_db_pool_total_connections",
		"open_idb_db_pool_acquired_connections",
		"open_idb_db_pool_idle_connections",
		"open_idb_db_pool_empty_acquire_total",
		"open_idb_db_pool_acquire_canceled_total",
		"open_idb_db_pool_acquire_timeout_total",
		"open_idb_db_pool_acquire_seconds_total",
	} {
		if !strings.Contains(rec.Body.String(), metric) {
			t.Fatalf("metrics body missing %q:\n%s", metric, rec.Body.String())
		}
	}
}
