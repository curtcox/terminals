package transport

import (
	"context"
	"encoding/binary"
	"sync"
	"testing"
	"time"

	diagnosticsv1 "github.com/curtcox/terminals/terminal_server/gen/go/diagnostics/v1"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/recording"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/telephony"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
	"google.golang.org/protobuf/proto"
)

type bugReportIntakeStub struct {
	ack        *diagnosticsv1.BugReportAck
	err        error
	lastReport *diagnosticsv1.BugReport
}

func (s *bugReportIntakeStub) File(_ context.Context, report *diagnosticsv1.BugReport) (*diagnosticsv1.BugReportAck, error) {
	if report != nil {
		s.lastReport = proto.Clone(report).(*diagnosticsv1.BugReport)
	}
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

type counterFramePublisherStub struct {
	mu       sync.Mutex
	deviceID []string
	counters []uint64
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

func (s *counterFramePublisherStub) Publish(deviceID string, chunk []byte) {
	if len(chunk) < 8 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deviceID = append(s.deviceID, deviceID)
	s.counters = append(s.counters, binary.BigEndian.Uint64(chunk[:8]))
}

func (s *counterFramePublisherStub) maxCounter() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	var maxVal uint64
	for _, value := range s.counters {
		if value > maxVal {
			maxVal = value
		}
	}
	return maxVal
}

func makeCounterPayload(counter uint64) []byte {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint64(payload[:8], counter)
	copy(payload[8:], []byte("test"))
	return payload
}

func TestOverlayInputPolicyAllowsMainStreamByMode(t *testing.T) {
	live := overlayInputPolicyConfig{Mode: overlayInputPolicyLive}
	if !policyAllowsMainStream(live, overlayStreamPointer) {
		t.Fatalf("LIVE policy should keep pointer stream live")
	}
	if !policyAllowsMainStream(live, overlayStreamAudio) {
		t.Fatalf("LIVE policy should keep audio stream live")
	}

	paused := overlayInputPolicyConfig{Mode: overlayInputPolicyPaused}
	if policyAllowsMainStream(paused, overlayStreamPointer) {
		t.Fatalf("PAUSED policy should block pointer stream by default")
	}
	if policyAllowsMainStream(paused, overlayStreamAudio) {
		t.Fatalf("PAUSED policy should block audio stream by default")
	}

	mixed := defaultOverlayInputPolicy()
	if policyAllowsMainStream(mixed, overlayStreamPointer) {
		t.Fatalf("MIXED policy should block pointer stream by default")
	}
	if !policyAllowsMainStream(mixed, overlayStreamAudio) {
		t.Fatalf("MIXED policy should keep audio stream live by default")
	}
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

func TestHandleMessageCapabilityDeltaRejectsStaleGeneration(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	handler := NewStreamHandler(service)

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Hello: &HelloRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	}); err != nil {
		t.Fatalf("HandleMessage(hello) error = %v", err)
	}

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 2,
			Capabilities: map[string]string{
				"screen.width": "1920",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability snapshot) error = %v", err)
	}

	baseline, ok := manager.Get("device-1")
	if !ok {
		t.Fatalf("expected device-1 in manager")
	}
	if baseline.LastSnapshot.IsZero() {
		t.Fatalf("expected LastSnapshot to be recorded after accepted snapshot")
	}
	if !baseline.LastDelta.IsZero() {
		t.Fatalf("LastDelta after snapshot = %v, want zero", baseline.LastDelta)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilityDelta: &CapabilityDeltaRequest{
			DeviceID:   "device-1",
			Generation: 1,
			Capabilities: map[string]string{
				"screen.width": "1280",
			},
		},
	})
	if err == nil {
		t.Fatalf("expected stale generation error")
	}
	if len(out) != 1 || out[0].ErrorCode != ErrorCodeProtocolViolation {
		t.Fatalf("error response = %+v, want protocol violation", out)
	}

	got, ok := manager.Get("device-1")
	if !ok {
		t.Fatalf("expected device-1 in manager")
	}
	if got.Generation != 2 {
		t.Fatalf("generation = %d, want 2", got.Generation)
	}
	if got.LastSnapshot != baseline.LastSnapshot {
		t.Fatalf("LastSnapshot = %v, want %v", got.LastSnapshot, baseline.LastSnapshot)
	}
	if !got.LastDelta.IsZero() {
		t.Fatalf("LastDelta = %v, want zero", got.LastDelta)
	}
	if got.Capabilities["screen.width"] != "1920" {
		t.Fatalf("screen.width = %q, want 1920", got.Capabilities["screen.width"])
	}
}

func TestHandleMessageCapabilitySnapshotRejectsStaleGeneration(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	handler := NewStreamHandler(service)

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Hello: &HelloRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	}); err != nil {
		t.Fatalf("HandleMessage(hello) error = %v", err)
	}

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 2,
			Capabilities: map[string]string{
				"screen.width": "1920",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability snapshot) error = %v", err)
	}

	baseline, ok := manager.Get("device-1")
	if !ok {
		t.Fatalf("expected device-1 in manager")
	}
	if baseline.LastSnapshot.IsZero() {
		t.Fatalf("expected LastSnapshot to be recorded after accepted snapshot")
	}
	if !baseline.LastDelta.IsZero() {
		t.Fatalf("LastDelta after snapshot = %v, want zero", baseline.LastDelta)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 2,
			Capabilities: map[string]string{
				"screen.width": "1280",
			},
		},
	})

	if err == nil {
		t.Fatalf("expected stale generation error")
	}
	if len(out) != 1 || out[0].ErrorCode != ErrorCodeProtocolViolation {
		t.Fatalf("error response = %+v, want protocol violation", out)
	}

	got, ok := manager.Get("device-1")
	if !ok {
		t.Fatalf("expected device-1 in manager")
	}
	if got.Generation != 2 {
		t.Fatalf("generation = %d, want 2", got.Generation)
	}
	if got.LastSnapshot != baseline.LastSnapshot {
		t.Fatalf("LastSnapshot = %v, want %v", got.LastSnapshot, baseline.LastSnapshot)
	}
	if !got.LastDelta.IsZero() {
		t.Fatalf("LastDelta = %v, want zero", got.LastDelta)
	}
	if got.Capabilities["screen.width"] != "1920" {
		t.Fatalf("screen.width = %q, want 1920", got.Capabilities["screen.width"])
	}
}

