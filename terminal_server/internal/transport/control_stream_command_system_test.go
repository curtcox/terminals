package transport

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/recording"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestHandleMessageSystemListDevices(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices: devices,
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook", Platform: "linux"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-2", DeviceName: "Hall Tablet", Platform: "android"},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-1",
			Kind:      "system",
			Intent:    "list_devices",
		},
	})
	if err != nil {
		t.Fatalf("system list devices error = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if out[0].CommandAck != "sys-1" {
		t.Fatalf("CommandAck = %q, want sys-1", out[0].CommandAck)
	}
	if len(out[0].Data) != 2 {
		t.Fatalf("len(Data) = %d, want 2", len(out[0].Data))
	}
}

func TestHandleMessageSystemActiveScenarios(t *testing.T) {
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
			RequestID: "cmd-start",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-2",
			Kind:      "system",
			Intent:    "active_scenarios",
		},
	})
	if err != nil {
		t.Fatalf("system active_scenarios error = %v", err)
	}
	if out[0].Data["device-1"] != "photo_frame" {
		t.Fatalf("active scenario = %q, want photo_frame", out[0].Data["device-1"])
	}
}

func TestHandleMessageSystemServerStatus(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	control.started = control.now().Add(-2 * time.Hour)
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
		Command: &CommandRequest{
			RequestID: "sys-status-1",
			Kind:      "system",
			Intent:    "server_status",
		},
	})
	if err != nil {
		t.Fatalf("system server_status error = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if out[0].Data["server_id"] != "srv-1" {
		t.Fatalf("server_id = %q, want srv-1", out[0].Data["server_id"])
	}
	if out[0].Data["devices_total"] != "1" {
		t.Fatalf("devices_total = %q, want 1", out[0].Data["devices_total"])
	}
	if out[0].CommandAck != "sys-status-1" {
		t.Fatalf("CommandAck = %q, want sys-status-1", out[0].CommandAck)
	}
}

func TestHandleMessageSystemRuntimeStatus(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	routes := io.NewRouter()
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices: devices,
		IO:      routes,
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-start-rs",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})
	_ = routes.Connect("device-1", "device-2", "audio")

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-runtime-1",
			Kind:      "system",
			Intent:    "runtime_status",
		},
	})
	if err != nil {
		t.Fatalf("system runtime_status error = %v", err)
	}
	if out[0].Data["active_scenarios"] != "1" {
		t.Fatalf("active_scenarios = %q, want 1", out[0].Data["active_scenarios"])
	}
	if out[0].Data["active_routes"] != "1" {
		t.Fatalf("active_routes = %q, want 1", out[0].Data["active_routes"])
	}
	if out[0].Data["registered_scenarios"] == "" {
		t.Fatalf("expected registered_scenarios in runtime_status")
	}
	if out[0].Data["pending_timers"] == "" {
		t.Fatalf("expected pending_timers in runtime_status")
	}
	if out[0].Data["media_streams_active"] != "0" {
		t.Fatalf("media_streams_active = %q, want 0", out[0].Data["media_streams_active"])
	}
	if out[0].Data["media_streams_ready"] != "0" {
		t.Fatalf("media_streams_ready = %q, want 0", out[0].Data["media_streams_ready"])
	}
	if out[0].Data["media_streams_pending"] != "0" {
		t.Fatalf("media_streams_pending = %q, want 0", out[0].Data["media_streams_pending"])
	}
	if out[0].Data["sensor_devices_reporting"] != "0" {
		t.Fatalf("sensor_devices_reporting = %q, want 0", out[0].Data["sensor_devices_reporting"])
	}
	if out[0].Data["sensor_latest_unix_ms"] != "0" {
		t.Fatalf("sensor_latest_unix_ms = %q, want 0", out[0].Data["sensor_latest_unix_ms"])
	}
	if out[0].Data["sensor_device_ids"] != "" {
		t.Fatalf("sensor_device_ids = %q, want empty", out[0].Data["sensor_device_ids"])
	}
	if out[0].Data["sensor_summaries"] != "" {
		t.Fatalf("sensor_summaries = %q, want empty", out[0].Data["sensor_summaries"])
	}
	if out[0].Data["recording_active_streams"] != "0" {
		t.Fatalf("recording_active_streams = %q, want 0", out[0].Data["recording_active_streams"])
	}
	if out[0].Data["recording_stream_ids"] != "" {
		t.Fatalf("recording_stream_ids = %q, want empty", out[0].Data["recording_stream_ids"])
	}
}

