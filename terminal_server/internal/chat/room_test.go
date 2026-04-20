package chat

import (
	"testing"
	"time"
)

func TestPostRecordsMessage(t *testing.T) {
	r := NewRoom()
	msg, ok := r.Post("dev-1", "alice", "hello")
	if !ok {
		t.Fatalf("Post ok = false, want true")
	}
	if msg.DeviceID != "dev-1" || msg.Name != "alice" || msg.Text != "hello" {
		t.Fatalf("message = %+v", msg)
	}
	if got := r.Messages(); len(got) != 1 || got[0].ID != msg.ID {
		t.Fatalf("Messages() = %+v, want one with ID %q", got, msg.ID)
	}
}

func TestPostRejectsEmpty(t *testing.T) {
	r := NewRoom()
	if _, ok := r.Post("dev-1", "alice", "   "); ok {
		t.Fatalf("Post returned ok for empty text")
	}
	if _, ok := r.Post("", "alice", "hi"); ok {
		t.Fatalf("Post returned ok for empty device")
	}
	if got := r.Messages(); len(got) != 0 {
		t.Fatalf("Messages() = %+v, want empty", got)
	}
}

func TestSetNameIsStickyAcrossPosts(t *testing.T) {
	r := NewRoom()
	r.SetName("dev-1", "alice")
	msg, _ := r.Post("dev-1", "", "hi")
	if msg.Name != "alice" {
		t.Fatalf("Name = %q, want alice", msg.Name)
	}
}

func TestPostWithoutNameFallsBackToDeviceID(t *testing.T) {
	r := NewRoom()
	msg, _ := r.Post("dev-1", "", "hi")
	if msg.Name != "dev-1" {
		t.Fatalf("Name = %q, want dev-1", msg.Name)
	}
}

func TestDuplicateNamesAllowed(t *testing.T) {
	r := NewRoom()
	r.SetName("dev-1", "alice")
	r.SetName("dev-2", "alice")
	if r.Name("dev-1") != "alice" || r.Name("dev-2") != "alice" {
		t.Fatalf("duplicate names should be allowed")
	}
	r.Post("dev-1", "alice", "hi from 1")
	r.Post("dev-2", "alice", "hi from 2")
	msgs := r.Messages()
	if len(msgs) != 2 {
		t.Fatalf("messages = %d, want 2", len(msgs))
	}
	if msgs[0].DeviceID == msgs[1].DeviceID {
		t.Fatalf("device ids collapsed: %+v", msgs)
	}
}

func TestRetentionTrimsOldMessages(t *testing.T) {
	r := NewRoomWithRetention(time.Hour)
	fake := time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC)
	r.now = func() time.Time { return fake }
	r.Post("dev-1", "alice", "old")

	fake = fake.Add(2 * time.Hour)
	r.Post("dev-1", "alice", "new")

	msgs := r.Messages()
	if len(msgs) != 1 {
		t.Fatalf("messages = %d, want 1 (old trimmed)", len(msgs))
	}
	if msgs[0].Text != "new" {
		t.Fatalf("retained text = %q, want new", msgs[0].Text)
	}
}

func TestNewJoinerSeesPriorHistory(t *testing.T) {
	r := NewRoom()
	r.Post("dev-1", "alice", "before bob joined")
	r.Join("dev-2")
	msgs := r.Messages()
	if len(msgs) != 1 || msgs[0].Text != "before bob joined" {
		t.Fatalf("joiner saw %+v, want prior history", msgs)
	}
}

func TestJoinReturnsExistingIdentity(t *testing.T) {
	r := NewRoom()
	r.SetName("dev-1", "alice")
	if name := r.Join("dev-1"); name != "alice" {
		t.Fatalf("Join returned %q, want alice", name)
	}
	if name := r.Join("dev-2"); name != "" {
		t.Fatalf("Join on unnamed device returned %q, want empty", name)
	}
}

func TestParticipantsSortedAndExcludeLeft(t *testing.T) {
	r := NewRoom()
	r.Join("dev-b")
	r.Join("dev-a")
	r.Join("dev-c")
	r.Leave("dev-b")
	parts := r.Participants()
	want := []string{"dev-a", "dev-c"}
	if len(parts) != len(want) {
		t.Fatalf("participants = %v, want %v", parts, want)
	}
	for i, id := range want {
		if parts[i] != id {
			t.Fatalf("participants[%d] = %q, want %q", i, parts[i], id)
		}
	}
}
