package transport

import (
	"context"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/telephony"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

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

func TestHandleMessageCapabilityDeltaStopsAudioRouteOnSpeakerEndpointAvailabilityLoss(t *testing.T) {
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
				"speakers.present":              "true",
				"speakers.endpoint_count":       "1",
				"speakers.endpoint.0.id":        "Kitchen Speaker",
				"speakers.endpoint.0.available": "true",
				"camera.present":                "true",
				"camera.endpoint_count":         "1",
				"camera.endpoint.0.id":          "Front Cam",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability snapshot) error = %v", err)
	}

	if err := router.Connect("device-2", "device-1", "audio"); err != nil {
		t.Fatalf("Connect(audio) error = %v", err)
	}
	if err := router.Connect("device-1", "device-2", "video"); err != nil {
		t.Fatalf("Connect(video) error = %v", err)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilityDelta: &CapabilityDeltaRequest{
			DeviceID:   "device-1",
			Generation: 2,
			Reason:     "speaker_endpoint_unavailable",
			Capabilities: map[string]string{
				"speakers.present":              "true",
				"speakers.endpoint_count":       "1",
				"speakers.endpoint.0.id":        "Kitchen Speaker",
				"speakers.endpoint.0.available": "false",
				"camera.present":                "true",
				"camera.endpoint_count":         "1",
				"camera.endpoint.0.id":          "Front Cam",
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
	if _, ok := stopStreamIDs["route:device-2|device-1|audio"]; !ok {
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
