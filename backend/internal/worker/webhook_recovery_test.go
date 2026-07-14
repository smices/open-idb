// SPDX-License-Identifier: MIT

package worker

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/smices/open-idb/internal/db/generated"
	"go.uber.org/zap"
)

type fakeWebhookRecoveryStore struct {
	mu    sync.Mutex
	calls int
	seen  chan generated.ClaimDueWebhookSyncSourcesParams
}

func (s *fakeWebhookRecoveryStore) ClaimDueWebhookSyncSources(_ context.Context, params generated.ClaimDueWebhookSyncSourcesParams) ([]generated.ClaimDueWebhookSyncSourcesRow, error) {
	s.mu.Lock()
	s.calls++
	s.mu.Unlock()
	select {
	case s.seen <- params:
	default:
	}
	return []generated.ClaimDueWebhookSyncSourcesRow{{
		EntityID:   "01HZZZZZZZ0000000000000001",
		SourceID:   "01HZZZZZZZ0000000000000002",
		Provider:   "feishu",
		ClaimToken: params.ClaimToken,
	}}, nil
}

func (s *fakeWebhookRecoveryStore) ReleaseWebhookSyncLease(context.Context, generated.ReleaseWebhookSyncLeaseParams) (int64, error) {
	return 1, nil
}

type fakeWebhookRecoveryDispatcher struct {
	dispatched chan recoveredWebhookSync
}

func (d *fakeWebhookRecoveryDispatcher) TriggerRecoveredWebhookSync(entityID, sourceID, provider, claimToken string) bool {
	dispatched := recoveredWebhookSync{
		entityID:   entityID,
		sourceID:   sourceID,
		provider:   provider,
		claimToken: claimToken,
	}
	select {
	case d.dispatched <- dispatched:
	default:
	}
	return true
}

type recoveredWebhookSync struct {
	entityID   string
	sourceID   string
	provider   string
	claimToken string
}

func TestWebhookRecoveryPollerPollsImmediatelyAndAtConfiguredInterval(t *testing.T) {
	store := &fakeWebhookRecoveryStore{seen: make(chan generated.ClaimDueWebhookSyncSourcesParams, 4)}
	dispatcher := &fakeWebhookRecoveryDispatcher{dispatched: make(chan recoveredWebhookSync, 4)}
	poller := NewWebhookRecoveryPoller(store, dispatcher, 20*time.Millisecond, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		poller.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})

	select {
	case params := <-store.seen:
		if params.ClaimToken == "" {
			t.Fatal("startup claim token is empty")
		}
		if params.LeaseSeconds <= 0 {
			t.Fatalf("startup lease seconds = %d, want positive", params.LeaseSeconds)
		}
	case <-time.After(time.Second):
		t.Fatal("poller did not inspect persisted jobs immediately on startup")
	}

	select {
	case got := <-dispatcher.dispatched:
		if got.entityID != "01HZZZZZZZ0000000000000001" || got.sourceID != "01HZZZZZZZ0000000000000002" || got.provider != "feishu" || got.claimToken == "" {
			t.Fatalf("unexpected recovered sync dispatch: %#v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("poller did not dispatch the startup recovery job")
	}

	select {
	case <-store.seen:
	case <-time.After(time.Second):
		t.Fatal("poller did not inspect persisted jobs again at PollInterval")
	}
}
