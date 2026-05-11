package transport

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/telephony"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestHandleMessageInputRoutesStartActionWhenScenarioIsActive(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-ui-route-photo-start",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "open_terminal_button",
			Action:      "start:terminal",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(input start:terminal) error = %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("len(out) = %d, want 3", len(out))
	}
	if out[0].ScenarioStart != "terminal" {
		t.Fatalf("ScenarioStart = %q, want terminal", out[0].ScenarioStart)
	}
	if out[1].SetUI == nil {
		t.Fatalf("expected SetUI response after starting terminal")
	}
	if out[2].TransitionUI == nil || out[2].TransitionUI.Transition != "terminal_enter" {
		t.Fatalf("expected terminal_enter transition, got %+v", out[2].TransitionUI)
	}
}

func TestHandleMessageInputRoutesStopActiveAction(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-ui-route-terminal-start",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "terminal",
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "stop_gesture",
			Action:      "stop_active",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(input stop_active) error = %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2", len(out))
	}
	if out[0].ScenarioStop != "terminal" {
		t.Fatalf("ScenarioStop = %q, want terminal", out[0].ScenarioStop)
	}
	if out[1].TransitionUI == nil || out[1].TransitionUI.Transition != "terminal_exit" {
		t.Fatalf("expected terminal_exit transition, got %+v", out[1].TransitionUI)
	}
	if _, ok := runtime.Engine.Active("device-1"); ok {
		t.Fatalf("expected no active scenario after stop_active")
	}
}

func TestHandleMessageInputCornerOpenTogglesMenuOverlayAndClaim(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	router := io.NewRouter()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        router,
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})

	openOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "act:device-1/__affordance.corner__",
			Action:      "open",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(input corner open) error = %v", err)
	}
	if len(openOut) != 1 || openOut[0].UpdateUI == nil {
		t.Fatalf("expected one UpdateUI response for menu open, got %+v", openOut)
	}
	if openOut[0].UpdateUI.ComponentID != ui.GlobalOverlayComponentID {
		t.Fatalf("UpdateUI.ComponentID = %q, want %q", openOut[0].UpdateUI.ComponentID, ui.GlobalOverlayComponentID)
	}
	if openOut[0].UpdateUI.Node.Type != "overlay" {
		t.Fatalf("UpdateUI node type = %q, want overlay", openOut[0].UpdateUI.Node.Type)
	}
	if findNodeByID(&openOut[0].UpdateUI.Node, "act:menu-overlay:device-1/menu.privacy_toggle") == nil {
		t.Fatalf("expected privacy toggle button in menu overlay descriptor")
	}
	if findNodeByID(&openOut[0].UpdateUI.Node, "act:menu-overlay:device-1/menu.bug_report") == nil {
		t.Fatalf("expected bug report button in menu overlay descriptor")
	}

	claims := router.Claims().Snapshot("device-1")
	if !hasClaim(claims, "menu-overlay:device-1", "screen.overlay") {
		t.Fatalf("expected screen.overlay claim for menu overlay activation, got %+v", claims)
	}

	closeOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "act:device-1/__affordance.corner__",
			Action:      "corner.open",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(input second corner open) error = %v", err)
	}
	if len(closeOut) != 1 || closeOut[0].UpdateUI == nil {
		t.Fatalf("expected one UpdateUI response for menu close, got %+v", closeOut)
	}
	if closeOut[0].UpdateUI.Node.Type != "overlay" {
		t.Fatalf("close UpdateUI node type = %q, want overlay", closeOut[0].UpdateUI.Node.Type)
	}
	if len(closeOut[0].UpdateUI.Node.Children) != 0 {
		t.Fatalf("expected menu overlay clear patch on close, got children=%d", len(closeOut[0].UpdateUI.Node.Children))
	}

	claims = router.Claims().Snapshot("device-1")
	if hasClaim(claims, "menu-overlay:device-1", "screen.overlay") {
		t.Fatalf("expected menu overlay claim released on close, got %+v", claims)
	}
}

