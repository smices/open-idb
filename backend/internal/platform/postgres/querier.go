// SPDX-License-Identifier: MIT

package postgres

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Querier bounds only the pool-acquisition phase. Once a connection has been
// acquired, query execution continues under the caller's context.
type Querier struct {
	pool           *pgxpool.Pool
	acquireTimeout time.Duration
	timeoutCount   atomic.Uint64
}

func NewQuerier(pool *pgxpool.Pool, acquireTimeout time.Duration) *Querier {
	if acquireTimeout <= 0 {
		acquireTimeout = 2 * time.Second
	}
	return &Querier{pool: pool, acquireTimeout: acquireTimeout}
}

func (q *Querier) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	conn, err := q.acquire(ctx)
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	defer conn.Release()
	return conn.Exec(ctx, sql, arguments...)
}

func (q *Querier) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	conn, err := q.acquire(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := conn.Query(ctx, sql, args...)
	if err != nil {
		conn.Release()
		return nil, err
	}
	return &releasingRows{Rows: rows, release: conn.Release}, nil
}

func (q *Querier) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	conn, err := q.acquire(ctx)
	if err != nil {
		return errorRow{err: err}
	}
	return &releasingRow{Row: conn.QueryRow(ctx, sql, args...), release: conn.Release}
}

func (q *Querier) Begin(ctx context.Context) (pgx.Tx, error) {
	return q.BeginTx(ctx, pgx.TxOptions{})
}

func (q *Querier) BeginTx(ctx context.Context, options pgx.TxOptions) (pgx.Tx, error) {
	conn, err := q.acquire(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := conn.BeginTx(ctx, options)
	if err != nil {
		conn.Release()
		return nil, err
	}
	return &releasingTx{Tx: tx, release: conn.Release}, nil
}

func (q *Querier) acquire(ctx context.Context) (*pgxpool.Conn, error) {
	acquireCtx, cancel := context.WithTimeout(ctx, q.acquireTimeout)
	defer cancel()
	conn, err := q.pool.Acquire(acquireCtx)
	if err != nil && ctx.Err() == nil && acquireCtx.Err() == context.DeadlineExceeded {
		q.timeoutCount.Add(1)
	}
	return conn, err
}

func (q *Querier) Stat() *pgxpool.Stat {
	return q.pool.Stat()
}

func (q *Querier) AcquireTimeoutCount() uint64 {
	return q.timeoutCount.Load()
}

type releasingRows struct {
	pgx.Rows
	once    sync.Once
	release func()
}

func (r *releasingRows) Close() {
	r.Rows.Close()
	r.once.Do(r.release)
}

type releasingRow struct {
	pgx.Row
	once    sync.Once
	release func()
}

func (r *releasingRow) Scan(dest ...any) error {
	defer r.once.Do(r.release)
	return r.Row.Scan(dest...)
}

type errorRow struct {
	err error
}

func (r errorRow) Scan(...any) error {
	return r.err
}

type releasingTx struct {
	pgx.Tx
	once    sync.Once
	release func()
}

func (t *releasingTx) Commit(ctx context.Context) error {
	err := t.Tx.Commit(ctx)
	t.once.Do(t.release)
	return err
}

func (t *releasingTx) Rollback(ctx context.Context) error {
	err := t.Tx.Rollback(ctx)
	t.once.Do(t.release)
	return err
}
