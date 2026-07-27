// SPDX-License-Identifier: MIT

package postgres

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestApplyPoolConfigUsesProductionSafeDefaults(t *testing.T) {
	cfg, err := pgxpool.ParseConfig("postgres://user:pass@localhost/db?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	applyPoolConfig(cfg, "postgres://user:pass@localhost/db?sslmode=disable", PoolConfig{})

	if cfg.MaxConns != 10 || cfg.MinConns != 2 {
		t.Fatalf("pool bounds = %d/%d, want 10/2", cfg.MaxConns, cfg.MinConns)
	}
	if cfg.MaxConnLifetime != time.Hour || cfg.MaxConnIdleTime != 15*time.Minute {
		t.Fatalf("pool lifetimes = %s/%s", cfg.MaxConnLifetime, cfg.MaxConnIdleTime)
	}
	if got := cfg.ConnConfig.RuntimeParams["application_name"]; got != "idbridge" {
		t.Fatalf("application_name = %q, want idbridge", got)
	}
}

func TestApplyPoolConfigPreservesURLParameters(t *testing.T) {
	const databaseURL = "postgres://user:pass@localhost/db?pool_max_conns=5&pool_min_conns=1&pool_max_conn_lifetime=30m&pool_max_conn_idle_time=5m&application_name=custom"
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	applyPoolConfig(cfg, databaseURL, PoolConfig{})

	if cfg.MaxConns != 5 || cfg.MinConns != 1 {
		t.Fatalf("pool bounds = %d/%d, want URL values 5/1", cfg.MaxConns, cfg.MinConns)
	}
	if cfg.MaxConnLifetime != 30*time.Minute || cfg.MaxConnIdleTime != 5*time.Minute {
		t.Fatalf("pool lifetimes = %s/%s", cfg.MaxConnLifetime, cfg.MaxConnIdleTime)
	}
	if got := cfg.ConnConfig.RuntimeParams["application_name"]; got != "custom" {
		t.Fatalf("application_name = %q, want custom", got)
	}
}

func TestApplyPoolConfigEnvironmentOptionsOverrideURL(t *testing.T) {
	const databaseURL = "postgres://user:pass@localhost/db?pool_max_conns=5&pool_min_conns=1"
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	applyPoolConfig(cfg, databaseURL, PoolConfig{MaxConns: 8, MinConns: 3})

	if cfg.MaxConns != 8 || cfg.MinConns != 3 {
		t.Fatalf("pool bounds = %d/%d, want options 8/3", cfg.MaxConns, cfg.MinConns)
	}
}

func TestApplyPoolConfigAllowsEnvironmentToSetMinimumToZero(t *testing.T) {
	const databaseURL = "postgres://user:pass@localhost/db?pool_min_conns=4"
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	applyPoolConfig(cfg, databaseURL, PoolConfig{MinConns: 0, MinConnsSet: true})

	if cfg.MinConns != 0 {
		t.Fatalf("MinConns = %d, want environment override 0", cfg.MinConns)
	}
}
