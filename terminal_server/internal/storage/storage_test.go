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

func TestMemorySchedulerScheduleRecordRoundTrip(t *testing.T) {
	s := NewMemoryScheduler()
	record := ScheduleRecord{
		Key:       "timer:device-1:123",
		Kind:      "timer",
		Subject:   "pasta",
		DeviceID:  "device-1",
		UnixMS:    123,
		Payload:   map[string]string{"duration_seconds": "600"},
		CreatedMS: 10,
	}

	if err := s.ScheduleRecord(context.Background(), record); err != nil {
		t.Fatalf("ScheduleRecord() error = %v", err)
	}

	due := s.DueRecords(123)
	if len(due) != 1 {
		t.Fatalf("len(DueRecords()) = %d, want 1", len(due))
	}
	got := due[0]
	if got.Key != record.Key || got.Kind != "timer" || got.Subject != "pasta" || got.DeviceID != "device-1" || got.UnixMS != 123 || got.CreatedMS != 10 {
		t.Fatalf("DueRecords()[0] = %+v", got)
	}
	if got.Payload["duration_seconds"] != "600" {
		t.Fatalf("payload = %+v, want duration_seconds=600", got.Payload)
	}

	got.Payload["duration_seconds"] = "changed"
	again := s.DueRecords(123)
	if again[0].Payload["duration_seconds"] != "600" {
		t.Fatalf("DueRecords() exposed mutable payload: %+v", again[0].Payload)
	}
}

func TestMemorySchedulerScheduleCompatibilityWritesRecord(t *testing.T) {
	s := NewMemoryScheduler()
	if err := s.Schedule(context.Background(), "timer:device-1:123", 123); err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}

	due := s.DueRecords(123)
	if len(due) != 1 {
		t.Fatalf("len(DueRecords()) = %d, want 1", len(due))
	}
	if due[0].Key != "timer:device-1:123" || due[0].Kind != "timer" || due[0].UnixMS != 123 {
		t.Fatalf("DueRecords()[0] = %+v", due[0])
	}
}

func TestMemorySchedulerDueRecordsOrderAndRemove(t *testing.T) {
	s := NewMemoryScheduler()
	_ = s.ScheduleRecord(context.Background(), ScheduleRecord{Key: "timer-b", Kind: "timer", UnixMS: 100})
	_ = s.ScheduleRecord(context.Background(), ScheduleRecord{Key: "timer-a", Kind: "timer", UnixMS: 100})
	_ = s.ScheduleRecord(context.Background(), ScheduleRecord{Key: "timer-c", Kind: "timer", UnixMS: 200})

	due := s.DueRecords(100)
	if len(due) != 2 || due[0].Key != "timer-a" || due[1].Key != "timer-b" {
		t.Fatalf("DueRecords(100) = %+v", due)
	}

	if err := s.Remove(context.Background(), "timer-a"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	due = s.DueRecords(100)
	if len(due) != 1 || due[0].Key != "timer-b" {
		t.Fatalf("DueRecords(100) after remove = %+v", due)
	}
}