func TestHandleMessageCapabilitySnapshotReturnsRegisterAckOnRebaseline(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	service.SetRegisterMetadata(map[string]string{
		"server_build_sha": "abc123",
	})
	handler := NewStreamHandler(service)

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Hello: &HelloRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	}); err != nil {
		t.Fatalf("HandleMessage(hello) error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 1,
			Capabilities: map[string]string{
				"screen.width": "1920",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(initial capability snapshot) error = %v", err)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 2,
			Capabilities: map[string]string{
				"screen.width": "1280",
			},
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(rebaseline capability snapshot) error = %v", err)
	}

	hasRegisterAck := false
	for _, msg := range out {
		if msg.RegisterAck == nil {
			continue
		}
		hasRegisterAck = true
		if msg.RegisterAck.Metadata["server_build_sha"] != "abc123" {
			t.Fatalf("register ack metadata server_build_sha = %q, want abc123", msg.RegisterAck.Metadata["server_build_sha"])
		}
		if msg.RegisterAck.Initial.Type != "" {
			t.Fatalf("register ack initial UI should be empty for rebaseline snapshot")
		}
	}
	if !hasRegisterAck {
		t.Fatalf("expected register ack on rebaseline capability snapshot")
	}
}

func TestHandleMessageCapabilityLifecycleAckReportsSnapshotAppliedAndGeneration(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	handler := NewStreamHandler(service)

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Hello: &HelloRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	}); err != nil {
		t.Fatalf("HandleMessage(hello) error = %v", err)
	}

	snapshotOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 1,
			Capabilities: map[string]string{
				"screen.width": "1920",
			},
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(capability snapshot) error = %v", err)
	}

	foundSnapshotAck := false
	for _, msg := range snapshotOut {
		if msg.CapabilityAck == nil {
			continue
		}
		foundSnapshotAck = true
		if !msg.CapabilityAck.SnapshotApplied {
			t.Fatalf("snapshot capability ack snapshot_applied = false, want true")
		}
		if msg.CapabilityAck.AcceptedGeneration != 1 {
			t.Fatalf("snapshot capability ack accepted_generation = %d, want 1", msg.CapabilityAck.AcceptedGeneration)
		}
	}
	if !foundSnapshotAck {
		t.Fatalf("expected capability ack for snapshot")
	}

	deltaOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilityDelta: &CapabilityDeltaRequest{
			DeviceID:   "device-1",
			Generation: 2,
			Reason:     "display_resize",
			Capabilities: map[string]string{
				"screen.width": "1280",
			},
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(capability delta) error = %v", err)
	}

	foundDeltaAck := false
	for _, msg := range deltaOut {
		if msg.CapabilityAck == nil {
			continue
		}
		foundDeltaAck = true
		if msg.CapabilityAck.SnapshotApplied {
			t.Fatalf("delta capability ack snapshot_applied = true, want false")
		}
		if msg.CapabilityAck.AcceptedGeneration != 2 {
			t.Fatalf("delta capability ack accepted_generation = %d, want 2", msg.CapabilityAck.AcceptedGeneration)
		}
	}
	if !foundDeltaAck {
		t.Fatalf("expected capability ack for delta")
	}
}

func TestHandleMessageCapabilitySnapshotBootstrapsUnknownDevice(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	handler := NewStreamHandler(service)

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 1,
			Capabilities: map[string]string{
				"device_name":  "Kitchen",
				"device_type":  "tablet",
				"platform":     "android",
				"screen.width": "1920",
			},
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(capability snapshot bootstrap) error = %v", err)
	}

	hasCapabilityAck := false
	for _, msg := range out {
		if msg.CapabilityAck == nil {
			continue
		}
		hasCapabilityAck = true
		if msg.CapabilityAck.DeviceID != "device-1" {
			t.Fatalf("capability ack device_id = %q, want device-1", msg.CapabilityAck.DeviceID)
		}
	}
	if !hasCapabilityAck {
		t.Fatalf("expected capability ack for snapshot bootstrap")
	}

	got, ok := manager.Get("device-1")
	if !ok {
		t.Fatalf("expected device record to be created")
	}
	if got.DeviceName != "Kitchen" {
		t.Fatalf("device name = %q, want Kitchen", got.DeviceName)
	}
	if got.Generation != 1 {
		t.Fatalf("generation = %d, want 1", got.Generation)
	}
}

func TestHandleMessageCapabilityLossReleasesClaimsAndStopsRoutes(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	router := iorouter.NewRouter()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   manager,
		IO:        router,
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(service, runtime)

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Hello: &HelloRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	}); err != nil {
		t.Fatalf("HandleMessage(hello) error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 1,
			Capabilities: map[string]string{
				"microphone.present": "true",
				"camera.present":     "true",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability snapshot) error = %v", err)
	}

	if _, err := router.Claims().Request(context.Background(), []iorouter.Claim{{
		ActivationID: "activation-mic",
		DeviceID:     "device-1",
		Resource:     "mic.capture",
		Mode:         iorouter.ClaimExclusive,
		Priority:     1,
	}, {
		ActivationID: "activation-camera",
		DeviceID:     "device-1",
		Resource:     "camera.capture",
		Mode:         iorouter.ClaimExclusive,
		Priority:     1,
	}}); err != nil {
		t.Fatalf("Claims().Request() error = %v", err)
	}
	if err := router.Connect("device-1", "device-2", "audio"); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if err := router.Connect("device-1", "device-2", "video"); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilityDelta: &CapabilityDeltaRequest{
			DeviceID:     "device-1",
			Generation:   2,
			Reason:       "privacy.toggle",
			Capabilities: map[string]string{},
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(capability delta) error = %v", err)
	}

	if len(router.Claims().Snapshot("device-1")) != 0 {
		t.Fatalf("expected claims to be released for lost resources")
	}
	if len(out) == 0 || out[0].CapabilityAck == nil {
		t.Fatalf("expected capability ack as first response")
	}
	invalidations := out[0].CapabilityAck.Invalidations
	if len(invalidations) != 4 {
		t.Fatalf("capability ack invalidations len = %d, want 4", len(invalidations))
	}
	invalidatedResources := map[string]struct{}{}
	for _, invalidation := range invalidations {
		if invalidation.Reason != "capability_lost" {
			t.Fatalf("invalidation reason = %q, want capability_lost", invalidation.Reason)
		}
		invalidatedResources[invalidation.Resource] = struct{}{}
	}
	for _, resource := range []string{"mic.capture", "mic.analyze", "camera.capture", "camera.analyze"} {
		if _, ok := invalidatedResources[resource]; !ok {
			t.Fatalf("missing invalidation for %q", resource)
		}
	}
	stopStreamIDs := map[string]struct{}{}
	for _, msg := range out {
		if msg.StopStream != nil {
			stopStreamIDs[msg.StopStream.StreamID] = struct{}{}
		}
	}
	expectedStopStreamIDs := map[string]struct{}{
		"route:device-1|device-2|audio": {},
		"route:device-1|device-2|video": {},
	}
	if len(stopStreamIDs) != len(expectedStopStreamIDs) {
		t.Fatalf("stop_stream count = %d, want %d (ids=%v)", len(stopStreamIDs), len(expectedStopStreamIDs), stopStreamIDs)
	}
	for streamID := range expectedStopStreamIDs {
		if _, ok := stopStreamIDs[streamID]; !ok {
			t.Fatalf("missing stop_stream for %q (ids=%v)", streamID, stopStreamIDs)
		}
	}
}

