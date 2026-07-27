// SPDX-License-Identifier: MIT

package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultMaxConns        int32 = 10
	defaultMinConns        int32 = 2
	defaultMaxConnLifetime       = time.Hour
	defaultMaxConnIdleTime       = 15 * time.Minute
)

type PoolConfig struct {
	MaxConns        int32
	MinConns        int32
	MinConnsSet     bool
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	AcquireTimeout  time.Duration
	ApplicationName string
}

func NewPool(ctx context.Context, databaseURL string, options ...PoolConfig) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	var option PoolConfig
	if len(options) > 0 {
		option = options[0]
	}
	applyPoolConfig(cfg, databaseURL, option)

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	acquireTimeout := option.AcquireTimeout
	if acquireTimeout <= 0 {
		acquireTimeout = 2 * time.Second
	}
	pingCtx, cancel := context.WithTimeout(ctx, acquireTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

func applyPoolConfig(cfg *pgxpool.Config, databaseURL string, option PoolConfig) {
	if option.MaxConns > 0 {
		cfg.MaxConns = option.MaxConns
	} else if !hasPoolParameter(databaseURL, "pool_max_conns") {
		cfg.MaxConns = defaultMaxConns
	}

	if option.MinConnsSet || option.MinConns > 0 {
		cfg.MinConns = option.MinConns
	} else if !hasPoolParameter(databaseURL, "pool_min_conns") {
		cfg.MinConns = min(defaultMinConns, cfg.MaxConns)
	}

	if option.MaxConnLifetime > 0 {
		cfg.MaxConnLifetime = option.MaxConnLifetime
	} else if !hasPoolParameter(databaseURL, "pool_max_conn_lifetime") {
		cfg.MaxConnLifetime = defaultMaxConnLifetime
	}

	if option.MaxConnIdleTime > 0 {
		cfg.MaxConnIdleTime = option.MaxConnIdleTime
	} else if !hasPoolParameter(databaseURL, "pool_max_conn_idle_time") {
		cfg.MaxConnIdleTime = defaultMaxConnIdleTime
	}

	applicationName := strings.TrimSpace(option.ApplicationName)
	if applicationName == "" {
		applicationName = "idbridge"
	}
	if _, configured := cfg.ConnConfig.RuntimeParams["application_name"]; !configured {
		cfg.ConnConfig.RuntimeParams["application_name"] = applicationName
	}
}

func hasPoolParameter(databaseURL string, name string) bool {
	lower := strings.ToLower(databaseURL)
	return strings.Contains(lower, name+"=")
}
