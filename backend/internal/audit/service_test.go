// SPDX-License-Identifier: MIT

package audit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/smices/open-idb/internal/db/generated"
)

// fakeQueries implements auditQuerier for unit tests.
type fakeQueries struct {
	createParams generated.CreateAuditLogParams
	createErr    error

	listParams generated.ListAuditLogsParams
	listRows   []generated.ListAuditLogsRow
	listErr    error

	countParams generated.CountAuditLogsParams
	countResult int64
	countErr    error

	deleteParams generated.DeleteAuditLogParams
	deleteResult int64
	deleteErr    error

	clearEntityID string
	clearResult   int64
	clearErr      error
}

func (f *fakeQueries) CreateAuditLog(_ context.Context, arg generated.CreateAuditLogParams) (generated.AuditLog, error) {
	f.createParams = arg
	if f.createErr != nil {
		return generated.AuditLog{}, f.createErr
	}
	return generated.AuditLog{
		ID:           arg.EntityID, // reuse for simplicity
		EntityID:     arg.EntityID,
		ActorUserID:  arg.ActorUserID,
		ActorType:    arg.ActorType,
		Action:       arg.Action,
		ResourceType: arg.ResourceType,
		ResourceID:   arg.ResourceID,
		BeforeState:  arg.BeforeState,
		AfterState:   arg.AfterState,
		Ip:           arg.Ip,
		UserAgent:    arg.UserAgent,
		TraceID:      arg.TraceID,
		CreatedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}, nil
}

func (f *fakeQueries) ListAuditLogs(_ context.Context, arg generated.ListAuditLogsParams) ([]generated.ListAuditLogsRow, error) {
	f.listParams = arg
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.listRows, nil
}

func (f *fakeQueries) CountAuditLogs(_ context.Context, arg generated.CountAuditLogsParams) (int64, error) {
	f.countParams = arg
	if f.countErr != nil {
		return 0, f.countErr
	}
	return f.countResult, nil
}

func (f *fakeQueries) DeleteAuditLog(_ context.Context, arg generated.DeleteAuditLogParams) (int64, error) {
	f.deleteParams = arg
	return f.deleteResult, f.deleteErr
}

func (f *fakeQueries) ClearAuditLogs(_ context.Context, entityID string) (int64, error) {
	f.clearEntityID = entityID
	return f.clearResult, f.clearErr
}

func TestWriteMapsEventToQueryParams(t *testing.T) {
	fq := &fakeQueries{}
	svc := &Service{queries: fq}

	event := Event{
		EntityID:     "01HZZZZZZZ0000000000000100",
		ActorUserID:  "01HZZZZZZZ0000000000000101",
		ActorType:    "user",
		Action:       ActionUserUpdated,
		ResourceType: "user",
		ResourceID:   "01HZZZZZZZ0000000000000101",
		Before:       map[string]string{"name": "old"},
		After:        map[string]string{"name": "new"},
		IP:           "192.168.1.1",
		UserAgent:    "Mozilla/5.0",
		TraceID:      "trace-abc",
	}

	if err := svc.Write(context.Background(), event); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	p := fq.createParams
	if p.Action != ActionUserUpdated {
		t.Errorf("Action = %q, want %q", p.Action, ActionUserUpdated)
	}
	if p.ActorType != "user" {
		t.Errorf("ActorType = %q, want %q", p.ActorType, "user")
	}
	if p.ResourceType != "user" {
		t.Errorf("ResourceType = %q, want %q", p.ResourceType, "user")
	}
	if p.ResourceID != "01HZZZZZZZ0000000000000101" {
		t.Errorf("ResourceID = %q", p.ResourceID)
	}
	if p.Ip != "192.168.1.1" {
		t.Errorf("Ip = %q", p.Ip)
	}
	if p.UserAgent != "Mozilla/5.0" {
		t.Errorf("UserAgent = %q", p.UserAgent)
	}
	if p.TraceID != "trace-abc" {
		t.Errorf("TraceID = %q", p.TraceID)
	}
	if p.EntityID == "" {
		t.Error("EntityID is empty")
	}
	if !p.ActorUserID.Valid {
		t.Error("ActorUserID.Valid = false, want true")
	}

	// Verify Before/After were marshaled to JSON
	var before map[string]string
	if err := json.Unmarshal(p.BeforeState, &before); err != nil {
		t.Fatalf("unmarshal BeforeState: %v", err)
	}
	if before["name"] != "old" {
		t.Errorf("before name = %q, want %q", before["name"], "old")
	}
	var after map[string]string
	if err := json.Unmarshal(p.AfterState, &after); err != nil {
		t.Fatalf("unmarshal AfterState: %v", err)
	}
	if after["name"] != "new" {
		t.Errorf("after name = %q, want %q", after["name"], "new")
	}
}

func TestWriteWithSystemActorOmitsActorUserID(t *testing.T) {
	fq := &fakeQueries{}
	svc := &Service{queries: fq}

	event := Event{
		EntityID:     "01HZZZZZZZ0000000000000100",
		ActorUserID:  "", // system actor
		ActorType:    "system",
		Action:       ActionSyncStarted,
		ResourceType: "sync_job",
		ResourceID:   "job-1",
	}

	if err := svc.Write(context.Background(), event); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	if fq.createParams.ActorUserID.Valid {
		t.Error("ActorUserID.Valid = true, want false for system actor")
	}
}

