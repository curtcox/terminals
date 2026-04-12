package storage

import (
	"context"
	"testing"
)

func TestMemoryStorePutGet(t *testing.T) {
	s := NewMemoryStore()
	if err := s.Put(context.Background(), "hello", "world"); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	got, err := s.Get(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != "world" {
		t.Fatalf("Get() = %q, want world", got)
	}
}

func TestMemoryStoreNotFound(t *testing.T) {
	s := NewMemoryStore()
	if _, err := s.Get(context.Background(), "missing"); err != ErrNotFound {
		t.Fatalf("Get() error = %v, want %v", err, ErrNotFound)
	}
}

func TestMemorySchedulerSchedule(t *testing.T) {
	s := NewMemoryScheduler()
	if err := s.Schedule(context.Background(), "timer-1", 12345); err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}
	items := s.List()
	if len(items) != 1 {
		t.Fatalf("len(List()) = %d, want 1", len(items))
	}
	if items[0].Key != "timer-1" || items[0].UnixMS != 12345 {
		t.Fatalf("List()[0] = %+v", items[0])
	}
}