func TestHandleMessageInputMenuCloseActionReleasesOverlayClaim(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	router := io.NewRouter()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        router,
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "act:device-1/__affordance.corner__",
			Action:      "open",
		},
	})

	closeOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "act:menu-overlay:device-1/menu.close",
			Action:      "close",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(input menu close) error = %v", err)
	}
	if len(closeOut) != 1 || closeOut[0].UpdateUI == nil {
		t.Fatalf("expected one UpdateUI response for menu close action, got %+v", closeOut)
	}
	if len(closeOut[0].UpdateUI.Node.Children) != 0 {
		t.Fatalf("expected menu overlay clear patch, got children=%d", len(closeOut[0].UpdateUI.Node.Children))
	}

	claims := router.Claims().Snapshot("device-1")
	if hasClaim(claims, "menu-overlay:device-1", "screen.overlay") {
		t.Fatalf("expected menu overlay claim released by close action, got %+v", claims)
	}
}

type stubIdentityService struct {
	actorsByDevice map[string]Actor
}

func (s stubIdentityService) ResolveActor(deviceID string) Actor {
	if actor, ok := s.actorsByDevice[deviceID]; ok {
		return actor
	}
	return Actor{Kind: "device", ID: strings.TrimSpace(deviceID)}
}

type fixtureMenuPolicy struct{}

func (fixtureMenuPolicy) VisibleApps(actor Actor, apps []string) []string {
	if strings.EqualFold(strings.TrimSpace(actor.Kind), "anonymous") {
		out := make([]string, 0, len(apps))
		for _, app := range apps {
			if app == "photo_frame" {
				out = append(out, app)
			}
		}
		return out
	}
	return append([]string(nil), apps...)
}

type countingAudioPublisher struct {
	mu     sync.Mutex
	count  int
	device string
}

func (p *countingAudioPublisher) Publish(deviceID string, _ []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.count++
	p.device = deviceID
}

func (p *countingAudioPublisher) Snapshot() (int, string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.count, p.device
}

func TestHandleMessageInputMenuOverlayCompositionVariesByResolvedActor(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	router := io.NewRouter()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        router,
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	handler.SetIdentityService(stubIdentityService{
		actorsByDevice: map[string]Actor{
			"device-anon":   {Kind: "anonymous", ID: "kiosk"},
			"device-person": {Kind: "person", ID: "alice"},
		},
	})
	handler.SetMenuAppPolicy(fixtureMenuPolicy{})

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-anon", DeviceName: "Kiosk"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-person", DeviceName: "Kitchen Tablet"},
	})

	anonOpenOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-anon",
			ComponentID: "act:device-anon/__affordance.corner__",
			Action:      "open",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(input corner open anonymous) error = %v", err)
	}
	personOpenOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-person",
			ComponentID: "act:device-person/__affordance.corner__",
			Action:      "open",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(input corner open person) error = %v", err)
	}

	if len(anonOpenOut) != 1 || anonOpenOut[0].UpdateUI == nil {
		t.Fatalf("expected one UpdateUI for anonymous menu open, got %+v", anonOpenOut)
	}
	if len(personOpenOut) != 1 || personOpenOut[0].UpdateUI == nil {
		t.Fatalf("expected one UpdateUI for person menu open, got %+v", personOpenOut)
	}

	anonApps := menuAppNamesFromDescriptor(&anonOpenOut[0].UpdateUI.Node)
	personApps := menuAppNamesFromDescriptor(&personOpenOut[0].UpdateUI.Node)

	if len(anonApps) == 0 {
		t.Fatalf("expected anonymous actor to see at least one app")
	}
	if _, ok := anonApps["terminal"]; ok {
		t.Fatalf("anonymous menu should hide terminal app, got %+v", anonApps)
	}
	if _, ok := personApps["terminal"]; !ok {
		t.Fatalf("person menu should include terminal app, got %+v", personApps)
	}
	if len(personApps) <= len(anonApps) {
		t.Fatalf("expected actor-variant menu app counts, anonymous=%d person=%d", len(anonApps), len(personApps))
	}
}

