// SPDX-License-Identifier: MIT

package worker

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type CleanupQueries interface {
	DeleteExpiredAuthorizationCodes(ctx context.Context) (int64, error)
	DeleteExpiredOAuthTokens(ctx context.Context) (int64, error)
	MarkExpiredSessions(ctx context.Context) (int64, error)
}

type CleanupRunner struct {
	queries  CleanupQueries
	interval time.Duration
	logger   *zap.Logger
	limiter  *backgroundLimiter
}

type CleanupResult struct {
	AuthorizationCodesDeleted int64
	OAuthTokensDeleted        int64
	SessionsExpired           int64
}

func NewCleanupRunner(queries CleanupQueries, interval time.Duration, logger *zap.Logger) *CleanupRunner {
	if interval <= 0 {
		interval = time.Hour
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &CleanupRunner{
		queries:  queries,
		interval: interval,
		logger:   logger,
	}
}

func (r *CleanupRunner) Run(ctx context.Context) {
	if r == nil || r.queries == nil {
		return
	}
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var result CleanupResult
			run := func(operationCtx context.Context) error {
				var err error
				result, err = r.RunOnce(operationCtx)
				return err
			}
			var err error
			if r.limiter != nil {
				err = r.limiter.do(ctx, run)
			} else {
				err = run(ctx)
			}
			if err != nil {
				r.logger.Warn("cleanup run failed", zap.Error(err))
				continue
			}
			r.logger.Info("cleanup run completed",
				zap.Int64("authorization_codes_deleted", result.AuthorizationCodesDeleted),
				zap.Int64("oauth_tokens_deleted", result.OAuthTokensDeleted),
				zap.Int64("sessions_expired", result.SessionsExpired),
			)
		}
	}
}

func (r *CleanupRunner) RunOnce(ctx context.Context) (CleanupResult, error) {
	var result CleanupResult
	var err error
	if result.AuthorizationCodesDeleted, err = r.queries.DeleteExpiredAuthorizationCodes(ctx); err != nil {
		return CleanupResult{}, err
	}
	if result.OAuthTokensDeleted, err = r.queries.DeleteExpiredOAuthTokens(ctx); err != nil {
		return CleanupResult{}, err
	}
	if result.SessionsExpired, err = r.queries.MarkExpiredSessions(ctx); err != nil {
		return CleanupResult{}, err
	}
	return result, nil
}
