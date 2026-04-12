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

func TestMemorySchedulerDueAndRemove(t *testing.T) {
	s := NewMemoryScheduler()
	_ = s.Schedule(context.Background(), "timer-2", 200)
	_ = s.Schedule(context.Background(), "timer-1", 100)
	_ = s.Schedule(context.Background(), "timer-3", 300)

	due := s.Due(200)
	if len(due) != 2 {
		t.Fatalf("len(Due(200)) = %d, want 2", len(due))
	}
	if due[0] != "timer-1" || due[1] != "timer-2" {
		t.Fatalf("Due(200) order = %+v", due)
	}

	if err := s.Remove(context.Background(), "timer-1"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	due = s.Due(200)
	if len(due) != 1 || due[0] != "timer-2" {
		t.Fatalf("Due(200) after remove = %+v", due)
	}
}
