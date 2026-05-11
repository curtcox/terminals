package transport

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/telephony"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestGeneratedSessionPASystemRelaysReceiverOverlayAndTransitions(t *testing.T) {
	globalSessionRelayRegistry = newSessionRelayRegistry()
	t.Cleanup(func() {
		globalSessionRelayRegistry = newSessionRelayRegistry()
	})

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

	stream1 := &asyncFakeProtoStream{
		ctx:    context.Background(),
		recvCh: make(chan ProtoClientEnvelope, 8),
		sentCh: make(chan ProtoServerEnvelope, 16),
	}
	stream2 := &asyncFakeProtoStream{
		ctx:    context.Background(),
		recvCh: make(chan ProtoClientEnvelope, 8),
		sentCh: make(chan ProtoServerEnvelope, 16),
	}

	var wg sync.WaitGroup
	wg.Add(2)
	var runErr1 error
	var runErr2 error
	go func() {
		defer wg.Done()
		runErr1 = RunProtoSession(NewStreamHandlerWithRuntime(control, runtime), control, stream1, GeneratedProtoAdapter{})
	}()
	go func() {
		defer wg.Done()
		runErr2 = RunProtoSession(NewStreamHandlerWithRuntime(control, runtime), control, stream2, GeneratedProtoAdapter{})
	}()

	waitFor := func(label string, ch <-chan ProtoServerEnvelope, pred func(*controlv1.ConnectResponse) bool) *controlv1.ConnectResponse {
		seen := make([]string, 0, 8)
		deadline := time.After(2 * time.Second)
		for {
			select {
			case env := <-ch:
				resp, ok := env.(*controlv1.ConnectResponse)
				if !ok {
					t.Fatalf("unexpected envelope type %T", env)
				}
				switch {
				case resp.GetCommandResult() != nil:
					seen = append(seen, "command_result:"+resp.GetCommandResult().GetScenarioStart()+"/"+resp.GetCommandResult().GetScenarioStop())
				case resp.GetUpdateUi() != nil:
					seen = append(seen, "update_ui:"+resp.GetUpdateUi().GetComponentId())
				case resp.GetTransitionUi() != nil:
					seen = append(seen, "transition_ui:"+resp.GetTransitionUi().GetTransition())
				case resp.GetStartStream() != nil:
					seen = append(seen, "start_stream:"+resp.GetStartStream().GetStreamId())
				case resp.GetStopStream() != nil:
					seen = append(seen, "stop_stream:"+resp.GetStopStream().GetStreamId())
				case resp.GetRouteStream() != nil:
					seen = append(seen, "route_stream:"+resp.GetRouteStream().GetStreamId())
				default:
					seen = append(seen, "other")
				}
				if pred(resp) {
					return resp
				}
			case <-deadline:
				t.Fatalf("timed out waiting for %s (seen=%v)", label, seen)
			}
		}
	}

	stream1.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "device-1",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
				},
			},
		},
	}
	stream2.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "device-2",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Hall"},
				},
			},
		},
	}

	for i := 0; i < 2; i++ {
		<-stream1.sentCh
		<-stream2.sentCh
	}

	stream1.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "pa-start",
				DeviceId:  "device-1",
				Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
				Intent:    "pa_system",
			},
		},
	}

	startDone := false
	sourceEnterDone := false
	waitFor("pa source start payloads", stream1.sentCh, func(resp *controlv1.ConnectResponse) bool {
		if resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStart() == "pa_system" {
			startDone = true
		}
		if resp.GetTransitionUi() != nil && resp.GetTransitionUi().GetTransition() == "pa_source_enter" {
			sourceEnterDone = true
		}
		return startDone && sourceEnterDone
	})

	receiverOverlayDone := false
	receiverEnterDone := false
	waitFor("pa receiver start payloads", stream2.sentCh, func(resp *controlv1.ConnectResponse) bool {
		if resp.GetUpdateUi() != nil &&
			(resp.GetUpdateUi().GetComponentId() == ui.GlobalOverlayComponentID ||
				strings.HasSuffix(resp.GetUpdateUi().GetComponentId(), "/"+ui.GlobalOverlayComponentID)) {
			if got := resp.GetUpdateUi().GetNode().GetProps()["id"]; got != ui.GlobalOverlayComponentID &&
				!strings.HasSuffix(got, "/"+ui.GlobalOverlayComponentID) {
				t.Fatalf("receiver overlay id prop = %q, want scoped or legacy %q", got, ui.GlobalOverlayComponentID)
			}
			receiverOverlayDone = true
		}
		if resp.GetTransitionUi() != nil && resp.GetTransitionUi().GetTransition() == "pa_receive_enter" {
			receiverEnterDone = true
		}
		return receiverOverlayDone && receiverEnterDone
	})

	stream1.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "pa-stop",
				DeviceId:  "device-1",
				Action:    controlv1.CommandAction_COMMAND_ACTION_STOP,
				Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
				Intent:    "pa_system",
			},
		},
	}

	stopDone := false
	sourceExitDone := false
	waitFor("pa source stop payloads", stream1.sentCh, func(resp *controlv1.ConnectResponse) bool {
		if resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStop() == "pa_system" {
			stopDone = true
		}
		if resp.GetTransitionUi() != nil && resp.GetTransitionUi().GetTransition() == "pa_source_exit" {
			sourceExitDone = true
		}
		return stopDone && sourceExitDone
	})

	receiverClearDone := false
	receiverExitDone := false
	waitFor("pa receiver stop payloads", stream2.sentCh, func(resp *controlv1.ConnectResponse) bool {
		if resp.GetUpdateUi() != nil &&
			(resp.GetUpdateUi().GetComponentId() == ui.GlobalOverlayComponentID ||
				strings.HasSuffix(resp.GetUpdateUi().GetComponentId(), "/"+ui.GlobalOverlayComponentID)) {
			node := resp.GetUpdateUi().GetNode()
			if (node.GetProps()["id"] == ui.GlobalOverlayComponentID ||
				strings.HasSuffix(node.GetProps()["id"], "/"+ui.GlobalOverlayComponentID)) &&
				len(node.GetChildren()) == 0 {
				receiverClearDone = true
			}
		}
		if resp.GetTransitionUi() != nil && resp.GetTransitionUi().GetTransition() == "pa_receive_exit" {
			receiverExitDone = true
		}
		return receiverClearDone && receiverExitDone
	})

	close(stream1.recvCh)
	close(stream2.recvCh)
	wg.Wait()
	if runErr1 != nil {
		t.Fatalf("session1 RunProtoSession() error = %v", runErr1)
	}
	if runErr2 != nil {
		t.Fatalf("session2 RunProtoSession() error = %v", runErr2)
	}
}

