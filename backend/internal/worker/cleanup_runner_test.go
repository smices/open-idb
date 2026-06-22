// SPDX-License-Identifier: MIT

package worker

import (
	"context"
	"testing"
)

func TestCleanupRunnerRunOnce(t *testing.T) {
	queries := &fakeCleanupQueries{
		codes:    2,
		tokens:   3,
		sessions: 4,
	}
	runner := NewCleanupRunner(queries, 0, nil)

	result, err := runner.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if result.AuthorizationCodesDeleted != 2 {
		t.Fatalf("AuthorizationCodesDeleted = %d, want 2", result.AuthorizationCodesDeleted)
	}
	if result.OAuthTokensDeleted != 3 {
		t.Fatalf("OAuthTokensDeleted = %d, want 3", result.OAuthTokensDeleted)
	}
	if result.SessionsExpired != 4 {
		t.Fatalf("SessionsExpired = %d, want 4", result.SessionsExpired)
	}
	if queries.calls != 3 {
		t.Fatalf("calls = %d, want 3", queries.calls)
	}
}

type fakeCleanupQueries struct {
	codes    int64
	tokens   int64
	sessions int64
	calls    int
}

func (f *fakeCleanupQueries) DeleteExpiredAuthorizationCodes(context.Context) (int64, error) {
	f.calls++
	return f.codes, nil
}

func (f *fakeCleanupQueries) DeleteExpiredOAuthTokens(context.Context) (int64, error) {
	f.calls++
	return f.tokens, nil
}

func (f *fakeCleanupQueries) MarkExpiredSessions(context.Context) (int64, error) {
	f.calls++
	return f.sessions, nil
}