func TestHandleMessageSystemRuntimeStatusTracksMediaStreamLifecycle(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	routes := io.NewRouter()
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices: devices,
		IO:      routes,
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	handler.SetRecordingManager(recording.NewMemoryManager())

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-2", DeviceName: "Hall Display"},
	})

	startOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "intercom-start-runtime-status",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "intercom",
		},
	})
	if err != nil {
		t.Fatalf("intercom start error = %v", err)
	}
	streamIDs := map[string]struct{}{}
	for _, msg := range startOut {
		if msg.StartStream != nil {
			streamIDs[msg.StartStream.StreamID] = struct{}{}
		}
	}
	if len(streamIDs) == 0 {
		t.Fatalf("expected start_stream message in start output")
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-runtime-pre-ready",
			Kind:      "system",
			Intent:    "runtime_status",
		},
	})
	if err != nil {
		t.Fatalf("runtime_status pre-ready error = %v", err)
	}
	if out[0].Data["media_streams_active"] != "2" {
		t.Fatalf("media_streams_active pre-ready = %q, want 2", out[0].Data["media_streams_active"])
	}
	if out[0].Data["media_streams_ready"] != "0" {
		t.Fatalf("media_streams_ready pre-ready = %q, want 0", out[0].Data["media_streams_ready"])
	}
	if out[0].Data["media_streams_pending"] != "2" {
		t.Fatalf("media_streams_pending pre-ready = %q, want 2", out[0].Data["media_streams_pending"])
	}
	if out[0].Data["recording_active_streams"] != "2" {
		t.Fatalf("recording_active_streams pre-ready = %q, want 2", out[0].Data["recording_active_streams"])
	}

	for streamID := range streamIDs {
		_, err = handler.HandleMessage(context.Background(), ClientMessage{
			StreamReady: &StreamReadyRequest{StreamID: streamID},
		})
		if err != nil {
			t.Fatalf("stream_ready error = %v", err)
		}
	}

	out, err = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-runtime-ready",
			Kind:      "system",
			Intent:    "runtime_status",
		},
	})
	if err != nil {
		t.Fatalf("runtime_status ready error = %v", err)
	}
	if out[0].Data["media_streams_active"] != "2" {
		t.Fatalf("media_streams_active ready = %q, want 2", out[0].Data["media_streams_active"])
	}
	if out[0].Data["media_streams_ready"] != "2" {
		t.Fatalf("media_streams_ready = %q, want 2", out[0].Data["media_streams_ready"])
	}
	if out[0].Data["media_streams_pending"] != "0" {
		t.Fatalf("media_streams_pending ready = %q, want 0", out[0].Data["media_streams_pending"])
	}
	if out[0].Data["recording_active_streams"] != "2" {
		t.Fatalf("recording_active_streams ready = %q, want 2", out[0].Data["recording_active_streams"])
	}
	if !strings.Contains(out[0].Data["media_streams"], "ready=true") {
		t.Fatalf("media_streams details should contain ready=true, got %q", out[0].Data["media_streams"])
	}

	_, err = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "intercom-stop-runtime-status",
			DeviceID:  "device-1",
			Action:    CommandActionStop,
			Kind:      "manual",
			Intent:    "intercom",
		},
	})
	if err != nil {
		t.Fatalf("intercom stop error = %v", err)
	}

	out, err = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-runtime-post-stop",
			Kind:      "system",
			Intent:    "runtime_status",
		},
	})
	if err != nil {
		t.Fatalf("runtime_status post-stop error = %v", err)
	}
	if out[0].Data["media_streams_active"] != "0" {
		t.Fatalf("media_streams_active post-stop = %q, want 0", out[0].Data["media_streams_active"])
	}
	if out[0].Data["media_streams_ready"] != "0" {
		t.Fatalf("media_streams_ready post-stop = %q, want 0", out[0].Data["media_streams_ready"])
	}
	if out[0].Data["media_streams_pending"] != "0" {
		t.Fatalf("media_streams_pending post-stop = %q, want 0", out[0].Data["media_streams_pending"])
	}
	if out[0].Data["media_streams"] != "" {
		t.Fatalf("media_streams post-stop = %q, want empty", out[0].Data["media_streams"])
	}
	if out[0].Data["recording_active_streams"] != "0" {
		t.Fatalf("recording_active_streams post-stop = %q, want 0", out[0].Data["recording_active_streams"])
	}
	if out[0].Data["recording_stream_ids"] != "" {
		t.Fatalf("recording_stream_ids post-stop = %q, want empty", out[0].Data["recording_stream_ids"])
	}
}

