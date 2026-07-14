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
	txStarter       txStarter
}

type SyncServiceConfig struct {
	Queries         *generated.Queries
	Provider        DirectoryProvider
	ProviderFactory func(ctx context.Context, entityID string, sourceID string, provider string) (DirectoryProvider, error)
	Audit           auditWriter
	TraceID         func() string
	TxStarter       txStarter
}

type txStarter interface {
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

// ErrSyncAlreadyRunning prevents two reconciliation passes for the same
// entity/source boundary from observing different upstream snapshots.
var ErrSyncAlreadyRunning = errors.New("sync already running for this identity source")

type FullSyncInput struct {
	EntityID string
	SourceID string
	Provider string
	SyncType SyncMode
}

type FullSyncResult struct {
	JobID                 string `json:"job_id"`
	DepartmentsUpserted   int    `json:"departments_upserted"`
	UsersUpserted         int    `json:"users_upserted"`
	ManagedUsersCreated   int    `json:"managed_users_created"`
	ManagedUsersUpdated   int    `json:"managed_users_updated"`
	DirectoryUsersDeleted int    `json:"directory_users_deleted"`
	ManagedUsersDeleted   int    `json:"managed_users_deleted"`
	BindingsCreated       int    `json:"bindings_created"`
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
		txStarter:       cfg.TxStarter,
	}, nil
}

