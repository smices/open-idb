// SPDX-License-Identifier: MIT

package ephemeral

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

type Store interface {
	Increment(ctx context.Context, key string, ttl time.Duration) (int64, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Delete(ctx context.Context, key string) error
	Close() error
}

type LimitResult struct {
	Allowed bool
	Count   int64
	Limit   int64
	ResetIn time.Duration
}

func CheckLimit(ctx context.Context, store Store, key string, limit int64, window time.Duration) (LimitResult, error) {
	if store == nil || key == "" || limit <= 0 || window <= 0 {
		return LimitResult{Allowed: true, Limit: limit, ResetIn: window}, nil
	}
	count, err := store.Increment(ctx, key, window)
	if err != nil {
		return LimitResult{}, err
	}
	return LimitResult{
		Allowed: count <= limit,
		Count:   count,
		Limit:   limit,
		ResetIn: window,
	}, nil
}

func Key(prefix string, parts ...string) string {
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		normalized = append(normalized, strings.TrimSpace(strings.ToLower(part)))
	}
	sum := sha256.Sum256([]byte(strings.Join(normalized, "\x00")))
	return prefix + ":" + hex.EncodeToString(sum[:])
}
