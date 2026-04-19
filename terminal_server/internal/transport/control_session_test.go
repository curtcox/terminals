package transport

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/telephony"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

type fakeStream struct {
	ctx       context.Context
	recvQueue []ClientMessage
	sent      []ServerMessage
}

func (f *fakeStream) Recv() (ClientMessage, error) {
	if len(f.recvQueue) == 0 {
		return ClientMessage{}, io.EOF
	}
	msg := f.recvQueue[0]
	f.recvQueue = f.recvQueue[1:]
	return msg, nil
}

func (f *fakeStream) Send(msg ServerMessage) error {
	f.sent = append(f.sent, msg)
	return nil
}

func (f *fakeStream) Context() context.Context {
	return f.ctx
}

func TestSessionRunRegisterAndDisconnect(t *testing.T) {
	manager := device.NewManager()
	control := NewControlService("srv-1", manager)
	handler := NewStreamHandler(control)
	session := NewSession(handler, control)

	stream := &fakeStream{
		ctx: context.Background(),
		recvQueue: []ClientMessage{
			{
				Register: &RegisterRequest{
					DeviceID:   "device-1",
					DeviceName: "Kitchen Chromebook",
				},
			},
		},
	}

	if err := session.Run(stream); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(stream.sent) != 2 {
		t.Fatalf("len(sent) = %d, want 2", len(stream.sent))
	}
	if stream.sent[1].SetUI == nil {
		t.Fatalf("second message should include SetUI")
	}
	if !sessionUIDescriptorHasBugButton(*stream.sent[1].SetUI) {
		t.Fatalf("register SetUI should include bug-report affordance")
	}

	got, ok := manager.Get("device-1")
	if !ok {
		t.Fatalf("expected registered device")
	}
	if got.State != device.StateDisconnected {
		t.Fatalf("state = %q, want %q", got.State, device.StateDisconnected)
	}
}

func TestSessionRunHelloSnapshotDeltaAndDisconnect(t *testing.T) {
	manager := device.NewManager()
	control := NewControlService("srv-1", manager)
	handler := NewStreamHandler(control)
	session := NewSession(handler, control)

	stream := &fakeStream{
		ctx: context.Background(),
		recvQueue: []ClientMessage{
			{Hello: &HelloRequest{DeviceID: "device-1", DeviceName: "Kitchen", DeviceType: "tablet", Platform: "android"}},
			{CapabilitySnap: &CapabilitySnapshotRequest{
				DeviceID:   "device-1",
				Generation: 1,
				Capabilities: map[string]string{
					"screen.width":  "1920",
					"screen.height": "1080",
				},
			}},
			{CapabilityDelta: &CapabilityDeltaRequest{
				DeviceID:   "device-1",
				Generation: 2,
				Reason:     "display_changed",
				Capabilities: map[string]string{
					"screen.width":  "1280",
					"screen.height": "720",
				},
			}},
		},
	}

	if err := session.Run(stream); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	got, ok := manager.Get("device-1")
	if !ok {
		t.Fatalf("expected registered device")
	}
	if got.Generation != 2 {
		t.Fatalf("generation = %d, want 2", got.Generation)
	}
	if got.Capabilities["screen.width"] != "1280" {
		t.Fatalf("screen.width = %q, want 1280", got.Capabilities["screen.width"])
	}
	if got.State != device.StateDisconnected {
		t.Fatalf("state = %q, want %q", got.State, device.StateDisconnected)
	}

	hasHelloAck := false
	hasCapabilityAck := false
	for _, sent := range stream.sent {
		if sent.HelloAck != nil {
			hasHelloAck = true
		}
		if sent.CapabilityAck != nil {
			hasCapabilityAck = true
		}
	}
	if !hasHelloAck {
		t.Fatalf("expected hello ack")
	}
	if !hasCapabilityAck {
		t.Fatalf("expected capability ack")
	}
}

func sessionUIDescriptorHasBugButton(root ui.Descriptor) bool {
	nodeID := root.ID
	if nodeID == "" {
		nodeID = root.Props["id"]
	}
	if nodeID == bugReportButtonID {
		return true
	}
	if root.Type == "button" && root.Props["action"] == bugReportActionPrefix+":device-1" {
		return true
	}
	for _, child := range root.Children {
		if sessionUIDescriptorHasBugButton(child) {
			return true
		}
	}
	return false
}

