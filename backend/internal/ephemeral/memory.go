// SPDX-License-Identifier: MIT

package ephemeral

import (
	"context"
	"sync"
	"time"
)

type MemoryStore struct {
	mu    sync.Mutex
	now   func() time.Time
	items map[string]memoryItem
}

type memoryItem struct {
	value     []byte
	count     int64
	expiresAt time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		now:   time.Now,
		items: map[string]memoryItem{},
	}
}

func (s *MemoryStore) Increment(_ context.Context, key string, ttl time.Duration) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	item := s.items[key]
	if item.expiresAt.IsZero() || !now.Before(item.expiresAt) {
		item = memoryItem{expiresAt: now.Add(ttl)}
	}
	item.count++
	s.items[key] = item
	return item.count, nil
}

func (s *MemoryStore) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cloned := append([]byte(nil), value...)
	s.items[key] = memoryItem{value: cloned, expiresAt: s.now().Add(ttl)}
	return nil
}

func (s *MemoryStore) Get(_ context.Context, key string) ([]byte, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[key]
	if !ok {
		return nil, false, nil
	}
	if !s.now().Before(item.expiresAt) {
		delete(s.items, key)
		return nil, false, nil
	}
	return append([]byte(nil), item.value...), true, nil
}

func (s *MemoryStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, key)
	return nil
}

func (s *MemoryStore) Close() error {
	return nil
}
