// SPDX-License-Identifier: MIT

package integration

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
	"github.com/smices/open-idb/internal/db/generated"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPostgresIdentitySchemaSupportsGeneratedQueries(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

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
	if conn == "" {
		t.Fatal("expected connection string")
	}

	applyMigrations(ctx, t, conn)

	pool, err := pgxpool.New(ctx, conn)
	if err != nil {
		t.Fatalf("open pgx pool: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
	})

	queries := generated.New(pool)
	entity, err := queries.CreateEntity(ctx, generated.CreateEntityParams{
		Name:          "Acme Identity",
		Slug:          "acme",
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
		EntityID:        entity.ID,
		SourceID:        source.ID,
		ExternalUserID:  "external-001",
		ExternalUnionID: pgtype.Text{String: "union-001", Valid: true},
		ExternalOpenID:  pgtype.Text{String: "open-001", Valid: true},
		Name:            "Ada Lovelace",
		Email:           pgtype.Text{String: "ada@example.com", Valid: true},
		Status:          "active",
		RawProfile:      []byte(`{"department":"engineering"}`),
	})
	if err != nil {
		t.Fatalf("upsert directory user: %v", err)
	}

	managedUser, err := queries.CreateManagedUser(ctx, generated.CreateManagedUserParams{
		EntityID:        entity.ID,
		Username:        "ada",
		DisplayName:     "Ada Lovelace",
		Email:           pgtype.Text{String: "ada@example.com", Valid: true},
		LifecycleStatus: "active",
		UserType:        "employee",
		PrimarySourceID: pgtype.Text{String: source.ID, Valid: true},
		Locale:          pgtype.Text{String: "en-US", Valid: true},
	})
	if err != nil {
		t.Fatalf("create managed user: %v", err)
	}

	binding, err := queries.CreateAccountBinding(ctx, generated.CreateAccountBindingParams{
		EntityID:        entity.ID,
		UserID:          managedUser.ID,
		SourceID:        source.ID,
		DirectoryUserID: directoryUser.ID,
		ProviderUid:     "external-001",
		ProviderUnionID: pgtype.Text{String: "union-001", Valid: true},
		IsPrimary:       true,
	})
	if err != nil {
		t.Fatalf("create account binding: %v", err)
	}
	if !binding.IsPrimary {
		t.Fatal("expected account binding to be primary")
	}

	otherEntity, err := queries.CreateEntity(ctx, generated.CreateEntityParams{
		Name:          "Other Identity",
		Slug:          "other",
		DefaultLocale: "en-US",
	})
	if err != nil {
		t.Fatalf("create second entity: %v", err)
	}
	_, err = queries.CreateAccountBinding(ctx, generated.CreateAccountBindingParams{
		EntityID:        otherEntity.ID,
		UserID:          managedUser.ID,
		SourceID:        source.ID,
		DirectoryUserID: directoryUser.ID,
		ProviderUid:     "external-001",
		IsPrimary:       true,
	})
	if err == nil {
		t.Fatal("expected cross-entity account binding to fail")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("expected postgres constraint error, got %T: %v", err, err)
	}
}

func applyMigrations(ctx context.Context, t *testing.T, conn string) {
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