func TestSessionRunCapabilityAndHeartbeat(t *testing.T) {
	manager := device.NewManager()
	control := NewControlService("srv-1", manager)
	handler := NewStreamHandler(control)
	session := NewSession(handler, control)

	stream := &fakeStream{
		ctx: context.Background(),
		recvQueue: []ClientMessage{
			{Register: &RegisterRequest{DeviceID: "d1", DeviceName: "Kitchen"}},
			{Capability: &CapabilityUpdateRequest{
				DeviceID: "d1",
				Capabilities: map[string]string{
					"screen.width": "1920",
				},
			}},
			{Heartbeat: &HeartbeatRequest{DeviceID: "d1"}},
		},
	}

	if err := session.Run(stream); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	got, ok := manager.Get("d1")
	if !ok {
		t.Fatalf("expected device d1")
	}
	if got.Capabilities["screen.width"] != "1920" {
		t.Fatalf("screen.width = %q, want 1920", got.Capabilities["screen.width"])
	}
}

func TestSessionRunContinuesAfterRecoverableError(t *testing.T) {
	manager := device.NewManager()
	control := NewControlService("srv-1", manager)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   manager,
		IO:        iorouter.NewRouter(),
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	session := NewSession(handler, control)

	stream := &fakeStream{
		ctx: context.Background(),
		recvQueue: []ClientMessage{
			{Register: &RegisterRequest{DeviceID: "d1", DeviceName: "Kitchen"}},
			{Command: &CommandRequest{
				RequestID: "bad-action",
				DeviceID:  "d1",
				Action:    "pause",
				Kind:      "manual",
				Intent:    "photo frame",
			}},
			{Heartbeat: &HeartbeatRequest{DeviceID: "d1"}},
		},
	}

	if err := session.Run(stream); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Register emits 2 responses. Bad command emits structured error response.
	if len(stream.sent) < 3 {
		t.Fatalf("len(sent) = %d, want at least 3", len(stream.sent))
	}
	last := stream.sent[len(stream.sent)-1]
	if last.Error == "" {
		t.Fatalf("expected structured error response for invalid command action")
	}

	got, ok := manager.Get("d1")
	if !ok {
		t.Fatalf("expected device d1")
	}
	// Stream EOF marks device disconnected, but heartbeat should have been processed before EOF.
	if got.LastHeartbeat.IsZero() {
		t.Fatalf("expected heartbeat to be processed after recoverable error")
	}
}

func TestSessionRunRejectsPreRegisterMessageButContinues(t *testing.T) {
	manager := device.NewManager()
	control := NewControlService("srv-1", manager)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   manager,
		IO:        iorouter.NewRouter(),
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	session := NewSession(handler, control)

	stream := &fakeStream{
		ctx: context.Background(),
		recvQueue: []ClientMessage{
			{Heartbeat: &HeartbeatRequest{DeviceID: "d1"}},
			{Register: &RegisterRequest{DeviceID: "d1", DeviceName: "Kitchen"}},
			{Heartbeat: &HeartbeatRequest{DeviceID: "d1"}},
		},
	}

	if err := session.Run(stream); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(stream.sent) < 3 {
		t.Fatalf("len(sent) = %d, want at least 3", len(stream.sent))
	}
	if stream.sent[0].Error == "" {
		t.Fatalf("first message should be structured pre-register error")
	}

	got, ok := manager.Get("d1")
	if !ok {
		t.Fatalf("expected device d1")
	}
	if got.LastHeartbeat.IsZero() {
		t.Fatalf("expected post-register heartbeat to be processed")
	}
	if handler.metrics.protocolErrors.Load() == 0 {
		t.Fatalf("expected protocol error metric increment")
	}
}

func TestSessionRunDefersHeartbeatUntilCapabilitySnapshot(t *testing.T) {
	manager := device.NewManager()
	control := NewControlService("srv-1", manager)
	handler := NewStreamHandler(control)
	session := NewSession(handler, control)

	stream := &fakeStream{
		ctx: context.Background(),
		recvQueue: []ClientMessage{
			{Hello: &HelloRequest{DeviceID: "device-1", DeviceName: "Kitchen"}},
			{Heartbeat: &HeartbeatRequest{DeviceID: "device-1"}},
			{CapabilitySnap: &CapabilitySnapshotRequest{
				DeviceID:   "device-1",
				Generation: 1,
				Capabilities: map[string]string{
					"screen.width":  "1920",
					"screen.height": "1080",
				},
			}},
			{Heartbeat: &HeartbeatRequest{DeviceID: "device-1"}},
		},
	}

	if err := session.Run(stream); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(stream.sent) < 3 {
		t.Fatalf("len(sent) = %d, want at least 3", len(stream.sent))
	}
	if stream.sent[1].ErrorCode != ErrorCodeProtocolViolation {
		t.Fatalf("pre-snapshot message should fail with protocol violation, got %+v", stream.sent[1])
	}

	got, ok := manager.Get("device-1")
	if !ok {
		t.Fatalf("expected registered device")
	}
	if got.LastHeartbeat.IsZero() {
		t.Fatalf("expected heartbeat after snapshot acceptance")
	}
}

