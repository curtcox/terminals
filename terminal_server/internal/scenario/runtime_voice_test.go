package scenario

import (
	"context"
	"testing"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestRuntimeHandleVoiceTextUsesLLMIntentResolutionForAmbiguousInput(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1"})
	engine := NewEngine()
	engine.Register(Registration{Scenario: &TerminalScenario{}, Priority: PriorityNormal})
	llm := &testLLM{
		text: `{"action":"terminal","object":"","slots":{"device_id":"d1"},"scope":{}}`,
	}
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: ui.NewMemoryBroadcaster(),
		LLM:       llm,
	})

	name, err := runtime.HandleVoiceText(
		context.Background(),
		"d1",
		"could you open a terminal for me",
		time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("HandleVoiceText() error = %v", err)
	}
	if name != "terminal" {
		t.Fatalf("resolved scenario = %q, want terminal", name)
	}
	if len(llm.queries) != 1 {
		t.Fatalf("LLM query count = %d, want 1", len(llm.queries))
	}
}

func TestRuntimeHandleVoiceTextSkipsLLMForKnownIntent(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1"})
	engine := NewEngine()
	engine.Register(Registration{Scenario: &TerminalScenario{}, Priority: PriorityNormal})
	llm := &testLLM{
		text: `{"action":"red_alert","slots":{},"scope":{}}`,
	}
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: ui.NewMemoryBroadcaster(),
		LLM:       llm,
	})

	name, err := runtime.HandleVoiceText(
		context.Background(),
		"d1",
		"open terminal",
		time.Date(2026, 4, 15, 10, 1, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("HandleVoiceText() error = %v", err)
	}
	if name != "terminal" {
		t.Fatalf("scenario = %q, want terminal", name)
	}
	if len(llm.queries) != 0 {
		t.Fatalf("expected no LLM query for known intent, got %d", len(llm.queries))
	}
}

func TestRuntimeStopVoiceTextStandDownStopsRedAlert(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	broadcaster := ui.NewMemoryBroadcaster()

	engine := NewEngine()
	engine.Register(Registration{
		Scenario: AlertScenario{},
		Priority: PriorityCritical,
	})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: broadcaster,
	})

	if _, err := runtime.HandleVoiceText(
		context.Background(),
		"d1",
		"red alert",
		time.Date(2026, 4, 11, 21, 0, 0, 0, time.UTC),
	); err != nil {
		t.Fatalf("HandleVoiceText(red alert) error = %v", err)
	}

	stopped, err := runtime.StopVoiceText(
		context.Background(),
		"d1",
		"stand down",
		time.Date(2026, 4, 11, 21, 1, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("StopVoiceText(stand down) error = %v", err)
	}
	if stopped != "red_alert" {
		t.Fatalf("stopped scenario = %q, want red_alert", stopped)
	}
	if _, ok := engine.Active("d1"); ok {
		t.Fatalf("expected no active scenario after stand down stop")
	}
}

func TestRuntimeStopVoiceTextEndPAStopsPASystem(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	broadcaster := ui.NewMemoryBroadcaster()
	router := iorouter.NewRouter()

	engine := NewEngine()
	engine.Register(Registration{Scenario: &PASystemScenario{}, Priority: PriorityHigh})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		IO:        router,
		Broadcast: broadcaster,
	})

	if _, err := runtime.HandleVoiceText(
		context.Background(),
		"d1",
		"pa mode",
		time.Date(2026, 4, 11, 21, 0, 0, 0, time.UTC),
	); err != nil {
		t.Fatalf("HandleVoiceText(pa mode) error = %v", err)
	}
	if router.RouteCount() != 1 {
		t.Fatalf("route count after pa start = %d, want 1", router.RouteCount())
	}

	stopped, err := runtime.StopVoiceText(
		context.Background(),
		"d1",
		"end pa",
		time.Date(2026, 4, 11, 21, 1, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("StopVoiceText(end pa) error = %v", err)
	}
	if stopped != "pa_system" {
		t.Fatalf("stopped scenario = %q, want pa_system", stopped)
	}
	if _, ok := engine.Active("d1"); ok {
		t.Fatalf("expected no active scenario after end pa stop")
	}
}

func TestRuntimeHandleVoiceTerminalTargetsSourceDevice(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	broadcaster := ui.NewMemoryBroadcaster()

	engine := NewEngine()
	engine.Register(Registration{
		Scenario: &TerminalScenario{},
		Priority: PriorityNormal,
	})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: broadcaster,
	})

	name, err := runtime.HandleVoiceText(
		context.Background(),
		"d1",
		"open terminal",
		time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("HandleVoiceText() error = %v", err)
	}
	if name != "terminal" {
		t.Fatalf("scenario name = %q, want terminal", name)
	}

	events := broadcaster.Events()
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}
	if events[0].Message != "Terminal active" {
		t.Fatalf("event message = %q, want Terminal active", events[0].Message)
	}
	if len(events[0].DeviceIDs) != 1 || events[0].DeviceIDs[0] != "d1" {
		t.Fatalf("event device IDs = %+v, want [d1]", events[0].DeviceIDs)
	}
}

