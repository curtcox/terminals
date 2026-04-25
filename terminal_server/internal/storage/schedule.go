package storage

import (
	"context"
	"sort"
	"strings"
	"sync"
)

// ScheduledItem is a simple scheduled timestamp entry.
type ScheduledItem struct {
	Key    string
	UnixMS int64
}

// ScheduleRecord stores a scheduled item with typed metadata.
type ScheduleRecord struct {
	Key       string
	Kind      string
	Subject   string
	DeviceID  string
	UnixMS    int64
	Payload   map[string]string
	CreatedMS int64
}

// MemoryScheduler stores scheduled actions in memory.
type MemoryScheduler struct {
	mu      sync.RWMutex
	records map[string]ScheduleRecord
}

// NewMemoryScheduler creates an empty scheduler.
func NewMemoryScheduler() *MemoryScheduler {
	return &MemoryScheduler{
		records: make(map[string]ScheduleRecord),
	}
}

// Schedule records or replaces a scheduled timestamp by key.
func (s *MemoryScheduler) Schedule(ctx context.Context, key string, unixMS int64) error {
	return s.ScheduleRecord(ctx, ScheduleRecord{
		Key:    key,
		Kind:   inferScheduleKind(key),
		UnixMS: unixMS,
	})
}

// ScheduleRecord records or replaces a structured scheduled item by key.
func (s *MemoryScheduler) ScheduleRecord(_ context.Context, record ScheduleRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	record.Payload = copyPayload(record.Payload)
	s.records[record.Key] = record
	return nil
}

// List returns all scheduled entries in key order for deterministic behavior.
func (s *MemoryScheduler) List() []ScheduledItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]ScheduledItem, 0, len(s.records))
	for k, record := range s.records {
		out = append(out, ScheduledItem{Key: k, UnixMS: record.UnixMS})
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
	for k, record := range s.records {
		if record.UnixMS <= unixMS {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// DueRecords returns scheduled records with trigger times at or before unixMS.
func (s *MemoryScheduler) DueRecords(unixMS int64) []ScheduleRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]ScheduleRecord, 0)
	for _, record := range s.records {
		if record.UnixMS <= unixMS {
			out = append(out, cloneRecord(record))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Key < out[j].Key
	})
	return out
}

// Remove deletes a scheduled item by key.
func (s *MemoryScheduler) Remove(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records, key)
	return nil
}

func inferScheduleKind(key string) string {
	prefix, _, ok := strings.Cut(key, ":")
	if !ok {
		return ""
	}
	switch prefix {
	case "timer", "timer_tick", "schedule_monitor":
		if prefix == "timer_tick" {
			return "timer.tick"
		}
		return prefix
	default:
		return ""
	}
}

func cloneRecord(record ScheduleRecord) ScheduleRecord {
	record.Payload = copyPayload(record.Payload)
	return record
}

func copyPayload(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