func TestHandleMessageInputMenuOverlayDefaultMixedPolicyBlocksMainPointerButKeepsAudioLive(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	pub := &countingAudioPublisher{}
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	handler.SetDeviceAudioPublisher(pub)

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	}); err != nil {
		t.Fatalf("register error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{RequestID: "cmd-terminal", DeviceID: "device-1", Kind: "manual", Intent: "terminal"},
	}); err != nil {
		t.Fatalf("terminal start error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{DeviceID: "device-1", ComponentID: "act:device-1/__affordance.corner__", Action: "open"},
	}); err != nil {
		t.Fatalf("menu open error = %v", err)
	}

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		VoiceAudio: &VoiceAudioRequest{DeviceID: "device-1", Audio: []byte("live-audio"), SampleRate: 16000, IsFinal: false},
	}); err != nil {
		t.Fatalf("voice audio error = %v", err)
	}
	if count, deviceID := pub.Snapshot(); count != 1 || deviceID != "device-1" {
		t.Fatalf("audio publish snapshot = (%d,%q), want (1,device-1)", count, deviceID)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{DeviceID: "device-1", ComponentID: "stop_gesture", Action: "stop_active"},
	})
	if err != nil {
		t.Fatalf("stop_active input error = %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected no routed main-layer response while overlay is open under MIXED policy, got %+v", out)
	}
	if active, ok := runtime.Engine.Active("device-1"); !ok || active != "terminal" {
		t.Fatalf("active scenario after stop_active = (%q, %v), want terminal,true", active, ok)
	}

	out, err = handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{DeviceID: "device-1", ComponentID: "act:menu-overlay:device-1/menu.close", Action: "close"},
	})
	if err != nil {
		t.Fatalf("menu close error = %v", err)
	}
	if len(out) == 0 || out[0].UpdateUI == nil {
		t.Fatalf("expected overlay clear patch on close, got %+v", out)
	}

	out, err = handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{DeviceID: "device-1", ComponentID: "stop_gesture", Action: "stop_active"},
	})
	if err != nil {
		t.Fatalf("stop_active post-close error = %v", err)
	}
	if len(out) == 0 || out[0].ScenarioStop != "terminal" {
		t.Fatalf("expected stop_active routed after menu close, got %+v", out)
	}
}

func TestCapabilityDeltaWhileMenuOverlayOpenPreservesMainAndOverlayActivations(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	router := io.NewRouter()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        router,
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	}); err != nil {
		t.Fatalf("register error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 1,
			Capabilities: map[string]string{
				"screen.width":       "1920",
				"screen.height":      "1080",
				"screen.orientation": "landscape",
			},
		},
	}); err != nil {
		t.Fatalf("capability snapshot error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{RequestID: "cmd-terminal", DeviceID: "device-1", Kind: "manual", Intent: "terminal"},
	}); err != nil {
		t.Fatalf("terminal start error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{DeviceID: "device-1", ComponentID: "act:device-1/__affordance.corner__", Action: "open"},
	}); err != nil {
		t.Fatalf("menu open error = %v", err)
	}

	if active, ok := runtime.Engine.Active("device-1"); !ok || active != "terminal" {
		t.Fatalf("active scenario before orientation delta = (%q, %v), want terminal,true", active, ok)
	}
	if !hasClaim(router.Claims().Snapshot("device-1"), "menu-overlay:device-1", "screen.overlay") {
		t.Fatalf("expected overlay claim before orientation delta")
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilityDelta: &CapabilityDeltaRequest{
			DeviceID:   "device-1",
			Generation: 2,
			Reason:     "display_geometry_change",
			Capabilities: map[string]string{
				"screen.width":       "1080",
				"screen.height":      "1920",
				"screen.orientation": "portrait",
			},
		},
	})
	if err != nil {
		t.Fatalf("capability delta error = %v", err)
	}
	if len(out) == 0 || out[0].CapabilityAck == nil {
		t.Fatalf("expected capability ack response, got %+v", out)
	}
	for _, msg := range out {
		if msg.UpdateUI != nil && len(msg.UpdateUI.Node.Children) == 0 {
			t.Fatalf("unexpected overlay clear patch on orientation delta: %+v", msg.UpdateUI)
		}
	}

	if active, ok := runtime.Engine.Active("device-1"); !ok || active != "terminal" {
		t.Fatalf("active scenario after orientation delta = (%q, %v), want terminal,true", active, ok)
	}
	if !hasClaim(router.Claims().Snapshot("device-1"), "menu-overlay:device-1", "screen.overlay") {
		t.Fatalf("expected overlay claim to remain active after orientation delta")
	}

	closeOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{DeviceID: "device-1", ComponentID: "act:menu-overlay:device-1/menu.close", Action: "close"},
	})
	if err != nil {
		t.Fatalf("menu close after orientation delta error = %v", err)
	}
	if len(closeOut) == 0 || closeOut[0].UpdateUI == nil {
		t.Fatalf("expected overlay close patch after orientation delta, got %+v", closeOut)
	}
	if len(closeOut[0].UpdateUI.Node.Children) != 0 {
		t.Fatalf("expected overlay close patch to clear children, got %+v", closeOut[0].UpdateUI.Node.Children)
	}
}