func TestHandleMessageSystemRecordingEvents(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices: devices,
		IO:      io.NewRouter(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	recorder, err := recording.NewDiskManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewDiskManager() error = %v", err)
	}
	handler.SetRecordingManager(recorder)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-2", DeviceName: "Hall Display"},
	})

	_, err = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "recording-events-start",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "intercom",
		},
	})
	if err != nil {
		t.Fatalf("intercom start error = %v", err)
	}
	_, err = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "recording-events-stop",
			DeviceID:  "device-1",
			Action:    CommandActionStop,
			Kind:      "manual",
			Intent:    "intercom",
		},
	})
	if err != nil {
		t.Fatalf("intercom stop error = %v", err)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-recording-events",
			Kind:      "system",
			Intent:    SystemIntentRecordingEvents,
		},
	})
	if err != nil {
		t.Fatalf("recording_events query error = %v", err)
	}
	if out[0].Notification != "System query: recording_events" {
		t.Fatalf("notification = %q, want recording_events", out[0].Notification)
	}
	if len(out[0].Data) == 0 {
		t.Fatalf("expected recording event rows")
	}
	foundStart := false
	foundStop := false
	for _, row := range out[0].Data {
		if strings.Contains(row, "|start|") {
			foundStart = true
		}
		if strings.Contains(row, "|stop|") {
			foundStop = true
		}
	}
	if !foundStart {
		t.Fatalf("recording event rows missing start action: %+v", out[0].Data)
	}
	if !foundStop {
		t.Fatalf("recording event rows missing stop action: %+v", out[0].Data)
	}
}

func TestHandleMessageSystemListPlaybackArtifacts(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)

	recorder, err := recording.NewDiskManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewDiskManager() error = %v", err)
	}
	handler.SetRecordingManager(recorder)
	if err := recorder.Start(context.Background(), recording.Stream{
		StreamID:       "route:device-1|device-2|audio",
		Kind:           "audio",
		SourceDeviceID: "device-1",
		TargetDeviceID: "device-2",
	}); err != nil {
		t.Fatalf("recorder.Start() error = %v", err)
	}
	if err := recorder.WriteDeviceAudio("device-1", []byte{0x01, 0x02}); err != nil {
		t.Fatalf("recorder.WriteDeviceAudio() error = %v", err)
	}
	if err := recorder.Stop(context.Background(), "route:device-1|device-2|audio"); err != nil {
		t.Fatalf("recorder.Stop() error = %v", err)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-playback-artifacts",
			Kind:      "system",
			Intent:    SystemIntentListPlaybackFiles,
		},
	})
	if err != nil {
		t.Fatalf("list_playback_artifacts query error = %v", err)
	}
	if out[0].Notification != "System query: list_playback_artifacts" {
		t.Fatalf("notification = %q, want list_playback_artifacts", out[0].Notification)
	}
	if len(out[0].Data) != 1 {
		t.Fatalf("len(Data) = %d, want 1", len(out[0].Data))
	}
	row := out[0].Data["000"]
	if !strings.Contains(row, "route:device-1|device-2|audio") {
		t.Fatalf("row = %q, want stream id", row)
	}
	if !strings.Contains(row, "|audio|device-1|device-2|") {
		t.Fatalf("row = %q, want kind/source/target columns", row)
	}
}

func TestHandleMessageManualPlaybackMetadata(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)

	recorder, err := recording.NewDiskManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewDiskManager() error = %v", err)
	}
	handler.SetRecordingManager(recorder)
	if err := recorder.Start(context.Background(), recording.Stream{
		StreamID:       "route:device-a|device-b|audio",
		Kind:           "audio",
		SourceDeviceID: "device-a",
		TargetDeviceID: "device-b",
	}); err != nil {
		t.Fatalf("recorder.Start() error = %v", err)
	}
	if err := recorder.WriteDeviceAudio("device-a", []byte{0xAA, 0xBB, 0xCC}); err != nil {
		t.Fatalf("recorder.WriteDeviceAudio() error = %v", err)
	}
	if err := recorder.Stop(context.Background(), "route:device-a|device-b|audio"); err != nil {
		t.Fatalf("recorder.Stop() error = %v", err)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "manual-playback-metadata",
			DeviceID:  "device-a",
			Kind:      "manual",
			Intent:    ManualIntentPlaybackMetadata,
			Arguments: map[string]string{
				"artifact_id":      "route:device-a|device-b|audio",
				"target_device_id": "hall-display",
			},
		},
	})
	if err != nil {
		t.Fatalf("manual playback_metadata error = %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2 (command result + play audio)", len(out))
	}
	if out[0].Notification != "Playback metadata ready" {
		t.Fatalf("notification = %q, want playback metadata ready", out[0].Notification)
	}
	if out[0].Data["artifact_id"] != "route:device-a|device-b|audio" {
		t.Fatalf("artifact_id = %q, want route:device-a|device-b|audio", out[0].Data["artifact_id"])
	}
	if out[0].Data["target_device_id"] != "hall-display" {
		t.Fatalf("target_device_id = %q, want hall-display", out[0].Data["target_device_id"])
	}
	if out[0].Data["size_bytes"] != "3" {
		t.Fatalf("size_bytes = %q, want 3", out[0].Data["size_bytes"])
	}
	if out[0].CommandAck != "manual-playback-metadata" {
		t.Fatalf("CommandAck = %q, want manual-playback-metadata", out[0].CommandAck)
	}
	if out[1].PlayAudio == nil {
		t.Fatalf("expected PlayAudio response")
	}
	if out[1].PlayAudio.DeviceID != "hall-display" {
		t.Fatalf("PlayAudio.DeviceID = %q, want hall-display", out[1].PlayAudio.DeviceID)
	}
	if out[1].PlayAudio.Format != "pcm16" {
		t.Fatalf("PlayAudio.Format = %q, want pcm16", out[1].PlayAudio.Format)
	}
	if out[1].RelayToDeviceID != "hall-display" {
		t.Fatalf("RelayToDeviceID = %q, want hall-display", out[1].RelayToDeviceID)
	}
	if string(out[1].PlayAudio.Audio) != string([]byte{0xAA, 0xBB, 0xCC}) {
		t.Fatalf("PlayAudio.Audio = %v, want %v", out[1].PlayAudio.Audio, []byte{0xAA, 0xBB, 0xCC})
	}
}