func TestGeneratedSessionRedAlertRelaysBroadcastNotification(t *testing.T) {
	globalSessionRelayRegistry = newSessionRelayRegistry()
	t.Cleanup(func() {
		globalSessionRelayRegistry = newSessionRelayRegistry()
	})

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

	stream1 := &asyncFakeProtoStream{
		ctx:    context.Background(),
		recvCh: make(chan ProtoClientEnvelope, 8),
		sentCh: make(chan ProtoServerEnvelope, 16),
	}
	stream2 := &asyncFakeProtoStream{
		ctx:    context.Background(),
		recvCh: make(chan ProtoClientEnvelope, 8),
		sentCh: make(chan ProtoServerEnvelope, 16),
	}

	var wg sync.WaitGroup
	wg.Add(2)
	var runErr1 error
	var runErr2 error
	go func() {
		defer wg.Done()
		runErr1 = RunProtoSession(NewStreamHandlerWithRuntime(control, runtime), control, stream1, GeneratedProtoAdapter{})
	}()
	go func() {
		defer wg.Done()
		runErr2 = RunProtoSession(NewStreamHandlerWithRuntime(control, runtime), control, stream2, GeneratedProtoAdapter{})
	}()

	waitFor := func(label string, ch <-chan ProtoServerEnvelope, pred func(*controlv1.ConnectResponse) bool) {
		deadline := time.After(2 * time.Second)
		for {
			select {
			case env := <-ch:
				resp, ok := env.(*controlv1.ConnectResponse)
				if !ok {
					t.Fatalf("unexpected envelope type %T", env)
				}
				if pred(resp) {
					return
				}
			case <-deadline:
				t.Fatalf("timed out waiting for %s", label)
			}
		}
	}

	stream1.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "d1",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
				},
			},
		},
	}
	stream2.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "d2",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Hall"},
				},
			},
		},
	}
	for i := 0; i < 2; i++ {
		<-stream1.sentCh
		<-stream2.sentCh
	}

	stream1.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "cmd-red-alert",
				DeviceId:  "d1",
				Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
				Text:      "red alert",
			},
		},
	}

	waitFor("source red_alert command result", stream1.sentCh, func(resp *controlv1.ConnectResponse) bool {
		return resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStart() == "red_alert"
	})
	waitFor("peer RED ALERT notification relay", stream2.sentCh, func(resp *controlv1.ConnectResponse) bool {
		return resp.GetCommandResult() != nil && resp.GetCommandResult().GetNotification() == "RED ALERT"
	})

	close(stream1.recvCh)
	close(stream2.recvCh)
	wg.Wait()
	if runErr1 != nil {
		t.Fatalf("session1 RunProtoSession() error = %v", runErr1)
	}
	if runErr2 != nil {
		t.Fatalf("session2 RunProtoSession() error = %v", runErr2)
	}
}

