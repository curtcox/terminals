package terminal

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestManagerStartRequiresDeviceID(t *testing.T) {
	t.Parallel()

	m := NewManager()
	_, err := m.Start(context.Background(), StartOptions{})
	if !errors.Is(err, ErrMissingDeviceID) {
		t.Fatalf("Start() error = %v, want %v", err, ErrMissingDeviceID)
	}
}

func TestManagerStartWriteReadClose(t *testing.T) {
	t.Parallel()

	m := NewManager()
	session, err := m.Start(context.Background(), StartOptions{
		DeviceID: "device-1",
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer m.CloseAll()

	if len(m.List()) != 1 {
		t.Fatalf("len(List()) = %d, want 1", len(m.List()))
	}

	if err := m.Write(session.ID, []byte("echo codex-terminal\n")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	combined := ""
	for time.Now().Before(deadline) {
		out, err := m.ReadAvailable(session.ID, 4096)
		if err != nil {
			t.Fatalf("ReadAvailable() error = %v", err)
		}
		if len(out) > 0 {
			combined += string(out)
			if strings.Contains(combined, "codex-terminal") {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}

	if !strings.Contains(combined, "codex-terminal") {
		t.Fatalf("terminal output missing marker, got: %q", combined)
	}

	if err := m.Close(session.ID); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if len(m.List()) != 0 {
		t.Fatalf("len(List()) = %d, want 0", len(m.List()))
	}
}

func TestManagerUnknownSession(t *testing.T) {
	t.Parallel()

	m := NewManager()
	if err := m.Write("missing", []byte("x")); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("Write() error = %v, want %v", err, ErrSessionNotFound)
	}
	if _, err := m.ReadAvailable("missing", 10); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("ReadAvailable() error = %v, want %v", err, ErrSessionNotFound)
	}
	if err := m.Close("missing"); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("Close() error = %v, want %v", err, ErrSessionNotFound)
	}
}