func TestHandleMessageManualPlaybackMetadataDefaultsTargetToCaller(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)

	recorder, err := recording.NewDiskManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewDiskManager() error = %v", err)
	}
	handler.SetRecordingManager(recorder)
	if err := recorder.Start(context.Background(), recording.Stream{
		StreamID:       "route:device-z|device-y|audio",
		Kind:           "audio",
		SourceDeviceID: "device-z",
		TargetDeviceID: "device-y",
	}); err != nil {
		t.Fatalf("recorder.Start() error = %v", err)
	}
	if err := recorder.WriteDeviceAudio("device-z", []byte{0x01, 0x02, 0x03, 0x04}); err != nil {
		t.Fatalf("recorder.WriteDeviceAudio() error = %v", err)
	}
	if err := recorder.Stop(context.Background(), "route:device-z|device-y|audio"); err != nil {
		t.Fatalf("recorder.Stop() error = %v", err)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "manual-playback-default-target",
			DeviceID:  "device-z",
			Kind:      "manual",
			Intent:    ManualIntentPlaybackMetadata,
			Arguments: map[string]string{
				"artifact_id": "route:device-z|device-y|audio",
			},
		},
	})
	if err != nil {
		t.Fatalf("manual playback_metadata default target error = %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2", len(out))
	}
	if out[1].PlayAudio == nil {
		t.Fatalf("expected PlayAudio response")
	}
	if out[1].PlayAudio.DeviceID != "device-z" {
		t.Fatalf("PlayAudio.DeviceID = %q, want device-z", out[1].PlayAudio.DeviceID)
	}
	if out[1].RelayToDeviceID != "" {
		t.Fatalf("RelayToDeviceID = %q, want empty for local playback", out[1].RelayToDeviceID)
	}
}

func TestHandleMessageSystemScenarioRegistry(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices: devices,
		IO:      io.NewRouter(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-registry-1",
			Kind:      "system",
			Intent:    "scenario_registry",
		},
	})
	if err != nil {
		t.Fatalf("system scenario_registry error = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if out[0].Data["red_alert"] == "" {
		t.Fatalf("expected red_alert in registry data")
	}
}

func TestHandleMessageSystemRunDueTimers(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	scheduler := storage.NewMemoryScheduler()
	broadcaster := ui.NewMemoryBroadcaster()
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Scheduler: scheduler,
		Broadcast: broadcaster,
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_ = scheduler.Schedule(context.Background(), "timer:device-1:1", control.now().Add(-1*time.Minute).UnixMilli())

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-run-due-1",
			Kind:      "system",
			Intent:    "run_due_timers",
		},
	})
	if err != nil {
		t.Fatalf("system run_due_timers error = %v", err)
	}
	if out[0].Data["processed"] != "1" {
		t.Fatalf("processed = %q, want 1", out[0].Data["processed"])
	}
	events := broadcaster.Events()
	if len(events) != 1 || events[0].Message != "Timer done!" {
		t.Fatalf("unexpected broadcast events: %+v", events)
	}
}

func TestHandleMessageSystemTransportMetrics(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices: devices,
		IO:      io.NewRouter(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Heartbeat: &HeartbeatRequest{DeviceID: "device-1"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "metrics-1",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-metrics-1",
			Kind:      "system",
			Intent:    "transport_metrics",
		},
	})
	if err != nil {
		t.Fatalf("system transport_metrics error = %v", err)
	}
	if out[0].Data["register_received"] != "1" {
		t.Fatalf("register_received = %q, want 1", out[0].Data["register_received"])
	}
	if out[0].Data["heartbeat_received"] != "1" {
		t.Fatalf("heartbeat_received = %q, want 1", out[0].Data["heartbeat_received"])
	}
	if out[0].Data["command_received"] != "2" {
		t.Fatalf("command_received = %q, want 2", out[0].Data["command_received"])
	}
	if out[0].Data["dedupe_hits"] != "0" {
		t.Fatalf("dedupe_hits = %q, want 0", out[0].Data["dedupe_hits"])
	}
}

func TestHandleMessageSystemWithoutRuntime(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-no-runtime",
			Kind:      "system",
			Intent:    "server_status",
		},
	})
	if err != nil {
		t.Fatalf("expected server_status to work without runtime, err=%v", err)
	}
	if len(out) != 1 || out[0].Data["server_id"] != "srv-1" {
		t.Fatalf("unexpected server_status response: %+v", out)
	}
}