func TestHandleMessageCapabilityDeltaStopsOnlyAffectedRoutesOnPartialLoss(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	router := iorouter.NewRouter()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   manager,
		IO:        router,
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(service, runtime)

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Hello: &HelloRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	}); err != nil {
		t.Fatalf("HandleMessage(hello) error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 1,
			Capabilities: map[string]string{
				"microphone.present": "true",
				"camera.present":     "true",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability snapshot) error = %v", err)
	}

	if err := router.Connect("device-1", "device-2", "audio"); err != nil {
		t.Fatalf("Connect(audio) error = %v", err)
	}
	if err := router.Connect("device-1", "device-2", "video"); err != nil {
		t.Fatalf("Connect(video) error = %v", err)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilityDelta: &CapabilityDeltaRequest{
			DeviceID:   "device-1",
			Generation: 2,
			Reason:     "microphone_lost",
			Capabilities: map[string]string{
				"camera.present": "true",
			},
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(capability delta) error = %v", err)
	}

	stopStreamIDs := map[string]struct{}{}
	for _, msg := range out {
		if msg.StopStream != nil {
			stopStreamIDs[msg.StopStream.StreamID] = struct{}{}
		}
	}
	if len(stopStreamIDs) != 1 {
		t.Fatalf("stop_stream count = %d, want 1 (ids=%v)", len(stopStreamIDs), stopStreamIDs)
	}
	if _, ok := stopStreamIDs["route:device-1|device-2|audio"]; !ok {
		t.Fatalf("missing stop_stream for audio route (ids=%v)", stopStreamIDs)
	}
	if _, ok := stopStreamIDs["route:device-1|device-2|video"]; ok {
		t.Fatalf("unexpected stop_stream for video route (ids=%v)", stopStreamIDs)
	}
}

func TestHandleMessageCapabilityDeltaStopsAudioRouteOnEndpointLoss(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	router := iorouter.NewRouter()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   manager,
		IO:        router,
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(service, runtime)

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Hello: &HelloRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	}); err != nil {
		t.Fatalf("HandleMessage(hello) error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 1,
			Capabilities: map[string]string{
				"microphone.present":        "true",
				"microphone.endpoint_count": "1",
				"microphone.endpoint.0.id":  "Mic USB",
				"camera.present":            "true",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability snapshot) error = %v", err)
	}

	if err := router.Connect("device-1", "device-2", "audio"); err != nil {
		t.Fatalf("Connect(audio) error = %v", err)
	}
	if err := router.Connect("device-1", "device-2", "video"); err != nil {
		t.Fatalf("Connect(video) error = %v", err)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilityDelta: &CapabilityDeltaRequest{
			DeviceID:   "device-1",
			Generation: 2,
			Reason:     "mic_endpoint_swapped",
			Capabilities: map[string]string{
				"microphone.present":        "true",
				"microphone.endpoint_count": "1",
				"microphone.endpoint.0.id":  "Mic Builtin",
				"camera.present":            "true",
			},
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(capability delta) error = %v", err)
	}

	stopStreamIDs := map[string]struct{}{}
	for _, msg := range out {
		if msg.StopStream != nil {
			stopStreamIDs[msg.StopStream.StreamID] = struct{}{}
		}
	}
	if len(stopStreamIDs) != 1 {
		t.Fatalf("stop_stream count = %d, want 1 (ids=%v)", len(stopStreamIDs), stopStreamIDs)
	}
	if _, ok := stopStreamIDs["route:device-1|device-2|audio"]; !ok {
		t.Fatalf("missing stop_stream for audio route (ids=%v)", stopStreamIDs)
	}
	if _, ok := stopStreamIDs["route:device-1|device-2|video"]; ok {
		t.Fatalf("unexpected stop_stream for video route (ids=%v)", stopStreamIDs)
	}
}

func TestHandleMessageCapabilityDeltaStopsVideoRouteOnEndpointLoss(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	router := iorouter.NewRouter()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   manager,
		IO:        router,
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(service, runtime)

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Hello: &HelloRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	}); err != nil {
		t.Fatalf("HandleMessage(hello) error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 1,
			Capabilities: map[string]string{
				"camera.present":            "true",
				"camera.endpoint_count":     "1",
				"camera.endpoint.0.id":      "Front Cam",
				"microphone.present":        "true",
				"microphone.endpoint_count": "1",
				"microphone.endpoint.0.id":  "Mic USB",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability snapshot) error = %v", err)
	}

	if err := router.Connect("device-1", "device-2", "audio"); err != nil {
		t.Fatalf("Connect(audio) error = %v", err)
	}
	if err := router.Connect("device-1", "device-2", "video"); err != nil {
		t.Fatalf("Connect(video) error = %v", err)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilityDelta: &CapabilityDeltaRequest{
			DeviceID:   "device-1",
			Generation: 2,
			Reason:     "camera_endpoint_swapped",
			Capabilities: map[string]string{
				"camera.present":            "true",
				"camera.endpoint_count":     "1",
				"camera.endpoint.0.id":      "Rear Cam",
				"microphone.present":        "true",
				"microphone.endpoint_count": "1",
				"microphone.endpoint.0.id":  "Mic USB",
			},
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(capability delta) error = %v", err)
	}

	stopStreamIDs := map[string]struct{}{}
	for _, msg := range out {
		if msg.StopStream != nil {
			stopStreamIDs[msg.StopStream.StreamID] = struct{}{}
		}
	}
	if len(stopStreamIDs) != 1 {
		t.Fatalf("stop_stream count = %d, want 1 (ids=%v)", len(stopStreamIDs), stopStreamIDs)
	}
	if _, ok := stopStreamIDs["route:device-1|device-2|video"]; !ok {
		t.Fatalf("missing stop_stream for video route (ids=%v)", stopStreamIDs)
	}
	if _, ok := stopStreamIDs["route:device-1|device-2|audio"]; ok {
		t.Fatalf("unexpected stop_stream for audio route (ids=%v)", stopStreamIDs)
	}
}

func TestHandleMessageCapabilityDeltaStopsVideoRouteOnEndpointAvailabilityLoss(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	router := iorouter.NewRouter()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   manager,
		IO:        router,
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(service, runtime)

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Hello: &HelloRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	}); err != nil {
		t.Fatalf("HandleMessage(hello) error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 1,
			Capabilities: map[string]string{
				"camera.present":              "true",
				"camera.endpoint_count":       "1",
				"camera.endpoint.0.id":        "Front Cam",
				"camera.endpoint.0.available": "true",
				"microphone.present":          "true",
				"microphone.endpoint_count":   "1",
				"microphone.endpoint.0.id":    "Mic USB",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability snapshot) error = %v", err)
	}

	if err := router.Connect("device-1", "device-2", "audio"); err != nil {
		t.Fatalf("Connect(audio) error = %v", err)
	}
	if err := router.Connect("device-1", "device-2", "video"); err != nil {
		t.Fatalf("Connect(video) error = %v", err)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilityDelta: &CapabilityDeltaRequest{
			DeviceID:   "device-1",
			Generation: 2,
			Reason:     "camera_endpoint_unavailable",
			Capabilities: map[string]string{
				"camera.present":              "true",
				"camera.endpoint_count":       "1",
				"camera.endpoint.0.id":        "Front Cam",
				"camera.endpoint.0.available": "false",
				"microphone.present":          "true",
				"microphone.endpoint_count":   "1",
				"microphone.endpoint.0.id":    "Mic USB",
			},
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(capability delta) error = %v", err)
	}

	stopStreamIDs := map[string]struct{}{}
	for _, msg := range out {
		if msg.StopStream != nil {
			stopStreamIDs[msg.StopStream.StreamID] = struct{}{}
		}
	}
	if len(stopStreamIDs) != 1 {
		t.Fatalf("stop_stream count = %d, want 1 (ids=%v)", len(stopStreamIDs), stopStreamIDs)
	}
	if _, ok := stopStreamIDs["route:device-1|device-2|video"]; !ok {
		t.Fatalf("missing stop_stream for video route (ids=%v)", stopStreamIDs)
	}
	if _, ok := stopStreamIDs["route:device-1|device-2|audio"]; ok {
		t.Fatalf("unexpected stop_stream for audio route (ids=%v)", stopStreamIDs)
	}
}

func TestHandleMessageCapabilityDeltaStopsAudioRouteOnEndpointAvailabilityLoss(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	router := iorouter.NewRouter()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   manager,
		IO:        router,
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(service, runtime)

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Hello: &HelloRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	}); err != nil {
		t.Fatalf("HandleMessage(hello) error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 1,
			Capabilities: map[string]string{
				"microphone.present":              "true",
				"microphone.endpoint_count":       "1",
				"microphone.endpoint.0.id":        "Mic USB",
				"microphone.endpoint.0.available": "true",
				"camera.present":                  "true",
				"camera.endpoint_count":           "1",
				"camera.endpoint.0.id":            "Front Cam",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability snapshot) error = %v", err)
	}

	if err := router.Connect("device-1", "device-2", "audio"); err != nil {
		t.Fatalf("Connect(audio) error = %v", err)
	}
	if err := router.Connect("device-1", "device-2", "video"); err != nil {
		t.Fatalf("Connect(video) error = %v", err)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilityDelta: &CapabilityDeltaRequest{
			DeviceID:   "device-1",
			Generation: 2,
			Reason:     "microphone_endpoint_unavailable",
			Capabilities: map[string]string{
				"microphone.present":              "true",
				"microphone.endpoint_count":       "1",
				"microphone.endpoint.0.id":        "Mic USB",
				"microphone.endpoint.0.available": "false",
				"camera.present":                  "true",
				"camera.endpoint_count":           "1",
				"camera.endpoint.0.id":            "Front Cam",
			},
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(capability delta) error = %v", err)
	}

	stopStreamIDs := map[string]struct{}{}
	for _, msg := range out {
		if msg.StopStream != nil {
			stopStreamIDs[msg.StopStream.StreamID] = struct{}{}
		}
	}
	if len(stopStreamIDs) != 1 {
		t.Fatalf("stop_stream count = %d, want 1 (ids=%v)", len(stopStreamIDs), stopStreamIDs)
	}
	if _, ok := stopStreamIDs["route:device-1|device-2|audio"]; !ok {
		t.Fatalf("missing stop_stream for audio route (ids=%v)", stopStreamIDs)
	}
	if _, ok := stopStreamIDs["route:device-1|device-2|video"]; ok {
		t.Fatalf("unexpected stop_stream for video route (ids=%v)", stopStreamIDs)
	}
}

func TestHandleMessageCapabilityDeltaStopsVideoRouteOnDisplayEndpointLoss(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	router := iorouter.NewRouter()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   manager,
		IO:        router,
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(service, runtime)

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Hello: &HelloRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	}); err != nil {
		t.Fatalf("HandleMessage(hello) error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 1,
			Capabilities: map[string]string{
				"display.count":      "1",
				"display.0.id":       "Main Display",
				"display.0.width":    "1920",
				"display.0.height":   "1080",
				"microphone.present": "true",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability snapshot) error = %v", err)
	}

	if err := router.Connect("device-2", "device-1", "video"); err != nil {
		t.Fatalf("Connect(video) error = %v", err)
	}
	if err := router.Connect("device-1", "device-2", "audio"); err != nil {
		t.Fatalf("Connect(audio) error = %v", err)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilityDelta: &CapabilityDeltaRequest{
			DeviceID:   "device-1",
			Generation: 2,
			Reason:     "display_hot_unplugged",
			Capabilities: map[string]string{
				"microphone.present": "true",
			},
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(capability delta) error = %v", err)
	}

	stopStreamIDs := map[string]struct{}{}
	for _, msg := range out {
		if msg.StopStream != nil {
			stopStreamIDs[msg.StopStream.StreamID] = struct{}{}
		}
	}
	if len(stopStreamIDs) != 1 {
		t.Fatalf("stop_stream count = %d, want 1 (ids=%v)", len(stopStreamIDs), stopStreamIDs)
	}
	if _, ok := stopStreamIDs["route:device-2|device-1|video"]; !ok {
		t.Fatalf("missing stop_stream for video route targeting display device (ids=%v)", stopStreamIDs)
	}
	if _, ok := stopStreamIDs["route:device-1|device-2|audio"]; ok {
		t.Fatalf("unexpected stop_stream for audio route (ids=%v)", stopStreamIDs)
	}
}

func TestHandleMessageCapabilitySnapshotStopsOnlyAffectedRoutesOnRebaselineLoss(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	router := iorouter.NewRouter()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   manager,
		IO:        router,
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(service, runtime)

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Hello: &HelloRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	}); err != nil {
		t.Fatalf("HandleMessage(hello) error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 1,
			Capabilities: map[string]string{
				"microphone.present": "true",
				"camera.present":     "true",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(initial capability snapshot) error = %v", err)
	}

	if err := router.Connect("device-1", "device-2", "audio"); err != nil {
		t.Fatalf("Connect(audio) error = %v", err)
	}
	if err := router.Connect("device-1", "device-2", "video"); err != nil {
		t.Fatalf("Connect(video) error = %v", err)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 2,
			Capabilities: map[string]string{
				"camera.present": "true",
			},
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(rebaseline capability snapshot) error = %v", err)
	}

	stopStreamIDs := map[string]struct{}{}
	for _, msg := range out {
		if msg.StopStream != nil {
			stopStreamIDs[msg.StopStream.StreamID] = struct{}{}
		}
	}
	if len(stopStreamIDs) != 1 {
		t.Fatalf("stop_stream count = %d, want 1 (ids=%v)", len(stopStreamIDs), stopStreamIDs)
	}
	if _, ok := stopStreamIDs["route:device-1|device-2|audio"]; !ok {
		t.Fatalf("missing stop_stream for audio route (ids=%v)", stopStreamIDs)
	}
	if _, ok := stopStreamIDs["route:device-1|device-2|video"]; ok {
		t.Fatalf("unexpected stop_stream for video route (ids=%v)", stopStreamIDs)
	}
}

func TestHandleMessageCapabilityDeltaRestoresOnlyMatchingSuspendedClaimsOnPartialRegain(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	router := iorouter.NewRouter()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   manager,
		IO:        router,
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(service, runtime)

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Hello: &HelloRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	}); err != nil {
		t.Fatalf("HandleMessage(hello) error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 1,
			Capabilities: map[string]string{
				"microphone.present": "true",
				"camera.present":     "true",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability snapshot) error = %v", err)
	}

	if _, err := router.Claims().Request(context.Background(), []iorouter.Claim{{
		ActivationID: "activation-mic",
		DeviceID:     "device-1",
		Resource:     "mic.capture",
		Mode:         iorouter.ClaimExclusive,
		Priority:     1,
	}, {
		ActivationID: "activation-camera",
		DeviceID:     "device-1",
		Resource:     "camera.capture",
		Mode:         iorouter.ClaimExclusive,
		Priority:     1,
	}}); err != nil {
		t.Fatalf("Claims().Request() error = %v", err)
	}

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilityDelta: &CapabilityDeltaRequest{
			DeviceID:     "device-1",
			Generation:   2,
			Reason:       "privacy.toggle",
			Capabilities: map[string]string{},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability loss delta) error = %v", err)
	}

	if claims := router.Claims().Snapshot("device-1"); len(claims) != 0 {
		t.Fatalf("active claims after capability loss = %d, want 0", len(claims))
	}

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilityDelta: &CapabilityDeltaRequest{
			DeviceID:   "device-1",
			Generation: 3,
			Reason:     "camera_readded",
			Capabilities: map[string]string{
				"camera.present": "true",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability partial regain delta) error = %v", err)
	}

	claimsAfterPartialRegain := router.Claims().Snapshot("device-1")
	if len(claimsAfterPartialRegain) != 1 {
		t.Fatalf("active claims after partial regain = %d, want 1", len(claimsAfterPartialRegain))
	}
	if claimsAfterPartialRegain[0].Resource != "camera.capture" {
		t.Fatalf("restored claim resource = %q, want camera.capture", claimsAfterPartialRegain[0].Resource)
	}
	if claimsAfterPartialRegain[0].ActivationID != "activation-camera" {
		t.Fatalf("camera claim activation = %q, want activation-camera", claimsAfterPartialRegain[0].ActivationID)
	}

	pending := handler.suspendedClaimsByDevice["device-1"]
	if len(pending) != 1 {
		t.Fatalf("suspended claim count after partial regain = %d, want 1", len(pending))
	}
	if pending[0].Resource != "mic.capture" {
		t.Fatalf("remaining suspended resource = %q, want mic.capture", pending[0].Resource)
	}

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilityDelta: &CapabilityDeltaRequest{
			DeviceID:   "device-1",
			Generation: 4,
			Reason:     "microphone_readded",
			Capabilities: map[string]string{
				"camera.present":     "true",
				"microphone.present": "true",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability full regain delta) error = %v", err)
	}

	claimsAfterFullRegain := router.Claims().Snapshot("device-1")
	if len(claimsAfterFullRegain) != 2 {
		t.Fatalf("active claims after full regain = %d, want 2", len(claimsAfterFullRegain))
	}
	claimByResource := map[string]string{}
	for _, claim := range claimsAfterFullRegain {
		claimByResource[claim.Resource] = claim.ActivationID
	}
	if claimByResource["camera.capture"] != "activation-camera" {
		t.Fatalf("camera.capture activation = %q, want activation-camera", claimByResource["camera.capture"])
	}
	if claimByResource["mic.capture"] != "activation-mic" {
		t.Fatalf("mic.capture activation = %q, want activation-mic", claimByResource["mic.capture"])
	}
	if len(handler.suspendedClaimsByDevice["device-1"]) != 0 {
		t.Fatalf("expected no suspended claims after full regain")
	}
}

