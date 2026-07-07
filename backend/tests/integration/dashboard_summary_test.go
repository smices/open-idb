// SPDX-License-Identifier: MIT

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/smices/open-idb/internal/db/generated"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestDashboardSummarySeparatesSyncedUsersFromAdmins(t *testing.T) {
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
	applyMigrations(ctx, t, conn)

	pool, err := pgxpool.New(ctx, conn)
	if err != nil {
		t.Fatalf("open pgx pool: %v", err)
	}
	t.Cleanup(pool.Close)
	queries := generated.New(pool)

	entity, err := queries.GetEntityBySlug(ctx, "default_enterprise")
	if err != nil {
		t.Fatalf("get default entity: %v", err)
	}
	summary, err := queries.GetDashboardSummary(ctx, entity.ID)
	if err != nil {
		t.Fatalf("get dashboard summary: %v", err)
	}
	if summary.Users != 0 || summary.ActiveUsers != 0 || summary.NewUsers != 0 {
		t.Fatalf("synced user counts = users:%d active:%d new:%d, want all 0", summary.Users, summary.ActiveUsers, summary.NewUsers)
	}
	if summary.AdminUsers != 1 {
		t.Fatalf("admin users = %d, want 1", summary.AdminUsers)
	}

	source, err := queries.CreateIdentitySource(ctx, generated.CreateIdentitySourceParams{
		EntityID:    entity.ID,
		Type:        "feishu",
		Name:        "Feishu",
		SyncEnabled: true,
	})
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	if _, err := queries.CreateManagedUser(ctx, generated.CreateManagedUserParams{
		EntityID:        entity.ID,
		Username:        "employee-1",
		DisplayName:     "Employee One",
		LifecycleStatus: "active",
		UserType:        "employee",
		PrimarySourceID: pgtype.Text{String: source.ID, Valid: true},
		Locale:          pgtype.Text{String: "zh-CN", Valid: true},
	}); err != nil {
		t.Fatalf("create synced managed user: %v", err)
	}

	summary, err = queries.GetDashboardSummary(ctx, entity.ID)
	if err != nil {
		t.Fatalf("get dashboard summary after synced user: %v", err)
	}
	if summary.Users != 1 || summary.ActiveUsers != 1 || summary.NewUsers != 1 {
		t.Fatalf("synced user counts = users:%d active:%d new:%d, want all 1", summary.Users, summary.ActiveUsers, summary.NewUsers)
	}
	if summary.AdminUsers != 1 {
		t.Fatalf("admin users = %d, want 1", summary.AdminUsers)
	}
}
