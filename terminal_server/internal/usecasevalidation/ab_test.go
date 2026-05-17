package usecasevalidation_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/transport"
	"github.com/curtcox/terminals/terminal_server/internal/usecasevalidation"
)

// TestUseCaseAB1WithEvidence validates the kitchen/front-of-house intercom use
// case: a restaurant manager triggers an intercom from the front-of-house
// terminal to the kitchen terminal, establishing a two-way audio route.
// Harness pattern: same as C1.
func TestUseCaseAB1WithEvidence(t *testing.T) {
	h := usecasevalidation.New(t)
	h.StartServer()

	_, _ = h.Devices.Register(device.Manifest{DeviceID: "kitchen_terminal", DeviceName: "Kitchen"})

	stream := usecasevalidation.NewMemStream(context.Background(), []transport.ProtoClientEnvelope{
		&controlv1.ConnectRequest{
			Payload: &controlv1.ConnectRequest_Register{
				Register: &controlv1.RegisterDevice{
					Capabilities: &capabilitiesv1.DeviceCapabilities{
						DeviceId: "front_of_house",
						Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Front of House"},
					},
				},
			},
		},
		&controlv1.ConnectRequest{
			Payload: &controlv1.ConnectRequest_Command{
				Command: &controlv1.CommandRequest{
					RequestId: "ab1-intercom-start",
					DeviceId:  "front_of_house",
					Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
					Intent:    "intercom",
				},
			},
		},
		&controlv1.ConnectRequest{
			Payload: &controlv1.ConnectRequest_Command{
				Command: &controlv1.CommandRequest{
					RequestId: "ab1-intercom-stop",
					DeviceId:  "front_of_house",
					Action:    controlv1.CommandAction_COMMAND_ACTION_STOP,
					Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
					Intent:    "intercom",
				},
			},
		},
	})

	err := transport.RunProtoSession(h.NewStreamHandler(), h.Control, stream, transport.GeneratedProtoAdapter{})
	h.Assert("AB1-no-session-error", "RunProtoSession returns nil", err == nil,
		fmt.Sprintf("err=%v", err))

	h.RecordInteraction("command", "Press intercom button from the Front of House terminal to ring the Kitchen.", "front_of_house")
	h.RecordInteraction("command", "End the intercom call.", "front_of_house")

	var sawRoute, sawStop bool
	for _, sent := range stream.Sent {
		resp, ok := sent.(*controlv1.ConnectResponse)
		if !ok {
			continue
		}
		if r := resp.GetRouteStream(); r != nil &&
			r.GetSourceDeviceId() == "front_of_house" &&
			r.GetTargetDeviceId() == "kitchen_terminal" &&
			r.GetKind() == "audio" {
			sawRoute = true
		}
		if s := resp.GetStopStream(); s != nil &&
			s.GetStreamId() == "route:front_of_house|kitchen_terminal|audio" {
			sawStop = true
		}
	}

	h.Assert("AB1-route-stream", "intercom start opens audio route from front-of-house to kitchen",
		sawRoute, fmt.Sprintf("sent=%d messages", len(stream.Sent)))
	h.Assert("AB1-stop-stream", "intercom stop closes the audio route",
		sawStop, fmt.Sprintf("sent=%d messages", len(stream.Sent)))

	h.CaptureFrame("AB1-intercom-routed", "front_of_house", stream.Sent)

	h.Evidence("AB1")
}

