// SPDX-License-Identifier: MIT

package ephemeral

import (
	"context"
	"testing"
	"time"
)

func TestMemoryStoreIncrementExpires(t *testing.T) {
	store := NewMemoryStore()
	now := time.Unix(1_800_000_000, 0)
	store.now = func() time.Time { return now }

	count, err := store.Increment(context.Background(), "rate:test", time.Minute)
	if err != nil {
		t.Fatalf("Increment first: %v", err)
	}
	if count != 1 {
		t.Fatalf("first count = %d, want 1", count)
	}
	count, err = store.Increment(context.Background(), "rate:test", time.Minute)
	if err != nil {
		t.Fatalf("Increment second: %v", err)
	}
	if count != 2 {
		t.Fatalf("second count = %d, want 2", count)
	}

	now = now.Add(time.Minute + time.Second)
	count, err = store.Increment(context.Background(), "rate:test", time.Minute)
	if err != nil {
		t.Fatalf("Increment after expiry: %v", err)
	}
	if count != 1 {
		t.Fatalf("expired count = %d, want 1", count)
	}
}

func TestCheckLimit(t *testing.T) {
	store := NewMemoryStore()

	for i := 0; i < 2; i++ {
		result, err := CheckLimit(context.Background(), store, "rate:test", 2, time.Minute)
		if err != nil {
			t.Fatalf("CheckLimit allowed: %v", err)
		}
		if !result.Allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	result, err := CheckLimit(context.Background(), store, "rate:test", 2, time.Minute)
	if err != nil {
		t.Fatalf("CheckLimit denied: %v", err)
	}
	if result.Allowed {
		t.Fatal("third request should be denied")
	}
}
