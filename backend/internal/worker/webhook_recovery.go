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
}

func NewWebhookRecoveryPoller(store webhookRecoveryStore, dispatcher webhookRecoveryDispatcher, interval time.Duration, logger *zap.Logger) *WebhookRecoveryPoller {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &WebhookRecoveryPoller{store: store, dispatcher: dispatcher, interval: interval, logger: logger}
}

func (p *WebhookRecoveryPoller) Run(ctx context.Context) {
	p.poll(ctx)
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

func (p *WebhookRecoveryPoller) poll(ctx context.Context) {
	claimToken := id.NewULID()
	sources, err := p.store.ClaimDueWebhookSyncSources(ctx, generated.ClaimDueWebhookSyncSourcesParams{
		BatchSize:    webhookRecoveryBatchSize,
		ClaimToken:   claimToken,
		LeaseSeconds: int32(webhookRecoveryLease / time.Second),
	})
	if err != nil {
		if ctx.Err() == nil {
			p.logger.Error("failed to claim persisted webhook jobs", zap.Error(err))
		}
		return
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
