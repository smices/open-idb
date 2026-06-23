// SPDX-License-Identifier: MIT

package auth

import "context"

type adminSessionContextKey struct{}

func WithAdminSession(ctx context.Context, session AdminSession) context.Context {
	return context.WithValue(ctx, adminSessionContextKey{}, session)
}

func AdminSessionFromContext(ctx context.Context) (AdminSession, bool) {
	session, ok := ctx.Value(adminSessionContextKey{}).(AdminSession)
	return session, ok
}
