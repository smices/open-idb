// SPDX-License-Identifier: MIT

package idp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/smices/open-idb/internal/audit"
	"github.com/smices/open-idb/internal/db/generated"
	"github.com/smices/open-idb/internal/id"
)

// auditWriter is the subset of audit.Service used by SyncService.
// Using an interface keeps the sync package decoupled from audit internals.
type auditWriter interface {
	Write(ctx context.Context, event audit.Event) error
}

type SyncService struct {
	queries         *generated.Queries
	provider        DirectoryProvider
	providerFactory func(ctx context.Context, entityID string, sourceID string, provider string) (DirectoryProvider, error)
	audit           auditWriter
	traceID         func() string
}

type SyncServiceConfig struct {
	Queries         *generated.Queries
	Provider        DirectoryProvider
	ProviderFactory func(ctx context.Context, entityID string, sourceID string, provider string) (DirectoryProvider, error)
	Audit           auditWriter
	TraceID         func() string
}

type FullSyncInput struct {
	EntityID string
	SourceID string
	Provider string
	SyncType SyncMode
}

type FullSyncResult struct {
	JobID               string `json:"job_id"`
	DepartmentsUpserted int    `json:"departments_upserted"`
	UsersUpserted       int    `json:"users_upserted"`
	ManagedUsersCreated int    `json:"managed_users_created"`
	ManagedUsersUpdated int    `json:"managed_users_updated"`
	BindingsCreated     int    `json:"bindings_created"`
}

func NewSyncService(cfg SyncServiceConfig) (*SyncService, error) {
	if cfg.Queries == nil {
		return nil, fmt.Errorf("queries are required")
	}
	if cfg.Provider == nil && cfg.ProviderFactory == nil {
		return nil, fmt.Errorf("directory provider is required")
	}
	if cfg.TraceID == nil {
		cfg.TraceID = func() string { return id.NewULID() }
	}
	return &SyncService{
		queries:         cfg.Queries,
		provider:        cfg.Provider,
		providerFactory: cfg.ProviderFactory,
		audit:           cfg.Audit,
		traceID:         cfg.TraceID,
	}, nil
}

func (s *SyncService) RunFullSync(ctx context.Context, input FullSyncInput) (FullSyncResult, error) {
	input.SyncType = SyncModeFull
	return s.runSync(ctx, input)
}

func (s *SyncService) RunIncrementalSync(ctx context.Context, input FullSyncInput) (FullSyncResult, error) {
	input.SyncType = SyncModeIncremental
	return s.runSync(ctx, input)
}

func (s *SyncService) SubmitWebhookEvent(ctx context.Context, entityID string, sourceID string, event DirectorySyncEvent) (string, error) {
	entity, err := ulidValue(entityID)
	if err != nil {
		return "", err
	}
	source, err := ulidValue(sourceID)
	if err != nil {
		return "", err
	}

	sourceRow, err := s.queries.GetIdentitySourceByID(ctx, generated.GetIdentitySourceByIDParams{
		EntityID: entity,
		ID:       source,
	})
	if err != nil {
		return "", err
	}
	if event.EventID == "" {
		event.EventID = id.NewULID()
	}
	trace, err := json.Marshal(event)
	if err != nil {
		return "", err
	}

	job, err := s.queries.CreateSyncJob(ctx, generated.CreateSyncJobParams{
		EntityID: entity,
		SourceID: source,
		Type:     "webhook",
		Provider: sourceRow.Type,
		TraceID:  string(trace),
	})
	if err != nil {
		return "", err
	}
	return ulidString(job.ID), nil
}

func (s *SyncService) ResolveDefaultFeishuWebhookTarget(ctx context.Context) (string, string, error) {
	entities, err := s.queries.ListEntities(ctx, generated.ListEntitiesParams{
		Limit:  1000,
		Offset: 0,
	})
	if err != nil {
		return "", "", err
	}

	var entityID string
	var sourceID string
	for _, entity := range entities {
		if entity.Status != "active" {
			continue
		}
		source, err := s.queries.GetFeishuSourceByEntity(ctx, entity.ID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return "", "", err
		}
		if entityID != "" {
			return "", "", fmt.Errorf("multiple active feishu webhook targets are configured")
		}
		entityID = entity.ID
		sourceID = source.ID
	}
	if entityID == "" || sourceID == "" {
		return "", "", fmt.Errorf("no active feishu identity source is configured")
	}
	return entityID, sourceID, nil
}