func TestSessionRunAllowsSnapshotRebaselineAfterStaleDelta(t *testing.T) {
	manager := device.NewManager()
	control := NewControlService("srv-1", manager)
	handler := NewStreamHandler(control)
	session := NewSession(handler, control)

	stream := &fakeStream{
		ctx: context.Background(),
		recvQueue: []ClientMessage{
			{Hello: &HelloRequest{DeviceID: "device-1", DeviceName: "Kitchen"}},
			{CapabilitySnap: &CapabilitySnapshotRequest{
				DeviceID:   "device-1",
				Generation: 3,
				Capabilities: map[string]string{
					"screen.width":  "1920",
					"screen.height": "1080",
				},
			}},
			{CapabilityDelta: &CapabilityDeltaRequest{
				DeviceID:   "device-1",
				Generation: 2,
				Reason:     "stale_delta",
				Capabilities: map[string]string{
					"screen.width":  "1280",
					"screen.height": "720",
				},
			}},
			{CapabilitySnap: &CapabilitySnapshotRequest{
				DeviceID:   "device-1",
				Generation: 4,
				Capabilities: map[string]string{
					"screen.width":  "1366",
					"screen.height": "768",
				},
			}},
		},
	}

	if err := session.Run(stream); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	got, ok := manager.Get("device-1")
	if !ok {
		t.Fatalf("expected registered device")
	}
	if got.Generation != 4 {
		t.Fatalf("generation = %d, want 4", got.Generation)
	}
	if got.Capabilities["screen.width"] != "1366" {
		t.Fatalf("screen.width = %q, want 1366", got.Capabilities["screen.width"])
	}

	sawStaleError := false
	sawRebaselineAck := false
	for _, msg := range stream.sent {
		if msg.ErrorCode == ErrorCodeProtocolViolation && msg.Error != "" {
			sawStaleError = true
		}
		if msg.CapabilityAck != nil && msg.CapabilityAck.AcceptedGeneration == 4 {
			sawRebaselineAck = true
		}
	}
	if !sawStaleError {
		t.Fatalf("expected stale delta protocol violation")
	}
	if !sawRebaselineAck {
		t.Fatalf("expected capability ack for rebaseline snapshot generation 4")
	}
}

func TestSessionRunRejectsDeviceIDMismatchButContinues(t *testing.T) {
	manager := device.NewManager()
	control := NewControlService("srv-1", manager)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   manager,
		IO:        iorouter.NewRouter(),
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	session := NewSession(handler, control)

	stream := &fakeStream{
		ctx: context.Background(),
		recvQueue: []ClientMessage{
			{Register: &RegisterRequest{DeviceID: "d1", DeviceName: "Kitchen"}},
			{Capability: &CapabilityUpdateRequest{
				DeviceID: "d2",
				Capabilities: map[string]string{
					"screen.width": "1920",
				},
			}},
			{Heartbeat: &HeartbeatRequest{DeviceID: "d1"}},
		},
	}

	if err := session.Run(stream); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	foundMismatchError := false
	for _, sent := range stream.sent {
		if sent.Error != "" {
			foundMismatchError = true
		}
	}
	if !foundMismatchError {
		t.Fatalf("expected mismatch structured error")
	}

	got, ok := manager.Get("d1")
	if !ok {
		t.Fatalf("expected device d1")
	}
	if got.LastHeartbeat.IsZero() {
		t.Fatalf("expected heartbeat to be processed after mismatch error")
	}
	if handler.metrics.protocolErrors.Load() == 0 {
		t.Fatalf("expected protocol error metric increment")
	}
}