func TestWriteWithNilStatesStoresEmptyObjects(t *testing.T) {
	fq := &fakeQueries{}
	svc := &Service{queries: fq}

	event := Event{
		EntityID:     "01HZZZZZZZ0000000000000100",
		ActorType:    "system",
		Action:       ActionSyncFinished,
		ResourceType: "sync_job",
		ResourceID:   "job-1",
		Before:       nil,
		After:        nil,
	}

	if err := svc.Write(context.Background(), event); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	if string(fq.createParams.BeforeState) != "{}" {
		t.Errorf("BeforeState = %s, want {}", fq.createParams.BeforeState)
	}
	if string(fq.createParams.AfterState) != "{}" {
		t.Errorf("AfterState = %s, want {}", fq.createParams.AfterState)
	}
}

func TestListPassesFiltersCorrectly(t *testing.T) {
	fq := &fakeQueries{
		countResult: 42,
		listRows: []generated.ListAuditLogsRow{
			{
				ID:           "01HZZZZZZZ0000000000000001",
				EntityID:     "01HZZZZZZZ0000000000000002",
				ActorUserID:  pgtype.Text{String: "01HZZZZZZZ0000000000000003", Valid: true},
				ActorType:    "user",
				Action:       ActionLoginSuccess,
				ResourceType: "session",
				ResourceID:   "sess-1",
				Ip:           "10.0.0.1",
				UserAgent:    "test-agent",
				TraceID:      "trace-1",
				CreatedAt:    pgtype.Timestamptz{Time: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
				Outcome:      "succeeded",
			},
		},
	}
	svc := &Service{queries: fq}

	result, err := svc.List(context.Background(), "01HZZZZZZZ0000000000000100", ListOptions{
		Action:       ActionLoginSuccess,
		ResourceType: "session",
		ActorType:    "user",
		Limit:        25,
		Offset:       10,
	})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	// Verify count params
	cp := fq.countParams
	if !cp.Action.Valid || cp.Action.String != ActionLoginSuccess {
		t.Errorf("count Action = %+v, want valid %q", cp.Action, ActionLoginSuccess)
	}
	if !cp.ResourceType.Valid || cp.ResourceType.String != "session" {
		t.Errorf("count ResourceType = %+v, want valid %q", cp.ResourceType, "session")
	}
	if !cp.ActorType.Valid || cp.ActorType.String != "user" {
		t.Errorf("count ActorType = %+v, want valid %q", cp.ActorType, "user")
	}

	// Verify list params
	lp := fq.listParams
	if lp.Limit != 25 {
		t.Errorf("Limit = %d, want 25", lp.Limit)
	}
	if lp.Offset != 10 {
		t.Errorf("Offset = %d, want 10", lp.Offset)
	}
	if !lp.Action.Valid || lp.Action.String != ActionLoginSuccess {
		t.Errorf("list Action = %+v, want valid %q", lp.Action, ActionLoginSuccess)
	}

	// Verify result
	if result.Total != 42 {
		t.Errorf("Total = %d, want 42", result.Total)
	}
	if len(result.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(result.Items))
	}
	entry := result.Items[0]
	if entry.Action != ActionLoginSuccess {
		t.Errorf("entry.Action = %q, want %q", entry.Action, ActionLoginSuccess)
	}
	if entry.IP != "10.0.0.1" {
		t.Errorf("entry.IP = %q", entry.IP)
	}
	if entry.CreatedAt.Year() != 2025 {
		t.Errorf("entry.CreatedAt.Year() = %d, want 2025", entry.CreatedAt.Year())
	}
}

func TestListWithEmptyFiltersSendsNullParams(t *testing.T) {
	fq := &fakeQueries{countResult: 0}
	svc := &Service{queries: fq}

	_, err := svc.List(context.Background(), "01HZZZZZZZ0000000000000100", ListOptions{
		Limit:  50,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	cp := fq.countParams
	if cp.Action.Valid {
		t.Error("count Action.Valid = true, want false (no filter)")
	}
	if cp.ResourceType.Valid {
		t.Error("count ResourceType.Valid = true, want false (no filter)")
	}
	if cp.ActorType.Valid {
		t.Error("count ActorType.Valid = true, want false (no filter)")
	}
}

func TestDeleteScopesQueryToEntity(t *testing.T) {
	fq := &fakeQueries{deleteResult: 1}
	svc := &Service{queries: fq}

	deleted, err := svc.Delete(
		context.Background(),
		"01HZZZZZZZ0000000000000100",
		"01HZZZZZZZ0000000000000200",
	)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}
	if fq.deleteParams.EntityID != "01HZZZZZZZ0000000000000100" {
		t.Fatalf("entity id = %q", fq.deleteParams.EntityID)
	}
	if fq.deleteParams.ID != "01HZZZZZZZ0000000000000200" {
		t.Fatalf("audit log id = %q", fq.deleteParams.ID)
	}
}

func TestClearScopesQueryToEntity(t *testing.T) {
	fq := &fakeQueries{clearResult: 4}
	svc := &Service{queries: fq}

	deleted, err := svc.Clear(context.Background(), "01HZZZZZZZ0000000000000100")
	if err != nil {
		t.Fatalf("Clear returned error: %v", err)
	}
	if deleted != 4 {
		t.Fatalf("deleted = %d, want 4", deleted)
	}
	if fq.clearEntityID != "01HZZZZZZZ0000000000000100" {
		t.Fatalf("entity id = %q", fq.clearEntityID)
	}
}