func (s *SyncService) runSync(ctx context.Context, input FullSyncInput) (FullSyncResult, error) {
	entityID, err := ulidValue(input.EntityID)
	if err != nil {
		return FullSyncResult{}, err
	}
	sourceID, err := ulidValue(input.SourceID)
	if err != nil {
		return FullSyncResult{}, err
	}
	provider := input.Provider
	if provider == "" {
		source, err := s.queries.GetIdentitySourceByID(ctx, generated.GetIdentitySourceByIDParams{
			EntityID: entityID,
			ID:       sourceID,
		})
		if err != nil {
			return FullSyncResult{}, err
		}
		provider = source.Type
	}

	syncProvider, err := s.resolveProvider(ctx, input.EntityID, input.SourceID, provider)
	if err != nil {
		return FullSyncResult{}, err
	}

	if input.SyncType != SyncModeFull && input.SyncType != SyncModeIncremental {
		input.SyncType = SyncModeFull
	}

	webhookJobs, err := s.pendingWebhookJobs(ctx, entityID, sourceID)
	if err != nil {
		return FullSyncResult{}, err
	}

	job, err := s.queries.CreateSyncJob(ctx, generated.CreateSyncJobParams{
		EntityID: entityID,
		SourceID: sourceID,
		Type:     string(input.SyncType),
		Provider: provider,
		TraceID:  s.traceID(),
	})
	if err != nil {
		return FullSyncResult{}, err
	}
	result := FullSyncResult{JobID: ulidString(job.ID)}
	traceID := job.TraceID

	// Emit sync started event
	s.writeAudit(ctx, audit.Event{
		EntityID:     input.EntityID,
		ActorType:    "sync_job",
		Action:       audit.ActionSyncStarted,
		ResourceType: "sync_job",
		ResourceID:   result.JobID,
		After:        map[string]string{"provider": provider, "trace_id": traceID},
		TraceID:      traceID,
	})

	data, processedWebhookJobs, err := s.loadDataByMode(ctx, syncProvider, entityID, input.SyncType, webhookJobs)
	webhookJobs = processedWebhookJobs
	if err != nil {
		_ = s.finishWebhookJobs(ctx, entityID, webhookJobs, result, err)
		_ = s.failJob(ctx, entityID, job.ID, result, err)
		s.writeAudit(ctx, audit.Event{
			EntityID:     input.EntityID,
			ActorType:    "sync_job",
			Action:       audit.ActionSyncFailed,
			ResourceType: "sync_job",
			ResourceID:   result.JobID,
			After:        map[string]string{"error": err.Error(), "trace_id": traceID},
			TraceID:      traceID,
		})
		return result, err
	}

	for _, department := range data.Departments {
		if _, err := s.queries.UpsertDirectoryDepartment(ctx, generated.UpsertDirectoryDepartmentParams{
			EntityID:                   entityID,
			SourceID:                   sourceID,
			ExternalDepartmentID:       department.ExternalDepartmentID,
			ParentExternalDepartmentID: textValue(department.ParentExternalDepartmentID),
			Name:                       department.Name,
			RawProfile:                 department.RawProfile,
		}); err != nil {
			_ = s.finishWebhookJobs(ctx, entityID, webhookJobs, result, err)
			_ = s.failJob(ctx, entityID, job.ID, result, err)
			return result, err
		}
		result.DepartmentsUpserted++

		// Per-department audit event
		s.writeAudit(ctx, audit.Event{
			EntityID:     input.EntityID,
			ActorType:    "sync_job",
			Action:       audit.ActionSyncDepartmentUpdated,
			ResourceType: "directory_department",
			ResourceID:   department.ExternalDepartmentID,
			After:        department,
			TraceID:      traceID,
		})
	}

	// Step 8: Map directory_departments to organizational departments
	s.mapDepartments(ctx, input, entityID, sourceID, data.Departments, traceID)

	for _, user := range data.Users {
		directoryUser, err := s.queries.UpsertDirectoryUser(ctx, generated.UpsertDirectoryUserParams{
			EntityID:        entityID,
			SourceID:        sourceID,
			ExternalUserID:  user.ExternalUserID,
			ExternalUnionID: textValue(user.ExternalUnionID),
			ExternalOpenID:  textValue(user.ExternalOpenID),
			Name:            user.Name,
			Email:           textValue(user.Email),
			Phone:           textValue(user.Phone),
			AvatarUrl:       textValue(user.AvatarURL),
			Status:          normalizeDirectoryStatus(user.Status),
			RawProfile:      user.RawProfile,
		})
		if err != nil {
			_ = s.finishWebhookJobs(ctx, entityID, webhookJobs, result, err)
			_ = s.failJob(ctx, entityID, job.ID, result, err)
			return result, err
		}
		result.UsersUpserted++

		binding, err := s.queries.GetAccountBindingByProviderUID(ctx, generated.GetAccountBindingByProviderUIDParams{
			EntityID:    entityID,
			SourceID:    sourceID,
			ProviderUid: user.ExternalUserID,
		})
		if err == nil {
			// Existing user — update and check for lifecycle changes
			newLifecycle := lifecycleForDirectoryStatus(user.Status)
			if _, err := s.queries.UpdateManagedUserFromDirectory(ctx, generated.UpdateManagedUserFromDirectoryParams{
				EntityID:        entityID,
				ID:              binding.UserID,
				PrimarySourceID: textValue(sourceID),
				DisplayName:     user.Name,
				Email:           textValue(user.Email),
				Phone:           textValue(user.Phone),
				AvatarUrl:       textValue(user.AvatarURL),
				LifecycleStatus: newLifecycle,
			}); err != nil {
				_ = s.finishWebhookJobs(ctx, entityID, webhookJobs, result, err)
				_ = s.failJob(ctx, entityID, job.ID, result, err)
				return result, err
			}
			result.ManagedUsersUpdated++

			// Emit sync.user.disabled when directory status disables the user
			if newLifecycle == "disabled" {
				s.writeAudit(ctx, audit.Event{
					EntityID:     input.EntityID,
					ActorType:    "sync_job",
					Action:       audit.ActionSyncUserDisabled,
					ResourceType: "user",
					ResourceID:   ulidString(binding.UserID),
					After:        map[string]string{"external_user_id": user.ExternalUserID, "lifecycle_status": newLifecycle},
					TraceID:      traceID,
				})
			}
			continue
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			_ = s.finishWebhookJobs(ctx, entityID, webhookJobs, result, err)
			_ = s.failJob(ctx, entityID, job.ID, result, err)
			return result, err
		}

		managedUser, err := s.queries.CreateManagedUser(ctx, generated.CreateManagedUserParams{
			EntityID:        entityID,
			Username:        usernameForDirectoryUser(user),
			DisplayName:     user.Name,
			Email:           textValue(user.Email),
			Phone:           textValue(user.Phone),
			AvatarUrl:       textValue(user.AvatarURL),
			LifecycleStatus: lifecycleForDirectoryStatus(user.Status),
			UserType:        "employee",
			PrimarySourceID: textValue(sourceID),
			Locale:          pgtype.Text{String: "en-US", Valid: true},
		})
		if err != nil {
			_ = s.finishWebhookJobs(ctx, entityID, webhookJobs, result, err)
			_ = s.failJob(ctx, entityID, job.ID, result, err)
			return result, err
		}
		result.ManagedUsersCreated++

		if _, err := s.queries.CreateAccountBinding(ctx, generated.CreateAccountBindingParams{
			EntityID:        entityID,
			UserID:          managedUser.ID,
			SourceID:        sourceID,
			DirectoryUserID: directoryUser.ID,
			ProviderUid:     user.ExternalUserID,
			ProviderUnionID: textValue(user.ExternalUnionID),
			IsPrimary:       true,
		}); err != nil {
			_ = s.finishWebhookJobs(ctx, entityID, webhookJobs, result, err)
			_ = s.failJob(ctx, entityID, job.ID, result, err)
			return result, err
		}
		result.BindingsCreated++

		// Per-user audit: new user created from directory sync
		s.writeAudit(ctx, audit.Event{
			EntityID:     input.EntityID,
			ActorType:    "sync_job",
			Action:       audit.ActionSyncUserCreated,
			ResourceType: "user",
			ResourceID:   ulidString(managedUser.ID),
			After: map[string]string{
				"external_user_id": user.ExternalUserID,
				"username":         managedUser.Username,
				"lifecycle_status": managedUser.LifecycleStatus,
			},
			TraceID: traceID,
		})
	}

	if _, err := s.queries.FinishSyncJob(ctx, generated.FinishSyncJobParams{
		EntityID: entityID,
		ID:       job.ID,
		Stats:    mustStatsJSON(result),
	}); err != nil {
		_ = s.finishWebhookJobs(ctx, entityID, webhookJobs, result, err)
		return result, err
	}

	_ = s.finishWebhookJobs(ctx, entityID, webhookJobs, result, nil)

	// Emit sync finished event
	s.writeAudit(ctx, audit.Event{
		EntityID:     input.EntityID,
		ActorType:    "sync_job",
		Action:       audit.ActionSyncFinished,
		ResourceType: "sync_job",
		ResourceID:   result.JobID,
		After:        result,
		TraceID:      traceID,
	})

	return result, nil
}