func TestHandleMessageDedupeEviction(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{Devices: devices})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	handler.seenLimit = 1

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "r1",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "r2",
			DeviceID:  "device-1",
			Action:    "stop",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})

	handler.mu.Lock()
	_, hasR1 := handler.seen["r1"]
	_, hasR2 := handler.seen["r2"]
	handler.mu.Unlock()
	if hasR1 || !hasR2 {
		t.Fatalf("expected r1 evicted and r2 retained, got hasR1=%v hasR2=%v", hasR1, hasR2)
	}
}

func TestHandleMessageTransportMetricsIncludesDedupeHits(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices: devices,
		IO:      io.NewRouter(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "dup-metrics-1",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "dup-metrics-1",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-metrics-dedupe",
			Kind:      "system",
			Intent:    "transport_metrics",
		},
	})
	if err != nil {
		t.Fatalf("transport_metrics query error = %v", err)
	}
	if out[0].Data["dedupe_hits"] != "1" {
		t.Fatalf("dedupe_hits = %q, want 1", out[0].Data["dedupe_hits"])
	}
}

func TestHandleMessageSystemHelp(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-help-1",
			Kind:      "system",
			Intent:    "system_help",
		},
	})
	if err != nil {
		t.Fatalf("system_help error = %v", err)
	}
	if out[0].Data["system_intents"] == "" || out[0].Data["command_kinds"] == "" {
		t.Fatalf("missing expected system_help fields: %+v", out[0].Data)
	}
	if out[0].Data["system_intents"] == "" || !contains(out[0].Data["system_intents"], "pending_timers") {
		t.Fatalf("system_help missing pending_timers intent: %+v", out[0].Data)
	}
	if !contains(out[0].Data["system_intents"], "recent_commands") {
		t.Fatalf("system_help missing recent_commands intent: %+v", out[0].Data)
	}
	if !contains(out[0].Data["system_intents"], SystemIntentListPlaybackFiles) {
		t.Fatalf("system_help missing list_playback_artifacts intent: %+v", out[0].Data)
	}
}

func TestHandleMessageSystemDeviceStatus(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{
			DeviceID:   "device-1",
			DeviceName: "Kitchen Chromebook",
			DeviceType: "laptop",
			Platform:   "linux",
			Capabilities: map[string]string{
				"screen.width": "1920",
			},
		},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Sensor: &SensorDataRequest{
			DeviceID: "device-1",
			UnixMS:   1713000000000,
			Values: map[string]float64{
				"accelerometer.x": 0.25,
				"accelerometer.y": -0.75,
			},
		},
	})
	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-device-1",
			Kind:      "system",
			Intent:    "device_status device-1",
		},
	})
	if err != nil {
		t.Fatalf("device_status error = %v", err)
	}
	if out[0].Data["device_id"] != "device-1" {
		t.Fatalf("device_id = %q, want device-1", out[0].Data["device_id"])
	}
	if out[0].Data["cap.screen.width"] != "1920" {
		t.Fatalf("cap.screen.width = %q, want 1920", out[0].Data["cap.screen.width"])
	}
	if out[0].Data["sensor.unix_ms"] != "1713000000000" {
		t.Fatalf("sensor.unix_ms = %q, want 1713000000000", out[0].Data["sensor.unix_ms"])
	}
	if out[0].Data["sensor.accelerometer.x"] != "0.25" {
		t.Fatalf("sensor.accelerometer.x = %q, want 0.25", out[0].Data["sensor.accelerometer.x"])
	}
	if out[0].Data["sensor.accelerometer.y"] != "-0.75" {
		t.Fatalf("sensor.accelerometer.y = %q, want -0.75", out[0].Data["sensor.accelerometer.y"])
	}
}

func TestHandleMessageSystemRuntimeStatusIncludesSensorSummary(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-2", DeviceName: "Hall Display"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Sensor: &SensorDataRequest{
			DeviceID: "device-1",
			UnixMS:   1713000000123,
			Values: map[string]float64{
				"temperature.c": 22.4,
			},
		},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Sensor: &SensorDataRequest{
			DeviceID: "device-2",
			UnixMS:   1713000000456,
			Values: map[string]float64{
				"temperature.c": 23.1,
				"humidity.pct":  45.5,
			},
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-runtime-sensor-summary",
			Kind:      "system",
			Intent:    "runtime_status",
		},
	})
	if err != nil {
		t.Fatalf("runtime_status error = %v", err)
	}
	if out[0].Data["sensor_devices_reporting"] != "2" {
		t.Fatalf("sensor_devices_reporting = %q, want 2", out[0].Data["sensor_devices_reporting"])
	}
	if out[0].Data["sensor_latest_unix_ms"] != "1713000000456" {
		t.Fatalf("sensor_latest_unix_ms = %q, want 1713000000456", out[0].Data["sensor_latest_unix_ms"])
	}
	if out[0].Data["sensor_device_ids"] != "device-1,device-2" {
		t.Fatalf("sensor_device_ids = %q, want device-1,device-2", out[0].Data["sensor_device_ids"])
	}
	if !strings.Contains(out[0].Data["sensor_summaries"], "device-1|unix_ms=1713000000123|keys=temperature.c") {
		t.Fatalf("sensor_summaries missing device-1 detail: %q", out[0].Data["sensor_summaries"])
	}
	if !strings.Contains(out[0].Data["sensor_summaries"], "device-2|unix_ms=1713000000456|keys=humidity.pct,temperature.c") {
		t.Fatalf("sensor_summaries missing device-2 detail: %q", out[0].Data["sensor_summaries"])
	}
}