func TestSessionRunRejectsInputDeviceIDMismatchButContinues(t *testing.T) {
	manager := device.NewManager()
	control := NewControlService("srv-1", manager)
	handler := NewStreamHandler(control)
	session := NewSession(handler, control)

	stream := &fakeStream{
		ctx: context.Background(),
		recvQueue: []ClientMessage{
			{Register: &RegisterRequest{DeviceID: "d1", DeviceName: "Kitchen"}},
			{Input: &InputRequest{
				DeviceID:    "d2",
				ComponentID: "terminal_input",
				Action:      "submit",
				Value:       "echo mismatch",
			}},
			{Heartbeat: &HeartbeatRequest{DeviceID: "d1"}},
		},
	}

	if err := session.Run(stream); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	foundMismatchError := false
	for _, sent := range stream.sent {
		if sent.Error != "" {
			foundMismatchError = true
		}
	}
	if !foundMismatchError {
		t.Fatalf("expected mismatch structured error")
	}

	got, ok := manager.Get("d1")
	if !ok {
		t.Fatalf("expected device d1")
	}
	if got.LastHeartbeat.IsZero() {
		t.Fatalf("expected heartbeat to be processed after mismatch error")
	}
}

func TestSessionRunDisconnectCleansTerminalSession(t *testing.T) {
	manager := device.NewManager()
	control := NewControlService("srv-1", manager)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   manager,
		IO:        iorouter.NewRouter(),
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	session := NewSession(handler, control)

	stream := &fakeStream{
		ctx: context.Background(),
		recvQueue: []ClientMessage{
			{Register: &RegisterRequest{DeviceID: "d1", DeviceName: "Kitchen"}},
			{Command: &CommandRequest{
				RequestID: "start-terminal",
				DeviceID:  "d1",
				Kind:      "manual",
				Intent:    "terminal",
			}},
		},
	}

	if err := session.Run(stream); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(handler.terminals.List()) != 0 {
		t.Fatalf("expected terminal sessions to be cleaned up on disconnect")
	}
	if sessionID, ok := handler.replSessionIDForDevice("d1"); ok || sessionID != "" {
		t.Fatalf("expected device session mapping removed on disconnect")
	}
}

func TestSessionRunDisconnectCleansIORoutes(t *testing.T) {
	manager := device.NewManager()
	control := NewControlService("srv-1", manager)
	routes := iorouter.NewRouter()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   manager,
		IO:        routes,
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	session := NewSession(handler, control)

	_ = routes.Connect("d1", "d2", "audio")
	_ = routes.Connect("d3", "d1", "video")
	_ = routes.Connect("x", "y", "audio")

	stream := &fakeStream{
		ctx: context.Background(),
		recvQueue: []ClientMessage{
			{Register: &RegisterRequest{DeviceID: "d1", DeviceName: "Kitchen"}},
		},
	}

	if err := session.Run(stream); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	remaining := routes.Routes()
	if len(remaining) != 1 {
		t.Fatalf("remaining routes = %d, want 1", len(remaining))
	}
	if remaining[0].SourceID != "x" || remaining[0].TargetID != "y" {
		t.Fatalf("unexpected remaining route: %+v", remaining[0])
	}
}

type asyncFakeStream struct {
	ctx    context.Context
	recvCh chan ClientMessage
	sentCh chan ServerMessage
}

func (a *asyncFakeStream) Recv() (ClientMessage, error) {
	msg, ok := <-a.recvCh
	if !ok {
		return ClientMessage{}, io.EOF
	}
	return msg, nil
}

func (a *asyncFakeStream) Send(msg ServerMessage) error {
	a.sentCh <- msg
	return nil
}

func (a *asyncFakeStream) Context() context.Context {
	return a.ctx
}