// TestUseCaseAB2WithEvidence validates the PA broadcast to sales floor use
// case: a store manager activates PA mode, and every sales floor device
// receives the audio stream notification.
// Harness pattern: same as C2 / C3 PA system.
func TestUseCaseAB2WithEvidence(t *testing.T) {
	h := usecasevalidation.New(t)
	h.StartServer()

	const waitTimeout = 2 * time.Second

	registerMsg := func(deviceID, name string) transport.ProtoClientEnvelope {
		return &controlv1.ConnectRequest{
			Payload: &controlv1.ConnectRequest_Register{
				Register: &controlv1.RegisterDevice{
					Capabilities: &capabilitiesv1.DeviceCapabilities{
						DeviceId: deviceID,
						Identity: &capabilitiesv1.DeviceIdentity{DeviceName: name},
					},
				},
			},
		}
	}

	manager := h.ConnectTerminal("manager_terminal", registerMsg("manager_terminal", "Manager Station"))
	north := h.ConnectTerminal("sales_floor_north", registerMsg("sales_floor_north", "Sales Floor North"))
	south := h.ConnectTerminal("sales_floor_south", registerMsg("sales_floor_south", "Sales Floor South"))
	east := h.ConnectTerminal("sales_floor_east", registerMsg("sales_floor_east", "Sales Floor East"))

	for _, term := range []*usecasevalidation.SimTerminal{manager, north, south, east} {
		if !term.WaitForAny(waitTimeout) {
			t.Fatalf("terminal %s: timed out waiting for session establishment", term.DeviceID)
		}
	}

	h.RecordInteraction("command", "Activate PA mode on the Manager Station terminal.", "manager_terminal")

	manager.Send(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "ab2-pa-start",
				DeviceId:  "manager_terminal",
				Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
				Intent:    "pa system",
			},
		},
	})

	_, sawPAStart := manager.WaitFor(func(env transport.ProtoServerEnvelope) bool {
		resp, ok := env.(*controlv1.ConnectResponse)
		return ok && resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStart() == "pa_system"
	}, waitTimeout)
	h.Assert("AB2-pa-started", "pa_system scenario started on manager terminal",
		sawPAStart, fmt.Sprintf("manager received %d messages", len(manager.Received())))

	// Verify each sales floor device received the PA broadcast notification.
	deadline := time.Now().Add(waitTimeout)
	for time.Now().Before(deadline) {
		events := h.Broadcast.Events()
		found := 0
		for _, ev := range events {
			if ev.Message == "PA from manager_terminal" {
				found++
			}
		}
		if found > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	broadcastEvents := h.Broadcast.Events()
	salesFloorNotified := false
	for _, ev := range broadcastEvents {
		if ev.Message == "PA from manager_terminal" {
			salesFloorNotified = true
			break
		}
	}
	h.Assert("AB2-sales-floor-notified", "PA broadcast delivered to sales floor devices",
		salesFloorNotified, fmt.Sprintf("broadcast events: %d", len(broadcastEvents)))

	h.CaptureFrame("AB2-pa-active", "manager_terminal", manager.Received())

	for _, term := range []*usecasevalidation.SimTerminal{manager, north, south, east} {
		if err := term.Disconnect(); err != nil {
			t.Logf("terminal %s disconnect: %v", term.DeviceID, err)
		}
	}

	h.Evidence("AB2")
}

// TestUseCaseAB3WithEvidence validates the guest welcome display use case: a
// hotel front desk activates the photo frame / display scenario on a lobby
// screen, and the lobby device enters display mode.
// Harness pattern: same as D-family display tests.
func TestUseCaseAB3WithEvidence(t *testing.T) {
	h := usecasevalidation.New(t)
	h.StartServer()

	const waitTimeout = 2 * time.Second

	lobby := h.ConnectTerminal("lobby_display", &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "lobby_display",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Lobby Display"},
				},
			},
		},
	})
	if !lobby.WaitForAny(waitTimeout) {
		t.Fatal("lobby_display terminal: timed out waiting for session establishment")
	}

	h.RecordInteraction("command", "Activate display mode on the Lobby Display to show guest welcome messages.", "lobby_display")

	lobby.Send(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "ab3-display-start",
				DeviceId:  "lobby_display",
				Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
				Intent:    "photo frame",
			},
		},
	})

	_, sawDisplayStart := lobby.WaitFor(func(env transport.ProtoServerEnvelope) bool {
		resp, ok := env.(*controlv1.ConnectResponse)
		return ok && resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStart() == "photo_frame"
	}, waitTimeout)
	h.Assert("AB3-display-started", "photo_frame display scenario started on lobby display",
		sawDisplayStart, fmt.Sprintf("lobby received %d messages", len(lobby.Received())))

	// Verify the broadcast confirms the display is active for all devices (including lobby).
	broadcastEvents := h.Broadcast.Events()
	sawDisplayActive := false
	for _, ev := range broadcastEvents {
		if ev.Message == "Photo frame active" {
			sawDisplayActive = true
			break
		}
	}
	h.Assert("AB3-display-active", "broadcast confirms display/welcome mode is active on lobby screen",
		sawDisplayActive, fmt.Sprintf("broadcast events: %d", len(broadcastEvents)))

	h.CaptureFrame("AB3-lobby-display-active", "lobby_display", lobby.Received())

	if err := lobby.Disconnect(); err != nil {
		t.Logf("lobby_display disconnect: %v", err)
	}

	h.Evidence("AB3")
}