func (s *SyncService) loadDataByMode(ctx context.Context, provider DirectoryProvider, entityID string, mode SyncMode, webhookJobs []generated.SyncJob) (FullSyncData, []generated.SyncJob, error) {
	if mode == SyncModeIncremental {
		type incremental interface {
			IncrementalSync(context.Context, []DirectorySyncEvent) (FullSyncData, error)
		}
		if p, ok := provider.(incremental); ok {
			events := make([]DirectorySyncEvent, 0, len(webhookJobs))
			remainingJobs := make([]generated.SyncJob, 0, len(webhookJobs))
			for _, job := range webhookJobs {
				var event DirectorySyncEvent
				if err := json.Unmarshal([]byte(job.TraceID), &event); err != nil {
					_ = s.failWebhookJob(ctx, entityID, job.ID, FullSyncResult{}, err)
					continue
				}
				events = append(events, event)
				remainingJobs = append(remainingJobs, job)
			}
			if len(events) == 0 {
				return FullSyncData{}, remainingJobs, nil
			}
			data, err := p.IncrementalSync(ctx, events)
			return data, remainingJobs, err
		}
	}
	data, err := provider.FullSync(ctx)
	return data, webhookJobs, err
}

func (s *SyncService) pendingWebhookJobs(ctx context.Context, entityID, sourceID string) ([]generated.SyncJob, error) {
	rows, err := s.queries.ListSyncJobsBySource(ctx, generated.ListSyncJobsBySourceParams{
		EntityID: entityID,
		SourceID: sourceID,
		Limit:    200,
	})
	if err != nil {
		return nil, err
	}
	webhookJobs := make([]generated.SyncJob, 0)
	for _, row := range rows {
		if row.Type != "webhook" {
			continue
		}
		if strings.TrimSpace(row.Status) != "running" {
			continue
		}
		webhookJobs = append(webhookJobs, row)
	}
	return webhookJobs, nil
}

