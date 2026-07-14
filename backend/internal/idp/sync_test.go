// SPDX-License-Identifier: MIT

package idp

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
	"github.com/smices/open-idb/internal/audit"
	"github.com/smices/open-idb/internal/db/generated"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestLifecycleForDirectoryStatus(t *testing.T) {
	tests := map[string]string{
		"active":   "active",
		"disabled": "disabled",
		"deleted":  "disabled",
		"unknown":  "locked",
		"other":    "locked",
	}
	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			if got := lifecycleForDirectoryStatus(input); got != want {
				t.Fatalf("lifecycleForDirectoryStatus(%q) = %q, want %q", input, got, want)
			}
		})
	}
}

func TestUsernameForDirectoryUser(t *testing.T) {
	if got := usernameForDirectoryUser(DirectoryUser{Email: "ada@example.test", ExternalUserID: "ou_1"}); got != "ada@example.test" {
		t.Fatalf("username = %q", got)
	}
	if got := usernameForDirectoryUser(DirectoryUser{ExternalUserID: "ou_1"}); got != "ou_1" {
		t.Fatalf("username = %q", got)
	}
}

func TestAcquireSourceLockRejectsConcurrentSyncForSameSource(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool := newSyncTestPool(ctx, t)
	queries := generated.New(pool)

	first, err := NewSyncService(SyncServiceConfig{Queries: queries, Provider: fakeSyncDirectoryProvider{}, TxStarter: pool})
	if err != nil {
		t.Fatalf("new first sync service: %v", err)
	}
	second, err := NewSyncService(SyncServiceConfig{Queries: queries, Provider: fakeSyncDirectoryProvider{}, TxStarter: pool})
	if err != nil {
		t.Fatalf("new second sync service: %v", err)
	}

	release, err := first.acquireSourceLock(ctx, "01HZZZZZZZ0000000000000001", "01HZZZZZZZ0000000000000002")
	if err != nil {
		t.Fatalf("acquire first lock: %v", err)
	}
	defer release()

	if _, err := second.acquireSourceLock(ctx, "01HZZZZZZZ0000000000000001", "01HZZZZZZZ0000000000000002"); !errors.Is(err, ErrSyncAlreadyRunning) {
		t.Fatalf("second lock error = %v, want ErrSyncAlreadyRunning", err)
	}
}