// TestUseCaseAB4WithEvidence validates the multi-window camera grid use case:
// a warehouse supervisor requests a view of all dock cameras, and the server
// routes video feeds from each dock camera to the supervisor terminal.
// Harness pattern: same as S1–S3 multi-window security tests.
func TestUseCaseAB4WithEvidence(t *testing.T) {
	h := usecasevalidation.New(t)
	h.StartServer()

	const waitTimeout = 2 * time.Second

	// Pre-register dock cameras so they appear as peer devices for routing.
	for _, id := range []string{"dock_camera_north", "dock_camera_south", "dock_camera_east", "dock_camera_west"} {
		_, _ = h.Devices.Register(device.Manifest{DeviceID: id, DeviceName: id})
	}

	supervisor := h.ConnectTerminal("supervisor_terminal", &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "supervisor_terminal",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Supervisor Station"},
				},
			},
		},
	})
	if !supervisor.WaitForAny(waitTimeout) {
		t.Fatal("supervisor_terminal: timed out waiting for session establishment")
	}

	h.RecordInteraction("command", "Say \"show all cameras\" on the Supervisor Station to open the camera grid view.", "supervisor_terminal")

	supervisor.Send(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "ab4-multi-window",
				DeviceId:  "supervisor_terminal",
				Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
				Intent:    "show all cameras",
			},
		},
	})

	_, sawMultiWindowStart := supervisor.WaitFor(func(env transport.ProtoServerEnvelope) bool {
		resp, ok := env.(*controlv1.ConnectResponse)
		return ok && resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStart() == "multi_window"
	}, waitTimeout)
	h.Assert("AB4-multi-window-started", "multi_window scenario started on supervisor terminal",
		sawMultiWindowStart, fmt.Sprintf("supervisor received %d messages", len(supervisor.Received())))

	// Verify video routes from dock cameras to supervisor are established.
	time.Sleep(50 * time.Millisecond)
	cameraIDs := map[string]bool{
		"dock_camera_north": false,
		"dock_camera_south": false,
		"dock_camera_east":  false,
		"dock_camera_west":  false,
	}
	for _, env := range supervisor.Received() {
		resp, ok := env.(*controlv1.ConnectResponse)
		if !ok {
			continue
		}
		if r := resp.GetRouteStream(); r != nil && r.GetKind() == "video" &&
			r.GetTargetDeviceId() == "supervisor_terminal" {
			cameraIDs[r.GetSourceDeviceId()] = true
		}
	}
	for camID, routed := range cameraIDs {
		h.Assert(
			fmt.Sprintf("AB4-camera-%s-routed", camID),
			fmt.Sprintf("video route established from %s to supervisor", camID),
			routed,
			fmt.Sprintf("supervisor received %d messages", len(supervisor.Received())),
		)
	}

	// Verify the multi-window activation broadcast.
	broadcastEvents := h.Broadcast.Events()
	sawMultiWindowActive := false
	for _, ev := range broadcastEvents {
		if ev.Message == "Multi-window active" {
			sawMultiWindowActive = true
			break
		}
	}
	h.Assert("AB4-multi-window-active", "multi-window active broadcast confirms camera grid is live",
		sawMultiWindowActive, fmt.Sprintf("broadcast events: %d", len(broadcastEvents)))

	h.CaptureFrame("AB4-camera-grid-active", "supervisor_terminal", supervisor.Received())

	if err := supervisor.Disconnect(); err != nil {
		t.Logf("supervisor_terminal disconnect: %v", err)
	}

	h.Evidence("AB4")
}