func TestHandleMessageSystemPendingTimers(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	scheduler := storage.NewMemoryScheduler()
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Scheduler: scheduler,
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_ = scheduler.Schedule(context.Background(), "timer:device-1:100", control.now().Add(5*time.Minute).UnixMilli())
	_ = scheduler.ScheduleRecord(context.Background(), storage.ScheduleRecord{
		Key:      "structured-timer-1",
		Kind:     "timer",
		Subject:  "pasta",
		DeviceID: "device-1",
		UnixMS:   control.now().Add(6 * time.Minute).UnixMilli(),
		Payload:  map[string]string{"duration_seconds": "360"},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-pending-1",
			Kind:      "system",
			Intent:    "pending_timers",
		},
	})
	if err != nil {
		t.Fatalf("pending_timers error = %v", err)
	}
	if out[0].Data["timer:device-1:100"] != "kind=timer" {
		t.Fatalf("pending timer missing from response: %+v", out[0].Data)
	}
	if out[0].Data["structured-timer-1"] != "kind=timer|device=device-1|subject=pasta|duration_seconds=360" {
		t.Fatalf("structured pending timer missing metadata: %+v", out[0].Data)
	}
}

func TestHandleMessageSystemRecentCommands(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "audit-1",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "audit-2",
			DeviceID:  "device-1",
			Action:    "stop",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-recent-1",
			Kind:      "system",
			Intent:    "recent_commands",
		},
	})
	if err != nil {
		t.Fatalf("recent_commands error = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if len(out[0].Data) < 2 {
		t.Fatalf("expected at least 2 recent command events, got %d", len(out[0].Data))
	}
	foundAudit1 := false
	for _, v := range out[0].Data {
		if strings.Contains(v, "audit-1") {
			foundAudit1 = true
			break
		}
	}
	if !foundAudit1 {
		t.Fatalf("expected recent_commands payload to include audit-1 event")
	}
}

func TestHandleMessageManualTerminalRefresh(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "manual-refresh-start-terminal",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "terminal",
		},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "terminal_input",
			Action:      "submit",
			Value:       "sleep 1; printf '\\x72\\x65\\x66\\x72\\x65\\x73\\x68\\x2d\\x6d\\x61\\x6e\\x75\\x61\\x6c\\n'",
		},
	})
	time.Sleep(1200 * time.Millisecond)

	out := waitForRefreshMarker(t, handler, ClientMessage{
		Command: &CommandRequest{
			RequestID: "manual-refresh-1",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    SystemIntentTerminalRefresh,
		},
	}, "refresh-manual", 10, 200*time.Millisecond)
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2", len(out))
	}
	if out[0].CommandAck != "manual-refresh-1" {
		t.Fatalf("CommandAck = %q, want manual-refresh-1", out[0].CommandAck)
	}
	if out[1].UpdateUI == nil {
		t.Fatalf("expected UpdateUI response after manual terminal_refresh")
	}
	if !strings.Contains(out[1].UpdateUI.Node.Props["value"], "refresh-manual") {
		t.Fatalf("manual terminal_refresh missing delayed output: %+v", out[1].UpdateUI.Node.Props)
	}
}

func TestHandleMessageSystemTerminalRefresh(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "system-refresh-start-terminal",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "terminal",
		},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "terminal_input",
			Action:      "submit",
			Value:       "sleep 1; printf '\\x72\\x65\\x66\\x72\\x65\\x73\\x68\\x2d\\x73\\x79\\x73\\x74\\x65\\x6d\\n'",
		},
	})
	time.Sleep(1200 * time.Millisecond)

	out := waitForRefreshMarker(t, handler, ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-refresh-1",
			Kind:      "system",
			Intent:    SystemIntentTerminalRefresh + " device-1",
		},
	}, "refresh-system", 10, 200*time.Millisecond)
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2", len(out))
	}
	if out[0].Data["device_id"] != "device-1" {
		t.Fatalf("device_id = %q, want device-1", out[0].Data["device_id"])
	}
	if out[1].UpdateUI == nil {
		t.Fatalf("expected UpdateUI response after system terminal_refresh")
	}
	if !strings.Contains(out[1].UpdateUI.Node.Props["value"], "refresh-system") {
		t.Fatalf("system terminal_refresh missing delayed output: %+v", out[1].UpdateUI.Node.Props)
	}
}