func (s *SyncService) finishWebhookJobs(ctx context.Context, entityID string, webhookJobs []generated.SyncJob, result FullSyncResult, cause error) error {
	if len(webhookJobs) == 0 {
		return nil
	}
	for _, job := range webhookJobs {
		var finishErr error
		if cause == nil {
			finishErr = s.finishWebhookJob(ctx, entityID, job.ID, result)
		} else {
			finishErr = s.failWebhookJob(ctx, entityID, job.ID, result, cause)
		}
		if finishErr != nil {
			// Best-effort for all webhook jobs.
			continue
		}
	}
	return nil
}

func (s *SyncService) finishWebhookJob(ctx context.Context, entityID string, jobID string, result FullSyncResult) error {
	_, err := s.queries.FinishSyncJob(ctx, generated.FinishSyncJobParams{
		EntityID: entityID,
		ID:       jobID,
		Stats:    mustStatsJSON(result),
	})
	return err
}

func (s *SyncService) failWebhookJob(ctx context.Context, entityID string, jobID string, result FullSyncResult, cause error) error {
	_, err := s.queries.FailSyncJob(ctx, generated.FailSyncJobParams{
		EntityID:     entityID,
		ID:           jobID,
		ErrorMessage: pgtype.Text{String: cause.Error(), Valid: true},
		Stats:        mustStatsJSON(result),
	})
	return err
}

