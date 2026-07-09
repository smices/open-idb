// SPDX-License-Identifier: MIT

package adminapi

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
	"github.com/smices/open-idb/internal/db/generated"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestArchiveUserWritesArchiveAndDeletesActiveRow(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool := newAdminServiceTestPool(ctx, t)
	queries := generated.New(pool)
	svc, err := NewAdminService(queries)
	if err != nil {
		t.Fatalf("new admin service: %v", err)
	}
	svc.SetTxStarter(pool)

	entity, err := queries.CreateEntity(ctx, generated.CreateEntityParams{
		Name:          "Acme Identity",
		Slug:          "acme-archive",
		DefaultLocale: "en-US",
	})
	if err != nil {
		t.Fatalf("create entity: %v", err)
	}
	source, err := queries.CreateIdentitySource(ctx, generated.CreateIdentitySourceParams{
		EntityID:    entity.ID,
		Type:        "local",
		Name:        "Primary directory",
		SyncEnabled: true,
	})
	if err != nil {
		t.Fatalf("create identity source: %v", err)
	}
	directoryUser, err := queries.UpsertDirectoryUser(ctx, generated.UpsertDirectoryUserParams{
		EntityID:       entity.ID,
		SourceID:       source.ID,
		ExternalUserID: "external-ada",
		Name:           "Ada Lovelace",
		Status:         "active",
		RawProfile:     []byte(`{"department":"engineering"}`),
	})
	if err != nil {
		t.Fatalf("create directory user: %v", err)
	}
	user, err := queries.CreateManagedUser(ctx, generated.CreateManagedUserParams{
		EntityID:        entity.ID,
		Username:        "ada@example.test",
		DisplayName:     "Ada Lovelace",
		Email:           pgtype.Text{String: "ada@example.test", Valid: true},
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
		UserID:          user.ID,
		SourceID:        source.ID,
		DirectoryUserID: directoryUser.ID,
		ProviderUid:     "ada@example.test",
		IsPrimary:       true,
	}); err != nil {
		t.Fatalf("create account binding: %v", err)
	}
	role, err := queries.CreateRole(ctx, generated.CreateRoleParams{
		EntityID: entity.ID,
		Name:     "Employee",
		Code:     "employee",
	})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}
	if err := queries.AssignRoleToUser(ctx, generated.AssignRoleToUserParams{
		EntityID: entity.ID,
		UserID:   user.ID,
		RoleID:   role.ID,
	}); err != nil {
		t.Fatalf("assign role: %v", err)
	}

	archive, err := svc.ArchiveUser(ctx, entity.ID, user.ID, "", "admin deleted user")
	if err != nil {
		t.Fatalf("archive user: %v", err)
	}

	if archive.OriginalUserID != user.ID {
		t.Fatalf("expected original user id %s, got %s", user.ID, archive.OriginalUserID)
	}
	if archive.Username != "ada@example.test" {
		t.Fatalf("expected username ada@example.test, got %s", archive.Username)
	}

	_, err = queries.GetUserByID(ctx, generated.GetUserByIDParams{EntityID: entity.ID, ID: user.ID})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected active user row to be deleted, got %v", err)
	}

	if _, err := queries.CreateManagedUser(ctx, generated.CreateManagedUserParams{
		EntityID:        entity.ID,
		Username:        "ada@example.test",
		DisplayName:     "Ada Recreated",
		Email:           pgtype.Text{String: "ada@example.test", Valid: true},
		LifecycleStatus: "active",
		UserType:        "employee",
		PrimarySourceID: pgtype.Text{String: source.ID, Valid: true},
		Locale:          pgtype.Text{String: "en-US", Valid: true},
	}); err != nil {
		t.Fatalf("expected username to be reusable after archive, got %v", err)
	}
}

func newAdminServiceTestPool(ctx context.Context, t *testing.T) *pgxpool.Pool {
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
	applyAdminServiceTestMigrations(ctx, t, conn)

	pool, err := pgxpool.New(ctx, conn)
	if err != nil {
		t.Fatalf("open pgx pool: %v", err)
	}
	t.Cleanup(pool.Close)

	return pool
}

func applyAdminServiceTestMigrations(ctx context.Context, t *testing.T, conn string) {
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
