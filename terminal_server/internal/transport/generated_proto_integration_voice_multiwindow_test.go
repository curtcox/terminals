package transport

import (
	"context"
	"strings"
	"testing"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	iov1 "github.com/curtcox/terminals/terminal_server/gen/go/io/v1"
	uiv1 "github.com/curtcox/terminals/terminal_server/gen/go/ui/v1"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/telephony"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestGeneratedSessionVoiceStandDownStopsRedAlert(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Telephony: telephony.NoopBridge{},
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	stream := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Register{
					Register: &controlv1.RegisterDevice{
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "device-1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "red-alert-start",
						DeviceId:  "device-1",
						Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
						Text:      "red alert",
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "red-alert-stop-stand-down",
						DeviceId:  "device-1",
						Action:    controlv1.CommandAction_COMMAND_ACTION_STOP,
						Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
						Text:      "stand down",
					},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}

	var sawStart bool
	var sawStop bool
	for _, sent := range stream.sent {
		resp, ok := sent.(*controlv1.ConnectResponse)
		if !ok || resp.GetCommandResult() == nil {
			continue
		}
		switch {
		case resp.GetCommandResult().GetScenarioStart() == "red_alert":
			sawStart = true
		case resp.GetCommandResult().GetScenarioStop() == "red_alert":
			sawStop = true
		}
	}
	if !sawStart {
		t.Fatalf("expected scenario_start=red_alert command result")
	}
	if !sawStop {
		t.Fatalf("expected scenario_stop=red_alert command result via stand down")
	}
}

func TestGeneratedSessionVoiceStopRedAlertStopsRedAlert(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Telephony: telephony.NoopBridge{},
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	stream := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Register{
					Register: &controlv1.RegisterDevice{
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "device-1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "red-alert-start",
						DeviceId:  "device-1",
						Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
						Text:      "red alert",
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "red-alert-stop-stop-red-alert",
						DeviceId:  "device-1",
						Action:    controlv1.CommandAction_COMMAND_ACTION_STOP,
						Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
						Text:      "stop red alert",
					},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}

	var sawStart bool
	var sawStop bool
	for _, sent := range stream.sent {
		resp, ok := sent.(*controlv1.ConnectResponse)
		if !ok || resp.GetCommandResult() == nil {
			continue
		}
		switch {
		case resp.GetCommandResult().GetScenarioStart() == "red_alert":
			sawStart = true
		case resp.GetCommandResult().GetScenarioStop() == "red_alert":
			sawStop = true
		}
	}
	if !sawStart {
		t.Fatalf("expected scenario_start=red_alert command result")
	}
	if !sawStop {
		t.Fatalf("expected scenario_stop=red_alert command result via stop red alert")
	}
}

func TestGeneratedSessionVoicePAModeStartsPASystem(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Telephony: telephony.NoopBridge{},
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	stream := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Register{
					Register: &controlv1.RegisterDevice{
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "device-1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "pa-mode-start",
						DeviceId:  "device-1",
						Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
						Text:      "pa mode",
					},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}

	var sawStart bool
	for _, sent := range stream.sent {
		resp, ok := sent.(*controlv1.ConnectResponse)
		if !ok || resp.GetCommandResult() == nil {
			continue
		}
		if resp.GetCommandResult().GetScenarioStart() == "pa_system" {
			sawStart = true
			break
		}
	}
	if !sawStart {
		t.Fatalf("expected scenario_start=pa_system command result via pa mode")
	}
}

func TestGeneratedSessionVoiceShowAllCamerasStartsMultiWindow(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Telephony: telephony.NoopBridge{},
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	stream := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Register{
					Register: &controlv1.RegisterDevice{
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "device-1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "show-all-cameras-start",
						DeviceId:  "device-1",
						Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
						Text:      "show all cameras",
					},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}

	var sawStart bool
	for _, sent := range stream.sent {
		resp, ok := sent.(*controlv1.ConnectResponse)
		if !ok || resp.GetCommandResult() == nil {
			continue
		}
		if resp.GetCommandResult().GetScenarioStart() == "multi_window" {
			sawStart = true
			break
		}
	}
	if !sawStart {
		t.Fatalf("expected scenario_start=multi_window command result via show all cameras")
	}
}

