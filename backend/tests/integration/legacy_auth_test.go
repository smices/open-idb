// SPDX-License-Identifier: MIT

package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/smices/open-idb/internal/auth"
	"github.com/smices/open-idb/internal/db/generated"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/crypto/bcrypt"
)

func TestLegacyLoginSuccess(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	queries, service, entity, user, app, pool := setupLegacyLoginHarness(t, ctx)
	if err := grantLegacyAppAccess(ctx, t, pool, entity.ID, app.ID, user.ID); err != nil {
		t.Fatalf("grant app access: %v", err)
	}
	if err := createLegacyMapping(ctx, t, queries, entity.ID, app.ID, user.ID, "legacy-user", "legacy-pass"); err != nil {
		t.Fatalf("create legacy mapping: %v", err)
	}

	result, err := service.AuthenticateLegacy(ctx, pgULIDString(entity.ID), pgULIDString(app.ID), "legacy-user", "legacy-pass", "127.0.0.1", "agent", "trace-success")
	if err != nil {
		t.Fatalf("AuthenticateLegacy: %v", err)
	}
	if result.UserID != pgULIDString(user.ID) {
		t.Fatalf("result user_id = %s, want %s", result.UserID, pgULIDString(user.ID))
	}
	if result.Username != "legacy-user" {
		t.Fatalf("result username = %s, want legacy-user", result.Username)
	}
	if result.SessionValue == "" {
		t.Fatal("session should not be empty")
	}
}

func TestLegacyLoginFailsWhenMappingMissing(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	_, service, entity, _, app, _ := setupLegacyLoginHarness(t, ctx)
	if _, err := service.AuthenticateLegacy(ctx, pgULIDString(entity.ID), pgULIDString(app.ID), "missing-user", "any", "127.0.0.1", "agent", "trace-missing"); err == nil {
		t.Fatal("AuthenticateLegacy() error = nil, want error")
	} else {
		assertLegacyLoginError(t, err, "legacy_auth_failed", 401)
	}
}

func TestLegacyLoginLocksAfterRepeatedFailures(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	queries, service, entity, user, app, pool := setupLegacyLoginHarness(t, ctx)
	if err := grantLegacyAppAccess(ctx, t, pool, entity.ID, app.ID, user.ID); err != nil {
		t.Fatalf("grant app access: %v", err)
	}
	if err := createLegacyMapping(ctx, t, queries, entity.ID, app.ID, user.ID, "legacy-user", "legacy-pass"); err != nil {
		t.Fatalf("create legacy mapping: %v", err)
	}
	service.SetFailurePolicy(2, 10*time.Minute)

	_, err := service.AuthenticateLegacy(ctx, pgULIDString(entity.ID), pgULIDString(app.ID), "legacy-user", "wrong-pass", "127.0.0.1", "agent", "trace-fail-1")
	if err == nil {
		t.Fatal("first failure expected")
	}
	assertLegacyLoginError(t, err, "legacy_auth_failed", 401)

	_, err = service.AuthenticateLegacy(ctx, pgULIDString(entity.ID), pgULIDString(app.ID), "legacy-user", "wrong-pass", "127.0.0.1", "agent", "trace-fail-2")
	if err == nil {
		t.Fatal("second failure expected")
	}
	assertLegacyLoginError(t, err, "legacy_auth_failed", 401)

	_, err = service.AuthenticateLegacy(ctx, pgULIDString(entity.ID), pgULIDString(app.ID), "legacy-user", "wrong-pass", "127.0.0.1", "agent", "trace-fail-3")
	if err == nil {
		t.Fatal("lock expected")
	}
	assertLegacyLoginError(t, err, "legacy_auth_locked", 423)
}

func TestLegacyLoginDeniedWithoutApplicationAccess(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	queries, service, entity, user, app, _ := setupLegacyLoginHarness(t, ctx)
	if err := createLegacyMapping(ctx, t, queries, entity.ID, app.ID, user.ID, "legacy-user", "legacy-pass"); err != nil {
		t.Fatalf("create legacy mapping: %v", err)
	}
	_, err := service.AuthenticateLegacy(ctx, pgULIDString(entity.ID), pgULIDString(app.ID), "legacy-user", "legacy-pass", "127.0.0.1", "agent", "trace-deny")
	if err == nil {
		t.Fatal("access denied expected")
	}
	assertLegacyLoginError(t, err, "access_denied", 403)
}

func setupLegacyLoginHarness(t *testing.T, ctx context.Context) (
	queries *generated.Queries,
	service *auth.LegacyLoginService,
	entity generated.BusinessEntity,
	user generated.User,
	app generated.Application,
	pool *pgxpool.Pool,
) {
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
	applyMigrations(ctx, t, conn)

	pool, err = pgxpool.New(ctx, conn)
	if err != nil {
		t.Fatalf("open pgx pool: %v", err)
	}
	t.Cleanup(pool.Close)

	queries = generated.New(pool)
	entity, user = createOIDCTestIdentity(ctx, t, queries)
	app, err = queries.CreateApplication(ctx, generated.CreateApplicationParams{
		EntityID: entity.ID,
		Name:     "Legacy Demo App",
		Type:     "api_client",
	})
	if err != nil {
		t.Fatalf("create application: %v", err)
	}
	service = auth.NewLegacyLoginService(queries)
	return
}

func grantLegacyAppAccess(
	ctx context.Context,
	t *testing.T,
	pool *pgxpool.Pool,
	entityID string,
	appID string,
	userID string,
) error {
	t.Helper()
	_, err := pool.Exec(ctx, `
		INSERT INTO application_assignments (entity_id, application_id, subject_type, subject_id, effect)
		VALUES ($1, $2, 'user', $3, 'allow')
	`, entityID, appID, userID)
	return err
}

func createLegacyMapping(
	ctx context.Context,
	t *testing.T,
	queries *generated.Queries,
	entityID string,
	appID string,
	userID string,
	username string,
	password string,
) error {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = queries.UpsertLegacyAppUser(ctx, generated.UpsertLegacyAppUserParams{
		EntityID:             entityID,
		ApplicationID:        appID,
		UserID:               userID,
		Username:             username,
		LegacyUserIdentifier: pgtype.Text{},
		AuthScheme:           "local",
		CredentialHash:       string(hash),
		IsActive:             true,
	})
	return err
}

func assertLegacyLoginError(t *testing.T, err error, wantCode string, wantStatus int) {
	t.Helper()
	var legacyErr auth.LegacyLoginError
	if !errors.As(err, &legacyErr) {
		t.Fatalf("error = %T %v, want LegacyLoginError", err, err)
	}
	if legacyErr.Code != wantCode {
		t.Fatalf("error code = %s, want %s", legacyErr.Code, wantCode)
	}
	if legacyErr.Status != wantStatus {
		t.Fatalf("error status = %d, want %d", legacyErr.Status, wantStatus)
	}
}
