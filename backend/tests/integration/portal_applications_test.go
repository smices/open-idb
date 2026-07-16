// SPDX-License-Identifier: MIT

package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/smices/open-idb/internal/auth"
	"github.com/smices/open-idb/internal/db/generated"
	"github.com/smices/open-idb/internal/httpserver"
	"github.com/smices/open-idb/internal/portal"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestPortalApplicationsReturnsOnlyActiveApplicationsForSessionEntity(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	conn := os.Getenv("OPEN_IDB_TEST_DATABASE_URL")
	if conn == "" {
		testcontainers.SkipIfProviderIsNotHealthy(t)
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
		conn, err = container.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			t.Fatalf("connection string: %v", err)
		}
	}
	applyMigrations(ctx, t, conn)

	pool, err := pgxpool.New(ctx, conn)
	if err != nil {
		t.Fatalf("open pgx pool: %v", err)
	}
	t.Cleanup(pool.Close)
	queries := generated.New(pool)

	entity, err := queries.CreateEntity(ctx, generated.CreateEntityParams{
		Name:          "Portal Entity",
		Slug:          "portal-entity",
		DefaultLocale: "en-US",
	})
	if err != nil {
		t.Fatalf("create portal entity: %v", err)
	}
	otherEntity, err := queries.CreateEntity(ctx, generated.CreateEntityParams{
		Name:          "Other Entity",
		Slug:          "other-entity",
		DefaultLocale: "en-US",
	})
	if err != nil {
		t.Fatalf("create other entity: %v", err)
	}

	active, err := queries.CreateApplication(ctx, generated.CreateApplicationParams{
		EntityID: entity.ID,
		Name:     "Expense",
		Type:     "internal_app",
		Config:   []byte(`{"description":"Submit expenses","app_url":"https://expense.example"}`),
	})
	if err != nil {
		t.Fatalf("create active application: %v", err)
	}
	disabled, err := queries.CreateApplication(ctx, generated.CreateApplicationParams{
		EntityID: entity.ID,
		Name:     "Disabled App",
		Type:     "internal_app",
		Status:   pgtype.Text{String: "disabled", Valid: true},
	})
	if err != nil {
		t.Fatalf("create disabled application: %v", err)
	}
	other, err := queries.CreateApplication(ctx, generated.CreateApplicationParams{
		EntityID: otherEntity.ID,
		Name:     "Other Entity App",
		Type:     "internal_app",
	})
	if err != nil {
		t.Fatalf("create other entity application: %v", err)
	}

	server := httptest.NewServer(httpserver.NewRouter(portal.NewHandler(queries).RegisterRoutes))
	t.Cleanup(server.Close)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/api/portal/applications", nil)
	if err != nil {
		t.Fatalf("new portal applications request: %v", err)
	}
	session, err := auth.EncodeSession(auth.Session{
		UserID:      "portal-user",
		EntityID:    entity.ID,
		Username:    "portal.user",
		DisplayName: "Portal User",
	})
	if err != nil {
		t.Fatalf("encode user session: %v", err)
	}
	req.AddCookie(&http.Cookie{Name: "idb_session", Value: session})

	res, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("get portal applications: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("portal applications status = %d, want %d, body=%s", res.StatusCode, http.StatusOK, body)
	}

	var response struct {
		Applications []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			EntryURL    string `json:"entry_url"`
		} `json:"applications"`
	}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		t.Fatalf("decode portal applications response: %v", err)
	}
	if len(response.Applications) != 1 {
		t.Fatalf("applications = %#v, want only active session-entity application", response.Applications)
	}
	if got := response.Applications[0]; got.ID != active.ID || got.Name != "Expense" || got.Description != "Submit expenses" || got.EntryURL != "https://expense.example" {
		t.Fatalf("application = %#v, want active session-entity application", got)
	}
	for _, application := range response.Applications {
		if application.ID == disabled.ID || application.ID == other.ID {
			t.Fatalf("unexpected application returned: %#v", application)
		}
	}
}
