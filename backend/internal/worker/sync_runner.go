// SPDX-License-Identifier: MIT

package worker

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/smices/open-idb/internal/audit"
	"github.com/smices/open-idb/internal/id"
	"github.com/smices/open-idb/internal/idp"
	"go.uber.org/zap"
)

// SyncRunner executes sync for a given identity source.  It
// delegates to idp.SyncService for the actual directory import and
// sync_job record management, then emits structured audit events.
type SyncRunner struct {
	syncService      *idp.SyncService
	logger           *zap.Logger
	cacheInvalidator organizationTreeCacheInvalidator
}

// NewSyncRunner creates a SyncRunner backed by the given SyncService.
func NewSyncRunner(syncService *idp.SyncService, logger *zap.Logger) *SyncRunner {
	return &SyncRunner{
		syncService: syncService,
		logger:      logger,
	}
}

type organizationTreeCacheInvalidator interface {
	InvalidateOrganizationTree(ctx context.Context, entityID string) error
}

func (r *SyncRunner) SetOrganizationTreeCacheInvalidator(invalidator organizationTreeCacheInvalidator) {
	r.cacheInvalidator = invalidator
}

// SyncJobRequest identifies a single sync job to execute.
type SyncJobRequest struct {
	EntityID string
	SourceID string
	Provider string // e.g. "feishu"
	SyncType string
}

// Run executes the full sync synchronously.  It:
//  1. Calls SyncService.RunFullSync or RunIncrementalSync (which creates and manages the
//     sync_job row internally).
//  2. Returns audit events for "sync started" and "sync finished" or
//     "sync failed" that the caller should send to the AuditProcessor.
//
// The returned events should be sent to the AuditProcessor by the
// caller (the Scheduler).  The error is the result from RunFullSync/RunIncrementalSync.
func (r *SyncRunner) Run(ctx context.Context, req SyncJobRequest) ([]audit.Event, error) {
	traceID := id.NewULID()
	log := r.logger.With(
		zap.String("entity_id", req.EntityID),
		zap.String("source_id", req.SourceID),
		zap.String("provider", req.Provider),
		zap.String("trace_id", traceID),
	)

	startEvent := audit.Event{
		EntityID:     req.EntityID,
		ActorType:    "sync_job",
		Action:       audit.ActionSyncStarted,
		ResourceType: "identity_source",
		ResourceID:   req.SourceID,
		TraceID:      traceID,
	}

	log.Info("sync job started")

	var (
		input = idp.FullSyncInput{
			EntityID: req.EntityID,
			SourceID: req.SourceID,
			Provider: req.Provider,
		}
		result idp.FullSyncResult
		err    error
	)

	if req.SyncType == string(idp.SyncModeIncremental) {
		result, err = r.syncService.RunIncrementalSync(ctx, input)
	} else {
		result, err = r.syncService.RunFullSync(ctx, input)
	}

	// Check for context cancellation — the caller is shutting down, so
	// we emit a cancelled event rather than a plain failure.
	if ctx.Err() != nil && errors.Is(err, context.Canceled) {
		log.Info("sync job cancelled")
		return []audit.Event{startEvent, {
			EntityID:     req.EntityID,
			ActorType:    "sync_job",
			Action:       audit.ActionSyncFailed,
			ResourceType: "identity_source",
			ResourceID:   req.SourceID,
			TraceID:      traceID,
			After:        map[string]string{"reason": "cancelled", "job_id": result.JobID},
		}}, err
	}

	if err != nil {
		log.Error("sync job failed", zap.Error(err))
		return []audit.Event{startEvent, {
			EntityID:     req.EntityID,
			ActorType:    "sync_job",
			Action:       audit.ActionSyncFailed,
			ResourceType: "identity_source",
			ResourceID:   req.SourceID,
			TraceID:      traceID,
			After:        map[string]string{"error": err.Error(), "job_id": result.JobID},
		}}, err
	}

	statsJSON, _ := json.Marshal(result)
	log.Info("sync job succeeded",
		zap.Int("departments_upserted", result.DepartmentsUpserted),
		zap.Int("users_upserted", result.UsersUpserted),
		zap.Int("managed_users_created", result.ManagedUsersCreated),
		zap.Int("managed_users_updated", result.ManagedUsersUpdated),
		zap.Int("bindings_created", result.BindingsCreated),
	)
	if r.cacheInvalidator != nil {
		if cacheErr := r.cacheInvalidator.InvalidateOrganizationTree(ctx, req.EntityID); cacheErr != nil {
			log.Warn("organization tree cache invalidation failed", zap.Error(cacheErr))
		}
	}

	return []audit.Event{startEvent, {
		EntityID:     req.EntityID,
		ActorType:    "sync_job",
		Action:       audit.ActionSyncFinished,
		ResourceType: "identity_source",
		ResourceID:   req.SourceID,
		TraceID:      traceID,
		After:        json.RawMessage(statsJSON),
	}}, nil
}