func (s *SyncService) SetTxStarter(starter txStarter) {
	s.txStarter = starter
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
	releaseLock, err := s.acquireSourceLock(ctx, entityID, sourceID)
	if err != nil {
		return FullSyncResult{}, err
	}
	defer releaseLock()
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

	users := uniqueDirectoryUsers(data.Users)
	presentExternalUserIDs := make([]string, 0, len(users))
	for _, user := range users {
		normalizedStatus := normalizeDirectoryStatus(user.Status)
		presentExternalUserIDs = append(presentExternalUserIDs, user.ExternalUserID)
		directoryUser, err := s.queries.UpsertDirectoryUser(ctx, generated.UpsertDirectoryUserParams{
			EntityID:        entityID,
			SourceID:        sourceID,
			ExternalUserID:  user.ExternalUserID,
			ExternalUnionID: textValue(user.ExternalUnionID),
			ExternalOpenID:  textValue(user.ExternalOpenID),
			Name:            user.Name,
			EnglishName:     user.EnglishName,
			EmployeeNo:      user.EmployeeNo,
			JobTitle:        user.JobTitle,
			Email:           textValue(user.Email),
			Phone:           textValue(user.Phone),
			AvatarUrl:       textValue(user.AvatarURL),
			Status:          normalizedStatus,
			RawProfile:      user.RawProfile,
		})
		if err != nil {
			_ = s.finishWebhookJobs(ctx, entityID, webhookJobs, result, err)
			_ = s.failJob(ctx, entityID, job.ID, result, err)
			return result, err
		}
		result.UsersUpserted++
		if normalizedStatus == "deleted" {
			continue
		}

		binding, hasBinding, err := s.resolveAccountBinding(ctx, entityID, sourceID, directoryUser.ID, user)
		if err != nil {
			_ = s.finishWebhookJobs(ctx, entityID, webhookJobs, result, err)
			_ = s.failJob(ctx, entityID, job.ID, result, err)
			return result, err
		}
		if hasBinding {
			// Existing user — update and check for lifecycle changes
			newLifecycle := lifecycleForDirectoryStatus(user.Status)
			if _, err := s.updateManagedUserFromDirectory(ctx, entityID, sourceID, binding.UserID, user, newLifecycle); err != nil {
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

		mergedUser, err := s.queries.GetManagedUserByUsername(ctx, generated.GetManagedUserByUsernameParams{
			EntityID: entityID,
			Username: usernameForDirectoryUser(user),
		})
		if err == nil {
			newLifecycle := lifecycleForDirectoryStatus(user.Status)
			if _, err := s.updateManagedUserFromDirectory(ctx, entityID, sourceID, mergedUser.ID, user, newLifecycle); err != nil {
				_ = s.finishWebhookJobs(ctx, entityID, webhookJobs, result, err)
				_ = s.failJob(ctx, entityID, job.ID, result, err)
				return result, err
			}
			if err := s.queries.AssignRoleToUserByCode(ctx, generated.AssignRoleToUserByCodeParams{
				EntityID: entityID,
				UserID:   mergedUser.ID,
				Code:     "employee",
			}); err != nil {
				_ = s.finishWebhookJobs(ctx, entityID, webhookJobs, result, err)
				_ = s.failJob(ctx, entityID, job.ID, result, err)
				return result, err
			}
			result.ManagedUsersUpdated++
			if _, err := s.queries.CreateAccountBinding(ctx, generated.CreateAccountBindingParams{
				EntityID:        entityID,
				UserID:          mergedUser.ID,
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
			EnglishName:     user.EnglishName,
			EmployeeNo:      user.EmployeeNo,
			JobTitle:        user.JobTitle,
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
		if err := s.queries.AssignRoleToUserByCode(ctx, generated.AssignRoleToUserByCodeParams{
			EntityID: entityID,
			UserID:   managedUser.ID,
			Code:     "employee",
		}); err != nil {
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

	if input.SyncType == SyncModeFull {
		archivedUsers, deletedDirectoryUsers, err := s.reconcileFullSync(ctx, entityID, sourceID, presentExternalUserIDs)
		if err != nil {
			_ = s.finishWebhookJobs(ctx, entityID, webhookJobs, result, err)
			_ = s.failJob(ctx, entityID, job.ID, result, err)
			return result, err
		}
		result.DirectoryUsersDeleted = deletedDirectoryUsers
		result.ManagedUsersDeleted = len(archivedUsers)
		for _, archivedUser := range archivedUsers {
			s.writeAudit(ctx, audit.Event{
				EntityID:     input.EntityID,
				ActorType:    "sync_job",
				Action:       audit.ActionSyncUserArchived,
				ResourceType: "archived_user",
				ResourceID:   ulidString(archivedUser.Archive.ID),
				After:        map[string]string{"username": archivedUser.Username, "archive_reason": "directory full sync removed user"},
				TraceID:      traceID,
			})
		}
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

func (s *SyncService) resolveAccountBinding(ctx context.Context, entityID, sourceID, directoryUserID string, user DirectoryUser) (generated.AccountBinding, bool, error) {
	binding, err := s.queries.GetAccountBindingByProviderUID(ctx, generated.GetAccountBindingByProviderUIDParams{
		EntityID:    entityID,
		SourceID:    sourceID,
		ProviderUid: user.ExternalUserID,
	})
	if err == nil {
		return s.ensureAccountBindingMatchesDirectory(ctx, entityID, sourceID, directoryUserID, binding, user)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return generated.AccountBinding{}, false, err
	}

	if strings.TrimSpace(user.ExternalUnionID) == "" {
		return generated.AccountBinding{}, false, nil
	}
	binding, err = s.queries.GetAccountBindingByProviderUnionID(ctx, generated.GetAccountBindingByProviderUnionIDParams{
		EntityID:        entityID,
		SourceID:        sourceID,
		ProviderUnionID: textValue(user.ExternalUnionID),
	})
	if err == nil {
		return s.ensureAccountBindingMatchesDirectory(ctx, entityID, sourceID, directoryUserID, binding, user)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return generated.AccountBinding{}, false, nil
	}
	return generated.AccountBinding{}, false, err
}

func (s *SyncService) ensureAccountBindingMatchesDirectory(ctx context.Context, entityID, sourceID, directoryUserID string, binding generated.AccountBinding, user DirectoryUser) (generated.AccountBinding, bool, error) {
	if binding.DirectoryUserID == directoryUserID &&
		binding.ProviderUid == user.ExternalUserID &&
		textMatches(binding.ProviderUnionID, user.ExternalUnionID) {
		return binding, true, nil
	}
	updated, err := s.queries.UpdateAccountBindingFromDirectory(ctx, generated.UpdateAccountBindingFromDirectoryParams{
		EntityID:        entityID,
		SourceID:        sourceID,
		ID:              binding.ID,
		DirectoryUserID: directoryUserID,
		ProviderUid:     user.ExternalUserID,
		ProviderUnionID: textValue(user.ExternalUnionID),
	})
	if err != nil {
		return generated.AccountBinding{}, false, err
	}
	return updated, true, nil
}

func (s *SyncService) updateManagedUserFromDirectory(ctx context.Context, entityID, sourceID, userID string, user DirectoryUser, lifecycle string) (generated.User, error) {
	return s.queries.UpdateManagedUserFromDirectory(ctx, generated.UpdateManagedUserFromDirectoryParams{
		EntityID:        entityID,
		ID:              userID,
		PrimarySourceID: textValue(sourceID),
		Username:        textValue(usernameForDirectoryUser(user)),
		DisplayName:     user.Name,
		EnglishName:     user.EnglishName,
		EmployeeNo:      user.EmployeeNo,
		JobTitle:        user.JobTitle,
		Email:           textValue(user.Email),
		Phone:           textValue(user.Phone),
		AvatarUrl:       textValue(user.AvatarURL),
		LifecycleStatus: lifecycle,
	})
}

func (s *SyncService) archiveManagedUser(ctx context.Context, entityID, userID, reason string) (generated.ArchivedUser, error) {
	if s.txStarter == nil {
		return generated.ArchivedUser{}, fmt.Errorf("sync service transaction starter is not configured")
	}
	tx, err := s.txStarter.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return generated.ArchivedUser{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	txQueries := s.queries.WithTx(tx)
	archive, err := txQueries.ArchiveUser(ctx, generated.ArchiveUserParams{
		EntityID:         entityID,
		UserID:           userID,
		ArchivedByUserID: pgtype.Text{},
		ArchiveReason:    reason,
	})
	if err != nil {
		return generated.ArchivedUser{}, err
	}
	if err := txQueries.DeleteUserActiveDependents(ctx, generated.DeleteUserActiveDependentsParams{
		EntityID: entityID,
		UserID:   userID,
	}); err != nil {
		return generated.ArchivedUser{}, err
	}
	if err := txQueries.DeleteUserActiveRow(ctx, generated.DeleteUserActiveRowParams{
		EntityID: entityID,
		UserID:   userID,
	}); err != nil {
		return generated.ArchivedUser{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return generated.ArchivedUser{}, err
	}
	committed = true
	return archive, nil
}

type archivedSyncUser struct {
	Username string
	Archive  generated.ArchivedUser
}

// reconcileFullSync performs the destructive tail only after the upstream
// snapshot has been loaded and all regular upserts have succeeded. Its single
// transaction prevents a failed archive from leaving directory records marked
// deleted while their managed accounts remain active.
func (s *SyncService) reconcileFullSync(ctx context.Context, entityID, sourceID string, presentExternalUserIDs []string) ([]archivedSyncUser, int, error) {
	if s.txStarter == nil {
		return nil, 0, fmt.Errorf("sync service transaction starter is not configured")
	}
	tx, err := s.txStarter.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, 0, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(context.Background())
		}
	}()
	queries := s.queries.WithTx(tx)
	deletedDirectoryUsers, err := queries.MarkMissingDirectoryUsersDeleted(ctx, generated.MarkMissingDirectoryUsersDeletedParams{
		EntityID:               entityID,
		SourceID:               sourceID,
		PresentExternalUserIds: presentExternalUserIDs,
	})
	if err != nil {
		return nil, 0, err
	}
	deletedManagedUsers, err := queries.ListManagedUsersForDeletedDirectoryUsers(ctx, generated.ListManagedUsersForDeletedDirectoryUsersParams{EntityID: entityID, SourceID: sourceID})
	if err != nil {
		return nil, 0, err
	}
	archivedUsers := make([]archivedSyncUser, 0, len(deletedManagedUsers))
	for _, user := range deletedManagedUsers {
		archive, err := queries.ArchiveUser(ctx, generated.ArchiveUserParams{
			EntityID: entityID, UserID: user.ID, ArchivedByUserID: pgtype.Text{}, ArchiveReason: "directory full sync removed user",
		})
		if err != nil {
			return nil, 0, err
		}
		if err := queries.DeleteUserActiveDependents(ctx, generated.DeleteUserActiveDependentsParams{EntityID: entityID, UserID: user.ID}); err != nil {
			return nil, 0, err
		}
		if err := queries.DeleteUserActiveRow(ctx, generated.DeleteUserActiveRowParams{EntityID: entityID, UserID: user.ID}); err != nil {
			return nil, 0, err
		}
		archivedUsers = append(archivedUsers, archivedSyncUser{Username: user.Username, Archive: archive})
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, 0, err
	}
	committed = true
	return archivedUsers, len(deletedDirectoryUsers), nil
}

// acquireSourceLock serializes sync runs for one identity source across all
// backend replicas. The lock is tied to a short, otherwise idle transaction;
// PostgreSQL releases it automatically if the process crashes.
func (s *SyncService) acquireSourceLock(ctx context.Context, entityID, sourceID string) (func(), error) {
	if s.txStarter == nil {
		// Preserve lightweight unit callers. The production app always sets the
		// PostgreSQL transaction starter before accepting sync requests.
		return func() {}, nil
	}
	tx, err := s.txStarter.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	var acquired bool
	err = tx.QueryRow(ctx, "SELECT pg_try_advisory_xact_lock(hashtextextended($1, 0), hashtextextended($2, 0))", entityID, sourceID).Scan(&acquired)
	if err != nil {
		_ = tx.Rollback(context.Background())
		return nil, err
	}
	if !acquired {
		_ = tx.Rollback(context.Background())
		return nil, ErrSyncAlreadyRunning
	}
	return func() {
		_ = tx.Commit(context.Background())
	}, nil
}

func uniqueDirectoryUsers(users []DirectoryUser) []DirectoryUser {
	result := make([]DirectoryUser, 0, len(users))
	seen := make(map[string]int, len(users))
	for _, user := range users {
		key := strings.TrimSpace(user.ExternalUserID)
		if key == "" {
			continue
		}
		user.ExternalUserID = key
		if index, ok := seen[key]; ok {
			result[index] = mergeDirectoryUser(result[index], user)
			continue
		}
		seen[key] = len(result)
		result = append(result, user)
	}
	return result
}

func mergeDirectoryUser(existing, next DirectoryUser) DirectoryUser {
	if existing.ExternalUnionID == "" {
		existing.ExternalUnionID = next.ExternalUnionID
	}
	if existing.ExternalOpenID == "" {
		existing.ExternalOpenID = next.ExternalOpenID
	}
	if existing.Name == "" {
		existing.Name = next.Name
	}
	if existing.EnglishName == "" {
		existing.EnglishName = next.EnglishName
	}
	if existing.EmployeeNo == "" {
		existing.EmployeeNo = next.EmployeeNo
	}
	if existing.JobTitle == "" {
		existing.JobTitle = next.JobTitle
	}
	if existing.Email == "" {
		existing.Email = next.Email
	}
	if existing.Phone == "" {
		existing.Phone = next.Phone
	}
	if existing.AvatarURL == "" {
		existing.AvatarURL = next.AvatarURL
	}
	if existing.Status == "" || existing.Status == "unknown" {
		existing.Status = next.Status
	}
	if len(existing.RawProfile) == 0 || string(existing.RawProfile) == "{}" {
		existing.RawProfile = next.RawProfile
	}
	return existing
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
	// Find or create a company root for this entity.
	org, err := s.queries.GetFirstOrganization(ctx, entityID)
	if err != nil {
		name := "Company"
		if entity, entityErr := s.queries.GetEntityByID(ctx, entityID); entityErr == nil {
			name = strings.TrimSpace(entity.BrandName)
			if name == "" {
				name = strings.TrimSpace(entity.Name)
			}
		}
		if name == "" {
			name = "Company"
		}
		org, err = s.queries.CreateOrganization(ctx, generated.CreateOrganizationParams{
			EntityID: entityID,
			Name:     name,
		})
		if err != nil {
			return // best-effort, log and continue
		}
	}

	// First pass: upsert all departments and index every known provider ID.
	deptIDs := make(map[string]string) // provider department/open_department ID → department ULID
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
		for _, key := range departmentLookupKeys(dept) {
			deptIDs[key] = row.ID
		}
	}

	// Second pass: resolve parent relationships
	for _, dept := range departments {
		parentExternalID := strings.TrimSpace(dept.ParentExternalDepartmentID)
		if parentExternalID == "" || parentExternalID == "0" {
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

func departmentLookupKeys(dept DirectoryDepartment) []string {
	keys := make([]string, 0, 3)
	appendKey := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" || value == "0" {
			return
		}
		for _, existing := range keys {
			if existing == value {
				return
			}
		}
		keys = append(keys, value)
	}
	appendKey(dept.ExternalDepartmentID)
	var raw struct {
		DepartmentID     string `json:"department_id"`
		OpenDepartmentID string `json:"open_department_id"`
	}
	if len(dept.RawProfile) > 0 && json.Unmarshal(dept.RawProfile, &raw) == nil {
		appendKey(raw.DepartmentID)
		appendKey(raw.OpenDepartmentID)
	}
	return keys
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

func textMatches(value pgtype.Text, expected string) bool {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return !value.Valid || strings.TrimSpace(value.String) == ""
	}
	return value.Valid && strings.TrimSpace(value.String) == expected
}