func TestHandleMessageCapabilitySnapshotRestoresSuspendedClaimsOnRebaselineRegain(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	router := iorouter.NewRouter()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   manager,
		IO:        router,
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(service, runtime)

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Hello: &HelloRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	}); err != nil {
		t.Fatalf("HandleMessage(hello) error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 1,
			Capabilities: map[string]string{
				"microphone.present": "true",
				"camera.present":     "true",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(initial capability snapshot) error = %v", err)
	}

	if _, err := router.Claims().Request(context.Background(), []iorouter.Claim{{
		ActivationID: "activation-mic",
		DeviceID:     "device-1",
		Resource:     "mic.capture",
		Mode:         iorouter.ClaimExclusive,
		Priority:     1,
	}, {
		ActivationID: "activation-camera",
		DeviceID:     "device-1",
		Resource:     "camera.capture",
		Mode:         iorouter.ClaimExclusive,
		Priority:     1,
	}}); err != nil {
		t.Fatalf("Claims().Request() error = %v", err)
	}

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 2,
			Capabilities: map[string]string{
				"camera.present": "true",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability rebaseline loss snapshot) error = %v", err)
	}

	claimsAfterLoss := router.Claims().Snapshot("device-1")
	if len(claimsAfterLoss) != 1 {
		t.Fatalf("active claims after snapshot loss = %d, want 1", len(claimsAfterLoss))
	}
	if claimsAfterLoss[0].Resource != "camera.capture" {
		t.Fatalf("remaining claim resource = %q, want camera.capture", claimsAfterLoss[0].Resource)
	}

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 3,
			Capabilities: map[string]string{
				"camera.present":     "true",
				"microphone.present": "true",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability rebaseline regain snapshot) error = %v", err)
	}

	claimsAfterRegain := router.Claims().Snapshot("device-1")
	if len(claimsAfterRegain) != 2 {
		t.Fatalf("active claims after snapshot regain = %d, want 2", len(claimsAfterRegain))
	}
	claimByResource := map[string]string{}
	for _, claim := range claimsAfterRegain {
		claimByResource[claim.Resource] = claim.ActivationID
	}
	if claimByResource["camera.capture"] != "activation-camera" {
		t.Fatalf("camera.capture activation = %q, want activation-camera", claimByResource["camera.capture"])
	}
	if claimByResource["mic.capture"] != "activation-mic" {
		t.Fatalf("mic.capture activation = %q, want activation-mic", claimByResource["mic.capture"])
	}
	if len(handler.suspendedClaimsByDevice["device-1"]) != 0 {
		t.Fatalf("expected no suspended claims after snapshot regain")
	}
}

