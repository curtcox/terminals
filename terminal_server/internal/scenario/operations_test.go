package scenario

import (
	"context"
	"testing"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestExecuteOperationsValidationPreventsPartialSideEffects(t *testing.T) {
	broadcaster := ui.NewMemoryBroadcaster()
	env := &Environment{
		Broadcast: broadcaster,
		Scheduler: storage.NewMemoryScheduler(),
	}

	err := ExecuteOperations(context.Background(), env, []Operation{
		{Kind: OperationBroadcastNotify, Target: "d1", Args: map[string]string{"message": "hello"}},
		{Kind: OperationSchedulerAfter, Target: "", Args: map[string]string{"unix_ms": "123"}},
	}, time.Date(2026, 4, 12, 9, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatalf("ExecuteOperations() error = nil, want validation error")
	}
	if events := broadcaster.Events(); len(events) != 0 {
		t.Fatalf("broadcast events = %+v, want none", events)
	}
	if due := env.Scheduler.Due(123); len(due) != 0 {
		t.Fatalf("scheduled due = %+v, want none", due)
	}
}

func TestExecuteOperationsSchedulerAndBroadcast(t *testing.T) {
	scheduler := storage.NewMemoryScheduler()
	broadcaster := ui.NewMemoryBroadcaster()
	env := &Environment{Scheduler: scheduler, Broadcast: broadcaster}

	err := ExecuteOperations(context.Background(), env, []Operation{
		{Kind: OperationSchedulerAfter, Target: "timer-1", Args: map[string]string{
			"unix_ms":          "123",
			"kind":             "timer",
			"device_id":        "d1",
			"subject":          "pasta",
			"duration_seconds": "600",
		}},
		{Kind: OperationBroadcastNotify, Target: "d1", Args: map[string]string{"message": "Timer set"}},
	}, time.Date(2026, 4, 12, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ExecuteOperations() error = %v", err)
	}

	records := scheduler.DueRecords(123)
	if len(records) != 1 || records[0].Kind != "timer" || records[0].Payload["duration_seconds"] != "600" {
		t.Fatalf("records = %+v, want structured timer", records)
	}
	events := broadcaster.Events()
	if len(events) != 1 || events[0].Message != "Timer set" {
		t.Fatalf("broadcast events = %+v, want Timer set", events)
	}
}

func TestExecuteOperationsTTSAndBusEmit(t *testing.T) {
	tts := &testTTS{}
	bus := NewIntentEventBus()
	env := &Environment{TTS: tts, TriggerBus: bus}
	events, cancel := bus.Subscribe(1)
	defer cancel()

	err := ExecuteOperations(context.Background(), env, []Operation{
		{Kind: OperationAITTS, Args: map[string]string{"text": "Your pasta is ready."}},
		{Kind: OperationBusEmit, Target: "timer.expired", Args: map[string]string{"subject": "pasta", "duration_seconds": "600"}},
	}, time.Date(2026, 4, 12, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ExecuteOperations() error = %v", err)
	}
	if len(tts.calls) != 1 || tts.calls[0] != "Your pasta is ready." {
		t.Fatalf("TTS calls = %+v", tts.calls)
	}
	select {
	case event := <-events:
		if event.EventV2 == nil || event.EventV2.Kind != "timer.expired" || event.EventV2.Subject != "pasta" || event.EventV2.Attributes["duration_seconds"] != "600" {
			t.Fatalf("bus event = %+v", event)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("expected bus event")
	}
}

func TestExecuteOperationsUISetPatchAndClear(t *testing.T) {
	uiHost := ui.NewMemoryHost()
	root := ui.New("stack", map[string]string{"id": "timer"})
	patch := ui.New("text", map[string]string{"id": "remaining", "value": "00:59"})
	env := &Environment{UI: uiHost}

	err := ExecuteOperations(context.Background(), env, []Operation{
		{Kind: OperationUISet, Target: "kitchen-screen", Node: &root},
		{Kind: OperationUIPatch, Target: "kitchen-screen", Args: map[string]string{"component_id": "remaining"}, Node: &patch},
		{Kind: OperationUIClear, Target: "kitchen-screen", Args: map[string]string{"root": "timer"}},
	}, time.Date(2026, 4, 12, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ExecuteOperations() error = %v", err)
	}
	events := uiHost.Events()
	if len(events) != 3 {
		t.Fatalf("len(events) = %d, want 3", len(events))
	}
	if events[0].Kind != "set" || events[1].ComponentID != "remaining" || events[2].Root != "timer" {
		t.Fatalf("unexpected UI events: %+v", events)
	}
}
