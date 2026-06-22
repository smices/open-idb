// SPDX-License-Identifier: MIT

package entity

import (
	"context"
	"time"
)

// Entity represents a entity in the system.
type Entity struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	Status        Status    `json:"status"`
	DefaultLocale string    `json:"default_locale"`
	CreatedAt     time.Time `json:"created_at"`
}

// IsActive returns true if the entity is active.
func (t Entity) IsActive() bool {
	return t.Status == StatusActive
}

type contextKey struct{}

// FromContext extracts the entity ID from the context.
func FromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(contextKey{}).(string)
	return v, ok
}

// WithEntityID returns a new context with the entity ID set.
func WithEntityID(ctx context.Context, entityID string) context.Context {
	return context.WithValue(ctx, contextKey{}, entityID)
}

// MustFromContext extracts the entity ID from the context or panics.
func MustFromContext(ctx context.Context) string {
	v, ok := FromContext(ctx)
	if !ok {
		panic("entity: entity ID not found in context")
	}
	return v
}