func TestHandleMessageCapabilityDeltaEmitsTypedCapabilityEvents(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	broadcaster := ui.NewMemoryBroadcaster()
	router := iorouter.NewRouter()
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

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Hello: &HelloRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	}); err != nil {
		t.Fatalf("HandleMessage(hello) error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 1,
			Capabilities: map[string]string{
				"screen.width":       "1920",
				"screen.height":      "1080",
				"microphone.present": "true",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability snapshot) error = %v", err)
	}

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilityDelta: &CapabilityDeltaRequest{
			DeviceID:   "device-1",
			Generation: 2,
			Reason:     "display_resize_and_mic_loss",
			Capabilities: map[string]string{
				"screen.width":  "1280",
				"screen.height": "720",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability delta) error = %v", err)
	}

	events := broadcaster.Events()
	messages := make([]string, 0, len(events))
	for _, event := range events {
		messages = append(messages, event.Message)
	}
	if !containsMessage(messages, "terminal.capability.updated") {
		t.Fatalf("expected terminal.capability.updated in events: %+v", messages)
	}
	if !containsMessage(messages, "terminal.capability.removed") {
		t.Fatalf("expected terminal.capability.removed in events: %+v", messages)
	}
	if !containsMessage(messages, "terminal.display.resized") {
		t.Fatalf("expected terminal.display.resized in events: %+v", messages)
	}
	if !containsMessage(messages, "terminal.audio_route.changed") {
		t.Fatalf("expected terminal.audio_route.changed in events: %+v", messages)
	}
	if !containsMessage(messages, "terminal.resource.lost") {
		t.Fatalf("expected terminal.resource.lost in events: %+v", messages)
	}
	if !containsMessage(messages, "terminal.resource.lost:mic.capture") {
		t.Fatalf("expected terminal.resource.lost:mic.capture in events: %+v", messages)
	}
}