func TestSessionRunRelaysWebRTCSignalsAcrossDeviceSessions(t *testing.T) {
	t.Cleanup(func() {
		globalSessionRelayRegistry = newSessionRelayRegistry()
	})

	manager := device.NewManager()
	control := NewControlService("srv-1", manager)
	routes := iorouter.NewRouter()
	_ = routes.Connect("d1", "d2", "audio")
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   manager,
		IO:        routes,
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})

	session1 := NewSession(NewStreamHandlerWithRuntime(control, runtime), control)
	session2 := NewSession(NewStreamHandlerWithRuntime(control, runtime), control)

	stream1 := &asyncFakeStream{
		ctx:    context.Background(),
		recvCh: make(chan ClientMessage, 8),
		sentCh: make(chan ServerMessage, 16),
	}
	stream2 := &asyncFakeStream{
		ctx:    context.Background(),
		recvCh: make(chan ClientMessage, 8),
		sentCh: make(chan ServerMessage, 16),
	}

	var wg sync.WaitGroup
	wg.Add(2)
	var runErr1 error
	var runErr2 error
	go func() {
		defer wg.Done()
		runErr1 = session1.Run(stream1)
	}()
	go func() {
		defer wg.Done()
		runErr2 = session2.Run(stream2)
	}()

	stream1.recvCh <- ClientMessage{Register: &RegisterRequest{DeviceID: "d1", DeviceName: "Kitchen"}}
	stream2.recvCh <- ClientMessage{Register: &RegisterRequest{DeviceID: "d2", DeviceName: "Hall"}}

	for i := 0; i < 2; i++ {
		<-stream1.sentCh
		<-stream2.sentCh
	}

	stream1.recvCh <- ClientMessage{
		WebRTCSignal: &WebRTCSignalRequest{
			StreamID:   "route:d1|d2|audio",
			SignalType: "offer",
			Payload:    "{\"sdp\":\"v=0-offer\"}",
		},
	}

	select {
	case relayed := <-stream2.sentCh:
		if relayed.WebRTCSignal == nil {
			t.Fatalf("expected relayed WebRTCSignal payload")
		}
		if relayed.WebRTCSignal.StreamID != "route:d1|d2|audio" {
			t.Fatalf("relayed stream_id = %q, want route:d1|d2|audio", relayed.WebRTCSignal.StreamID)
		}
		if relayed.WebRTCSignal.SignalType != "offer" {
			t.Fatalf("relayed signal_type = %q, want offer", relayed.WebRTCSignal.SignalType)
		}
		if relayed.WebRTCSignal.Payload != "{\"sdp\":\"v=0-offer\"}" {
			t.Fatalf("relayed payload = %q, want offer payload", relayed.WebRTCSignal.Payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for relayed WebRTC signal")
	}

	close(stream1.recvCh)
	close(stream2.recvCh)
	wg.Wait()
	if runErr1 != nil {
		t.Fatalf("session1.Run() error = %v", runErr1)
	}
	if runErr2 != nil {
		t.Fatalf("session2.Run() error = %v", runErr2)
	}
}

func TestSessionRunIntercomRoutesFanOutToPeerSession(t *testing.T) {
	t.Cleanup(func() {
		globalSessionRelayRegistry = newSessionRelayRegistry()
	})

	manager := device.NewManager()
	control := NewControlService("srv-1", manager)
	routes := iorouter.NewRouter()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   manager,
		IO:        routes,
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})

	session1 := NewSession(NewStreamHandlerWithRuntime(control, runtime), control)
	session2 := NewSession(NewStreamHandlerWithRuntime(control, runtime), control)

	stream1 := &asyncFakeStream{
		ctx:    context.Background(),
		recvCh: make(chan ClientMessage, 8),
		sentCh: make(chan ServerMessage, 16),
	}
	stream2 := &asyncFakeStream{
		ctx:    context.Background(),
		recvCh: make(chan ClientMessage, 8),
		sentCh: make(chan ServerMessage, 16),
	}

	var wg sync.WaitGroup
	wg.Add(2)
	var runErr1 error
	var runErr2 error
	go func() {
		defer wg.Done()
		runErr1 = session1.Run(stream1)
	}()
	go func() {
		defer wg.Done()
		runErr2 = session2.Run(stream2)
	}()

	waitFor := func(ch <-chan ServerMessage, pred func(ServerMessage) bool) ServerMessage {
		deadline := time.After(2 * time.Second)
		for {
			select {
			case msg := <-ch:
				if pred(msg) {
					return msg
				}
			case <-deadline:
				t.Fatalf("timed out waiting for expected message")
			}
		}
	}

	stream1.recvCh <- ClientMessage{Register: &RegisterRequest{DeviceID: "d1", DeviceName: "Kitchen"}}
	stream2.recvCh <- ClientMessage{Register: &RegisterRequest{DeviceID: "d2", DeviceName: "Hall"}}

	for i := 0; i < 2; i++ {
		<-stream1.sentCh
		<-stream2.sentCh
	}

	stream1.recvCh <- ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-intercom-start",
			DeviceID:  "d1",
			Kind:      "manual",
			Intent:    "intercom",
		},
	}

	waitFor(stream1.sentCh, func(msg ServerMessage) bool { return msg.ScenarioStart == "intercom" })
	waitFor(stream1.sentCh, func(msg ServerMessage) bool {
		return msg.StartStream != nil && msg.StartStream.StreamID == "route:d1|d2|audio"
	})
	waitFor(stream1.sentCh, func(msg ServerMessage) bool {
		return msg.RouteStream != nil && msg.RouteStream.StreamID == "route:d1|d2|audio"
	})

	waitFor(stream2.sentCh, func(msg ServerMessage) bool {
		return msg.StartStream != nil &&
			msg.StartStream.StreamID == "route:d1|d2|audio" &&
			msg.StartStream.SourceDeviceID == "d1" &&
			msg.StartStream.TargetDeviceID == "d2"
	})
	waitFor(stream2.sentCh, func(msg ServerMessage) bool {
		return msg.RouteStream != nil &&
			msg.RouteStream.StreamID == "route:d1|d2|audio" &&
			msg.RouteStream.SourceDeviceID == "d1" &&
			msg.RouteStream.TargetDeviceID == "d2"
	})

	stream1.recvCh <- ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-intercom-stop",
			DeviceID:  "d1",
			Action:    CommandActionStop,
			Kind:      "manual",
			Intent:    "intercom",
		},
	}

	waitFor(stream1.sentCh, func(msg ServerMessage) bool { return msg.ScenarioStop == "intercom" })
	waitFor(stream1.sentCh, func(msg ServerMessage) bool {
		return msg.StopStream != nil && msg.StopStream.StreamID == "route:d1|d2|audio"
	})
	waitFor(stream2.sentCh, func(msg ServerMessage) bool {
		return msg.StopStream != nil && msg.StopStream.StreamID == "route:d1|d2|audio"
	})

	close(stream1.recvCh)
	close(stream2.recvCh)
	wg.Wait()
	if runErr1 != nil {
		t.Fatalf("session1.Run() error = %v", runErr1)
	}
	if runErr2 != nil {
		t.Fatalf("session2.Run() error = %v", runErr2)
	}
}