func TestGeneratedSessionPASystemVoiceStopAliasesRelayCleanup(t *testing.T) {
	for _, spoken := range []string{"end pa", "stop pa"} {
		t.Run(spoken, func(t *testing.T) {
			globalSessionRelayRegistry = newSessionRelayRegistry()
			t.Cleanup(func() {
				globalSessionRelayRegistry = newSessionRelayRegistry()
			})

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

			stream1 := &asyncFakeProtoStream{
				ctx:    context.Background(),
				recvCh: make(chan ProtoClientEnvelope, 8),
				sentCh: make(chan ProtoServerEnvelope, 24),
			}
			stream2 := &asyncFakeProtoStream{
				ctx:    context.Background(),
				recvCh: make(chan ProtoClientEnvelope, 8),
				sentCh: make(chan ProtoServerEnvelope, 24),
			}

			var wg sync.WaitGroup
			wg.Add(2)
			var runErr1 error
			var runErr2 error
			go func() {
				defer wg.Done()
				runErr1 = RunProtoSession(NewStreamHandlerWithRuntime(control, runtime), control, stream1, GeneratedProtoAdapter{})
			}()
			go func() {
				defer wg.Done()
				runErr2 = RunProtoSession(NewStreamHandlerWithRuntime(control, runtime), control, stream2, GeneratedProtoAdapter{})
			}()

			waitFor := func(label string, ch <-chan ProtoServerEnvelope, pred func(*controlv1.ConnectResponse) bool) {
				deadline := time.After(2 * time.Second)
				for {
					select {
					case env := <-ch:
						resp, ok := env.(*controlv1.ConnectResponse)
						if !ok {
							t.Fatalf("unexpected envelope type %T", env)
						}
						if pred(resp) {
							return
						}
					case <-deadline:
						t.Fatalf("timed out waiting for %s", label)
					}
				}
			}

			stream1.recvCh <- &controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Register{
					Register: &controlv1.RegisterDevice{
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "d1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
						},
					},
				},
			}
			stream2.recvCh <- &controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Register{
					Register: &controlv1.RegisterDevice{
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "d2",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Hall"},
						},
					},
				},
			}
			for i := 0; i < 2; i++ {
				<-stream1.sentCh
				<-stream2.sentCh
			}

			stream1.recvCh <- &controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "pa-start",
						DeviceId:  "d1",
						Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
						Intent:    "pa_system",
					},
				},
			}

			waitFor("source pa start", stream1.sentCh, func(resp *controlv1.ConnectResponse) bool {
				return resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStart() == "pa_system"
			})
			waitFor("peer pa start route", stream2.sentCh, func(resp *controlv1.ConnectResponse) bool {
				return resp.GetStartStream() != nil && resp.GetStartStream().GetStreamId() == "route:d1|d2|pa_audio"
			})

			stream1.recvCh <- &controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "pa-stop-voice",
						DeviceId:  "d1",
						Action:    controlv1.CommandAction_COMMAND_ACTION_STOP,
						Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
						Text:      spoken,
					},
				},
			}

			waitFor("source pa stop via voice alias", stream1.sentCh, func(resp *controlv1.ConnectResponse) bool {
				return resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStop() == "pa_system"
			})
			waitFor("source pa source_exit transition", stream1.sentCh, func(resp *controlv1.ConnectResponse) bool {
				return resp.GetTransitionUi() != nil && resp.GetTransitionUi().GetTransition() == "pa_source_exit"
			})

			peerStopSeen := false
			peerOverlayClearSeen := false
			peerReceiveExitSeen := false
			waitFor("peer pa stop cleanup relays", stream2.sentCh, func(resp *controlv1.ConnectResponse) bool {
				if stop := resp.GetStopStream(); stop != nil && stop.GetStreamId() == "route:d1|d2|pa_audio" {
					peerStopSeen = true
				}
				if update := resp.GetUpdateUi(); update != nil &&
					(update.GetComponentId() == ui.GlobalOverlayComponentID ||
						strings.HasSuffix(update.GetComponentId(), "/"+ui.GlobalOverlayComponentID)) {
					node := update.GetNode()
					if (node.GetProps()["id"] == ui.GlobalOverlayComponentID ||
						strings.HasSuffix(node.GetProps()["id"], "/"+ui.GlobalOverlayComponentID)) &&
						len(node.GetChildren()) == 0 {
						peerOverlayClearSeen = true
					}
				}
				if transition := resp.GetTransitionUi(); transition != nil && transition.GetTransition() == "pa_receive_exit" {
					peerReceiveExitSeen = true
				}
				return peerStopSeen && peerOverlayClearSeen && peerReceiveExitSeen
			})

			close(stream1.recvCh)
			close(stream2.recvCh)
			wg.Wait()
			if runErr1 != nil {
				t.Fatalf("session1 RunProtoSession() error = %v", runErr1)
			}
			if runErr2 != nil {
				t.Fatalf("session2 RunProtoSession() error = %v", runErr2)
			}
		})
	}
}