func (s *SyncService) resolveProvider(ctx context.Context, entityID string, sourceID string, provider string) (DirectoryProvider, error) {
	if provider == "" {
		return nil, fmt.Errorf("provider is required")
	}
	switch strings.TrimSpace(provider) {
	case "":
		return nil, fmt.Errorf("provider is required")
	case "feishu", "dingtalk", "wecom", "ldap", "local":
		if s.providerFactory != nil {
			return s.providerFactory(ctx, entityID, sourceID, provider)
		}
		if s.provider == nil {
			return nil, fmt.Errorf("directory provider is not configured")
		}
		return s.provider, nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// writeAudit emits an audit event if the audit writer is configured.
// Audit write failures are logged but do not fail the sync operation
// (sync runs are long-running and partial progress should be preserved).
func (s *SyncService) writeAudit(ctx context.Context, event audit.Event) {
	if s.audit == nil {
		return
	}
	// Best-effort: sync audit writes should not block sync progress
	_ = s.audit.Write(ctx, event)
}

func (s *SyncService) failJob(ctx context.Context, entityID string, jobID string, result FullSyncResult, cause error) error {
	_, err := s.queries.FailSyncJob(ctx, generated.FailSyncJobParams{
		EntityID:     entityID,
		ID:           jobID,
		ErrorMessage: pgtype.Text{String: cause.Error(), Valid: true},
		Stats:        mustStatsJSON(result),
	})
	return err
}

// mapDepartments maps directory_departments to organizational departments.
// It finds or creates a default organization for the entity, then upserts
// department records linked to that organization. Parent relationships are
// resolved in two passes: first upsert all departments, then update parent_id.
// Errors are logged but do not fail the sync job — department mapping is
// best-effort and can be retried on the next sync.
func (s *SyncService) mapDepartments(ctx context.Context, input FullSyncInput, entityID, sourceID string, departments []DirectoryDepartment, traceID string) {
	// Find or create a default organization for this entity
	org, err := s.queries.GetFirstOrganization(ctx, entityID)
	if err != nil {
		// No organization exists — create a default one
		org, err = s.queries.CreateOrganization(ctx, generated.CreateOrganizationParams{
			EntityID: entityID,
			Name:     "Default",
		})
		if err != nil {
			return // best-effort, log and continue
		}
	}

	// First pass: upsert all departments (without parent_id)
	deptIDs := make(map[string]string) // external_id → department ULID
	for _, dept := range departments {
		row, err := s.queries.UpsertDepartmentBySource(ctx, generated.UpsertDepartmentBySourceParams{
			EntityID:             entityID,
			OrganizationID:       org.ID,
			Name:                 dept.Name,
			SourceID:             pgtypeULID(sourceID),
			ExternalDepartmentID: pgtypeText(dept.ExternalDepartmentID),
		})
		if err != nil {
			continue // best-effort
		}
		deptIDs[dept.ExternalDepartmentID] = row.ID
	}

	// Second pass: resolve parent relationships
	for _, dept := range departments {
		if dept.ParentExternalDepartmentID == "" {
			continue
		}
		childID, ok := deptIDs[dept.ExternalDepartmentID]
		if !ok {
			continue
		}
		parentID, ok := deptIDs[dept.ParentExternalDepartmentID]
		if !ok {
			continue
		}
		_, _ = s.queries.UpdateDepartment(ctx, generated.UpdateDepartmentParams{
			EntityID: entityID,
			ID:       childID,
			Name:     pgtype.Text{String: dept.Name, Valid: true},
			ParentID: pgtypeULID(parentID),
		})
	}
}

func pgtypeULID(v string) pgtype.Text {
	return pgtype.Text{String: v, Valid: v != ""}
}

func pgtypeText(v string) pgtype.Text {
	return pgtype.Text{String: v, Valid: v != ""}
}

func normalizeDirectoryStatus(status string) string {
	switch status {
	case "active", "disabled", "deleted", "unknown":
		return status
	default:
		return "unknown"
	}
}

func lifecycleForDirectoryStatus(status string) string {
	switch normalizeDirectoryStatus(status) {
	case "active":
		return "active"
	case "disabled", "deleted":
		return "disabled"
	default:
		return "locked"
	}
}

func usernameForDirectoryUser(user DirectoryUser) string {
	if user.Email != "" {
		return user.Email
	}
	return user.ExternalUserID
}

func mustStatsJSON(result FullSyncResult) []byte {
	payload, err := json.Marshal(result)
	if err != nil {
		panic(err)
	}
	return payload
}

func ulidValue(value string) (string, error) {
	if err := id.ValidateULID(value); err != nil {
		return "", err
	}
	return value, nil
}

func ulidString(value string) string {
	return value
}

func textValue(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: value != ""}
}