func TestGeneratedSessionVoiceAllCamerasStartsMultiWindow(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Telephony: telephony.NoopBridge{},
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	stream := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Register{
					Register: &controlv1.RegisterDevice{
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "device-1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "all-cameras-start",
						DeviceId:  "device-1",
						Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
						Text:      "all cameras",
					},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}

	var sawStart bool
	for _, sent := range stream.sent {
		resp, ok := sent.(*controlv1.ConnectResponse)
		if !ok || resp.GetCommandResult() == nil {
			continue
		}
		if resp.GetCommandResult().GetScenarioStart() == "multi_window" {
			sawStart = true
			break
		}
	}
	if !sawStart {
		t.Fatalf("expected scenario_start=multi_window command result via all cameras")
	}
}

func TestGeneratedSessionMultiWindowAudioMixAndFocusSelection(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d3", DeviceName: "Office"})
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Telephony: telephony.NoopBridge{},
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	stream := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Register{
					Register: &controlv1.RegisterDevice{
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "d1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "all-cameras-start",
						DeviceId:  "d1",
						Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
						Text:      "all cameras",
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "all-cameras-stop",
						DeviceId:  "d1",
						Action:    controlv1.CommandAction_COMMAND_ACTION_STOP,
						Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
						Text:      "all cameras",
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "all-cameras-focus-start",
						DeviceId:  "d1",
						Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
						Text:      "all cameras focus d2",
					},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}

	sawMixD2 := false
	sawMixD3 := false
	sawMixStopD2 := false
	sawMixStopD3 := false
	focusStartIdx := -1
	for idx, sent := range stream.sent {
		resp, ok := sent.(*controlv1.ConnectResponse)
		if !ok {
			continue
		}
		if start := resp.GetStartStream(); start != nil {
			switch start.GetStreamId() {
			case "route:d2|d1|audio_mix":
				sawMixD2 = true
			case "route:d3|d1|audio_mix":
				sawMixD3 = true
			}
		}
		if stop := resp.GetStopStream(); stop != nil {
			switch stop.GetStreamId() {
			case "route:d2|d1|audio_mix":
				sawMixStopD2 = true
			case "route:d3|d1|audio_mix":
				sawMixStopD3 = true
			}
		}
		if result := resp.GetCommandResult(); result != nil &&
			result.GetRequestId() == "all-cameras-focus-start" &&
			result.GetScenarioStart() == "multi_window" {
			focusStartIdx = idx
		}
	}

	if !sawMixD2 || !sawMixD3 {
		t.Fatalf("expected initial audio_mix start routes for d2 and d3")
	}
	if !sawMixStopD2 || !sawMixStopD3 {
		t.Fatalf("expected multi_window stop to emit stop_stream for both audio_mix routes")
	}
	if focusStartIdx == -1 {
		t.Fatalf("expected focused multi_window command_result")
	}

	sawFocusedAudio := false
	sawFocusedAudioMix := false
	for _, sent := range stream.sent[focusStartIdx+1:] {
		resp, ok := sent.(*controlv1.ConnectResponse)
		if !ok {
			continue
		}
		start := resp.GetStartStream()
		if start == nil {
			continue
		}
		if start.GetStreamId() == "route:d2|d1|audio" {
			sawFocusedAudio = true
		}
		if start.GetStreamId() == "route:d2|d1|audio_mix" || start.GetStreamId() == "route:d3|d1|audio_mix" {
			sawFocusedAudioMix = true
		}
	}
	if !sawFocusedAudio {
		t.Fatalf("expected focused audio start route route:d2|d1|audio")
	}
	if sawFocusedAudioMix {
		t.Fatalf("did not expect audio_mix start routes after focused restart")
	}
}

