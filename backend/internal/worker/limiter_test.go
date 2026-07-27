// SPDX-License-Identifier: MIT

package worker

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestBackgroundLimiterTimesOutInsteadOfWaitingIndefinitely(t *testing.T) {
	limiter := newBackgroundLimiter(1, 20*time.Millisecond)
	release, err := limiter.acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	start := time.Now()
	_, err = limiter.acquire(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("acquire error = %v, want deadline exceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("acquire took %s, want fast failure", elapsed)
	}
}