func TestSessionRunRelaysScenarioBroadcastNotificationsToPeerSessions(t *testing.T) {
	t.Cleanup(func() {
		globalSessionRelayRegistry = newSessionRelayRegistry()
	})

	manager := device.NewManager()
	control := NewControlService("srv-1", manager)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   manager,
		IO:        iorouter.NewRouter(),
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})

	session1 := NewSession(NewStreamHandlerWithRuntime(control, runtime), control)
	session2 := NewSession(NewStreamHandlerWithRuntime(control, runtime), control)

	stream1 := &asyncFakeStream{
		ctx:    context.Background(),
		recvCh: make(chan ClientMessage, 8),
		sentCh: make(chan ServerMessage, 16),
	}
	stream2 := &asyncFakeStream{
		ctx:    context.Background(),
		recvCh: make(chan ClientMessage, 8),
		sentCh: make(chan ServerMessage, 16),
	}

	var wg sync.WaitGroup
	wg.Add(2)
	var runErr1 error
	var runErr2 error
	go func() {
		defer wg.Done()
		runErr1 = session1.Run(stream1)
	}()
	go func() {
		defer wg.Done()
		runErr2 = session2.Run(stream2)
	}()

	waitFor := func(ch <-chan ServerMessage, pred func(ServerMessage) bool) ServerMessage {
		deadline := time.After(2 * time.Second)
		for {
			select {
			case msg := <-ch:
				if pred(msg) {
					return msg
				}
			case <-deadline:
				t.Fatalf("timed out waiting for expected message")
			}
		}
	}

	stream1.recvCh <- ClientMessage{Register: &RegisterRequest{DeviceID: "d1", DeviceName: "Kitchen"}}
	stream2.recvCh <- ClientMessage{Register: &RegisterRequest{DeviceID: "d2", DeviceName: "Hall"}}

	for i := 0; i < 2; i++ {
		<-stream1.sentCh
		<-stream2.sentCh
	}

	stream1.recvCh <- ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-red-alert",
			DeviceID:  "d1",
			Kind:      CommandKindVoice,
			Text:      "red alert",
		},
	}

	waitFor(stream1.sentCh, func(msg ServerMessage) bool { return msg.ScenarioStart == "red_alert" })
	waitFor(stream2.sentCh, func(msg ServerMessage) bool { return msg.Notification == "RED ALERT" })

	close(stream1.recvCh)
	close(stream2.recvCh)
	wg.Wait()
	if runErr1 != nil {
		t.Fatalf("session1.Run() error = %v", runErr1)
	}
	if runErr2 != nil {
		t.Fatalf("session2.Run() error = %v", runErr2)
	}
}