func TestGeneratedSessionMultiWindowSetUIAndFocusActionRouting(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d3", DeviceName: "Office"})
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Telephony: telephony.NoopBridge{},
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	stream := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Register{
					Register: &controlv1.RegisterDevice{
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "d1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "all-cameras-start",
						DeviceId:  "d1",
						Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
						Text:      "all cameras",
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Input{
					Input: &iov1.InputEvent{
						DeviceId: "d1",
						Payload: &iov1.InputEvent_UiAction{
							UiAction: &iov1.UIAction{
								ComponentId: "multi_window_focus_d2",
								Action:      "multi_window_focus:d2",
							},
						},
					},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}

	var sawGridColumns bool
	var sawFocusAction bool
	var sawFocusedLabel bool
	var sawEndAction bool
	walkNode := func(_ *uiv1.Node, _ func(*uiv1.Node)) {}
	walkNode = func(node *uiv1.Node, fn func(*uiv1.Node)) {
		if node == nil {
			return
		}
		fn(node)
		for _, child := range node.GetChildren() {
			walkNode(child, fn)
		}
	}

	scenarioStartCount := 0
	mixStopCount := 0
	focusAudioStartCount := 0
	for _, sent := range stream.sent {
		resp, ok := sent.(*controlv1.ConnectResponse)
		if !ok {
			continue
		}
		if result := resp.GetCommandResult(); result != nil && result.GetScenarioStart() == "multi_window" {
			scenarioStartCount++
		}
		if stop := resp.GetStopStream(); stop != nil {
			if stop.GetStreamId() == "route:d2|d1|audio_mix" || stop.GetStreamId() == "route:d3|d1|audio_mix" {
				mixStopCount++
			}
		}
		if start := resp.GetStartStream(); start != nil && start.GetStreamId() == "route:d2|d1|audio" {
			focusAudioStartCount++
		}
		if set := resp.GetSetUi(); set != nil {
			walkNode(set.GetRoot(), func(node *uiv1.Node) {
				propID := node.GetProps()["id"]
				if (propID == "multi_window_grid" || strings.HasSuffix(propID, "/multi_window_grid")) && node.GetGrid() != nil && node.GetGrid().GetColumns() == 2 {
					sawGridColumns = true
				}
				if button := node.GetButton(); button != nil {
					if button.GetAction() == "multi_window_end" {
						sawEndAction = true
					}
					if button.GetAction() == "multi_window_focus:d2" {
						sawFocusAction = true
					}
					if button.GetLabel() == "Hearing d2" {
						sawFocusedLabel = true
					}
				}
			})
		}
	}

	if !sawGridColumns {
		t.Fatalf("expected multi_window grid columns to be set to 2")
	}
	if !sawFocusAction {
		t.Fatalf("expected multi_window focus button action for d2")
	}
	if !sawEndAction {
		t.Fatalf("expected multi_window end button action")
	}
	if scenarioStartCount < 2 {
		t.Fatalf("expected two multi_window starts (voice + focus action), got %d", scenarioStartCount)
	}
	if mixStopCount < 2 {
		t.Fatalf("expected focus action to stop both audio_mix routes, got %d", mixStopCount)
	}
	if focusAudioStartCount == 0 {
		t.Fatalf("expected focus action to start focused audio route")
	}
	if !sawFocusedLabel {
		t.Fatalf("expected re-rendered UI with focused label Hearing d2")
	}
}

func TestGeneratedSessionMultiWindowEndActionRestoresPriorUIAndTransition(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Telephony: telephony.NoopBridge{},
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	stream := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Register{
					Register: &controlv1.RegisterDevice{
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "d1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "terminal-start",
						DeviceId:  "d1",
						Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
						Intent:    "terminal",
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "multi-window-start",
						DeviceId:  "d1",
						Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
						Text:      "all cameras",
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Input{
					Input: &iov1.InputEvent{
						DeviceId: "d1",
						Payload: &iov1.InputEvent_UiAction{
							UiAction: &iov1.UIAction{
								ComponentId: "multi_window_end",
								Action:      "multi_window_end",
							},
						},
					},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}

	var sawMultiWindowStop bool
	var sawRestoredTerminalUI bool
	var sawTerminalEnterTransition bool
	var sawVideoStop bool
	for _, sent := range stream.sent {
		resp, ok := sent.(*controlv1.ConnectResponse)
		if !ok {
			continue
		}
		if result := resp.GetCommandResult(); result != nil && result.GetScenarioStop() == "multi_window" {
			sawMultiWindowStop = true
		}
		if set := resp.GetSetUi(); set != nil {
			if root := set.GetRoot(); root != nil &&
				(root.GetProps()["id"] == "terminal_root" || strings.HasSuffix(root.GetProps()["id"], "/terminal_root")) {
				sawRestoredTerminalUI = true
			}
		}
		if transition := resp.GetTransitionUi(); transition != nil && transition.GetTransition() == "terminal_enter" {
			sawTerminalEnterTransition = true
		}
		if stop := resp.GetStopStream(); stop != nil && stop.GetStreamId() == "route:d2|d1|video" {
			sawVideoStop = true
		}
	}

	if !sawMultiWindowStop {
		t.Fatalf("expected multi_window scenario stop from UI end action")
	}
	if !sawRestoredTerminalUI {
		t.Fatalf("expected restored terminal SetUI after multi-window end")
	}
	if !sawTerminalEnterTransition {
		t.Fatalf("expected terminal_enter transition restored after multi-window end")
	}
	if !sawVideoStop {
		t.Fatalf("expected video stop_stream for multi-window teardown")
	}
}

func TestGeneratedSessionInternalVideoCallStartSetUIAndHangupFlow(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Telephony: telephony.NoopBridge{},
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	stream := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Register{
					Register: &controlv1.RegisterDevice{
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "d1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "video-call-start",
						DeviceId:  "d1",
						Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
						Text:      "video call d2",
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Input{
					Input: &iov1.InputEvent{
						DeviceId: "d1",
						Payload: &iov1.InputEvent_UiAction{
							UiAction: &iov1.UIAction{
								ComponentId: "internal_video_call_hangup",
								Action:      "internal_video_call_end",
							},
						},
					},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}

	var sawScenarioStart bool
	var sawScenarioStop bool
	var sawHangupAction bool
	var sawEnterTransition bool
	var sawExitTransition bool
	startStreams := map[string]bool{}
	stopStreams := map[string]bool{}
	walkNode := func(_ *uiv1.Node, _ func(*uiv1.Node)) {}
	walkNode = func(node *uiv1.Node, fn func(*uiv1.Node)) {
		if node == nil {
			return
		}
		fn(node)
		for _, child := range node.GetChildren() {
			walkNode(child, fn)
		}
	}

	for _, sent := range stream.sent {
		resp, ok := sent.(*controlv1.ConnectResponse)
		if !ok {
			continue
		}
		if result := resp.GetCommandResult(); result != nil {
			if result.GetScenarioStart() == "internal_video_call" {
				sawScenarioStart = true
			}
			if result.GetScenarioStop() == "internal_video_call" {
				sawScenarioStop = true
			}
		}
		if start := resp.GetStartStream(); start != nil {
			startStreams[start.GetStreamId()] = true
		}
		if stop := resp.GetStopStream(); stop != nil {
			stopStreams[stop.GetStreamId()] = true
		}
		if set := resp.GetSetUi(); set != nil {
			walkNode(set.GetRoot(), func(node *uiv1.Node) {
				if button := node.GetButton(); button != nil && button.GetAction() == "internal_video_call_end" {
					sawHangupAction = true
				}
			})
		}
		if transition := resp.GetTransitionUi(); transition != nil {
			if transition.GetTransition() == "internal_video_call_enter" {
				sawEnterTransition = true
			}
			if transition.GetTransition() == "internal_video_call_exit" {
				sawExitTransition = true
			}
		}
	}

	if !sawScenarioStart {
		t.Fatalf("expected internal_video_call scenario start")
	}
	if !sawScenarioStop {
		t.Fatalf("expected internal_video_call scenario stop from hangup action")
	}
	if !sawHangupAction {
		t.Fatalf("expected internal video call SetUI to include hangup action")
	}
	if !sawEnterTransition {
		t.Fatalf("expected internal_video_call_enter transition")
	}
	if !sawExitTransition {
		t.Fatalf("expected internal_video_call_exit transition")
	}
	for _, streamID := range []string{
		"route:d1|d2|audio",
		"route:d2|d1|audio",
		"route:d1|d2|video",
		"route:d2|d1|video",
	} {
		if !startStreams[streamID] {
			t.Fatalf("expected start_stream for %s", streamID)
		}
		if !stopStreams[streamID] {
			t.Fatalf("expected stop_stream for %s", streamID)
		}
	}
}