func TestHandleMessageInputTerminalRefreshAction(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "ui-refresh-start-terminal",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "terminal",
		},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "terminal_input",
			Action:      "submit",
			Value:       "sleep 1; printf '\\x72\\x65\\x66\\x72\\x65\\x73\\x68\\x2d\\x75\\x69\\n'",
		},
	})
	time.Sleep(1200 * time.Millisecond)

	out := waitForRefreshMarker(t, handler, ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "terminal_refresh_button",
			Action:      SystemIntentTerminalRefresh,
		},
	}, "refresh-ui", 10, 200*time.Millisecond)
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2", len(out))
	}
	if out[0].Notification == "" {
		t.Fatalf("expected notification response before ui update")
	}
	if out[1].UpdateUI == nil {
		t.Fatalf("expected UpdateUI response after terminal_refresh UIAction")
	}
	if !strings.Contains(out[1].UpdateUI.Node.Props["value"], "refresh-ui") {
		t.Fatalf("terminal_refresh UIAction missing delayed output: %+v", out[1].UpdateUI.Node.Props)
	}
}

func TestHandleMessageSystemTerminalRefreshRequiresDeviceID(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	})
	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-refresh-missing-device",
			Kind:      "system",
			Intent:    SystemIntentTerminalRefresh,
		},
	})
	if err != ErrMissingCommandDeviceID {
		t.Fatalf("error = %v, want %v", err, ErrMissingCommandDeviceID)
	}
}

func TestHandleMessageSystemReconcileLivenessDefault(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	base := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)
	stale := base.Add(-10 * time.Minute)
	control.now = func() time.Time { return stale }
	handler := NewStreamHandler(control)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Heartbeat: &HeartbeatRequest{DeviceID: "device-1"},
	})
	control.now = func() time.Time { return base }

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-reconcile-1",
			Kind:      "system",
			Intent:    "reconcile_liveness",
		},
	})
	if err != nil {
		t.Fatalf("reconcile_liveness default error = %v", err)
	}
	if out[0].Data["updated"] != "1" {
		t.Fatalf("updated = %q, want 1", out[0].Data["updated"])
	}
	if out[0].Data["timeout_seconds"] != "120" {
		t.Fatalf("timeout_seconds = %q, want 120", out[0].Data["timeout_seconds"])
	}
}

func TestHandleMessageSystemReconcileLivenessCustom(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	base := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)
	stale := base.Add(-45 * time.Second)
	control.now = func() time.Time { return stale }
	handler := NewStreamHandler(control)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Heartbeat: &HeartbeatRequest{DeviceID: "device-1"},
	})
	control.now = func() time.Time { return base }

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-reconcile-2",
			Kind:      "system",
			Intent:    "reconcile_liveness 30",
		},
	})
	if err != nil {
		t.Fatalf("reconcile_liveness custom error = %v", err)
	}
	if out[0].Data["updated"] != "1" {
		t.Fatalf("updated = %q, want 1", out[0].Data["updated"])
	}
	if out[0].Data["timeout_seconds"] != "30" {
		t.Fatalf("timeout_seconds = %q, want 30", out[0].Data["timeout_seconds"])
	}
}

func TestHandleMessageSystemReconcileLivenessInvalidSeconds(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	})
	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-reconcile-bad",
			Kind:      "system",
			Intent:    "reconcile_liveness nope",
		},
	})
	if err == nil {
		t.Fatalf("expected error for invalid reconcile_liveness seconds")
	}
}

func TestHandleMessageRecentCommandsEviction(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices: devices,
		IO:      io.NewRouter(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	handler.commandDispatcher.SetRecentLimit(2)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{RequestID: "evict-1", DeviceID: "device-1", Kind: "manual", Intent: "photo frame"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{RequestID: "evict-2", DeviceID: "device-1", Action: "stop", Kind: "manual", Intent: "photo frame"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{RequestID: "evict-3", Kind: "system", Intent: "server_status"},
	})

	events := handler.commandDispatcher.Recent()
	if len(events) != 2 {
		t.Fatalf("len(recent) = %d, want 2", len(events))
	}
	if events[0].RequestID != "evict-2" || events[1].RequestID != "evict-3" {
		t.Fatalf("unexpected recent eviction order: %+v", events)
	}
}

func TestHandleMessageRejectsInvalidCommandKind(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{Devices: devices})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	})
	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "bad-kind",
			DeviceID:  "device-1",
			Kind:      "remote",
			Intent:    "photo frame",
		},
	})
	if err != ErrInvalidCommandKind {
		t.Fatalf("error = %v, want %v", err, ErrInvalidCommandKind)
	}
}

