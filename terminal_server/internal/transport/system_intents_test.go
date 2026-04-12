package transport

import (
	"strings"
	"testing"
)

func TestSystemHelpIntentsString(t *testing.T) {
	s := SystemHelpIntentsString()
	if s == "" {
		t.Fatalf("SystemHelpIntentsString() should not be empty")
	}
	if !strings.Contains(s, SystemIntentRecentCommands) {
		t.Fatalf("SystemHelpIntentsString() missing %q", SystemIntentRecentCommands)
	}
}

func TestParseSystemIntent(t *testing.T) {
	got, err := ParseSystemIntent("  " + SystemIntentServerStatus + " ")
	if err != nil {
		t.Fatalf("ParseSystemIntent(server_status) error = %v", err)
	}
	if got.Name != SystemIntentServerStatus {
		t.Fatalf("Name = %q, want %q", got.Name, SystemIntentServerStatus)
	}

	device, err := ParseSystemIntent(SystemIntentDeviceStatus + " device-1")
	if err != nil {
		t.Fatalf("ParseSystemIntent(device_status) error = %v", err)
	}
	if device.Name != SystemIntentDeviceStatus || device.Arg != "device-1" {
		t.Fatalf("unexpected parsed device intent: %+v", device)
	}

	reconcile, err := ParseSystemIntent(SystemIntentReconcileLiveness + " 30")
	if err != nil {
		t.Fatalf("ParseSystemIntent(reconcile_liveness) error = %v", err)
	}
	if reconcile.Name != SystemIntentReconcileLiveness || reconcile.Arg != "30" {
		t.Fatalf("unexpected parsed reconcile intent: %+v", reconcile)
	}
}

func TestParseSystemIntentErrors(t *testing.T) {
	if _, err := ParseSystemIntent("   "); err != ErrMissingCommandIntent {
		t.Fatalf("expected ErrMissingCommandIntent, got %v", err)
	}
	if _, err := ParseSystemIntent(SystemIntentDeviceStatus + "   "); err == nil {
		t.Fatalf("expected device_status arg error")
	}
	if _, err := ParseSystemIntent("unknown_thing"); err == nil {
		t.Fatalf("expected unknown intent error")
	}
}