func TestLifecycleCapabilityDeltaWhileMenuOverlayOpenPreservesMainAndOverlayActivations(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	router := io.NewRouter()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        router,
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	}); err != nil {
		t.Fatalf("register error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 1,
			Capabilities: map[string]string{
				"screen.width":          "1920",
				"screen.height":         "1080",
				"screen.orientation":    "landscape",
				"edge.operators":        "monitor.lifecycle.foreground",
				"monitor.runtime_state": "foreground",
			},
		},
	}); err != nil {
		t.Fatalf("capability snapshot error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{RequestID: "cmd-terminal", DeviceID: "device-1", Kind: "manual", Intent: "terminal"},
	}); err != nil {
		t.Fatalf("terminal start error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{DeviceID: "device-1", ComponentID: "act:device-1/__affordance.corner__", Action: "open"},
	}); err != nil {
		t.Fatalf("menu open error = %v", err)
	}

	if active, ok := runtime.Engine.Active("device-1"); !ok || active != "terminal" {
		t.Fatalf("active scenario before lifecycle delta = (%q, %v), want terminal,true", active, ok)
	}
	if !hasClaim(router.Claims().Snapshot("device-1"), "menu-overlay:device-1", "screen.overlay") {
		t.Fatalf("expected overlay claim before lifecycle delta")
	}

	backgroundOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilityDelta: &CapabilityDeltaRequest{
			DeviceID:   "device-1",
			Generation: 2,
			Reason:     "app_lifecycle_change",
			Capabilities: map[string]string{
				"screen.width":          "1920",
				"screen.height":         "1080",
				"screen.orientation":    "landscape",
				"edge.operators":        "monitor.lifecycle.background",
				"monitor.runtime_state": "background",
			},
		},
	})
	if err != nil {
		t.Fatalf("background lifecycle capability delta error = %v", err)
	}
	if len(backgroundOut) == 0 || backgroundOut[0].CapabilityAck == nil {
		t.Fatalf("expected capability ack for background lifecycle delta, got %+v", backgroundOut)
	}
	for _, msg := range backgroundOut {
		if msg.UpdateUI != nil && len(msg.UpdateUI.Node.Children) == 0 {
			t.Fatalf("unexpected overlay clear patch on background lifecycle delta: %+v", msg.UpdateUI)
		}
	}
	if active, ok := runtime.Engine.Active("device-1"); !ok || active != "terminal" {
		t.Fatalf("active scenario after background lifecycle delta = (%q, %v), want terminal,true", active, ok)
	}
	if !hasClaim(router.Claims().Snapshot("device-1"), "menu-overlay:device-1", "screen.overlay") {
		t.Fatalf("expected overlay claim to remain active after background lifecycle delta")
	}

	foregroundOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilityDelta: &CapabilityDeltaRequest{
			DeviceID:   "device-1",
			Generation: 3,
			Reason:     "app_lifecycle_change",
			Capabilities: map[string]string{
				"screen.width":          "1920",
				"screen.height":         "1080",
				"screen.orientation":    "landscape",
				"edge.operators":        "monitor.lifecycle.foreground",
				"monitor.runtime_state": "foreground",
			},
		},
	})
	if err != nil {
		t.Fatalf("foreground lifecycle capability delta error = %v", err)
	}
	if len(foregroundOut) == 0 || foregroundOut[0].CapabilityAck == nil {
		t.Fatalf("expected capability ack for foreground lifecycle delta, got %+v", foregroundOut)
	}
	for _, msg := range foregroundOut {
		if msg.UpdateUI != nil && len(msg.UpdateUI.Node.Children) == 0 {
			t.Fatalf("unexpected overlay clear patch on foreground lifecycle delta: %+v", msg.UpdateUI)
		}
	}
	if active, ok := runtime.Engine.Active("device-1"); !ok || active != "terminal" {
		t.Fatalf("active scenario after foreground lifecycle delta = (%q, %v), want terminal,true", active, ok)
	}
	if !hasClaim(router.Claims().Snapshot("device-1"), "menu-overlay:device-1", "screen.overlay") {
		t.Fatalf("expected overlay claim to remain active after foreground lifecycle delta")
	}

	closeOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{DeviceID: "device-1", ComponentID: "act:menu-overlay:device-1/menu.close", Action: "close"},
	})
	if err != nil {
		t.Fatalf("menu close after lifecycle deltas error = %v", err)
	}
	if len(closeOut) == 0 || closeOut[0].UpdateUI == nil {
		t.Fatalf("expected overlay close patch after lifecycle deltas, got %+v", closeOut)
	}
	if len(closeOut[0].UpdateUI.Node.Children) != 0 {
		t.Fatalf("expected overlay close patch to clear children, got %+v", closeOut[0].UpdateUI.Node.Children)
	}
}

