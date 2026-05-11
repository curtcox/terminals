package scenario

import (
	"context"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestRuntimeIntercomPreemptedByRedAlertSuspendsAndResumesRoutes(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d3", DeviceName: "Office"})
	broadcaster := ui.NewMemoryBroadcaster()
	router := iorouter.NewRouter()

	engine := NewEngine()
	engine.Register(Registration{Scenario: &IntercomScenario{}, Priority: PriorityHigh})
	engine.Register(Registration{Scenario: AlertScenario{}, Priority: PriorityCritical})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		IO:        router,
		Broadcast: broadcaster,
	})

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "intercom",
	}); err != nil {
		t.Fatalf("HandleTrigger(intercom) error = %v", err)
	}
	if got := router.RouteCount(); got != 4 {
		t.Fatalf("route count after intercom start = %d, want 4", got)
	}

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "red_alert",
	}); err != nil {
		t.Fatalf("HandleTrigger(red_alert) error = %v", err)
	}
	if got := router.RouteCount(); got != 0 {
		t.Fatalf("route count after red_alert preemption = %d, want 0", got)
	}

	if _, err := runtime.StopTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "red_alert",
	}); err != nil {
		t.Fatalf("StopTrigger(red_alert) error = %v", err)
	}
	if got := router.RouteCount(); got != 4 {
		t.Fatalf("route count after red_alert stop resume = %d, want 4", got)
	}
}

func TestRuntimePAPreemptedByRedAlertSuspendsAndResumesRoutes(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d3", DeviceName: "Office"})
	broadcaster := ui.NewMemoryBroadcaster()
	router := iorouter.NewRouter()

	engine := NewEngine()
	engine.Register(Registration{Scenario: &PASystemScenario{}, Priority: PriorityHigh})
	engine.Register(Registration{Scenario: AlertScenario{}, Priority: PriorityCritical})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		IO:        router,
		Broadcast: broadcaster,
	})

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "pa_system",
	}); err != nil {
		t.Fatalf("HandleTrigger(pa_system) error = %v", err)
	}
	if got := router.RouteCount(); got != 2 {
		t.Fatalf("route count after pa start = %d, want 2", got)
	}

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "red_alert",
	}); err != nil {
		t.Fatalf("HandleTrigger(red_alert) error = %v", err)
	}
	if got := router.RouteCount(); got != 0 {
		t.Fatalf("route count after red_alert preemption = %d, want 0", got)
	}

	if _, err := runtime.StopTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "red_alert",
	}); err != nil {
		t.Fatalf("StopTrigger(red_alert) error = %v", err)
	}
	if got := router.RouteCount(); got != 2 {
		t.Fatalf("route count after red_alert stop resume = %d, want 2", got)
	}
}

func TestRuntimePAClaimsEndpointScopedResourcesWhenAvailable(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{
		DeviceID:   "d1",
		DeviceName: "Kitchen",
		Capabilities: device.CapabilitySet{
			"microphone.endpoint_count": "1",
			"microphone.endpoint.0.id":  "Mic Main",
		},
	})
	_, _ = devices.Register(device.Manifest{
		DeviceID:   "d2",
		DeviceName: "Hall",
		Capabilities: device.CapabilitySet{
			"speakers.endpoint_count": "1",
			"speakers.endpoint.0.id":  "Hall Speaker",
		},
	})
	router := iorouter.NewRouter()

	engine := NewEngine()
	engine.Register(Registration{Scenario: &PASystemScenario{}, Priority: PriorityHigh})
	runtime := NewRuntime(engine, &Environment{
		Devices: devices,
		IO:      router,
	})

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "pa_system",
	}); err != nil {
		t.Fatalf("HandleTrigger(pa_system) error = %v", err)
	}

	sourceClaims := router.Claims().Snapshot("d1")
	if len(sourceClaims) != 1 {
		t.Fatalf("source claims len = %d, want 1", len(sourceClaims))
	}
	if sourceClaims[0].Resource != "audio_in.mic-main.capture" {
		t.Fatalf("source claim resource = %q, want audio_in.mic-main.capture", sourceClaims[0].Resource)
	}
	targetClaims := router.Claims().Snapshot("d2")
	if len(targetClaims) != 1 {
		t.Fatalf("target claims len = %d, want 1", len(targetClaims))
	}
	if targetClaims[0].Resource != "audio_out.hall-speaker" {
		t.Fatalf("target claim resource = %q, want audio_out.hall-speaker", targetClaims[0].Resource)
	}
}

func TestRuntimeNestedPreemptionSoak(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	router := iorouter.NewRouter()
	broadcaster := ui.NewMemoryBroadcaster()

	engine := NewEngine()
	RegisterBuiltins(engine)
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		IO:        router,
		Broadcast: broadcaster,
	})

	for i := 0; i < 25; i++ {
		if _, err := runtime.HandleTrigger(context.Background(), Trigger{
			Kind:     TriggerManual,
			SourceID: "d1",
			Intent:   "photo frame",
		}); err != nil {
			t.Fatalf("iteration %d photo_frame error = %v", i, err)
		}
		if _, err := runtime.HandleTrigger(context.Background(), Trigger{
			Kind:     TriggerManual,
			SourceID: "d1",
			Intent:   "voice_assistant",
		}); err != nil {
			t.Fatalf("iteration %d voice_assistant error = %v", i, err)
		}
		if _, err := runtime.HandleTrigger(context.Background(), Trigger{
			Kind:     TriggerManual,
			SourceID: "d1",
			Intent:   "pa_system",
		}); err != nil {
			t.Fatalf("iteration %d pa_system error = %v", i, err)
		}
		if _, err := runtime.HandleTrigger(context.Background(), Trigger{
			Kind:     TriggerManual,
			SourceID: "d1",
			Intent:   "red_alert",
		}); err != nil {
			t.Fatalf("iteration %d red_alert start error = %v", i, err)
		}
		if _, err := runtime.StopTrigger(context.Background(), Trigger{
			Kind:     TriggerManual,
			SourceID: "d1",
			Intent:   "red_alert",
		}); err != nil {
			t.Fatalf("iteration %d red_alert stop error = %v", i, err)
		}
	}
}