func TestHandleMessageCapabilityDeltaNoOpDoesNotEmitCapabilityEvents(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	broadcaster := ui.NewMemoryBroadcaster()
	router := iorouter.NewRouter()
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

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Hello: &HelloRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	}); err != nil {
		t.Fatalf("HandleMessage(hello) error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 1,
			Capabilities: map[string]string{
				"screen.width":       "1920",
				"screen.height":      "1080",
				"microphone.present": "true",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability snapshot) error = %v", err)
	}

	if len(broadcaster.Events()) != 0 {
		t.Fatalf("expected no events after initial baseline snapshot")
	}

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilityDelta: &CapabilityDeltaRequest{
			DeviceID:   "device-1",
			Generation: 2,
			Reason:     "noop_refresh",
			Capabilities: map[string]string{
				"screen.width":       "1920",
				"screen.height":      "1080",
				"microphone.present": "true",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability delta) error = %v", err)
	}

	if len(broadcaster.Events()) != 0 {
		messages := make([]string, 0, len(broadcaster.Events()))
		for _, event := range broadcaster.Events() {
			messages = append(messages, event.Message)
		}
		t.Fatalf("expected no capability lifecycle events for no-op delta: %+v", messages)
	}
}

func TestHandleMessageCapabilitySnapshotNoOpRebaselineDoesNotEmitCapabilityEvents(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	broadcaster := ui.NewMemoryBroadcaster()
	router := iorouter.NewRouter()
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

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Hello: &HelloRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	}); err != nil {
		t.Fatalf("HandleMessage(hello) error = %v", err)
	}

	baseline := map[string]string{
		"screen.width":       "1920",
		"screen.height":      "1080",
		"microphone.present": "true",
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:     "device-1",
			Generation:   1,
			Capabilities: baseline,
		},
	}); err != nil {
		t.Fatalf("HandleMessage(initial capability snapshot) error = %v", err)
	}

	if len(broadcaster.Events()) != 0 {
		t.Fatalf("expected no events after initial baseline snapshot")
	}

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 2,
			Capabilities: map[string]string{
				"screen.width":       "1920",
				"screen.height":      "1080",
				"microphone.present": "true",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(no-op rebaseline capability snapshot) error = %v", err)
	}

	if len(broadcaster.Events()) != 0 {
		messages := make([]string, 0, len(broadcaster.Events()))
		for _, event := range broadcaster.Events() {
			messages = append(messages, event.Message)
		}
		t.Fatalf("expected no capability lifecycle events for no-op rebaseline snapshot: %+v", messages)
	}
}

func TestHandleMessageCapabilitySnapshotInitialBaselineDoesNotEmitCapabilityEvents(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	broadcaster := ui.NewMemoryBroadcaster()
	router := iorouter.NewRouter()
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

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Hello: &HelloRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	}); err != nil {
		t.Fatalf("HandleMessage(hello) error = %v", err)
	}

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 1,
			Capabilities: map[string]string{
				"screen.width":       "1920",
				"screen.height":      "1080",
				"microphone.present": "true",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability snapshot) error = %v", err)
	}

	events := broadcaster.Events()
	if len(events) != 0 {
		messages := make([]string, 0, len(events))
		for _, event := range events {
			messages = append(messages, event.Message)
		}
		t.Fatalf("expected no capability events on initial baseline snapshot: %+v", messages)
	}
}

