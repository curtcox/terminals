package transport

import (
	"context"
	"sync"
	"testing"
	"time"

	diagnosticsv1 "github.com/curtcox/terminals/terminal_server/gen/go/diagnostics/v1"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/recording"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/telephony"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

type bugReportIntakeStub struct {
	ack *diagnosticsv1.BugReportAck
	err error
}

func (s bugReportIntakeStub) File(_ context.Context, _ *diagnosticsv1.BugReport) (*diagnosticsv1.BugReportAck, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.ack, nil
}

type audioChunkRecordingStub struct {
	mu      sync.Mutex
	writes  int
	devices []string
	audio   []byte
}

func (s *audioChunkRecordingStub) Start(context.Context, recording.Stream) error { return nil }

func (s *audioChunkRecordingStub) Stop(context.Context, string) error { return nil }

func (s *audioChunkRecordingStub) WriteDeviceAudio(deviceID string, chunk []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writes++
	s.devices = append(s.devices, deviceID)
	s.audio = append(s.audio, chunk...)
	return nil
}

func TestHandleMessageRegisterSendsAckAndUI(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	handler := NewStreamHandler(service)

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{
			DeviceID:   "device-1",
			DeviceName: "Kitchen Chromebook",
			DeviceType: "laptop",
			Platform:   "chromeos",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(register) error = %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2", len(out))
	}
	if out[0].RegisterAck == nil {
		t.Fatalf("first response should contain register ack")
	}
	if out[1].SetUI == nil {
		t.Fatalf("second response should contain SetUI")
	}
}

func TestHandleMessageCapabilityAndHeartbeat(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	now := time.Date(2026, 4, 11, 20, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	handler := NewStreamHandler(service)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{
			DeviceID:   "device-1",
			DeviceName: "Kitchen Chromebook",
		},
	})

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Capability: &CapabilityUpdateRequest{
			DeviceID: "device-1",
			Capabilities: map[string]string{
				"screen.width":  "1920",
				"screen.height": "1080",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability) error = %v", err)
	}

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Heartbeat: &HeartbeatRequest{DeviceID: "device-1"},
	}); err != nil {
		t.Fatalf("HandleMessage(heartbeat) error = %v", err)
	}

	got, ok := manager.Get("device-1")
	if !ok {
		t.Fatalf("expected registered device")
	}
	if got.Capabilities["screen.width"] != "1920" {
		t.Fatalf("screen.width = %q, want 1920", got.Capabilities["screen.width"])
	}
	if got.LastHeartbeat != now {
		t.Fatalf("LastHeartbeat = %v, want %v", got.LastHeartbeat, now)
	}
}

func TestHandleMessageSensorAndStreamReady(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	handler := NewStreamHandler(service)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{
			DeviceID:   "device-1",
			DeviceName: "Kitchen Chromebook",
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Sensor: &SensorDataRequest{
			DeviceID: "device-1",
			UnixMS:   1713000000000,
			Values: map[string]float64{
				"accelerometer.x": 0.12,
			},
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(sensor) error = %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("len(sensor out) = %d, want 0", len(out))
	}

	out, err = handler.HandleMessage(context.Background(), ClientMessage{
		StreamReady: &StreamReadyRequest{StreamID: "stream-1"},
	})
	if err != nil {
		t.Fatalf("HandleMessage(stream_ready) error = %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("len(stream_ready out) = %d, want 0", len(out))
	}
}

func TestHandleMessageVoiceAudioWritesChunksToRecordingManager(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	handler := NewStreamHandler(service)
	recorder := &audioChunkRecordingStub{}
	handler.SetRecordingManager(recorder)

	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		VoiceAudio: &VoiceAudioRequest{
			DeviceID:   "device-1",
			Audio:      []byte{0x10, 0x20, 0x30},
			SampleRate: 16000,
			IsFinal:    false,
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(voice_audio non-final) error = %v", err)
	}

	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.writes != 1 {
		t.Fatalf("writes = %d, want 1", recorder.writes)
	}
	if len(recorder.devices) != 1 || recorder.devices[0] != "device-1" {
		t.Fatalf("devices = %+v, want [device-1]", recorder.devices)
	}
	if got := recorder.audio; len(got) != 3 || got[0] != 0x10 || got[1] != 0x20 || got[2] != 0x30 {
		t.Fatalf("audio bytes = %v, want [16 32 48]", got)
	}
}

func TestHandleMessageSensorTriggersActiveScenarioHook(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	broadcaster := ui.NewMemoryBroadcaster()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   manager,
		IO:        io.NewRouter(),
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: broadcaster,
	})
	handler := NewStreamHandlerWithRuntime(service, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{
			DeviceID:   "device-1",
			DeviceName: "Kitchen Chromebook",
		},
	})

	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-schedule-monitor",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "schedule monitor",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(command schedule monitor) error = %v", err)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Sensor: &SensorDataRequest{
			DeviceID: "device-1",
			UnixMS:   1713000000000,
			Values: map[string]float64{
				"motion.magnitude": 1.8,
			},
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(sensor) error = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(sensor out) = %d, want 1", len(out))
	}
	if out[0].Notification != "Schedule monitor activity detected: magnitude=1.80" {
		t.Fatalf("notification = %q, want schedule monitor activity notification", out[0].Notification)
	}
	if out[0].RelayToDeviceID != "" {
		t.Fatalf("RelayToDeviceID = %q, want empty for local notification", out[0].RelayToDeviceID)
	}
}

