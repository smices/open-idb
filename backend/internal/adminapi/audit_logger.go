// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"fmt"

	"github.com/smices/open-idb/internal/audit"
)

// AuditLogger is the subset of audit.Service needed for admin operation
// audit logging. Defining it here allows testing without the real audit
// service.
type AuditLogger interface {
	Write(ctx context.Context, event audit.Event) error
}

// auditWriter wraps an AuditLogger with convenience methods for common
// admin audit patterns. Per spec: "Critical admin audit write failure:
// fail the admin operation." All methods return errors so callers can
// propagate audit failures.
type auditWriter struct {
	logger AuditLogger
}

// logCreate writes a create audit event with the after state.
// Returns an error if the audit write fails, which the caller should
// treat as a fatal error for the admin operation.
func (a *auditWriter) logCreate(ctx context.Context, entityID, actorUserID, resourceType, resourceID string, after interface{}) error {
	if a == nil || a.logger == nil {
		return nil
	}
	err := a.logger.Write(ctx, audit.Event{
		EntityID:     entityID,
		ActorUserID:  actorUserID,
		ActorType:    "user",
		Action:       actionForCreate(resourceType),
		ResourceType: resourceType,
		ResourceID:   resourceID,
		After:        after,
	})
	if err != nil {
		return fmt.Errorf("audit write failed for %s create: %w", resourceType, err)
	}
	return nil
}

// logUpdate writes an update audit event with before and after states.
func (a *auditWriter) logUpdate(ctx context.Context, entityID, actorUserID, resourceType, resourceID string, before, after interface{}) error {
	if a == nil || a.logger == nil {
		return nil
	}
	err := a.logger.Write(ctx, audit.Event{
		EntityID:     entityID,
		ActorUserID:  actorUserID,
		ActorType:    "user",
		Action:       actionForUpdate(resourceType),
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Before:       before,
		After:        after,
	})
	if err != nil {
		return fmt.Errorf("audit write failed for %s update: %w", resourceType, err)
	}
	return nil
}

// logDelete writes a delete audit event with the before state.
func (a *auditWriter) logDelete(ctx context.Context, entityID, actorUserID, resourceType, resourceID string, before interface{}) error {
	if a == nil || a.logger == nil {
		return nil
	}
	err := a.logger.Write(ctx, audit.Event{
		EntityID:     entityID,
		ActorUserID:  actorUserID,
		ActorType:    "user",
		Action:       "admin.deleted",
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Before:       before,
	})
	if err != nil {
		return fmt.Errorf("audit write failed for %s delete: %w", resourceType, err)
	}
	return nil
}

// logAction writes a custom audit event.
func (a *auditWriter) logAction(ctx context.Context, event audit.Event) error {
	if a == nil || a.logger == nil {
		return nil
	}
	err := a.logger.Write(ctx, event)
	if err != nil {
		return fmt.Errorf("audit write failed for %s: %w", event.Action, err)
	}
	return nil
}

// actionForCreate returns the appropriate audit action for a create operation.
func actionForCreate(resourceType string) string {
	switch resourceType {
	case "role":
		return audit.ActionRoleCreated
	case "application":
		return audit.ActionAppCreated
	default:
		return "admin.created"
	}
}

// actionForUpdate returns the appropriate audit action for an update operation.
func actionForUpdate(resourceType string) string {
	switch resourceType {
	case "role":
		return audit.ActionRoleUpdated
	case "user":
		return audit.ActionUserUpdated
	case "permission":
		return audit.ActionRolePermChanged
	case "oidc_client":
		return audit.ActionOIDCClientUpdated
	default:
		return "admin.updated"
	}
}