func TestHandleMessageRejectsMissingManualIntent(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{Devices: devices})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	})
	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "bad-intent",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "   ",
		},
	})
	if err != ErrMissingCommandIntent {
		t.Fatalf("error = %v, want %v", err, ErrMissingCommandIntent)
	}
}

func TestHandleMessageRejectsMissingVoiceText(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{Devices: devices})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	})
	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "bad-text",
			DeviceID:  "device-1",
			Kind:      "voice",
			Text:      "",
		},
	})
	if err != ErrMissingCommandText {
		t.Fatalf("error = %v, want %v", err, ErrMissingCommandText)
	}
}

func TestHandleMessageRejectsMissingCommandDeviceID(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{Devices: devices})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	})
	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "bad-device",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})
	if err != ErrMissingCommandDeviceID {
		t.Fatalf("error = %v, want %v", err, ErrMissingCommandDeviceID)
	}
}

func TestHandleMessageManualBluetoothScanUsesPassthroughScenario(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	bridge := &testRuntimePassthroughBridge{}
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:     devices,
		Broadcast:   ui.NewMemoryBroadcaster(),
		Passthrough: bridge,
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "ble-scan-1",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    ManualIntentBluetoothScan,
			Arguments: map[string]string{
				"window_ms": "5000",
			},
		},
	})
	if err != nil {
		t.Fatalf("manual bluetooth_scan error = %v", err)
	}
	if len(out) == 0 || out[0].ScenarioStart != "bluetooth_passthrough" {
		t.Fatalf("unexpected response: %+v", out)
	}
	if len(bridge.bluetooth) != 1 {
		t.Fatalf("len(bluetooth commands) = %d, want 1", len(bridge.bluetooth))
	}
	if bridge.bluetooth[0].Action != "scan" {
		t.Fatalf("bluetooth action = %q, want scan", bridge.bluetooth[0].Action)
	}
	if bridge.bluetooth[0].Parameters["window_ms"] != "5000" {
		t.Fatalf("window_ms = %q, want 5000", bridge.bluetooth[0].Parameters["window_ms"])
	}
}

func TestHandleMessageManualUSBClaimUsesPassthroughScenario(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	bridge := &testRuntimePassthroughBridge{}
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:     devices,
		Broadcast:   ui.NewMemoryBroadcaster(),
		Passthrough: bridge,
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "usb-claim-1",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    ManualIntentUSBClaim,
			Arguments: map[string]string{
				"vendor_id":  "1a2b",
				"product_id": "3c4d",
			},
		},
	})
	if err != nil {
		t.Fatalf("manual usb_claim error = %v", err)
	}
	if len(out) == 0 || out[0].ScenarioStart != "usb_passthrough" {
		t.Fatalf("unexpected response: %+v", out)
	}
	if len(bridge.usb) != 1 {
		t.Fatalf("len(usb commands) = %d, want 1", len(bridge.usb))
	}
	if bridge.usb[0].Action != "claim" {
		t.Fatalf("usb action = %q, want claim", bridge.usb[0].Action)
	}
	if bridge.usb[0].VendorID != "1a2b" || bridge.usb[0].ProductID != "3c4d" {
		t.Fatalf("unexpected usb cmd: %+v", bridge.usb[0])
	}
}

type testRuntimePassthroughBridge struct {
	bluetooth []scenario.BluetoothCommand
	usb       []scenario.USBCommand
}

func (t *testRuntimePassthroughBridge) DispatchBluetoothCommand(_ context.Context, cmd scenario.BluetoothCommand) error {
	t.bluetooth = append(t.bluetooth, cmd)
	return nil
}

func (t *testRuntimePassthroughBridge) DispatchUSBCommand(_ context.Context, cmd scenario.USBCommand) error {
	t.usb = append(t.usb, cmd)
	return nil
}

func contains(s, needle string) bool {
	return strings.Contains(s, needle)
}

func findNodePropValue(node *ui.Descriptor, nodeID, prop string) string {
	if node == nil {
		return ""
	}
	if node.Props["id"] == nodeID {
		return node.Props[prop]
	}
	for i := range node.Children {
		child := &node.Children[i]
		if got := findNodePropValue(child, nodeID, prop); got != "" {
			return got
		}
	}
	return ""
}

func waitForRefreshMarker(
	t *testing.T,
	handler *StreamHandler,
	request ClientMessage,
	marker string,
	attempts int,
	delay time.Duration,
) []ServerMessage {
	t.Helper()
	for i := 0; i < attempts; i++ {
		out, err := handler.HandleMessage(context.Background(), request)
		if err != nil {
			t.Fatalf("refresh request error = %v", err)
		}
		if len(out) >= 2 && out[len(out)-1].UpdateUI != nil &&
			strings.Contains(out[len(out)-1].UpdateUI.Node.Props["value"], marker) {
			return out
		}
		time.Sleep(delay)
	}
	t.Fatalf("timed out waiting for refresh marker %q", marker)
	return nil
}
