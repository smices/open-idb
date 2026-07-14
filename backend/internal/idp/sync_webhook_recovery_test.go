// SPDX-License-Identifier: MIT

package idp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/smices/open-idb/internal/db/generated"
)

func TestWebhookRetryDelayUsesBoundedBackoff(t *testing.T) {
	tests := []struct {
		attempt int32
		want    time.Duration
	}{
		{attempt: 1, want: time.Minute},
		{attempt: 2, want: 5 * time.Minute},
		{attempt: 3, want: 30 * time.Minute},
		{attempt: 4, want: 2 * time.Hour},
		{attempt: 20, want: 2 * time.Hour},
	}
	for _, tc := range tests {
		if got := webhookRetryDelay(tc.attempt); got != tc.want {
			t.Fatalf("webhookRetryDelay(%d) = %s, want %s", tc.attempt, got, tc.want)
		}
	}
}

func TestWebhookJobsAreIdempotentRecoverableAndEventuallyTerminal(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool := newAtomicSyncTestPool(ctx, t)
	queries := generated.New(pool)

	entity, err := queries.CreateEntity(ctx, generated.CreateEntityParams{
		Name:          "Webhook Recovery Entity",
		Slug:          "webhook-recovery",
		DefaultLocale: "en-US",
	})
	if err != nil {
		t.Fatalf("create entity: %v", err)
	}
	source, err := queries.CreateIdentitySource(ctx, generated.CreateIdentitySourceParams{
		EntityID: entity.ID, Type: "feishu", Name: "Feishu", SyncEnabled: true,
	})
	if err != nil {
		t.Fatalf("create identity source: %v", err)
	}

	providerFailure := errors.New("temporary provider outage")
	service, err := NewSyncService(SyncServiceConfig{
		Queries:   queries,
		Provider:  fakeSyncDirectoryProvider{err: providerFailure},
		TraceID:   func() string { return "trace-webhook-recovery" },
		TxStarter: pool,
	})
	if err != nil {
		t.Fatalf("new sync service: %v", err)
	}
	event := DirectorySyncEvent{
		EventType: "user.updated", ObjectType: "user", ObjectID: "ou_retry", EventID: "evt_retry_1",
	}
	type submitResult struct {
		jobID string
		err   error
	}
	const concurrentDeliveries = 8
	startDeliveries := make(chan struct{})
	submitResults := make(chan submitResult, concurrentDeliveries)
	for range concurrentDeliveries {
		go func() {
			<-startDeliveries
			jobID, submitErr := service.SubmitWebhookEvent(ctx, entity.ID, source.ID, event)
			submitResults <- submitResult{jobID: jobID, err: submitErr}
		}()
	}
	close(startDeliveries)
	jobID := ""
	for range concurrentDeliveries {
		result := <-submitResults
		if result.err != nil {
			t.Fatalf("submit concurrent webhook event: %v", result.err)
		}
		if jobID == "" {
			jobID = result.jobID
		} else if result.jobID != jobID {
			t.Fatalf("duplicate event job id = %q, want existing %q", result.jobID, jobID)
		}
	}
	var duplicateCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM sync_jobs WHERE entity_id=$1 AND source_id=$2 AND type='webhook' AND event_id=$3`, entity.ID, source.ID, event.EventID).Scan(&duplicateCount); err != nil {
		t.Fatalf("count duplicate webhook jobs: %v", err)
	}
	if duplicateCount != 1 {
		t.Fatalf("duplicate webhook job count = %d, want 1", duplicateCount)
	}

	type claimResult struct {
		claims []generated.ClaimDueWebhookSyncSourcesRow
		err    error
	}
	startClaims := make(chan struct{})
	claimResults := make(chan claimResult, 2)
	for _, token := range []string{"claim-a", "claim-b"} {
		go func() {
			<-startClaims
			claims, claimErr := queries.ClaimDueWebhookSyncSources(ctx, generated.ClaimDueWebhookSyncSourcesParams{
				ClaimToken: token, LeaseSeconds: 300, BatchSize: 20,
			})
			claimResults <- claimResult{claims: claims, err: claimErr}
		}()
	}
	close(startClaims)
	claimedSources := 0
	for range 2 {
		result := <-claimResults
		if result.err != nil {
			t.Fatalf("claim due webhook source: %v", result.err)
		}
		for _, claim := range result.claims {
			if claim.EntityID == entity.ID && claim.SourceID == source.ID {
				claimedSources++
			}
		}
	}
	if claimedSources != 1 {
		t.Fatalf("concurrent replicas claimed source %d times, want exactly 1", claimedSources)
	}
	if _, err := pool.Exec(ctx, `UPDATE webhook_sync_leases SET lease_expires_at=now()-interval '1 second' WHERE entity_id=$1 AND source_id=$2`, entity.ID, source.ID); err != nil {
		t.Fatalf("expire source lease: %v", err)
	}
	recoveredClaims, err := queries.ClaimDueWebhookSyncSources(ctx, generated.ClaimDueWebhookSyncSourcesParams{
		ClaimToken: "claim-c", LeaseSeconds: 300, BatchSize: 20,
	})
	if err != nil {
		t.Fatalf("reclaim expired webhook source: %v", err)
	}
	if len(recoveredClaims) != 1 || recoveredClaims[0].ClaimToken != "claim-c" {
		t.Fatalf("expired source lease was not recoverable: %#v", recoveredClaims)
	}
	for attempt := int32(1); attempt <= webhookMaxAttempts; attempt++ {
		if _, err := pool.Exec(ctx, `UPDATE sync_jobs SET next_attempt_at=now()-interval '1 second' WHERE entity_id=$1 AND id=$2`, entity.ID, jobID); err != nil {
			t.Fatalf("make webhook job due for attempt %d: %v", attempt, err)
		}
		claimToken := ""
		if attempt == 1 {
			claimToken = "claim-c"
		}
		if _, err := service.RunIncrementalSync(ctx, FullSyncInput{EntityID: entity.ID, SourceID: source.ID, Provider: "feishu", RecoveryClaimToken: claimToken}); !errors.Is(err, providerFailure) {
			t.Fatalf("attempt %d error = %v, want provider failure", attempt, err)
		}

		var status string
		var attempts int32
		var nextAttempt time.Time
		var finishedAt pgtype.Timestamptz
		var errorMessage pgtype.Text
		if err := pool.QueryRow(ctx, `SELECT status, attempt_count, next_attempt_at, finished_at, error_message FROM sync_jobs WHERE entity_id=$1 AND id=$2`, entity.ID, jobID).Scan(&status, &attempts, &nextAttempt, &finishedAt, &errorMessage); err != nil {
			t.Fatalf("read webhook job after attempt %d: %v", attempt, err)
		}
		if attempts != attempt || !errorMessage.Valid {
			t.Fatalf("attempt %d persisted attempts/error = %d/%#v", attempt, attempts, errorMessage)
		}
		if attempt < webhookMaxAttempts {
			if status != "running" || finishedAt.Valid || !nextAttempt.After(time.Now()) {
				t.Fatalf("attempt %d state = %q finished=%#v next=%s, want retryable running job", attempt, status, finishedAt, nextAttempt)
			}
		} else if status != "failed" || !finishedAt.Valid {
			t.Fatalf("terminal attempt state = %q finished=%#v, want failed", status, finishedAt)
		}
	}
}

func TestSuccessfulWebhookConsumptionMarksJobSucceeded(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool := newAtomicSyncTestPool(ctx, t)
	queries := generated.New(pool)

	entity, err := queries.CreateEntity(ctx, generated.CreateEntityParams{Name: "Webhook Success Entity", Slug: "webhook-success", DefaultLocale: "en-US"})
	if err != nil {
		t.Fatalf("create entity: %v", err)
	}
	source, err := queries.CreateIdentitySource(ctx, generated.CreateIdentitySourceParams{EntityID: entity.ID, Type: "feishu", Name: "Feishu", SyncEnabled: true})
	if err != nil {
		t.Fatalf("create identity source: %v", err)
	}
	service, err := NewSyncService(SyncServiceConfig{Queries: queries, Provider: fakeSyncDirectoryProvider{}, TraceID: func() string { return "trace-webhook-success" }, TxStarter: pool})
	if err != nil {
		t.Fatalf("new sync service: %v", err)
	}
	jobID, err := service.SubmitWebhookEvent(ctx, entity.ID, source.ID, DirectorySyncEvent{EventType: "user.updated", ObjectType: "user", ObjectID: "ou_success"})
	if err != nil {
		t.Fatalf("submit webhook event: %v", err)
	}
	var persistedEventID string
	if err := pool.QueryRow(ctx, `SELECT event_id FROM sync_jobs WHERE entity_id=$1 AND id=$2`, entity.ID, jobID).Scan(&persistedEventID); err != nil {
		t.Fatalf("read generated webhook event id: %v", err)
	}
	if persistedEventID == "" {
		t.Fatal("missing provider event id did not receive a durable fallback")
	}
	if _, err := service.RunIncrementalSync(ctx, FullSyncInput{EntityID: entity.ID, SourceID: source.ID, Provider: "feishu"}); err != nil {
		t.Fatalf("run incremental sync: %v", err)
	}
	var status string
	var attempts int32
	var finishedAt pgtype.Timestamptz
	if err := pool.QueryRow(ctx, `SELECT status, attempt_count, finished_at FROM sync_jobs WHERE entity_id=$1 AND id=$2`, entity.ID, jobID).Scan(&status, &attempts, &finishedAt); err != nil {
		t.Fatalf("read successful webhook job: %v", err)
	}
	if status != "succeeded" || attempts != 1 || !finishedAt.Valid {
		t.Fatalf("successful webhook state = %q attempts=%d finished=%#v", status, attempts, finishedAt)
	}
}

func TestRecoveredWebhookReleasesSourceLeaseWhenSyncLockIsBusy(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool := newAtomicSyncTestPool(ctx, t)
	queries := generated.New(pool)

	entity, err := queries.CreateEntity(ctx, generated.CreateEntityParams{Name: "Webhook Lock Entity", Slug: "webhook-lock", DefaultLocale: "en-US"})
	if err != nil {
		t.Fatalf("create entity: %v", err)
	}
	source, err := queries.CreateIdentitySource(ctx, generated.CreateIdentitySourceParams{EntityID: entity.ID, Type: "feishu", Name: "Feishu", SyncEnabled: true})
	if err != nil {
		t.Fatalf("create identity source: %v", err)
	}
	service, err := NewSyncService(SyncServiceConfig{Queries: queries, Provider: fakeSyncDirectoryProvider{}, TraceID: func() string { return "trace-webhook-lock" }, TxStarter: pool})
	if err != nil {
		t.Fatalf("new sync service: %v", err)
	}
	if _, err := service.SubmitWebhookEvent(ctx, entity.ID, source.ID, DirectorySyncEvent{EventType: "user.updated", ObjectType: "user", ObjectID: "ou_lock", EventID: "evt_lock"}); err != nil {
		t.Fatalf("submit webhook event: %v", err)
	}
	const claimToken = "claim-lock"
	claims, err := queries.ClaimDueWebhookSyncSources(ctx, generated.ClaimDueWebhookSyncSourcesParams{ClaimToken: claimToken, LeaseSeconds: 300, BatchSize: 1})
	if err != nil || len(claims) != 1 {
		t.Fatalf("claim webhook source: claims=%d err=%v", len(claims), err)
	}

	releaseLock, err := service.acquireSourceLock(ctx, entity.ID, source.ID)
	if err != nil {
		t.Fatalf("hold source lock: %v", err)
	}
	defer releaseLock()
	_, err = service.RunIncrementalSync(ctx, FullSyncInput{EntityID: entity.ID, SourceID: source.ID, Provider: "feishu", RecoveryClaimToken: claimToken})
	if !errors.Is(err, ErrSyncAlreadyRunning) {
		t.Fatalf("run recovered webhook error = %v, want ErrSyncAlreadyRunning", err)
	}

	var leaseCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM webhook_sync_leases WHERE entity_id=$1 AND source_id=$2`, entity.ID, source.ID).Scan(&leaseCount); err != nil {
		t.Fatalf("count webhook source leases: %v", err)
	}
	if leaseCount != 0 {
		t.Fatalf("webhook source lease count = %d, want 0 after lock conflict", leaseCount)
	}
}