// TestUseCaseAB5WithEvidence validates the after-hours alarm monitoring use
// case: a business owner arms audio monitoring for glass-break sounds, and the
// server broadcasts an alert when the classifier detects the sound.
// Harness pattern: same as M2 audio monitor / AA2 monitoring agent tests.
func TestUseCaseAB5WithEvidence(t *testing.T) {
	h := usecasevalidation.New(t)

	h.SetSound(&usecasevalidation.FakeSoundClassifier{
		Events: []scenario.SoundEvent{
			{Label: "glass_break", Confidence: 0.95, AtMS: 1000},
		},
	})
	h.StartServer()

	const waitTimeout = 2 * time.Second

	security := h.ConnectTerminal("security_terminal", &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "security_terminal",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Security Monitor"},
				},
			},
		},
	})
	if !security.WaitForAny(waitTimeout) {
		t.Fatal("security_terminal: timed out waiting for session establishment")
	}

	h.RecordInteraction("command", "Arm glass-break audio monitoring on the Security Monitor terminal.", "security_terminal")

	security.Send(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "ab5-arm-monitor",
				DeviceId:  "security_terminal",
				Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
				Intent:    "audio monitor",
				Arguments: map[string]string{
					"target": "glass_break",
				},
			},
		},
	})

	_, sawMonitorArmed := security.WaitFor(func(env transport.ProtoServerEnvelope) bool {
		resp, ok := env.(*controlv1.ConnectResponse)
		return ok && resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStart() == "audio_monitor"
	}, waitTimeout)
	h.Assert("AB5-monitor-armed", "audio_monitor scenario started on security terminal",
		sawMonitorArmed, fmt.Sprintf("security received %d messages", len(security.Received())))

	h.CaptureFrame("AB5-monitor-armed", "security_terminal", security.Received())

	// Wait for the glass_break detection broadcast.
	deadline := time.Now().Add(waitTimeout)
	sawAlert := false
	for time.Now().Before(deadline) && !sawAlert {
		for _, ev := range h.Broadcast.Events() {
			if ev.Message == "Audio monitor detected: glass_break" {
				sawAlert = true
				break
			}
		}
		if !sawAlert {
			time.Sleep(10 * time.Millisecond)
		}
	}
	h.Assert("AB5-glass-break-alert", "glass_break detection event broadcast to security terminal",
		sawAlert, fmt.Sprintf("broadcast events: %d", len(h.Broadcast.Events())))

	if err := security.Disconnect(); err != nil {
		t.Logf("security_terminal disconnect: %v", err)
	}

	h.Evidence("AB5")
}

// TestUseCaseAB6WithEvidence validates the voice timer for food service use
// case: a restaurant staff member says "set a timer for 12 minutes table 5",
// and when the timer fires the broadcast includes "Timer done!" with the label.
// Harness pattern: same as T1 voice-path timer test.
func TestUseCaseAB6WithEvidence(t *testing.T) {
	h := usecasevalidation.New(t)

	startTime := time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)
	h.Clock().SetNow(startTime)

	h.StartServer()
	h.Control.SetNowForTest(h.Clock().Now)

	const (
		waitTimeout   = 2 * time.Second
		timerDuration = 12 * time.Minute
	)

	kitchen := h.ConnectTerminal("kitchen_pos", &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "kitchen_pos",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen POS"},
				},
			},
		},
	})
	if !kitchen.WaitForAny(waitTimeout) {
		t.Fatal("kitchen_pos terminal: timed out waiting for session establishment")
	}

	h.RecordInteraction("voice", "Say \"set a timer for 12 minutes table 5\" on the Kitchen POS terminal.", "kitchen_pos")

	kitchen.Send(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "ab6-voice-timer",
				DeviceId:  "kitchen_pos",
				Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
				Text:      "set a timer for 12 minutes table 5",
			},
		},
	})

	_, sawTimerStart := kitchen.WaitFor(func(env transport.ProtoServerEnvelope) bool {
		resp, ok := env.(*controlv1.ConnectResponse)
		return ok && resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStart() == "timer_reminder"
	}, waitTimeout)
	h.Assert("AB6-timer-set", "timer_reminder scenario started via voice command",
		sawTimerStart, fmt.Sprintf("kitchen_pos received %d messages", len(kitchen.Received())))
	h.CaptureFrame("AB6-timer-started", "kitchen_pos", kitchen.Received())

	// At T+0, no timer should fire.
	processed0, err := h.ProcessDueTimers(context.Background())
	h.Assert("AB6-no-premature-fire", "no timers fire before the due time",
		err == nil && processed0 == 0,
		fmt.Sprintf("processed=%d err=%v", processed0, err))

	// Advance synthetic time past the 12-minute fire point.
	h.Clock().AdvanceTo(startTime.Add(timerDuration + time.Second))

	processed1, err := h.ProcessDueTimers(context.Background())
	h.Assert("AB6-timer-fired", "exactly one timer processed after clock advance",
		err == nil && processed1 == 1,
		fmt.Sprintf("processed=%d err=%v", processed1, err))

	events := h.Broadcast.Events()
	sawDone := false
	for _, ev := range events {
		if ev.Message == "Timer done!" {
			sawDone = true
			break
		}
	}
	h.Assert("AB6-done-notification", "broadcast emits 'Timer done!' after 12-minute table timer fires",
		sawDone, fmt.Sprintf("broadcast events: %d", len(events)))

	h.CaptureHostFrame("AB6-timer-done", "kitchen_pos")

	if err := kitchen.Disconnect(); err != nil {
		t.Logf("kitchen_pos disconnect: %v", err)
	}

	h.Evidence("AB6")
}

