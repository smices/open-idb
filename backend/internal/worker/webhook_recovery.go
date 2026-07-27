// SPDX-License-Identifier: MIT

package worker

import (
	"context"
	"time"

	"github.com/smices/open-idb/internal/db/generated"
	"github.com/smices/open-idb/internal/id"
	"go.uber.org/zap"
)

const (
	webhookRecoveryBatchSize      int32 = 100
	webhookRecoveryLease                = 5 * time.Minute
	webhookRecoveryReleaseTimeout       = 2 * time.Second
)

type webhookRecoveryStore interface {
	ClaimDueWebhookSyncSources(context.Context, generated.ClaimDueWebhookSyncSourcesParams) ([]generated.ClaimDueWebhookSyncSourcesRow, error)
	ReleaseWebhookSyncLease(context.Context, generated.ReleaseWebhookSyncLeaseParams) (int64, error)
}

type webhookRecoveryDispatcher interface {
	TriggerRecoveredWebhookSync(entityID, sourceID, provider, claimToken string) bool
}

// WebhookRecoveryPoller turns durable webhook rows back into incremental sync
// requests. The database lease prevents two application replicas from claiming
// the same identity source at the same time.
type WebhookRecoveryPoller struct {
	store      webhookRecoveryStore
	dispatcher webhookRecoveryDispatcher
	interval   time.Duration
	logger     *zap.Logger
	limiter    *backgroundLimiter
}

func NewWebhookRecoveryPoller(store webhookRecoveryStore, dispatcher webhookRecoveryDispatcher, interval time.Duration, logger *zap.Logger) *WebhookRecoveryPoller {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &WebhookRecoveryPoller{store: store, dispatcher: dispatcher, interval: interval, logger: logger}
}

func (p *WebhookRecoveryPoller) Run(ctx context.Context) {
	delay := time.Duration(0)
	failures := 0
	for {
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		if err := p.poll(ctx); err != nil {
			failures++
			delay = webhookPollRetryDelay(p.interval, failures)
		} else {
			failures = 0
			delay = p.interval
		}
	}
}

func (p *WebhookRecoveryPoller) poll(ctx context.Context) error {
	claimToken := id.NewULID()
	var sources []generated.ClaimDueWebhookSyncSourcesRow
	claim := func(operationCtx context.Context) error {
		var err error
		sources, err = p.store.ClaimDueWebhookSyncSources(operationCtx, generated.ClaimDueWebhookSyncSourcesParams{
			BatchSize:    webhookRecoveryBatchSize,
			ClaimToken:   claimToken,
			LeaseSeconds: int32(webhookRecoveryLease / time.Second),
		})
		return err
	}
	var err error
	if p.limiter != nil {
		err = p.limiter.do(ctx, claim)
	} else {
		err = claim(ctx)
	}
	if err != nil {
		if ctx.Err() == nil {
			p.logger.Error("failed to claim persisted webhook jobs", zap.Error(err))
		}
		return err
	}
	for _, source := range sources {
		if p.dispatcher.TriggerRecoveredWebhookSync(source.EntityID, source.SourceID, source.Provider, source.ClaimToken) {
			continue
		}
		p.releaseLease(source)
		p.logger.Warn("sync queue full; released persisted webhook source lease",
			zap.String("entity_id", source.EntityID),
			zap.String("source_id", source.SourceID),
		)
	}
	return nil
}

func webhookPollRetryDelay(interval time.Duration, failures int) time.Duration {
	if failures < 1 {
		return interval
	}
	delay := interval * time.Duration(1<<min(failures-1, 5))
	maxDelay := 15 * time.Minute
	if delay > maxDelay {
		delay = maxDelay
	}
	if delay == maxDelay {
		return maxDelay
	}
	jitter := time.Duration(time.Now().UnixNano() % max(1, int64(delay/4)))
	return delay + jitter
}

func (p *WebhookRecoveryPoller) releaseLease(source generated.ClaimDueWebhookSyncSourcesRow) {
	ctx, cancel := context.WithTimeout(context.Background(), webhookRecoveryReleaseTimeout)
	defer cancel()
	if _, err := p.store.ReleaseWebhookSyncLease(ctx, generated.ReleaseWebhookSyncLeaseParams{
		EntityID: source.EntityID, SourceID: source.SourceID, ClaimToken: source.ClaimToken,
	}); err != nil {
		p.logger.Warn("failed to release persisted webhook source lease", zap.Error(err))
	}
}
