package transport

import (
	"context"
	"io"
	"testing"

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

	got, ok := manager.Get("device-1")
	if !ok {
		t.Fatalf("expected registered device")
	}
	if got.State != device.StateDisconnected {
		t.Fatalf("state = %q, want %q", got.State, device.StateDisconnected)
	}
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
	if _, exists := handler.terminalByDevice["d1"]; exists {
		t.Fatalf("expected terminalByDevice entry removed on disconnect")
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