func TestHandleMessageInputMenuOverlayLivePolicyKeepsMainPointerActive(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	handler.SetMenuOverlayInputPolicyForTesting("LIVE", nil)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{RequestID: "cmd-terminal", DeviceID: "device-1", Kind: "manual", Intent: "terminal"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{DeviceID: "device-1", ComponentID: "act:device-1/__affordance.corner__", Action: "open"},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{DeviceID: "device-1", ComponentID: "stop_gesture", Action: "stop_active"},
	})
	if err != nil {
		t.Fatalf("stop_active input error = %v", err)
	}
	if len(out) == 0 || out[0].ScenarioStop != "terminal" {
		t.Fatalf("expected stop_active routed with LIVE policy while overlay open, got %+v", out)
	}
}

func TestHandleMessageInputMenuOverlayPausedPolicyTearsDownAndRestoresRoutes(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	router := io.NewRouter()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        router,
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	handler.SetMenuOverlayInputPolicyForTesting("PAUSED", map[string]bool{
		"audio": false,
	})

	if err := router.Connect("device-1", "device-2", "audio"); err != nil {
		t.Fatalf("connect audio route error = %v", err)
	}
	if got := len(router.RoutesForDevice("device-1")); got != 1 {
		t.Fatalf("routes before menu open = %d, want 1", got)
	}

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{RequestID: "cmd-terminal", DeviceID: "device-1", Kind: "manual", Intent: "terminal"},
	})
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{DeviceID: "device-1", ComponentID: "act:device-1/__affordance.corner__", Action: "open"},
	}); err != nil {
		t.Fatalf("menu open error = %v", err)
	}
	if got := len(router.RoutesForDevice("device-1")); got != 0 {
		t.Fatalf("routes after menu open with PAUSED policy = %d, want 0", got)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{DeviceID: "device-1", ComponentID: "stop_gesture", Action: "stop_active"},
	})
	if err != nil {
		t.Fatalf("stop_active while paused error = %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected main pointer action blocked while PAUSED overlay open, got %+v", out)
	}

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{DeviceID: "device-1", ComponentID: "act:menu-overlay:device-1/menu.close", Action: "close"},
	}); err != nil {
		t.Fatalf("menu close error = %v", err)
	}
	if got := len(router.RoutesForDevice("device-1")); got != 1 {
		t.Fatalf("routes after menu close with PAUSED policy = %d, want 1", got)
	}
}

func menuAppNamesFromDescriptor(node *ui.Descriptor) map[string]struct{} {
	out := map[string]struct{}{}
	if node == nil {
		return out
	}
	if id := node.Props["id"]; strings.Contains(id, "/menu.app.") {
		parts := strings.SplitN(id, "/menu.app.", 2)
		if len(parts) == 2 {
			out[parts[1]] = struct{}{}
		}
	}
	for i := range node.Children {
		for name := range menuAppNamesFromDescriptor(&node.Children[i]) {
			out[name] = struct{}{}
		}
	}
	return out
}