func TestHandleMessageCapabilitySnapshotRebaselineEmitsTypedCapabilityEvents(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	broadcaster := ui.NewMemoryBroadcaster()
	router := iorouter.NewRouter()
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

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Hello: &HelloRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	}); err != nil {
		t.Fatalf("HandleMessage(hello) error = %v", err)
	}

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 1,
			Capabilities: map[string]string{
				"screen.width":       "1920",
				"screen.height":      "1080",
				"microphone.present": "true",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(initial capability snapshot) error = %v", err)
	}

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 2,
			Capabilities: map[string]string{
				"screen.width":  "1280",
				"screen.height": "720",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(rebaseline capability snapshot) error = %v", err)
	}

	events := broadcaster.Events()
	messages := make([]string, 0, len(events))
	for _, event := range events {
		messages = append(messages, event.Message)
	}
	if !containsMessage(messages, "terminal.capability.updated") {
		t.Fatalf("expected terminal.capability.updated in events: %+v", messages)
	}
	if !containsMessage(messages, "terminal.capability.removed") {
		t.Fatalf("expected terminal.capability.removed in events: %+v", messages)
	}
	if !containsMessage(messages, "terminal.display.resized") {
		t.Fatalf("expected terminal.display.resized in events: %+v", messages)
	}
	if !containsMessage(messages, "terminal.audio_route.changed") {
		t.Fatalf("expected terminal.audio_route.changed in events: %+v", messages)
	}
	if !containsMessage(messages, "terminal.resource.lost") {
		t.Fatalf("expected terminal.resource.lost in events: %+v", messages)
	}
	if !containsMessage(messages, "terminal.resource.lost:mic.capture") {
		t.Fatalf("expected terminal.resource.lost:mic.capture in events: %+v", messages)
	}
}

func TestHandleMessageCapabilityDeltaEmitsDisplayResizedForDisplayCapabilityGeometry(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	broadcaster := ui.NewMemoryBroadcaster()
	router := iorouter.NewRouter()
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

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Hello: &HelloRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	}); err != nil {
		t.Fatalf("HandleMessage(hello) error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 1,
			Capabilities: map[string]string{
				"display.count":      "1",
				"display.0.id":       "main",
				"display.0.width":    "1920",
				"display.0.height":   "1080",
				"display.0.density":  "2.0",
				"display.0.safe.top": "16",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability snapshot) error = %v", err)
	}

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilityDelta: &CapabilityDeltaRequest{
			DeviceID:   "device-1",
			Generation: 2,
			Reason:     "display_geometry_changed",
			Capabilities: map[string]string{
				"display.count":      "1",
				"display.0.id":       "main",
				"display.0.width":    "1280",
				"display.0.height":   "720",
				"display.0.density":  "2.0",
				"display.0.safe.top": "24",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability delta) error = %v", err)
	}

	events := broadcaster.Events()
	messages := make([]string, 0, len(events))
	for _, event := range events {
		messages = append(messages, event.Message)
	}
	if !containsMessage(messages, "terminal.display.resized") {
		t.Fatalf("expected terminal.display.resized in events: %+v", messages)
	}
}

func TestHandleMessageCapabilityDeltaEmitsCapabilityAddedEventOnEndpointGain(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	broadcaster := ui.NewMemoryBroadcaster()
	router := iorouter.NewRouter()
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

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Hello: &HelloRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	}); err != nil {
		t.Fatalf("HandleMessage(hello) error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:     "device-1",
			Generation:   1,
			Capabilities: map[string]string{},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability snapshot) error = %v", err)
	}

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilityDelta: &CapabilityDeltaRequest{
			DeviceID:   "device-1",
			Generation: 2,
			Reason:     "headset_connected",
			Capabilities: map[string]string{
				"speakers.present":        "true",
				"speakers.endpoint_count": "1",
				"speakers.endpoint.0.id":  "headset",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability delta) error = %v", err)
	}

	events := broadcaster.Events()
	messages := make([]string, 0, len(events))
	for _, event := range events {
		messages = append(messages, event.Message)
	}
	if !containsMessage(messages, "terminal.capability.added") {
		t.Fatalf("expected terminal.capability.added in events: %+v", messages)
	}
	if !containsMessage(messages, "terminal.audio_route.changed") {
		t.Fatalf("expected terminal.audio_route.changed in events: %+v", messages)
	}
	if containsMessage(messages, "terminal.capability.removed") {
		t.Fatalf("did not expect terminal.capability.removed in events: %+v", messages)
	}
}

func containsMessage(messages []string, target string) bool {
	for _, message := range messages {
		if message == target {
			return true
		}
	}
	return false
}

func TestCapabilityResourcesCompilesEndpointScopedResources(t *testing.T) {
	resources := capabilityResources(map[string]string{
		"screen.width":                    "1920",
		"screen.height":                   "1080",
		"display.count":                   "1",
		"display.0.id":                    "Main Display",
		"speakers.present":                "true",
		"speakers.endpoint_count":         "1",
		"speakers.endpoint.0.id":          "Kitchen Speaker",
		"speakers.endpoint.1.id":          "Muted Speaker",
		"speakers.endpoint.1.available":   "false",
		"microphone.present":              "true",
		"microphone.endpoint_count":       "1",
		"microphone.endpoint.0.id":        "Mic USB",
		"microphone.endpoint.1.id":        "Muted Mic",
		"microphone.endpoint.1.available": "false",
		"camera.present":                  "true",
		"camera.endpoint_count":           "1",
		"camera.endpoint.0.id":            "Front Cam",
		"camera.endpoint.1.available":     "true", // no id -> deterministic fallback token
		"camera.endpoint.2.id":            "Unavailable Cam",
		"camera.endpoint.2.available":     "false",
	})

	want := []string{
		"screen.main",
		"screen.overlay",
		"display.main-display.main",
		"display.main-display.overlay",
		"speaker.main",
		"audio_out.kitchen-speaker",
		"mic.capture",
		"mic.analyze",
		"audio_in.mic-usb.capture",
		"audio_in.mic-usb.analyze",
		"camera.capture",
		"camera.analyze",
		"camera.front-cam.capture",
		"camera.front-cam.analyze",
		"camera.endpoint-1.capture",
		"camera.endpoint-1.analyze",
	}
	for _, resource := range want {
		if _, ok := resources[resource]; !ok {
			t.Fatalf("resource %q missing from %+v", resource, resources)
		}
	}
	forbidden := []string{
		"audio_out.muted-speaker",
		"audio_in.muted-mic.capture",
		"audio_in.muted-mic.analyze",
		"camera.unavailable-cam.capture",
		"camera.unavailable-cam.analyze",
	}
	for _, resource := range forbidden {
		if _, ok := resources[resource]; ok {
			t.Fatalf("resource %q unexpectedly present in %+v", resource, resources)
		}
	}
}

