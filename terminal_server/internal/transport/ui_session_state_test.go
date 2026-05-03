package transport

import (
	"reflect"
	"sync"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestUISessionState_RememberAndRecall(t *testing.T) {
	s := NewUISessionState()

	if _, ok := s.LastSetUI("dev-a"); ok {
		t.Fatalf("expected no UI for fresh device")
	}

	descA := ui.HelloWorld("dev-a-name")
	descB := ui.HelloWorld("dev-b-name")
	s.RememberSetUI("dev-a", []ServerMessage{
		{SetUI: &descA},
	})
	s.RememberSetUI("dev-b", []ServerMessage{
		{SetUI: &descB},
	})

	gotA, ok := s.LastSetUI("dev-a")
	if !ok || !reflect.DeepEqual(gotA, descA) {
		t.Fatalf("expected descA for dev-a, got %+v ok=%v", gotA, ok)
	}
	gotB, ok := s.LastSetUI("dev-b")
	if !ok || !reflect.DeepEqual(gotB, descB) {
		t.Fatalf("expected descB for dev-b, got %+v ok=%v", gotB, ok)
	}
	if _, ok := s.LastSetUI("dev-c"); ok {
		t.Fatalf("expected no UI for unrelated device")
	}
}

func TestUISessionState_RememberSetUIIgnoresRelays(t *testing.T) {
	s := NewUISessionState()
	desc := ui.HelloWorld("relayed")
	s.RememberSetUI("dev-a", []ServerMessage{
		{SetUI: &desc, RelayToDeviceID: "dev-other"},
	})
	if _, ok := s.LastSetUI("dev-a"); ok {
		t.Fatalf("relayed SetUI should not be remembered for sender")
	}
}

func TestUISessionState_RememberSetUIBlankDeviceNoop(t *testing.T) {
	s := NewUISessionState()
	desc := ui.HelloWorld("x")
	s.RememberSetUI("   ", []ServerMessage{{SetUI: &desc}})
	if _, ok := s.LastSetUI(""); ok {
		t.Fatalf("blank device must not store anything")
	}
}

func TestUISessionState_RememberSetUITakesLastSetUIInSlice(t *testing.T) {
	s := NewUISessionState()
	first := ui.HelloWorld("first")
	second := ui.HelloWorld("second")
	s.RememberSetUI("dev-a", []ServerMessage{
		{SetUI: &first},
		{SetUI: &second},
	})
	got, ok := s.LastSetUI("dev-a")
	if !ok || !reflect.DeepEqual(got, second) {
		t.Fatalf("expected second SetUI to win, got %+v ok=%v", got, ok)
	}
}

func TestUISessionState_SwapMainUIActivation(t *testing.T) {
	s := NewUISessionState()
	if prior := s.SwapMainUIActivation("dev-a", "act-1"); prior != "" {
		t.Fatalf("expected empty prior, got %q", prior)
	}
	if prior := s.SwapMainUIActivation("dev-a", "act-2"); prior != "act-1" {
		t.Fatalf("expected act-1, got %q", prior)
	}
	if prior := s.SwapMainUIActivation("dev-b", "act-3"); prior != "" {
		t.Fatalf("expected empty prior for dev-b, got %q", prior)
	}
	if prior := s.SwapMainUIActivation("", "act-x"); prior != "" {
		t.Fatalf("blank device must be a no-op, got %q", prior)
	}
}

func TestUISessionState_ForgetMainUIActivation(t *testing.T) {
	s := NewUISessionState()
	s.SwapMainUIActivation("dev-a", "act-1")
	s.ForgetMainUIActivation("dev-a")
	if prior := s.SwapMainUIActivation("dev-a", "act-2"); prior != "" {
		t.Fatalf("expected forgotten activation, got %q", prior)
	}
}

func TestUISessionState_MultiWindowResumeCaptureAndTake(t *testing.T) {
	s := NewUISessionState()
	desc := ui.HelloWorld("ui-a")
	s.RememberSetUI("dev-a", []ServerMessage{{SetUI: &desc}})
	s.CaptureMultiWindowResume("dev-a", "terminal")

	priorScenario, priorUI, hasPriorUI, taken := s.TakeMultiWindowResume("dev-a")
	if !taken {
		t.Fatalf("expected captured resume state")
	}
	if priorScenario != "terminal" || !hasPriorUI || !reflect.DeepEqual(priorUI, desc) {
		t.Fatalf("unexpected resume: scenario=%q hasUI=%v ui=%+v", priorScenario, hasPriorUI, priorUI)
	}

	if _, _, _, taken := s.TakeMultiWindowResume("dev-a"); taken {
		t.Fatalf("take should consume the captured state")
	}
}

func TestUISessionState_CaptureMultiWindowResumeIgnoresMultiWindow(t *testing.T) {
	s := NewUISessionState()
	s.CaptureMultiWindowResume("dev-a", "multi_window")
	if _, _, _, taken := s.TakeMultiWindowResume("dev-a"); taken {
		t.Fatalf("capture must skip the multi_window scenario")
	}
}

func TestUISessionState_CaptureMultiWindowResumeFirstWriteWins(t *testing.T) {
	s := NewUISessionState()
	s.CaptureMultiWindowResume("dev-a", "terminal")
	s.CaptureMultiWindowResume("dev-a", "photo_frame")
	priorScenario, _, _, taken := s.TakeMultiWindowResume("dev-a")
	if !taken || priorScenario != "terminal" {
		t.Fatalf("first capture should stick, got %q taken=%v", priorScenario, taken)
	}
}

func TestUISessionState_UIHostBeforeCountAndAdvance(t *testing.T) {
	s := NewUISessionState()
	if got := s.UIHostBeforeCountAndAdvance("dev-a", 5); got != 0 {
		t.Fatalf("first read should be 0, got %d", got)
	}
	if got := s.UIHostBeforeCountAndAdvance("dev-a", 9); got != 5 {
		t.Fatalf("expected prior 5, got %d", got)
	}
	if got := s.UIHostBeforeCountAndAdvance("dev-b", 3); got != 0 {
		t.Fatalf("dev-b should be isolated, got %d", got)
	}
}

func TestUISessionState_MarkUIHostDelivered(t *testing.T) {
	s := NewUISessionState()
	delivered := map[string]struct{}{"dev-a": {}, "dev-b": {}, "": {}}
	s.MarkUIHostDelivered(delivered, 12)

	if got := s.UIHostBeforeCountAndAdvance("dev-a", 12); got != 12 {
		t.Fatalf("dev-a expected 12, got %d", got)
	}
	if got := s.UIHostBeforeCountAndAdvance("dev-b", 12); got != 12 {
		t.Fatalf("dev-b expected 12, got %d", got)
	}
	if got := s.UIHostBeforeCountAndAdvance("dev-c", 12); got != 0 {
		t.Fatalf("dev-c should be unaffected, got %d", got)
	}
}

func TestUISessionState_PerDeviceIsolation(t *testing.T) {
	s := NewUISessionState()
	descA := ui.HelloWorld("a")
	descB := ui.HelloWorld("b")
	s.RememberSetUI("dev-a", []ServerMessage{{SetUI: &descA}})
	s.RememberSetUI("dev-b", []ServerMessage{{SetUI: &descB}})
	s.SwapMainUIActivation("dev-a", "act-a")
	s.SwapMainUIActivation("dev-b", "act-b")

	if prior := s.SwapMainUIActivation("dev-a", "act-a2"); prior != "act-a" {
		t.Fatalf("dev-a expected act-a, got %q", prior)
	}
	gotB, _ := s.LastSetUI("dev-b")
	if !reflect.DeepEqual(gotB, descB) {
		t.Fatalf("dev-b UI cross-contaminated: %+v", gotB)
	}
}

func TestUISessionState_ConcurrentReadWriteRace(_ *testing.T) {
	s := NewUISessionState()
	const workers = 8
	const iterations = 200

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			deviceID := "dev-" + string(rune('a'+id))
			desc := ui.HelloWorld(deviceID)
			for i := 0; i < iterations; i++ {
				s.RememberSetUI(deviceID, []ServerMessage{{SetUI: &desc}})
				_, _ = s.LastSetUI(deviceID)
				s.SwapMainUIActivation(deviceID, "act")
				s.UIHostBeforeCountAndAdvance(deviceID, i)
				s.CaptureMultiWindowResume(deviceID, "terminal")
				s.TakeMultiWindowResume(deviceID)
				s.ForgetMainUIActivation(deviceID)
			}
		}(w)
	}
	wg.Wait()
}