func TestGeneratedSessionWebRTCSignalRelayAcrossSessions(t *testing.T) {
	globalSessionRelayRegistry = newSessionRelayRegistry()
	t.Cleanup(func() {
		globalSessionRelayRegistry = newSessionRelayRegistry()
	})

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

	stream1 := &asyncFakeProtoStream{
		ctx:    context.Background(),
		recvCh: make(chan ProtoClientEnvelope, 8),
		sentCh: make(chan ProtoServerEnvelope, 24),
	}
	stream2 := &asyncFakeProtoStream{
		ctx:    context.Background(),
		recvCh: make(chan ProtoClientEnvelope, 8),
		sentCh: make(chan ProtoServerEnvelope, 24),
	}

	var wg sync.WaitGroup
	wg.Add(2)
	var runErr1 error
	var runErr2 error
	go func() {
		defer wg.Done()
		runErr1 = RunProtoSession(NewStreamHandlerWithRuntime(control, runtime), control, stream1, GeneratedProtoAdapter{})
	}()
	go func() {
		defer wg.Done()
		runErr2 = RunProtoSession(NewStreamHandlerWithRuntime(control, runtime), control, stream2, GeneratedProtoAdapter{})
	}()

	waitFor := func(label string, ch <-chan ProtoServerEnvelope, pred func(*controlv1.ConnectResponse) bool) {
		deadline := time.After(2 * time.Second)
		for {
			select {
			case env := <-ch:
				resp, ok := env.(*controlv1.ConnectResponse)
				if !ok {
					t.Fatalf("unexpected envelope type %T", env)
				}
				if pred(resp) {
					return
				}
			case <-deadline:
				t.Fatalf("timed out waiting for %s", label)
			}
		}
	}

	stream1.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "d1",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
				},
			},
		},
	}
	stream2.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "d2",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Hall"},
				},
			},
		},
	}
	for i := 0; i < 2; i++ {
		<-stream1.sentCh
		<-stream2.sentCh
	}

	stream1.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "intercom-start",
				DeviceId:  "d1",
				Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
				Intent:    "intercom",
			},
		},
	}
	waitFor("source intercom start", stream1.sentCh, func(resp *controlv1.ConnectResponse) bool {
		return resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStart() == "intercom"
	})
	waitFor("peer intercom route", stream2.sentCh, func(resp *controlv1.ConnectResponse) bool {
		return resp.GetRouteStream() != nil && resp.GetRouteStream().GetStreamId() == "route:d1|d2|audio"
	})

	stream1.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_WebrtcSignal{
			WebrtcSignal: &controlv1.WebRTCSignal{
				StreamId:   "route:d1|d2|audio",
				SignalType: "offer",
				Payload:    "{\"sdp\":\"v=0-offer\"}",
			},
		},
	}
	waitFor("relayed offer to peer", stream2.sentCh, func(resp *controlv1.ConnectResponse) bool {
		signal := resp.GetWebrtcSignal()
		return signal != nil &&
			signal.GetStreamId() == "route:d1|d2|audio" &&
			signal.GetSignalType() == "offer" &&
			signal.GetPayload() == "{\"sdp\":\"v=0-offer\"}"
	})

	stream2.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_WebrtcSignal{
			WebrtcSignal: &controlv1.WebRTCSignal{
				StreamId:   "route:d1|d2|audio",
				SignalType: "answer",
				Payload:    "{\"sdp\":\"v=0-answer\"}",
			},
		},
	}
	waitFor("relayed answer to source", stream1.sentCh, func(resp *controlv1.ConnectResponse) bool {
		signal := resp.GetWebrtcSignal()
		return signal != nil &&
			signal.GetStreamId() == "route:d1|d2|audio" &&
			signal.GetSignalType() == "answer" &&
			signal.GetPayload() == "{\"sdp\":\"v=0-answer\"}"
	})

	close(stream1.recvCh)
	close(stream2.recvCh)
	wg.Wait()
	if runErr1 != nil {
		t.Fatalf("session1 RunProtoSession() error = %v", runErr1)
	}
	if runErr2 != nil {
		t.Fatalf("session2 RunProtoSession() error = %v", runErr2)
	}
}