func TestHandleMessageBugReportRequiresIntake(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	handler := NewStreamHandler(service)

	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		BugReport: &diagnosticsv1.BugReport{ReportId: "bug-1"},
	})
	if err == nil {
		t.Fatalf("expected error when bug report intake is missing")
	}
	if err != ErrBugReportIntakeUnavailable {
		t.Fatalf("err = %v, want %v", err, ErrBugReportIntakeUnavailable)
	}
}

func TestHandleMessageBugReportReturnsAck(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	handler := NewStreamHandler(service)
	handler.SetBugReportIntake(bugReportIntakeStub{
		ack: &diagnosticsv1.BugReportAck{
			ReportId:      "bug-2",
			CorrelationId: "bug:bug-2",
			Status:        diagnosticsv1.BugReportStatus_BUG_REPORT_STATUS_FILED,
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		BugReport: &diagnosticsv1.BugReport{ReportId: "bug-2"},
	})
	if err != nil {
		t.Fatalf("HandleMessage(bug_report) error = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if out[0].BugReportAck == nil || out[0].BugReportAck.GetReportId() != "bug-2" {
		t.Fatalf("bug_report_ack = %+v, want report_id bug-2", out[0].BugReportAck)
	}
}

func TestHandleMessageInputBugReportActionFilesReport(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	handler := NewStreamHandler(service)
	handler.SetBugReportIntake(bugReportIntakeStub{
		ack: &diagnosticsv1.BugReportAck{
			ReportId:      "bug-from-ui-action",
			CorrelationId: "bug:bug-from-ui-action",
			Status:        diagnosticsv1.BugReportStatus_BUG_REPORT_STATUS_FILED,
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: bugReportButtonID,
			Action:      "bug_report:subject-1",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(input bug_report) error = %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2", len(out))
	}
	if out[0].BugReportAck == nil || out[0].BugReportAck.GetReportId() != "bug-from-ui-action" {
		t.Fatalf("first response bug_report_ack = %+v", out[0].BugReportAck)
	}
	if out[1].Notification == "" {
		t.Fatalf("second response should include filing notification")
	}
}

func TestHandleDisconnectStopsRecordingForDisconnectedDeviceRoutes(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	broadcaster := ui.NewMemoryBroadcaster()
	router := io.NewRouter()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   manager,
		IO:        router,
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: broadcaster,
	})
	handler := NewStreamHandlerWithRuntime(service, runtime)
	recorder := recording.NewMemoryManager()
	handler.SetRecordingManager(recorder)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-2", DeviceName: "Hall"},
	})

	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-intercom-start-disconnect-recording",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "intercom",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(intercom start) error = %v", err)
	}
	if len(recorder.Active()) != 2 {
		t.Fatalf("len(recorder.Active()) = %d, want 2 before disconnect", len(recorder.Active()))
	}

	handler.HandleDisconnect("device-1")
	if len(recorder.Active()) != 0 {
		t.Fatalf("len(recorder.Active()) = %d, want 0 after disconnect", len(recorder.Active()))
	}
}

func TestHandleMessageInvalid(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	handler := NewStreamHandler(service)

	out, err := handler.HandleMessage(context.Background(), ClientMessage{})
	if err != ErrInvalidClientMessage {
		t.Fatalf("err = %v, want %v", err, ErrInvalidClientMessage)
	}
	if len(out) != 1 || out[0].Error == "" {
		t.Fatalf("expected one error response")
	}
}
