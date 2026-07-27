// SPDX-License-Identifier: MIT

package httpserver

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PoolStats interface {
	Stat() *pgxpool.Stat
}

type acquireTimeoutStats interface {
	AcquireTimeoutCount() uint64
}

func WithPoolStats(pool PoolStats) Option {
	return func(r chi.Router) {
		r.Get("/metrics", DatabaseMetricsHandler(pool))
	}
}

func DatabaseMetricsHandler(pool PoolStats) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if pool == nil {
			http.Error(w, "database pool unavailable", http.StatusServiceUnavailable)
			return
		}
		stat := pool.Stat()
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		writeGauge(w, "open_idb_db_pool_max_connections", "Configured maximum database connections.", stat.MaxConns())
		writeGauge(w, "open_idb_db_pool_total_connections", "Current database connections.", stat.TotalConns())
		writeGauge(w, "open_idb_db_pool_acquired_connections", "Connections currently checked out.", stat.AcquiredConns())
		writeGauge(w, "open_idb_db_pool_idle_connections", "Connections currently idle.", stat.IdleConns())
		writeCounter(w, "open_idb_db_pool_acquire_total", "Successful pool acquisitions.", stat.AcquireCount())
		writeCounter(w, "open_idb_db_pool_empty_acquire_total", "Acquisitions that waited for a connection.", stat.EmptyAcquireCount())
		writeCounter(w, "open_idb_db_pool_acquire_canceled_total", "Acquisitions canceled by their context.", stat.CanceledAcquireCount())
		var timeoutCount uint64
		if timeoutStats, ok := pool.(acquireTimeoutStats); ok {
			timeoutCount = timeoutStats.AcquireTimeoutCount()
		}
		writeCounter(w, "open_idb_db_pool_acquire_timeout_total", "Acquisitions that exceeded DB_POOL_ACQUIRE_TIMEOUT.", timeoutCount)
		writeCounter(w, "open_idb_db_pool_acquire_seconds_total", "Cumulative connection acquisition time.", stat.AcquireDuration().Seconds())
	}
}

func writeGauge(w http.ResponseWriter, name, help string, value int32) {
	_, _ = fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s gauge\n%s %d\n", name, help, name, name, value)
}

func writeCounter(w http.ResponseWriter, name, help string, value any) {
	_, _ = fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s counter\n%s %v\n", name, help, name, name, value)
}