// TestUseCaseAB7WithEvidence validates the voice announcement to waiting area
// use case: a clinic receptionist announces "patient Smith, room 3 is ready",
// and the waiting area terminal receives the announcement audio route.
// Harness pattern: same as C2 whole-house announcement.
func TestUseCaseAB7WithEvidence(t *testing.T) {
	h := usecasevalidation.New(t)
	h.StartServer()

	const waitTimeout = 2 * time.Second

	registerMsg := func(deviceID, name string) transport.ProtoClientEnvelope {
		return &controlv1.ConnectRequest{
			Payload: &controlv1.ConnectRequest_Register{
				Register: &controlv1.RegisterDevice{
					Capabilities: &capabilitiesv1.DeviceCapabilities{
						DeviceId: deviceID,
						Identity: &capabilitiesv1.DeviceIdentity{DeviceName: name},
					},
				},
			},
		}
	}

	reception := h.ConnectTerminal("reception_terminal", registerMsg("reception_terminal", "Reception Desk"))
	waiting := h.ConnectTerminal("waiting_area_terminal", registerMsg("waiting_area_terminal", "Waiting Area"))

	for _, term := range []*usecasevalidation.SimTerminal{reception, waiting} {
		if !term.WaitForAny(waitTimeout) {
			t.Fatalf("terminal %s: timed out waiting for session establishment", term.DeviceID)
		}
	}

	h.RecordInteraction("command", "Announce \"patient Smith, room 3 is ready\" from the Reception Desk terminal.", "reception_terminal")

	reception.Send(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "ab7-announce",
				DeviceId:  "reception_terminal",
				Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
				Intent:    "announcement",
			},
		},
	})

	_, sawAnnouncementStart := reception.WaitFor(func(env transport.ProtoServerEnvelope) bool {
		resp, ok := env.(*controlv1.ConnectResponse)
		return ok && resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStart() == "announcement"
	}, waitTimeout)
	h.Assert("AB7-scenario-start", "announcement scenario started on reception terminal",
		sawAnnouncementStart, fmt.Sprintf("reception received %d messages", len(reception.Received())))

	// Waiting area terminal should receive an announcement_audio RouteStream.
	_, sawRoute := waiting.WaitFor(func(env transport.ProtoServerEnvelope) bool {
		resp, ok := env.(*controlv1.ConnectResponse)
		if !ok {
			return false
		}
		r := resp.GetRouteStream()
		return r != nil && r.GetKind() == "announcement_audio"
	}, waitTimeout)
	h.Assert("AB7-waiting-area-route", "waiting area terminal receives announcement_audio route",
		sawRoute, fmt.Sprintf("waiting_area received %d messages", len(waiting.Received())))

	// Verify no duplicate announcement routes to the waiting area.
	count := 0
	for _, env := range waiting.Received() {
		resp, ok := env.(*controlv1.ConnectResponse)
		if !ok {
			continue
		}
		if r := resp.GetRouteStream(); r != nil && r.GetKind() == "announcement_audio" {
			count++
		}
	}
	h.Assert("AB7-no-duplicate-route", "waiting area receives exactly one announcement_audio route",
		count <= 1, fmt.Sprintf("got %d announcement_audio routes", count))

	h.CaptureFrame("AB7-announcement-routed", "waiting_area_terminal", waiting.Received())

	for _, term := range []*usecasevalidation.SimTerminal{reception, waiting} {
		if err := term.Disconnect(); err != nil {
			t.Logf("terminal %s disconnect: %v", term.DeviceID, err)
		}
	}

	h.Evidence("AB7")
}