func TestHandleInputActionMapTurnoverDropsPriorMainActivationScopedIDs(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)

	initialUI := ui.New("stack", map[string]string{
		"id": "act:main-a/root",
	}, ui.New("button", map[string]string{
		"id":     "act:main-a/__affordance.corner__",
		"label":  "Menu",
		"action": "corner.open",
	}))
	if _, err := handler.prepareOutboundUI("device-1", ServerMessage{SetUI: &initialUI}); err != nil {
		t.Fatalf("prepareOutboundUI(initial) error = %v", err)
	}

	swappedUI := ui.New("stack", map[string]string{
		"id": "act:main-b/root",
	}, ui.New("button", map[string]string{
		"id":     "act:main-b/__affordance.corner__",
		"label":  "Menu",
		"action": "corner.open",
	}))
	if _, err := handler.prepareOutboundUI("device-1", ServerMessage{SetUI: &swappedUI}); err != nil {
		t.Fatalf("prepareOutboundUI(swapped) error = %v", err)
	}

	openNew, err := handler.handleInput(context.Background(), &InputRequest{
		DeviceID:    "device-1",
		ComponentID: "act:main-b/__affordance.corner__",
		Action:      "open",
	})
	if err != nil {
		t.Fatalf("handleInput(new activation) error = %v", err)
	}
	if len(openNew) != 1 || openNew[0].UpdateUI == nil {
		t.Fatalf("expected overlay update for new activation component id, got %+v", openNew)
	}

	snapshot := handler.metrics.Snapshot()
	if snapshot[`ui_action_unknown_component_total{reason="unknown_activation"}`] != "0" {
		t.Fatalf("unknown_activation counter after new action = %q, want 0", snapshot[`ui_action_unknown_component_total{reason="unknown_activation"}`])
	}

	oldOut, err := handler.handleInput(context.Background(), &InputRequest{
		DeviceID:    "device-1",
		ComponentID: "act:main-a/__affordance.corner__",
		Action:      "open",
	})
	if err != nil {
		t.Fatalf("handleInput(old activation) error = %v", err)
	}
	if len(oldOut) != 0 {
		t.Fatalf("old activation action should be dropped, got %+v", oldOut)
	}

	snapshot = handler.metrics.Snapshot()
	if snapshot[`ui_action_unknown_component_total{reason="unknown_activation"}`] != "1" {
		t.Fatalf("unknown_activation counter after stale action = %q, want 1", snapshot[`ui_action_unknown_component_total{reason="unknown_activation"}`])
	}
}

func hasClaim(claims []io.Claim, activationID, resource string) bool {
	for _, claim := range claims {
		if claim.ActivationID == activationID && claim.Resource == resource {
			return true
		}
	}
	return false
}

func TestHandleMessageInputRoutesMultiWindowEndActionAndRestoresPriorTerminal(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "device-2", DeviceName: "Hall"})
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-terminal-start",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "terminal",
		},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-multi-window-start",
			DeviceID:  "device-1",
			Kind:      "voice",
			Text:      "all cameras",
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "multi_window_end",
			Action:      "multi_window_end",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(input multi_window_end) error = %v", err)
	}

	if len(out) < 3 {
		t.Fatalf("len(out) = %d, want at least 3", len(out))
	}
	if out[0].ScenarioStop != "multi_window" {
		t.Fatalf("ScenarioStop = %q, want multi_window", out[0].ScenarioStop)
	}

	sawTerminalRoot := false
	sawTerminalEnter := false
	for _, msg := range out {
		if set := msg.SetUI; set != nil && (set.Props["id"] == "terminal_root" || strings.HasSuffix(set.Props["id"], "/terminal_root")) {
			sawTerminalRoot = true
		}
		if transition := msg.TransitionUI; transition != nil && transition.Transition == "terminal_enter" {
			sawTerminalEnter = true
		}
	}
	if !sawTerminalRoot {
		t.Fatalf("expected restored terminal SetUI after multi_window_end")
	}
	if !sawTerminalEnter {
		t.Fatalf("expected terminal_enter transition after multi_window_end")
	}
}

func TestHandleMessageInputRoutesInternalVideoCallEndAction(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "device-2", DeviceName: "Hall"})
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-video-call-start",
			DeviceID:  "device-1",
			Kind:      "voice",
			Text:      "video call device-2",
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "internal_video_call_hangup",
			Action:      "internal_video_call_end",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(input internal_video_call_end) error = %v", err)
	}

	if len(out) < 2 {
		t.Fatalf("len(out) = %d, want at least 2", len(out))
	}
	if out[0].ScenarioStop != "internal_video_call" {
		t.Fatalf("ScenarioStop = %q, want internal_video_call", out[0].ScenarioStop)
	}
	if out[1].TransitionUI == nil || out[1].TransitionUI.Transition != "internal_video_call_exit" {
		t.Fatalf("expected internal_video_call_exit transition, got %+v", out[1].TransitionUI)
	}
}

func TestHandleMessageInputActionIgnoredWithoutActiveScenario(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices: devices,
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "open_terminal_button",
			Action:      "start:terminal",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(input action without active scenario) error = %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("len(out) = %d, want 0", len(out))
	}
}

func TestHandleMessageInputTapIgnoredWithActiveScenario(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices: devices,
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-ui-route-photo-start-tap",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "generic_tap",
			Action:      "tap",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(input tap) error = %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("len(out) = %d, want 0", len(out))
	}
	active, ok := runtime.Engine.Active("device-1")
	if !ok || active != "photo_frame" {
		t.Fatalf("active scenario = %q (ok=%t), want photo_frame", active, ok)
	}
}
