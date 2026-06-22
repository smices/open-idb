// SPDX-License-Identifier: MIT

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/smices/open-idb/internal/auth"
	"github.com/smices/open-idb/internal/db/generated"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestDefaultAdminCanAuthenticateAndRequiresPasswordChange(t *testing.T) {
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

	service, err := auth.NewService(generated.New(pool))
	if err != nil {
		t.Fatalf("new auth service: %v", err)
	}
	result, err := service.AuthenticateLocal(ctx, "admin", "admin123")
	if err != nil {
		t.Fatalf("authenticate default admin: %v", err)
	}
	if result.Username != "admin" {
		t.Fatalf("Username = %q, want admin", result.Username)
	}
	if !result.MustChangePassword {
		t.Fatal("MustChangePassword = false, want true")
	}
	if !result.WeakPassword {
		t.Fatal("WeakPassword = false, want true")
	}

	if _, err := service.AuthenticateLocal(ctx, "admin", "wrong"); err == nil {
		t.Fatal("AuthenticateLocal wrong password error = nil, want error")
	}
}
