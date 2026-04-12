package storage

import (
	"context"
	"sort"
	"sync"
)

// ScheduledItem is a simple scheduled timestamp entry.
type ScheduledItem struct {
	Key    string
	UnixMS int64
}

// MemoryScheduler stores scheduled actions in memory.
type MemoryScheduler struct {
	mu    sync.RWMutex
	items map[string]int64
}

// NewMemoryScheduler creates an empty scheduler.
func NewMemoryScheduler() *MemoryScheduler {
	return &MemoryScheduler{
		items: make(map[string]int64),
	}
}

// Schedule records or replaces a scheduled timestamp by key.
func (s *MemoryScheduler) Schedule(_ context.Context, key string, unixMS int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = unixMS
	return nil
}

// List returns all scheduled entries in key order for deterministic behavior.
func (s *MemoryScheduler) List() []ScheduledItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]ScheduledItem, 0, len(s.items))
	for k, v := range s.items {
		out = append(out, ScheduledItem{Key: k, UnixMS: v})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Key < out[j].Key
	})
	return out
}

// Due returns scheduled keys with trigger times at or before unixMS.
func (s *MemoryScheduler) Due(unixMS int64) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]string, 0)
	for k, ts := range s.items {
		if ts <= unixMS {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// Remove deletes a scheduled item by key.
func (s *MemoryScheduler) Remove(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, key)
	return nil
}
