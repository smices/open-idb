// SPDX-License-Identifier: MIT

package entity

import (
	"context"
	"net/http"
)

const (
	// HeaderEntityID is the HTTP header name for entity ID.
	HeaderEntityID = "X-IDB-Entity-ID"
)

// Middleware extracts the entity ID from the X-IDB-Entity-ID header
// and injects it into the request context.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entityID := r.Header.Get(HeaderEntityID)
		if entityID == "" {
			next.ServeHTTP(w, r)
			return
		}
		ctx := WithEntityID(r.Context(), entityID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireMiddleware requires a entity ID in the request. If missing,
// it returns 400 Bad Request.
func RequireMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entityID := r.Header.Get(HeaderEntityID)
		if entityID == "" {
			http.Error(w, `{"error":"entity_id_required","error_description":"X-IDB-Entity-ID header is required"}`, http.StatusBadRequest)
			return
		}
		ctx := WithEntityID(r.Context(), entityID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// FromRequest extracts the entity ID from an HTTP request.
// It checks the context first (set by Middleware), then the header.
func FromRequest(r *http.Request) string {
	if id, ok := FromContext(r.Context()); ok {
		return id
	}
	return r.Header.Get(HeaderEntityID)
}

// InjectContext returns a middleware that injects a fixed entity ID
// into the context. Useful for single-entity deployments.
func InjectContext(entityID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := WithEntityID(r.Context(), entityID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ExtractAndInject is a convenience that combines Middleware with
// context propagation for downstream handlers.
func ExtractAndInject() func(http.Handler) http.Handler {
	return Middleware
}

// ContextWithEntity is a convenience alias for WithEntityID.
func ContextWithEntity(ctx context.Context, entityID string) context.Context {
	return WithEntityID(ctx, entityID)
}
