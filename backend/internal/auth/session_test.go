// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/smices/open-idb/internal/db/generated"
)

func TestResolveSessionUsesConfiguredResolver(t *testing.T) {
	legacyValue, err := EncodeSession(Session{
		UserID:      "legacy-user",
		EntityID:    "legacy-entity",
		Username:    "legacy",
		DisplayName: "Legacy",
	})
	if err != nil {
		t.Fatalf("EncodeSession: %v", err)
	}

	SetSessionResolver(fakeSessionResolver{
		session: Session{
			ID:          "opaque-session",
			UserID:      "resolved-user",
			EntityID:    "resolved-entity",
			Username:    "resolved",
			DisplayName: "Resolved User",
		},
	})
	defer SetSessionResolver(nil)

	session, err := ResolveSession(context.Background(), legacyValue)
	if err != nil {
		t.Fatalf("ResolveSession: %v", err)
	}
	if session.UserID != "resolved-user" {
		t.Fatalf("UserID = %q, want resolver result", session.UserID)
	}
}

func TestCreateSessionValueUsesDatabaseCreator(t *testing.T) {
	now := time.Now().UTC()
	creator := &fakeSessionCreator{session: generated.Session{
		ID:       "db-session-id",
		EntityID: "entity-1",
		UserID:   "user-1",
		ExpiresAt: pgtype.Timestamptz{
			Time:  now.Add(time.Hour),
			Valid: true,
		},
	}}

	session, err := createSessionValue(context.Background(), creator, Session{
		EntityID:    "entity-1",
		UserID:      "user-1",
		Username:    "admin",
		DisplayName: "Administrator",
	}, SessionMetadata{
		LoginMethod: "password",
		TTL:         time.Hour,
	})
	if err != nil {
		t.Fatalf("createSessionValue: %v", err)
	}
	if session.ID != "db-session-id" {
		t.Fatalf("ID = %q, want db-session-id", session.ID)
	}
	if creator.arg.LoginMethod != "password" {
		t.Fatalf("LoginMethod = %q, want password", creator.arg.LoginMethod)
	}
}

type fakeSessionResolver struct {
	session Session
}

func (f fakeSessionResolver) ResolveSession(context.Context, string) (Session, error) {
	return f.session, nil
}

type fakeSessionCreator struct {
	session generated.Session
	arg     generated.CreateSessionParams
}

func (f *fakeSessionCreator) CreateSession(_ context.Context, arg generated.CreateSessionParams) (generated.Session, error) {
	f.arg = arg
	return f.session, nil
}