func TestRuntimeHandleVoiceIntercomCreatesAudioRoutes(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d3", DeviceName: "Office"})
	router := iorouter.NewRouter()
	broadcaster := ui.NewMemoryBroadcaster()

	engine := NewEngine()
	engine.Register(Registration{
		Scenario: &IntercomScenario{},
		Priority: PriorityHigh,
	})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		IO:        router,
		Broadcast: broadcaster,
	})

	name, err := runtime.HandleVoiceText(
		context.Background(),
		"d1",
		"intercom",
		time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("HandleVoiceText() error = %v", err)
	}
	if name != "intercom" {
		t.Fatalf("scenario name = %q, want intercom", name)
	}
	if router.RouteCount() != 4 {
		t.Fatalf("route count = %d, want 4", router.RouteCount())
	}
	events := broadcaster.Events()
	if len(events) != 1 || events[0].Message != "Intercom active" {
		t.Fatalf("unexpected broadcast events: %+v", events)
	}
	if len(events[0].DeviceIDs) != 1 || events[0].DeviceIDs[0] != "d1" {
		t.Fatalf("broadcast target = %+v, want [d1]", events[0].DeviceIDs)
	}
}

func TestRuntimeHandleVoiceAssistantAndPhoneCall(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	broadcaster := ui.NewMemoryBroadcaster()
	aiBackend := &testAIBackend{response: "assistant response"}
	telephony := &testTelephonyBridge{}

	engine := NewEngine()
	engine.Register(Registration{
		Scenario: &VoiceAssistantScenario{},
		Priority: PriorityNormal,
	})
	engine.Register(Registration{
		Scenario: &PhoneCallScenario{},
		Priority: PriorityHigh,
	})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: broadcaster,
		AI:        aiBackend,
		Telephony: telephony,
	})

	assistantName, err := runtime.HandleVoiceText(
		context.Background(),
		"d1",
		"assistant what is on my calendar",
		time.Date(2026, 4, 12, 0, 5, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("HandleVoiceText(assistant) error = %v", err)
	}
	if assistantName != "voice_assistant" {
		t.Fatalf("assistant scenario name = %q, want voice_assistant", assistantName)
	}
	if aiBackend.lastInput != "what is on my calendar" {
		t.Fatalf("assistant query = %q, want what is on my calendar", aiBackend.lastInput)
	}

	callName, err := runtime.HandleVoiceText(
		context.Background(),
		"d1",
		"call 5551212",
		time.Date(2026, 4, 12, 0, 6, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("HandleVoiceText(call) error = %v", err)
	}
	if callName != "phone_call" {
		t.Fatalf("call scenario name = %q, want phone_call", callName)
	}
	if telephony.lastTarget != "5551212" {
		t.Fatalf("last telephony target = %q, want 5551212", telephony.lastTarget)
	}

	events := broadcaster.Events()
	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}
	if events[0].Message != "assistant response" {
		t.Fatalf("assistant event message = %q, want assistant response", events[0].Message)
	}
	if events[1].Message != "Calling 5551212" {
		t.Fatalf("call event message = %q, want Calling 5551212", events[1].Message)
	}
}

func TestRuntimeHandleVoiceInternalVideoCallCreatesBidirectionalAVRoutes(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	router := iorouter.NewRouter()
	broadcaster := ui.NewMemoryBroadcaster()

	engine := NewEngine()
	engine.Register(Registration{
		Scenario: &InternalVideoCallScenario{},
		Priority: PriorityHigh,
	})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		IO:        router,
		Broadcast: broadcaster,
	})

	name, err := runtime.HandleVoiceText(
		context.Background(),
		"d1",
		"video call d2",
		time.Date(2026, 4, 12, 0, 7, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("HandleVoiceText(video call) error = %v", err)
	}
	if name != "internal_video_call" {
		t.Fatalf("scenario name = %q, want internal_video_call", name)
	}
	if router.RouteCount() != 4 {
		t.Fatalf("route count = %d, want 4", router.RouteCount())
	}

	events := broadcaster.Events()
	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}
	if events[0].Message != "Video call active: d2" {
		t.Fatalf("source event message = %q, want Video call active: d2", events[0].Message)
	}
	if len(events[0].DeviceIDs) != 1 || events[0].DeviceIDs[0] != "d1" {
		t.Fatalf("source event device IDs = %+v, want [d1]", events[0].DeviceIDs)
	}
	if events[1].Message != "Incoming video call: d1" {
		t.Fatalf("target event message = %q, want Incoming video call: d1", events[1].Message)
	}
	if len(events[1].DeviceIDs) != 1 || events[1].DeviceIDs[0] != "d2" {
		t.Fatalf("target event device IDs = %+v, want [d2]", events[1].DeviceIDs)
	}
}
