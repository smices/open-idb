// SPDX-License-Identifier: MIT

package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/smices/open-idb/internal/db/generated"
	"github.com/smices/open-idb/internal/id"
)

// auditQuerier abstracts the generated query methods for testability.
type auditQuerier interface {
	CreateAuditLog(ctx context.Context, arg generated.CreateAuditLogParams) (generated.AuditLog, error)
	ListAuditLogs(ctx context.Context, arg generated.ListAuditLogsParams) ([]generated.ListAuditLogsRow, error)
	CountAuditLogs(ctx context.Context, arg generated.CountAuditLogsParams) (int64, error)
}

// Service provides audit log write and query operations.
type Service struct {
	queries auditQuerier
}

// NewService creates an audit Service backed by the given queries.
func NewService(queries *generated.Queries) *Service {
	return &Service{queries: queries}
}

// ListOptions controls filtering and pagination for audit log queries.
type ListOptions struct {
	Action       string
	ResourceType string
	ActorType    string
	Limit        int
	Offset       int
}

// ListResult holds a page of audit log entries and the total count.
type ListResult struct {
	Items []AuditLogEntry
	Total int64
}

// AuditLogEntry is the application-level representation of an audit log row.
type AuditLogEntry struct {
	ID           string          `json:"id"`
	EntityID     string          `json:"entity_id"`
	ActorUserID  string          `json:"actor_user_id"`
	ActorType    string          `json:"actor_type"`
	ActorName    string          `json:"actor_display_name"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resource_type"`
	ResourceID   string          `json:"resource_id"`
	ResourceName string          `json:"resource_display_name"`
	Outcome      string          `json:"outcome"`
	Before       json.RawMessage `json:"before"`
	After        json.RawMessage `json:"after"`
	IP           string          `json:"ip"`
	UserAgent    string          `json:"user_agent"`
	TraceID      string          `json:"trace_id"`
	CreatedAt    time.Time       `json:"created_at"`
}

// Write records an audit event. A critical write failure must fail the
// calling admin operation.
func (s *Service) Write(ctx context.Context, event Event) error {
	entityID, err := ulidValue(event.EntityID)
	if err != nil {
		return fmt.Errorf("invalid entity_id: %w", err)
	}

	var actorUserID pgtype.Text
	if event.ActorUserID != "" {
		parsedActorUserID, err := ulidValue(event.ActorUserID)
		if err != nil {
			return fmt.Errorf("invalid actor_user_id: %w", err)
		}
		actorUserID = pgtype.Text{String: parsedActorUserID, Valid: true}
	}

	beforeState, err := marshalJSON(event.Before)
	if err != nil {
		return fmt.Errorf("marshal before state: %w", err)
	}
	afterState, err := marshalJSON(event.After)
	if err != nil {
		return fmt.Errorf("marshal after state: %w", err)
	}

	_, err = s.queries.CreateAuditLog(ctx, generated.CreateAuditLogParams{
		EntityID:     entityID,
		ActorUserID:  actorUserID,
		ActorType:    event.ActorType,
		Action:       event.Action,
		ResourceType: event.ResourceType,
		ResourceID:   event.ResourceID,
		BeforeState:  beforeState,
		AfterState:   afterState,
		Ip:           event.IP,
		UserAgent:    event.UserAgent,
		TraceID:      event.TraceID,
	})
	return err
}

// List returns paginated audit logs with optional filters for a entity.
func (s *Service) List(ctx context.Context, entityID string, opts ListOptions) (ListResult, error) {
	entityULID, err := ulidValue(entityID)
	if err != nil {
		return ListResult{}, fmt.Errorf("invalid entity_id: %w", err)
	}

	f := filterParams(opts)

	total, err := s.queries.CountAuditLogs(ctx, generated.CountAuditLogsParams{
		EntityID:     entityULID,
		Action:       f.action,
		ResourceType: f.resourceType,
		ActorType:    f.actorType,
	})
	if err != nil {
		return ListResult{}, fmt.Errorf("count audit logs: %w", err)
	}

	rows, err := s.queries.ListAuditLogs(ctx, generated.ListAuditLogsParams{
		EntityID:     entityULID,
		Limit:        int32(opts.Limit),
		Offset:       int32(opts.Offset),
		Action:       f.action,
		ResourceType: f.resourceType,
		ActorType:    f.actorType,
	})
	if err != nil {
		return ListResult{}, fmt.Errorf("list audit logs: %w", err)
	}

	items := make([]AuditLogEntry, 0, len(rows))
	for _, row := range rows {
		items = append(items, toEntry(row))
	}

	return ListResult{Items: items, Total: total}, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

type optFilters struct {
	action       pgtype.Text
	resourceType pgtype.Text
	actorType    pgtype.Text
}

func filterParams(opts ListOptions) optFilters {
	return optFilters{
		action:       optionalText(opts.Action),
		resourceType: optionalText(opts.ResourceType),
		actorType:    optionalText(opts.ActorType),
	}
}

func toEntry(row generated.ListAuditLogsRow) AuditLogEntry {
	return AuditLogEntry{
		ID:           ulidString(row.ID),
		EntityID:     ulidString(row.EntityID),
		ActorUserID:  textString(row.ActorUserID),
		ActorType:    row.ActorType,
		ActorName:    interfaceString(row.ActorDisplayName),
		Action:       row.Action,
		ResourceType: row.ResourceType,
		ResourceID:   row.ResourceID,
		ResourceName: interfaceString(row.ResourceDisplayName),
		Outcome:      row.Outcome,
		Before:       json.RawMessage(row.BeforeState),
		After:        json.RawMessage(row.AfterState),
		IP:           row.Ip,
		UserAgent:    row.UserAgent,
		TraceID:      row.TraceID,
		CreatedAt:    row.CreatedAt.Time,
	}
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

func textString(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func interfaceString(value interface{}) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
}

func optionalText(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: value != ""}
}

func marshalJSON(v interface{}) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}