func TestHandleMessageCapabilityLossReleasesEndpointScopedClaims(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	router := iorouter.NewRouter()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   manager,
		IO:        router,
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(service, runtime)

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Hello: &HelloRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	}); err != nil {
		t.Fatalf("HandleMessage(hello) error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 1,
			Capabilities: map[string]string{
				"microphone.present":        "true",
				"microphone.endpoint_count": "1",
				"microphone.endpoint.0.id":  "Mic USB",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability snapshot) error = %v", err)
	}

	if _, err := router.Claims().Request(context.Background(), []iorouter.Claim{{
		ActivationID: "activation-endpoint-mic",
		DeviceID:     "device-1",
		Resource:     "audio_in.mic-usb.capture",
		Mode:         iorouter.ClaimExclusive,
		Priority:     1,
	}}); err != nil {
		t.Fatalf("Claims().Request() error = %v", err)
	}

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilityDelta: &CapabilityDeltaRequest{
			DeviceID:     "device-1",
			Generation:   2,
			Reason:       "mic_unplugged",
			Capabilities: map[string]string{},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability delta) error = %v", err)
	}

	if got := router.Claims().Snapshot("device-1"); len(got) != 0 {
		t.Fatalf("expected endpoint claims to be released for lost endpoint resource, got %+v", got)
	}
}

func TestHandleMessageCapabilityLossKeepsUnaffectedClaimsForSameActivation(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	router := iorouter.NewRouter()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   manager,
		IO:        router,
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(service, runtime)

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Hello: &HelloRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	}); err != nil {
		t.Fatalf("HandleMessage(hello) error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 1,
			Capabilities: map[string]string{
				"microphone.present": "true",
				"speakers.present":   "true",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability snapshot) error = %v", err)
	}

	if _, err := router.Claims().Request(context.Background(), []iorouter.Claim{{
		ActivationID: "activation-media",
		DeviceID:     "device-1",
		Resource:     "mic.capture",
		Mode:         iorouter.ClaimExclusive,
		Priority:     1,
	}, {
		ActivationID: "activation-media",
		DeviceID:     "device-1",
		Resource:     "speaker.main",
		Mode:         iorouter.ClaimExclusive,
		Priority:     1,
	}}); err != nil {
		t.Fatalf("Claims().Request() error = %v", err)
	}

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilityDelta: &CapabilityDeltaRequest{
			DeviceID:   "device-1",
			Generation: 2,
			Reason:     "mic_unplugged",
			Capabilities: map[string]string{
				"speakers.present": "true",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability delta) error = %v", err)
	}

	claims := router.Claims().Snapshot("device-1")
	if len(claims) != 1 {
		t.Fatalf("len(claims) = %d, want 1", len(claims))
	}
	if claims[0].ActivationID != "activation-media" || claims[0].Resource != "speaker.main" {
		t.Fatalf("remaining claim = %+v, want activation-media speaker.main", claims[0])
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

func TestHandleMessageVoiceAudioDropsPostPrivacyCutoverFrames(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	handler := NewStreamHandler(service)
	publisher := &counterFramePublisherStub{}
	handler.SetDeviceAudioPublisher(publisher)

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Hello: &HelloRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	}); err != nil {
		t.Fatalf("HandleMessage(hello) error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 1,
			Capabilities: map[string]string{
				"microphone.present": "true",
				"camera.present":     "true",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability snapshot) error = %v", err)
	}

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		VoiceAudio: &VoiceAudioRequest{
			DeviceID:   "device-1",
			Audio:      makeCounterPayload(1),
			SampleRate: 16000,
			IsFinal:    false,
		},
	}); err != nil {
		t.Fatalf("HandleMessage(voice_audio pre-cutover) error = %v", err)
	}

	cutoverCounter := publisher.maxCounter()
	if cutoverCounter != 1 {
		t.Fatalf("cutover counter = %d, want 1", cutoverCounter)
	}

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilityDelta: &CapabilityDeltaRequest{
			DeviceID:     "device-1",
			Generation:   2,
			Reason:       "privacy.toggle",
			Capabilities: map[string]string{},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability delta) error = %v", err)
	}

	for _, counter := range []uint64{2, 3, 4} {
		if _, err := handler.HandleMessage(context.Background(), ClientMessage{
			VoiceAudio: &VoiceAudioRequest{
				DeviceID:   "device-1",
				Audio:      makeCounterPayload(counter),
				SampleRate: 16000,
				IsFinal:    false,
			},
		}); err != nil {
			t.Fatalf("HandleMessage(voice_audio post-cutover counter=%d) error = %v", counter, err)
		}
	}

	if got := publisher.maxCounter(); got > cutoverCounter {
		t.Fatalf("max delivered counter after cutover = %d, want <= %d", got, cutoverCounter)
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
		IO:        iorouter.NewRouter(),
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
	handler.SetBugReportIntake(&bugReportIntakeStub{
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
	handler.SetBugReportIntake(&bugReportIntakeStub{
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

func TestHandleMessageInputBugReportActionRespectsModalitySources(t *testing.T) {
	tests := []struct {
		name       string
		action     string
		wantSource diagnosticsv1.BugReportSource
	}{
		{name: "screen button", action: "bug_report", wantSource: diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_SCREEN_BUTTON},
		{name: "gesture", action: "bug_report.gesture", wantSource: diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_GESTURE},
		{name: "shake", action: "bug_report.shake", wantSource: diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_SHAKE},
		{name: "keyboard", action: "bug_report.keyboard", wantSource: diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_KEYBOARD},
		{name: "voice", action: "bug_report.voice", wantSource: diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_VOICE},
		{name: "qr", action: "bug_report.qr:subject-2", wantSource: diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_QR},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			manager := device.NewManager()
			service := NewControlService("srv-1", manager)
			handler := NewStreamHandler(service)
			intake := &bugReportIntakeStub{
				ack: &diagnosticsv1.BugReportAck{
					ReportId:      "bug-from-modality",
					CorrelationId: "bug:bug-from-modality",
					Status:        diagnosticsv1.BugReportStatus_BUG_REPORT_STATUS_FILED,
				},
			}
			handler.SetBugReportIntake(intake)

			_, err := handler.HandleMessage(context.Background(), ClientMessage{
				Input: &InputRequest{
					DeviceID:    "device-1",
					ComponentID: bugReportButtonID,
					Action:      tc.action,
				},
			})
			if err != nil {
				t.Fatalf("HandleMessage(input bug_report) error = %v", err)
			}
			if intake.lastReport == nil {
				t.Fatalf("expected intake to receive bug report payload")
			}
			if got := intake.lastReport.GetSource(); got != tc.wantSource {
				t.Fatalf("source = %v, want %v", got, tc.wantSource)
			}
			if tc.wantSource == diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_QR {
				if got := intake.lastReport.GetSubjectDeviceId(); got != "subject-2" {
					t.Fatalf("subject_device_id = %q, want subject-2", got)
				}
				return
			}
			if got := intake.lastReport.GetSubjectDeviceId(); got != "device-1" {
				t.Fatalf("subject_device_id = %q, want device-1", got)
			}
		})
	}
}

func TestHandleDisconnectStopsRecordingForDisconnectedDeviceRoutes(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	broadcaster := ui.NewMemoryBroadcaster()
	router := iorouter.NewRouter()
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