func TestFullSyncArchivesMissingManagedUsers(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool := newSyncTestPool(ctx, t)
	queries := generated.New(pool)

	entity, err := queries.CreateEntity(ctx, generated.CreateEntityParams{
		Name:          "Sync Archive Entity",
		Slug:          "sync-archive",
		DefaultLocale: "en-US",
	})
	if err != nil {
		t.Fatalf("create entity: %v", err)
	}
	source, err := queries.CreateIdentitySource(ctx, generated.CreateIdentitySourceParams{
		EntityID:    entity.ID,
		Type:        "feishu",
		Name:        "Feishu",
		SyncEnabled: true,
	})
	if err != nil {
		t.Fatalf("create identity source: %v", err)
	}
	directoryUser, err := queries.UpsertDirectoryUser(ctx, generated.UpsertDirectoryUserParams{
		EntityID:       entity.ID,
		SourceID:       source.ID,
		ExternalUserID: "ou_archive_1",
		Name:           "Archived User",
		Status:         "active",
		RawProfile:     []byte(`{"name":"Archived User"}`),
	})
	if err != nil {
		t.Fatalf("create directory user: %v", err)
	}
	managedUser, err := queries.CreateManagedUser(ctx, generated.CreateManagedUserParams{
		EntityID:        entity.ID,
		Username:        "archived.user@example.test",
		DisplayName:     "Archived User",
		Email:           pgtype.Text{String: "archived.user@example.test", Valid: true},
		LifecycleStatus: "active",
		UserType:        "employee",
		PrimarySourceID: pgtype.Text{String: source.ID, Valid: true},
		Locale:          pgtype.Text{String: "en-US", Valid: true},
	})
	if err != nil {
		t.Fatalf("create managed user: %v", err)
	}
	if _, err := queries.CreateAccountBinding(ctx, generated.CreateAccountBindingParams{
		EntityID:        entity.ID,
		UserID:          managedUser.ID,
		SourceID:        source.ID,
		DirectoryUserID: directoryUser.ID,
		ProviderUid:     "ou_archive_1",
		IsPrimary:       true,
	}); err != nil {
		t.Fatalf("create account binding: %v", err)
	}
	if _, err := queries.UpsertDirectoryDepartment(ctx, generated.UpsertDirectoryDepartmentParams{EntityID: entity.ID, SourceID: source.ID, ExternalDepartmentID: "od_removed", Name: "Removed Department", RawProfile: []byte(`{}`)}); err != nil {
		t.Fatalf("create directory department: %v", err)
	}
	org, err := queries.CreateOrganization(ctx, generated.CreateOrganizationParams{EntityID: entity.ID, Name: "Company"})
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	if _, err := queries.UpsertDepartmentBySource(ctx, generated.UpsertDepartmentBySourceParams{EntityID: entity.ID, OrganizationID: org.ID, Name: "Removed Department", SourceID: pgtype.Text{String: source.ID, Valid: true}, ExternalDepartmentID: pgtype.Text{String: "od_removed", Valid: true}}); err != nil {
		t.Fatalf("create mapped department: %v", err)
	}

	service, err := NewSyncService(SyncServiceConfig{
		Queries:   queries,
		Provider:  fakeSyncDirectoryProvider{data: FullSyncData{}},
		Audit:     &capturingAuditWriter{},
		TraceID:   func() string { return "trace-sync-archive" },
		TxStarter: pool,
	})
	if err != nil {
		t.Fatalf("new sync service: %v", err)
	}
	auditWriter := service.audit.(*capturingAuditWriter)

	result, err := service.RunFullSync(ctx, FullSyncInput{
		EntityID: entity.ID,
		SourceID: source.ID,
		Provider: "feishu",
	})
	if err != nil {
		t.Fatalf("run full sync: %v", err)
	}

	if result.ManagedUsersDeleted != 1 {
		t.Fatalf("ManagedUsersDeleted = %d, want 1", result.ManagedUsersDeleted)
	}
	if result.DirectoryUsersDeleted != 1 {
		t.Fatalf("DirectoryUsersDeleted = %d, want 1", result.DirectoryUsersDeleted)
	}
	if result.DepartmentsDeleted != 1 {
		t.Fatalf("DepartmentsDeleted = %d, want 1", result.DepartmentsDeleted)
	}
	var directoryDepartmentCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM directory_departments WHERE entity_id = $1 AND source_id = $2 AND external_department_id = 'od_removed'`, entity.ID, source.ID).Scan(&directoryDepartmentCount); err != nil {
		t.Fatalf("count directory departments: %v", err)
	}
	if directoryDepartmentCount != 0 {
		t.Fatalf("directory department count = %d, want 0", directoryDepartmentCount)
	}

	_, err = queries.GetUserByID(ctx, generated.GetUserByIDParams{
		EntityID: entity.ID,
		ID:       managedUser.ID,
	})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected active user row to be deleted, got %v", err)
	}

	archive, err := queries.GetArchivedUserByOriginalID(ctx, generated.GetArchivedUserByOriginalIDParams{
		EntityID:       entity.ID,
		OriginalUserID: managedUser.ID,
	})
	if err != nil {
		t.Fatalf("get archived user: %v", err)
	}
	if archive.Username != managedUser.Username {
		t.Fatalf("archived username = %q, want %q", archive.Username, managedUser.Username)
	}

	foundArchivedAction := false
	for _, event := range auditWriter.events {
		if event.ResourceType != "archived_user" {
			continue
		}
		if event.Action == audit.ActionSyncUserDisabled {
			t.Fatalf("archived_user audit action = %q, do not want disabled action", event.Action)
		}
		if event.Action == audit.ActionSyncUserArchived {
			foundArchivedAction = true
		}
	}
	if !foundArchivedAction {
		t.Fatal("expected archived_user audit event with sync.user.archived action")
	}
}

func TestFullSyncProviderReturnedDeletedUserArchivesWithoutDisabledAudit(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool := newSyncTestPool(ctx, t)
	queries := generated.New(pool)

	entity, err := queries.CreateEntity(ctx, generated.CreateEntityParams{
		Name:          "Sync Provider Deleted Entity",
		Slug:          "sync-provider-deleted",
		DefaultLocale: "en-US",
	})
	if err != nil {
		t.Fatalf("create entity: %v", err)
	}
	source, err := queries.CreateIdentitySource(ctx, generated.CreateIdentitySourceParams{
		EntityID:    entity.ID,
		Type:        "feishu",
		Name:        "Feishu",
		SyncEnabled: true,
	})
	if err != nil {
		t.Fatalf("create identity source: %v", err)
	}
	directoryUser, err := queries.UpsertDirectoryUser(ctx, generated.UpsertDirectoryUserParams{
		EntityID:       entity.ID,
		SourceID:       source.ID,
		ExternalUserID: "ou_deleted_1",
		Name:           "Deleted User",
		Status:         "active",
		RawProfile:     []byte(`{"name":"Deleted User"}`),
	})
	if err != nil {
		t.Fatalf("create directory user: %v", err)
	}
	managedUser, err := queries.CreateManagedUser(ctx, generated.CreateManagedUserParams{
		EntityID:        entity.ID,
		Username:        "deleted.user@example.test",
		DisplayName:     "Deleted User",
		Email:           pgtype.Text{String: "deleted.user@example.test", Valid: true},
		LifecycleStatus: "active",
		UserType:        "employee",
		PrimarySourceID: pgtype.Text{String: source.ID, Valid: true},
		Locale:          pgtype.Text{String: "en-US", Valid: true},
	})
	if err != nil {
		t.Fatalf("create managed user: %v", err)
	}
	if _, err := queries.CreateAccountBinding(ctx, generated.CreateAccountBindingParams{
		EntityID:        entity.ID,
		UserID:          managedUser.ID,
		SourceID:        source.ID,
		DirectoryUserID: directoryUser.ID,
		ProviderUid:     "ou_deleted_1",
		IsPrimary:       true,
	}); err != nil {
		t.Fatalf("create account binding: %v", err)
	}

	service, err := NewSyncService(SyncServiceConfig{
		Queries: queries,
		Provider: fakeSyncDirectoryProvider{data: FullSyncData{
			Users: []DirectoryUser{{
				ExternalUserID: "ou_deleted_1",
				Name:           "Deleted User",
				Email:          "deleted.user@example.test",
				Status:         "deleted",
				RawProfile:     []byte(`{"name":"Deleted User","status":"deleted"}`),
			}},
		}},
		Audit:     &capturingAuditWriter{},
		TraceID:   func() string { return "trace-sync-provider-deleted" },
		TxStarter: pool,
	})
	if err != nil {
		t.Fatalf("new sync service: %v", err)
	}
	auditWriter := service.audit.(*capturingAuditWriter)

	result, err := service.RunFullSync(ctx, FullSyncInput{
		EntityID: entity.ID,
		SourceID: source.ID,
		Provider: "feishu",
	})
	if err != nil {
		t.Fatalf("run full sync: %v", err)
	}

	if result.ManagedUsersDeleted != 1 {
		t.Fatalf("ManagedUsersDeleted = %d, want 1", result.ManagedUsersDeleted)
	}

	_, err = queries.GetUserByID(ctx, generated.GetUserByIDParams{
		EntityID: entity.ID,
		ID:       managedUser.ID,
	})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected active user row to be deleted, got %v", err)
	}

	archive, err := queries.GetArchivedUserByOriginalID(ctx, generated.GetArchivedUserByOriginalIDParams{
		EntityID:       entity.ID,
		OriginalUserID: managedUser.ID,
	})
	if err != nil {
		t.Fatalf("get archived user: %v", err)
	}
	if archive.Username != managedUser.Username {
		t.Fatalf("archived username = %q, want %q", archive.Username, managedUser.Username)
	}

	foundArchivedAction := false
	for _, event := range auditWriter.events {
		if event.Action == audit.ActionSyncUserDisabled {
			t.Fatalf("unexpected disabled audit action for provider-returned deleted user: %+v", event)
		}
		if event.ResourceType == "archived_user" && event.Action == audit.ActionSyncUserArchived {
			foundArchivedAction = true
		}
	}
	if !foundArchivedAction {
		t.Fatal("expected archived_user audit event with sync.user.archived action")
	}
}

func TestFullSyncProviderReturnedDeletedUserWithoutBindingDoesNotMergeByUsername(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool := newSyncTestPool(ctx, t)
	queries := generated.New(pool)

	entity, err := queries.CreateEntity(ctx, generated.CreateEntityParams{
		Name:          "Sync Deleted Merge Guard Entity",
		Slug:          "sync-deleted-merge-guard",
		DefaultLocale: "en-US",
	})
	if err != nil {
		t.Fatalf("create entity: %v", err)
	}
	source, err := queries.CreateIdentitySource(ctx, generated.CreateIdentitySourceParams{
		EntityID:    entity.ID,
		Type:        "feishu",
		Name:        "Feishu",
		SyncEnabled: true,
	})
	if err != nil {
		t.Fatalf("create identity source: %v", err)
	}
	managedUser, err := queries.CreateManagedUser(ctx, generated.CreateManagedUserParams{
		EntityID:        entity.ID,
		Username:        "preserve.user@example.test",
		DisplayName:     "Preserve Me",
		Email:           pgtype.Text{String: "preserve.user@example.test", Valid: true},
		LifecycleStatus: "active",
		UserType:        "employee",
		PrimarySourceID: pgtype.Text{String: source.ID, Valid: true},
		Locale:          pgtype.Text{String: "en-US", Valid: true},
	})
	if err != nil {
		t.Fatalf("create managed user: %v", err)
	}

	service, err := NewSyncService(SyncServiceConfig{
		Queries: queries,
		Provider: fakeSyncDirectoryProvider{data: FullSyncData{
			Users: []DirectoryUser{{
				ExternalUserID: "ou_deleted_unbound_match",
				Name:           "Directory Deleted Name",
				Email:          managedUser.Username,
				Status:         "deleted",
				RawProfile:     []byte(`{"name":"Directory Deleted Name","status":"deleted"}`),
			}},
		}},
		Audit:     &capturingAuditWriter{},
		TraceID:   func() string { return "trace-sync-deleted-merge-guard" },
		TxStarter: pool,
	})
	if err != nil {
		t.Fatalf("new sync service: %v", err)
	}

	result, err := service.RunFullSync(ctx, FullSyncInput{
		EntityID: entity.ID,
		SourceID: source.ID,
		Provider: "feishu",
	})
	if err != nil {
		t.Fatalf("run full sync: %v", err)
	}

	if result.ManagedUsersUpdated != 0 {
		t.Fatalf("ManagedUsersUpdated = %d, want 0", result.ManagedUsersUpdated)
	}
	if result.ManagedUsersCreated != 0 {
		t.Fatalf("ManagedUsersCreated = %d, want 0", result.ManagedUsersCreated)
	}
	if result.ManagedUsersDeleted != 0 {
		t.Fatalf("ManagedUsersDeleted = %d, want 0", result.ManagedUsersDeleted)
	}
	if result.BindingsCreated != 0 {
		t.Fatalf("BindingsCreated = %d, want 0", result.BindingsCreated)
	}

	gotUser, err := queries.GetUserByID(ctx, generated.GetUserByIDParams{
		EntityID: entity.ID,
		ID:       managedUser.ID,
	})
	if err != nil {
		t.Fatalf("get managed user: %v", err)
	}
	if gotUser.DisplayName != managedUser.DisplayName {
		t.Fatalf("display_name = %q, want %q", gotUser.DisplayName, managedUser.DisplayName)
	}
	if gotUser.LifecycleStatus != "active" {
		t.Fatalf("lifecycle_status = %q, want active", gotUser.LifecycleStatus)
	}

	bindings, err := queries.ListAccountBindingsByUser(ctx, generated.ListAccountBindingsByUserParams{
		EntityID: entity.ID,
		UserID:   managedUser.ID,
	})
	if err != nil {
		t.Fatalf("list account bindings: %v", err)
	}
	if len(bindings) != 0 {
		t.Fatalf("account bindings = %d, want 0", len(bindings))
	}

	_, err = queries.GetArchivedUserByOriginalID(ctx, generated.GetArchivedUserByOriginalIDParams{
		EntityID:       entity.ID,
		OriginalUserID: managedUser.ID,
	})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected no archived user row, got %v", err)
	}

	directoryUser, err := queries.GetDirectoryUserByExternalID(ctx, generated.GetDirectoryUserByExternalIDParams{
		EntityID:       entity.ID,
		SourceID:       source.ID,
		ExternalUserID: "ou_deleted_unbound_match",
	})
	if err != nil {
		t.Fatalf("get directory user: %v", err)
	}
	if directoryUser.Status != "deleted" {
		t.Fatalf("directory user status = %q, want deleted", directoryUser.Status)
	}
}

func TestFullSyncProviderReturnedDeletedUserWithoutBindingDoesNotCreateManagedUser(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool := newSyncTestPool(ctx, t)
	queries := generated.New(pool)

	entity, err := queries.CreateEntity(ctx, generated.CreateEntityParams{
		Name:          "Sync Deleted Create Guard Entity",
		Slug:          "sync-deleted-create-guard",
		DefaultLocale: "en-US",
	})
	if err != nil {
		t.Fatalf("create entity: %v", err)
	}
	source, err := queries.CreateIdentitySource(ctx, generated.CreateIdentitySourceParams{
		EntityID:    entity.ID,
		Type:        "feishu",
		Name:        "Feishu",
		SyncEnabled: true,
	})
	if err != nil {
		t.Fatalf("create identity source: %v", err)
	}

	service, err := NewSyncService(SyncServiceConfig{
		Queries: queries,
		Provider: fakeSyncDirectoryProvider{data: FullSyncData{
			Users: []DirectoryUser{{
				ExternalUserID: "ou_deleted_unbound_new",
				Name:           "Deleted User",
				Email:          "deleted.unbound.new@example.test",
				Status:         "deleted",
				RawProfile:     []byte(`{"name":"Deleted User","status":"deleted"}`),
			}},
		}},
		Audit:     &capturingAuditWriter{},
		TraceID:   func() string { return "trace-sync-deleted-create-guard" },
		TxStarter: pool,
	})
	if err != nil {
		t.Fatalf("new sync service: %v", err)
	}

	result, err := service.RunFullSync(ctx, FullSyncInput{
		EntityID: entity.ID,
		SourceID: source.ID,
		Provider: "feishu",
	})
	if err != nil {
		t.Fatalf("run full sync: %v", err)
	}

	if result.ManagedUsersCreated != 0 {
		t.Fatalf("ManagedUsersCreated = %d, want 0", result.ManagedUsersCreated)
	}
	if result.ManagedUsersDeleted != 0 {
		t.Fatalf("ManagedUsersDeleted = %d, want 0", result.ManagedUsersDeleted)
	}
	if result.BindingsCreated != 0 {
		t.Fatalf("BindingsCreated = %d, want 0", result.BindingsCreated)
	}

	_, err = queries.GetManagedUserByUsername(ctx, generated.GetManagedUserByUsernameParams{
		EntityID: entity.ID,
		Username: "deleted.unbound.new@example.test",
	})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected no managed user row, got %v", err)
	}

	_, err = queries.GetAccountBindingByProviderUID(ctx, generated.GetAccountBindingByProviderUIDParams{
		EntityID:    entity.ID,
		SourceID:    source.ID,
		ProviderUid: "ou_deleted_unbound_new",
	})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected no account binding row, got %v", err)
	}

	directoryUser, err := queries.GetDirectoryUserByExternalID(ctx, generated.GetDirectoryUserByExternalIDParams{
		EntityID:       entity.ID,
		SourceID:       source.ID,
		ExternalUserID: "ou_deleted_unbound_new",
	})
	if err != nil {
		t.Fatalf("get directory user: %v", err)
	}
	if directoryUser.Status != "deleted" {
		t.Fatalf("directory user status = %q, want deleted", directoryUser.Status)
	}
}

type fakeSyncDirectoryProvider struct {
	data FullSyncData
	err  error
}

type capturingAuditWriter struct {
	events []audit.Event
}

func (w *capturingAuditWriter) Write(_ context.Context, event audit.Event) error {
	w.events = append(w.events, event)
	return nil
}

func (f fakeSyncDirectoryProvider) FullSync(context.Context) (FullSyncData, error) {
	return f.data, f.err
}

func (f fakeSyncDirectoryProvider) IncrementalSync(context.Context, []DirectorySyncEvent) (FullSyncData, error) {
	return f.data, f.err
}

func newSyncTestPool(ctx context.Context, t *testing.T) *pgxpool.Pool {
	t.Helper()

	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("idbridge"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("5432/tcp")),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if err := container.Terminate(cleanupCtx); err != nil {
			t.Errorf("terminate postgres container: %v", err)
		}
	})

	conn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	applySyncTestMigrations(ctx, t, conn)

	pool, err := pgxpool.New(ctx, conn)
	if err != nil {
		t.Fatalf("open pgx pool: %v", err)
	}
	t.Cleanup(pool.Close)

	return pool
}

func applySyncTestMigrations(ctx context.Context, t *testing.T, conn string) {
	t.Helper()

	db, err := sql.Open("pgx", conn)
	if err != nil {
		t.Fatalf("open sql db: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close sql db: %v", err)
		}
	})
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping sql db: %v", err)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("set goose dialect: %v", err)
	}
	if err := goose.UpContext(ctx, db, "../../migrations"); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
}
