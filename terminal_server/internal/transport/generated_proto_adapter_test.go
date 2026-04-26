package transport

import (
	"testing"
	"time"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	diagnosticsv1 "github.com/curtcox/terminals/terminal_server/gen/go/diagnostics/v1"
	iov1 "github.com/curtcox/terminals/terminal_server/gen/go/io/v1"
	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
	"google.golang.org/protobuf/proto"
)

func TestGeneratedProtoAdapterToInternalRegister(t *testing.T) {
	adapter := GeneratedProtoAdapter{}
	msg, err := adapter.ToInternal(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "device-1",
					Identity: &capabilitiesv1.DeviceIdentity{
						DeviceName: "Kitchen Display",
						DeviceType: "tablet",
						Platform:   "android",
					},
					Screen: &capabilitiesv1.ScreenCapability{
						Width:   1920,
						Height:  1080,
						Density: 2.0,
						Touch:   true,
					},
					Speakers: &capabilitiesv1.AudioOutputCapability{
						Channels:    2,
						SampleRates: []int32{44100, 48000},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("ToInternal() error = %v", err)
	}
	if msg.Register == nil {
		t.Fatalf("expected register message")
	}
	if msg.Register.DeviceID != "device-1" {
		t.Fatalf("device_id = %q, want %q", msg.Register.DeviceID, "device-1")
	}
	if msg.Register.DeviceName != "Kitchen Display" {
		t.Fatalf("device_name = %q, want %q", msg.Register.DeviceName, "Kitchen Display")
	}
	if msg.Register.Capabilities["platform"] != "android" {
		t.Fatalf("platform capability = %q, want %q", msg.Register.Capabilities["platform"], "android")
	}
	if msg.Register.Capabilities["screen.width"] != "1920" {
		t.Fatalf("screen.width capability = %q, want 1920", msg.Register.Capabilities["screen.width"])
	}
	if msg.Register.Capabilities["speakers.sample_rates"] != "44100,48000" {
		t.Fatalf(
			"speakers.sample_rates capability = %q, want 44100,48000",
			msg.Register.Capabilities["speakers.sample_rates"],
		)
	}
}

func TestGeneratedProtoAdapterToInternalHello(t *testing.T) {
	adapter := GeneratedProtoAdapter{}
	msg, err := adapter.ToInternal(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Hello{
			Hello: &controlv1.Hello{
				DeviceId: "device-1",
				Identity: &capabilitiesv1.DeviceIdentity{
					DeviceName: "Kitchen Display",
					DeviceType: "tablet",
					Platform:   "android",
				},
				ClientVersion: "1.2.3",
			},
		},
	})
	if err != nil {
		t.Fatalf("ToInternal(hello) error = %v", err)
	}
	if msg.Hello == nil {
		t.Fatalf("expected hello message")
	}
	if msg.Hello.DeviceID != "device-1" {
		t.Fatalf("device_id = %q, want device-1", msg.Hello.DeviceID)
	}
	if msg.Hello.DeviceName != "Kitchen Display" {
		t.Fatalf("device_name = %q, want Kitchen Display", msg.Hello.DeviceName)
	}
}

func TestGeneratedProtoAdapterToInternalCapabilitySnapshotAndDelta(t *testing.T) {
	adapter := GeneratedProtoAdapter{}

	snapshotMsg, err := adapter.ToInternal(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_CapabilitySnapshot{
			CapabilitySnapshot: &controlv1.CapabilitySnapshot{
				DeviceId:   "device-1",
				Generation: 1,
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "device-1",
					Screen: &capabilitiesv1.ScreenCapability{
						Width:  1920,
						Height: 1080,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("ToInternal(capability_snapshot) error = %v", err)
	}
	if snapshotMsg.CapabilitySnap == nil {
		t.Fatalf("expected capability snapshot message")
	}
	if snapshotMsg.CapabilitySnap.Generation != 1 {
		t.Fatalf("snapshot generation = %d, want 1", snapshotMsg.CapabilitySnap.Generation)
	}

	deltaMsg, err := adapter.ToInternal(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_CapabilityDelta{
			CapabilityDelta: &controlv1.CapabilityDelta{
				DeviceId:   "device-1",
				Generation: 2,
				Reason:     "display_changed",
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "device-1",
					Screen: &capabilitiesv1.ScreenCapability{
						Width:  1280,
						Height: 720,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("ToInternal(capability_delta) error = %v", err)
	}
	if deltaMsg.CapabilityDelta == nil {
		t.Fatalf("expected capability delta message")
	}
	if deltaMsg.CapabilityDelta.Generation != 2 {
		t.Fatalf("delta generation = %d, want 2", deltaMsg.CapabilityDelta.Generation)
	}
	if deltaMsg.CapabilityDelta.Reason != "display_changed" {
		t.Fatalf("delta reason = %q, want display_changed", deltaMsg.CapabilityDelta.Reason)
	}
}

func TestProtoRoundTripCapabilityDeltaPrivacyWithdrawalOmitsMicAndCameraFields(t *testing.T) {
	adapter := GeneratedProtoAdapter{}
	original := &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_CapabilityDelta{
			CapabilityDelta: &controlv1.CapabilityDelta{
				DeviceId:   "device-privacy",
				Generation: 7,
				Reason:     "privacy.toggle",
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "device-privacy",
					Screen: &capabilitiesv1.ScreenCapability{
						Width:  1920,
						Height: 1080,
					},
				},
			},
		},
	}

	encoded, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("proto.Marshal(capability_delta privacy) error = %v", err)
	}

	var decoded controlv1.ConnectRequest
	if err := proto.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("proto.Unmarshal(capability_delta privacy) error = %v", err)
	}

	delta := decoded.GetCapabilityDelta()
	if delta == nil {
		t.Fatalf("decoded capability delta is nil")
	}
	caps := delta.GetCapabilities()
	if caps == nil {
		t.Fatalf("decoded capabilities are nil")
	}
	fields := caps.ProtoReflect().Descriptor().Fields()
	microphoneField := fields.ByName("microphone")
	if microphoneField == nil {
		t.Fatalf("microphone field descriptor not found")
	}
	if caps.ProtoReflect().Has(microphoneField) {
		t.Fatalf("microphone field should be absent for privacy withdrawal encoding")
	}
	cameraField := fields.ByName("camera")
	if cameraField == nil {
		t.Fatalf("camera field descriptor not found")
	}
	if caps.ProtoReflect().Has(cameraField) {
		t.Fatalf("camera field should be absent for privacy withdrawal encoding")
	}

	internalMsg, err := adapter.ToInternal(&decoded)
	if err != nil {
		t.Fatalf("ToInternal(decoded capability_delta privacy) error = %v", err)
	}
	if internalMsg.CapabilityDelta == nil {
		t.Fatalf("expected internal capability delta message")
	}
	if internalMsg.CapabilityDelta.Reason != "privacy.toggle" {
		t.Fatalf("delta reason = %q, want privacy.toggle", internalMsg.CapabilityDelta.Reason)
	}
	if _, ok := internalMsg.CapabilityDelta.Capabilities["microphone.present"]; ok {
		t.Fatalf("microphone.present should be omitted when capability field is absent")
	}
	if _, ok := internalMsg.CapabilityDelta.Capabilities["camera.present"]; ok {
		t.Fatalf("camera.present should be omitted when capability field is absent")
	}
}

func TestGeneratedProtoAdapterFromInternalHelloAndCapabilityAck(t *testing.T) {
	adapter := GeneratedProtoAdapter{}

	env, err := adapter.FromInternal(ServerMessage{
		HelloAck: &HelloResponse{
			ServerID:            "srv-1",
			SessionID:           "device-1:123",
			HeartbeatIntervalMS: 5000,
		},
	})
	if err != nil {
		t.Fatalf("FromInternal(hello_ack) error = %v", err)
	}
	resp, ok := env.(*controlv1.ConnectResponse)
	if !ok {
		t.Fatalf("unexpected proto envelope type %T", env)
	}
	if resp.GetHelloAck() == nil {
		t.Fatalf("expected hello_ack payload")
	}

	env, err = adapter.FromInternal(ServerMessage{
		CapabilityAck: &CapabilityLifecycleAck{
			DeviceID:           "device-1",
			AcceptedGeneration: 2,
			SnapshotApplied:    false,
		},
	})
	if err != nil {
		t.Fatalf("FromInternal(capability_ack) error = %v", err)
	}
	resp, ok = env.(*controlv1.ConnectResponse)
	if !ok {
		t.Fatalf("unexpected proto envelope type %T", env)
	}
	if resp.GetCapabilityAck() == nil {
		t.Fatalf("expected capability_ack payload")
	}
	if resp.GetCapabilityAck().GetAcceptedGeneration() != 2 {
		t.Fatalf("accepted_generation = %d, want 2", resp.GetCapabilityAck().GetAcceptedGeneration())
	}
}

func TestCapabilitiesToDataMapPresenceOnlyForSparseMediaProbes(t *testing.T) {
	got := capabilitiesToDataMap(&capabilitiesv1.DeviceCapabilities{
		DeviceId: "device-1",
		Screen:   &capabilitiesv1.ScreenCapability{},
		Displays: []*capabilitiesv1.DisplayCapability{{
			DisplayId: "display-1",
		}},
		Microphone: &capabilitiesv1.AudioInputCapability{
			Channels: 0,
			Endpoints: []*capabilitiesv1.AudioEndpoint{{
				EndpointId: "mic-1",
			}},
		},
		Speakers: &capabilitiesv1.AudioOutputCapability{
			Channels: 0,
			Endpoints: []*capabilitiesv1.AudioEndpoint{{
				EndpointId: "spk-1",
			}},
		},
		Camera: &capabilitiesv1.CameraCapability{
			Endpoints: []*capabilitiesv1.CameraEndpoint{{
				EndpointId: "cam-1",
			}},
		},
	})

	if got["camera.present"] != "true" {
		t.Fatalf("camera.present = %q, want true", got["camera.present"])
	}
	if got["microphone.present"] != "true" {
		t.Fatalf("microphone.present = %q, want true", got["microphone.present"])
	}
	if got["speakers.present"] != "true" {
		t.Fatalf("speakers.present = %q, want true", got["speakers.present"])
	}
	if _, ok := got["microphone.channels"]; ok {
		t.Fatalf("microphone.channels should be omitted when value is zero")
	}
	if _, ok := got["screen.touch"]; ok {
		t.Fatalf("screen.touch should be omitted when touch capability was not explicitly true")
	}
	if _, ok := got["screen.fullscreen_supported"]; ok {
		t.Fatalf("screen.fullscreen_supported should be omitted when value is default false")
	}
	if _, ok := got["screen.multi_window_supported"]; ok {
		t.Fatalf("screen.multi_window_supported should be omitted when value is default false")
	}
	if _, ok := got["display.0.primary"]; ok {
		t.Fatalf("display.0.primary should be omitted when value is default false")
	}
	if _, ok := got["microphone.endpoint.0.channels"]; ok {
		t.Fatalf("microphone.endpoint.0.channels should be omitted when value is zero")
	}
	if _, ok := got["speakers.endpoint.0.channels"]; ok {
		t.Fatalf("speakers.endpoint.0.channels should be omitted when value is zero")
	}
	if _, ok := got["microphone.endpoint.0.available"]; ok {
		t.Fatalf("microphone.endpoint.0.available should be omitted when value is default false")
	}
	if _, ok := got["speakers.endpoint.0.available"]; ok {
		t.Fatalf("speakers.endpoint.0.available should be omitted when value is default false")
	}
	if _, ok := got["camera.endpoint.0.available"]; ok {
		t.Fatalf("camera.endpoint.0.available should be omitted when value is default false")
	}
	if _, ok := got["camera.front.width"]; ok {
		t.Fatalf("camera.front.width should be omitted when no lens dimensions were provided")
	}
}

func TestCapabilitiesToDataMapIncludesMonitoringTierKeys(t *testing.T) {
	got := capabilitiesToDataMap(&capabilitiesv1.DeviceCapabilities{
		DeviceId: "device-monitor",
		Edge: &capabilitiesv1.EdgeCapability{
			Operators: []string{
				"monitor.tier.foreground_only",
				"monitor.lifecycle.background",
			},
		},
	})
	if got["monitor.support_tier"] != "foreground_only" {
		t.Fatalf("monitor.support_tier = %q, want foreground_only", got["monitor.support_tier"])
	}
	if got["monitor.foreground_only"] != "true" {
		t.Fatalf("monitor.foreground_only = %q, want true", got["monitor.foreground_only"])
	}
	if got["monitor.background_capable"] != "false" {
		t.Fatalf("monitor.background_capable = %q, want false", got["monitor.background_capable"])
	}
	if got["monitor.runtime_state"] != "background" {
		t.Fatalf("monitor.runtime_state = %q, want background", got["monitor.runtime_state"])
	}
}

func TestCapabilitiesToDataMapIncludesEndpointInventory(t *testing.T) {
	got := capabilitiesToDataMap(&capabilitiesv1.DeviceCapabilities{
		DeviceId: "device-endpoints",
		Displays: []*capabilitiesv1.DisplayCapability{
			{
				DisplayId:   "main",
				DisplayName: "Primary",
				Primary:     true,
				Screen: &capabilitiesv1.ScreenCapability{
					Width:       1920,
					Height:      1080,
					Orientation: "landscape",
				},
			},
		},
		Microphone: &capabilitiesv1.AudioInputCapability{
			Endpoints: []*capabilitiesv1.AudioEndpoint{{
				EndpointId:     "mic-1",
				EndpointName:   "Built-in Mic",
				ConnectionType: "built_in",
				Channels:       1,
				Available:      true,
			}},
		},
		Speakers: &capabilitiesv1.AudioOutputCapability{
			Endpoints: []*capabilitiesv1.AudioEndpoint{{
				EndpointId:     "spk-1",
				EndpointName:   "Bluetooth Speaker",
				ConnectionType: "bluetooth",
				Channels:       2,
				Available:      true,
			}},
		},
		Camera: &capabilitiesv1.CameraCapability{
			Endpoints: []*capabilitiesv1.CameraEndpoint{{
				EndpointId:     "cam-1",
				EndpointName:   "USB Camera",
				ConnectionType: "usb",
				Facing:         "front",
				Available:      true,
			}},
		},
		Haptics: &capabilitiesv1.HapticCapability{
			Supported:     true,
			Vibration:     true,
			HapticsEngine: false,
		},
	})

	if got["display.count"] != "1" {
		t.Fatalf("display.count = %q, want 1", got["display.count"])
	}
	if got["display.0.id"] != "main" {
		t.Fatalf("display.0.id = %q, want main", got["display.0.id"])
	}
	if got["microphone.endpoint_count"] != "1" {
		t.Fatalf("microphone.endpoint_count = %q, want 1", got["microphone.endpoint_count"])
	}
	if got["speakers.endpoint_count"] != "1" {
		t.Fatalf("speakers.endpoint_count = %q, want 1", got["speakers.endpoint_count"])
	}
	if got["camera.endpoint_count"] != "1" {
		t.Fatalf("camera.endpoint_count = %q, want 1", got["camera.endpoint_count"])
	}
	if got["haptics.supported"] != "true" {
		t.Fatalf("haptics.supported = %q, want true", got["haptics.supported"])
	}
	if got["haptics.vibration"] != "true" {
		t.Fatalf("haptics.vibration = %q, want true", got["haptics.vibration"])
	}
	if _, ok := got["haptics.engine"]; ok {
		t.Fatalf("haptics.engine should be omitted when value is default false")
	}
}

func TestCapabilitiesToDataMapOmitsDefaultSensorConnectivityAndEdgeListFields(t *testing.T) {
	got := capabilitiesToDataMap(&capabilitiesv1.DeviceCapabilities{
		DeviceId: "device-omissions",
		Sensors: &capabilitiesv1.SensorCapability{
			Accelerometer: false,
			Gyroscope:     false,
			Compass:       false,
			AmbientLight:  false,
			Proximity:     false,
			Gps:           false,
		},
		Connectivity: &capabilitiesv1.ConnectivityCapability{
			BluetoothVersion:   "",
			WifiSignalStrength: false,
			UsbHost:            false,
			UsbPorts:           0,
			Nfc:                false,
		},
		Edge: &capabilitiesv1.EdgeCapability{},
	})

	if _, ok := got["sensors.accelerometer"]; ok {
		t.Fatalf("sensors.accelerometer should be omitted when value is default false")
	}
	if _, ok := got["sensors.gyroscope"]; ok {
		t.Fatalf("sensors.gyroscope should be omitted when value is default false")
	}
	if _, ok := got["sensors.compass"]; ok {
		t.Fatalf("sensors.compass should be omitted when value is default false")
	}
	if _, ok := got["sensors.ambient_light"]; ok {
		t.Fatalf("sensors.ambient_light should be omitted when value is default false")
	}
	if _, ok := got["sensors.proximity"]; ok {
		t.Fatalf("sensors.proximity should be omitted when value is default false")
	}
	if _, ok := got["sensors.gps"]; ok {
		t.Fatalf("sensors.gps should be omitted when value is default false")
	}
	if _, ok := got["connectivity.bluetooth_version"]; ok {
		t.Fatalf("connectivity.bluetooth_version should be omitted when value is empty")
	}
	if _, ok := got["connectivity.wifi_signal_strength"]; ok {
		t.Fatalf("connectivity.wifi_signal_strength should be omitted when value is default false")
	}
	if _, ok := got["connectivity.usb_host"]; ok {
		t.Fatalf("connectivity.usb_host should be omitted when value is default false")
	}
	if _, ok := got["connectivity.usb_ports"]; ok {
		t.Fatalf("connectivity.usb_ports should be omitted when value is zero")
	}
	if _, ok := got["connectivity.nfc"]; ok {
		t.Fatalf("connectivity.nfc should be omitted when value is default false")
	}
	if _, ok := got["edge.runtimes"]; ok {
		t.Fatalf("edge.runtimes should be omitted when list is empty")
	}
	if _, ok := got["edge.operators"]; ok {
		t.Fatalf("edge.operators should be omitted when list is empty")
	}
}

func TestGeneratedProtoAdapterToInternalInput(t *testing.T) {
	adapter := GeneratedProtoAdapter{}
	msg, err := adapter.ToInternal(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Input{
			Input: &iov1.InputEvent{
				DeviceId: "device-2",
				Payload: &iov1.InputEvent_UiAction{
					UiAction: &iov1.UIAction{
						ComponentId: "terminal_input",
						Action:      "submit",
						Value:       "echo hello",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("ToInternal() error = %v", err)
	}
	if msg.Input == nil {
		t.Fatalf("expected input message")
	}
	if msg.Input.DeviceID != "device-2" {
		t.Fatalf("input device_id = %q, want device-2", msg.Input.DeviceID)
	}
	if msg.Input.ComponentID != "terminal_input" || msg.Input.Action != "submit" {
		t.Fatalf("unexpected input mapping: %+v", msg.Input)
	}
	if msg.Input.Value != "echo hello" {
		t.Fatalf("input value = %q, want echo hello", msg.Input.Value)
	}
}

func TestGeneratedProtoAdapterToInternalCommandArguments(t *testing.T) {
	adapter := GeneratedProtoAdapter{}
	msg, err := adapter.ToInternal(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "cmd-args-1",
				DeviceId:  "device-1",
				Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
				Intent:    "photo frame",
				Arguments: map[string]string{
					"device_ids": "device-1,device-2",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("ToInternal(command arguments) error = %v", err)
	}
	if msg.Command == nil {
		t.Fatalf("expected command message")
	}
	if got := msg.Command.Arguments["device_ids"]; got != "device-1,device-2" {
		t.Fatalf("device_ids argument = %q, want device-1,device-2", got)
	}
}

func TestGeneratedProtoAdapterToInternalSensorAndStreamReady(t *testing.T) {
	adapter := GeneratedProtoAdapter{}

	sensorMsg, err := adapter.ToInternal(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Sensor{
			Sensor: &iov1.SensorData{
				DeviceId: "device-3",
				UnixMs:   1713000000000,
				Values: map[string]float64{
					"accelerometer.x": 0.12,
					"accelerometer.y": -0.45,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("ToInternal(sensor) error = %v", err)
	}
	if sensorMsg.Sensor == nil {
		t.Fatalf("expected sensor message")
	}
	if sensorMsg.Sensor.DeviceID != "device-3" {
		t.Fatalf("sensor device_id = %q, want device-3", sensorMsg.Sensor.DeviceID)
	}
	if sensorMsg.Sensor.UnixMS != 1713000000000 {
		t.Fatalf("sensor unix_ms = %d, want 1713000000000", sensorMsg.Sensor.UnixMS)
	}
	if sensorMsg.Sensor.Values["accelerometer.y"] != -0.45 {
		t.Fatalf("sensor value accelerometer.y = %f, want -0.45", sensorMsg.Sensor.Values["accelerometer.y"])
	}

	streamReadyMsg, err := adapter.ToInternal(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_StreamReady{
			StreamReady: &controlv1.StreamReady{
				StreamId: "stream-7",
			},
		},
	})
	if err != nil {
		t.Fatalf("ToInternal(stream_ready) error = %v", err)
	}
	if streamReadyMsg.StreamReady == nil {
		t.Fatalf("expected stream_ready message")
	}
	if streamReadyMsg.StreamReady.StreamID != "stream-7" {
		t.Fatalf("stream_ready stream_id = %q, want stream-7", streamReadyMsg.StreamReady.StreamID)
	}

	webrtcMsg, err := adapter.ToInternal(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_WebrtcSignal{
			WebrtcSignal: &controlv1.WebRTCSignal{
				StreamId:   "stream-7",
				SignalType: "offer",
				Payload:    "{\"sdp\":\"v=0...\"}",
			},
		},
	})
	if err != nil {
		t.Fatalf("ToInternal(webrtc_signal) error = %v", err)
	}
	if webrtcMsg.WebRTCSignal == nil {
		t.Fatalf("expected webrtc_signal message")
	}
	if webrtcMsg.WebRTCSignal.StreamID != "stream-7" {
		t.Fatalf("webrtc_signal stream_id = %q, want stream-7", webrtcMsg.WebRTCSignal.StreamID)
	}
	if webrtcMsg.WebRTCSignal.SignalType != "offer" {
		t.Fatalf("webrtc_signal signal_type = %q, want offer", webrtcMsg.WebRTCSignal.SignalType)
	}
	if webrtcMsg.WebRTCSignal.Payload != "{\"sdp\":\"v=0...\"}" {
		t.Fatalf("webrtc_signal payload = %q, want {\"sdp\":\"v=0...\"}", webrtcMsg.WebRTCSignal.Payload)
	}
}

func TestGeneratedProtoAdapterToInternalObservationArtifactAndFlowStats(t *testing.T) {
	adapter := GeneratedProtoAdapter{}
	observedAt := time.UnixMilli(1713000100000).UTC()

	observationMsg, err := adapter.ToInternal(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_ObservationMessage{
			ObservationMessage: &iov1.ObservationMessage{
				Observation: &iov1.Observation{
					Kind:           "sound.detected",
					Subject:        "kitchen",
					SourceDevice:   &iov1.DeviceRef{DeviceId: "device-1"},
					OccurredUnixMs: observedAt.UnixMilli(),
					Confidence:     0.91,
					Zone:           "kitchen",
					TrackId:        "track-7",
					Attributes: map[string]string{
						"label": "beep",
					},
					Provenance: &iov1.ObservationProvenance{
						FlowId:             "flow-1",
						NodeId:             "analyze-1",
						ExecSite:           "client:device-1",
						ModelId:            "sound-v2",
						CalibrationVersion: "cal-3",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("ToInternal(observation_message) error = %v", err)
	}
	if observationMsg.Observation == nil {
		t.Fatalf("expected observation message")
	}
	if got := observationMsg.Observation.Observation.Kind; got != "sound.detected" {
		t.Fatalf("observation kind = %q, want sound.detected", got)
	}
	if got := observationMsg.Observation.Observation.SourceDevice.DeviceID; got != "device-1" {
		t.Fatalf("observation source_device = %q, want device-1", got)
	}
	if got := observationMsg.Observation.Observation.OccurredAt; !got.Equal(observedAt) {
		t.Fatalf("observation occurred_at = %v, want %v", got, observedAt)
	}
	if got := observationMsg.Observation.Observation.Provenance.FlowID; got != "flow-1" {
		t.Fatalf("observation provenance.flow_id = %q, want flow-1", got)
	}

	artifactMsg, err := adapter.ToInternal(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_ArtifactAvailable{
			ArtifactAvailable: &iov1.ArtifactAvailable{
				Artifact: &iov1.ArtifactRef{
					Id:          "artifact-1",
					Kind:        "audio_clip",
					Source:      &iov1.DeviceRef{DeviceId: "device-1"},
					StartUnixMs: observedAt.UnixMilli(),
					EndUnixMs:   observedAt.Add(3 * time.Second).UnixMilli(),
					Uri:         "file:///tmp/a.wav",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("ToInternal(artifact_available) error = %v", err)
	}
	if artifactMsg.ArtifactReady == nil {
		t.Fatalf("expected artifact_available message")
	}
	if got := artifactMsg.ArtifactReady.Artifact.ID; got != "artifact-1" {
		t.Fatalf("artifact id = %q, want artifact-1", got)
	}
	if got := artifactMsg.ArtifactReady.Artifact.Source.DeviceID; got != "device-1" {
		t.Fatalf("artifact source_device = %q, want device-1", got)
	}

	flowStatsMsg, err := adapter.ToInternal(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_FlowStats{
			FlowStats: &iov1.FlowStats{
				FlowId:        "flow-1",
				CpuPct:        22.5,
				MemMb:         144.75,
				DroppedFrames: 3,
				State:         "healthy",
				Error:         "",
			},
		},
	})
	if err != nil {
		t.Fatalf("ToInternal(flow_stats) error = %v", err)
	}
	if flowStatsMsg.FlowStats == nil {
		t.Fatalf("expected flow_stats message")
	}
	if got := flowStatsMsg.FlowStats.FlowID; got != "flow-1" {
		t.Fatalf("flow_stats flow_id = %q, want flow-1", got)
	}
	if got := flowStatsMsg.FlowStats.CPUPct; got != 22.5 {
		t.Fatalf("flow_stats cpu_pct = %v, want 22.5", got)
	}
}

func TestGeneratedProtoAdapterFromInternalFlowAndArtifactControl(t *testing.T) {
	adapter := GeneratedProtoAdapter{}

	envelope, err := adapter.FromInternal(ServerMessage{
		StartFlow: &StartFlowResponse{
			FlowID: "flow-1",
			Plan: iorouter.FlowPlan{
				Nodes: []iorouter.FlowNode{{
					ID:   "mic",
					Kind: iorouter.NodeSourceMic,
					Args: map[string]string{"device_id": "d1"},
					Exec: iorouter.ExecPreferClient,
				}},
				Edges: []iorouter.FlowEdge{},
			},
		},
	})
	if err != nil {
		t.Fatalf("FromInternal(start_flow) error = %v", err)
	}
	resp, ok := envelope.(*controlv1.ConnectResponse)
	if !ok {
		t.Fatalf("start_flow envelope type = %T, want *controlv1.ConnectResponse", envelope)
	}
	if resp.GetStartFlow() == nil {
		t.Fatalf("expected start_flow payload")
	}
	if got := resp.GetStartFlow().GetFlowId(); got != "flow-1" {
		t.Fatalf("start_flow flow_id = %q, want flow-1", got)
	}
	if got := resp.GetStartFlow().GetPlan().GetNodes()[0].GetKind(); got != string(iorouter.NodeSourceMic) {
		t.Fatalf("start_flow node kind = %q, want %q", got, iorouter.NodeSourceMic)
	}

	envelope, err = adapter.FromInternal(ServerMessage{
		PatchFlow: &PatchFlowResponse{
			FlowID: "flow-1",
			Plan: iorouter.FlowPlan{
				Nodes: []iorouter.FlowNode{{
					ID:   "speaker",
					Kind: iorouter.NodeSinkSpeaker,
					Args: map[string]string{"device_id": "d2"},
					Exec: iorouter.ExecServerOnly,
				}},
				Edges: []iorouter.FlowEdge{},
			},
		},
	})
	if err != nil {
		t.Fatalf("FromInternal(patch_flow) error = %v", err)
	}
	resp, ok = envelope.(*controlv1.ConnectResponse)
	if !ok {
		t.Fatalf("patch_flow envelope type = %T, want *controlv1.ConnectResponse", envelope)
	}
	if resp.GetPatchFlow() == nil {
		t.Fatalf("expected patch_flow payload")
	}
	if got := resp.GetPatchFlow().GetFlowId(); got != "flow-1" {
		t.Fatalf("patch_flow flow_id = %q, want flow-1", got)
	}
	if got := resp.GetPatchFlow().GetPlan().GetNodes()[0].GetExec(); got != string(iorouter.ExecServerOnly) {
		t.Fatalf("patch_flow node exec = %q, want %q", got, iorouter.ExecServerOnly)
	}

	envelope, err = adapter.FromInternal(ServerMessage{
		StopFlow: &StopFlowResponse{FlowID: "flow-1"},
	})
	if err != nil {
		t.Fatalf("FromInternal(stop_flow) error = %v", err)
	}
	resp, ok = envelope.(*controlv1.ConnectResponse)
	if !ok {
		t.Fatalf("stop_flow envelope type = %T, want *controlv1.ConnectResponse", envelope)
	}
	if got := resp.GetStopFlow().GetFlowId(); got != "flow-1" {
		t.Fatalf("stop_flow flow_id = %q, want flow-1", got)
	}

	envelope, err = adapter.FromInternal(ServerMessage{
		RequestArtifact: &RequestArtifactResponse{ArtifactID: "artifact-1"},
	})
	if err != nil {
		t.Fatalf("FromInternal(request_artifact) error = %v", err)
	}
	resp, ok = envelope.(*controlv1.ConnectResponse)
	if !ok {
		t.Fatalf("request_artifact envelope type = %T, want *controlv1.ConnectResponse", envelope)
	}
	if got := resp.GetRequestArtifact().GetArtifactId(); got != "artifact-1" {
		t.Fatalf("request_artifact artifact_id = %q, want artifact-1", got)
	}
}

func TestGeneratedProtoAdapterToInternalBugReport(t *testing.T) {
	adapter := GeneratedProtoAdapter{}
	msg, err := adapter.ToInternal(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_BugReport{
			BugReport: &diagnosticsv1.BugReport{
				ReportId:         "bug-1",
				ReporterDeviceId: "d1",
			},
		},
	})
	if err != nil {
		t.Fatalf("ToInternal(bug_report) error = %v", err)
	}
	if msg.BugReport == nil {
		t.Fatalf("expected bug_report message")
	}
	if msg.BugReport.GetReportId() != "bug-1" {
		t.Fatalf("report_id = %q, want bug-1", msg.BugReport.GetReportId())
	}
}

func TestGeneratedProtoAdapterFromInternalBugReportAck(t *testing.T) {
	adapter := GeneratedProtoAdapter{}
	envelope, err := adapter.FromInternal(ServerMessage{
		BugReportAck: &diagnosticsv1.BugReportAck{
			ReportId:      "bug-2",
			CorrelationId: "bug:bug-2",
			Status:        diagnosticsv1.BugReportStatus_BUG_REPORT_STATUS_FILED,
		},
	})
	if err != nil {
		t.Fatalf("FromInternal(bug_report_ack) error = %v", err)
	}
	resp, ok := envelope.(*controlv1.ConnectResponse)
	if !ok {
		t.Fatalf("response envelope type = %T, want *controlv1.ConnectResponse", envelope)
	}
	if got := resp.GetBugReportAck(); got == nil || got.GetReportId() != "bug-2" {
		t.Fatalf("bug_report_ack = %+v, want report_id bug-2", got)
	}
}

func TestGeneratedProtoAdapterFromInternal(t *testing.T) {
	adapter := GeneratedProtoAdapter{}

	envelope, err := adapter.FromInternal(ServerMessage{
		CommandAck:    "req-1",
		ScenarioStart: "photo_frame",
		Data: map[string]string{
			"a": "1",
			"b": "2",
		},
	})
	if err != nil {
		t.Fatalf("FromInternal() error = %v", err)
	}

	resp, ok := envelope.(*controlv1.ConnectResponse)
	if !ok {
		t.Fatalf("response envelope type = %T, want *controlv1.ConnectResponse", envelope)
	}
	result := resp.GetCommandResult()
	if result == nil {
		t.Fatalf("expected command_result payload")
	}
	if result.GetRequestId() != "req-1" {
		t.Fatalf("request_id = %q, want %q", result.GetRequestId(), "req-1")
	}
	if result.GetData()["a"] != "1" || result.GetData()["b"] != "2" {
		t.Fatalf("unexpected data map: %+v", result.GetData())
	}

	envelope, err = adapter.FromInternal(ServerMessage{
		SetUI: &ui.Descriptor{
			Type: "stack",
			Children: []ui.Descriptor{
				{
					Type: "text",
					Props: map[string]string{
						"value": "hello",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("FromInternal() set_ui error = %v", err)
	}
	resp, ok = envelope.(*controlv1.ConnectResponse)
	if !ok {
		t.Fatalf("set_ui envelope type = %T, want *controlv1.ConnectResponse", envelope)
	}
	if resp.GetSetUi() == nil || resp.GetSetUi().GetRoot() == nil {
		t.Fatalf("expected set_ui root payload")
	}
	if resp.GetSetUi().GetRoot().GetText() != nil {
		t.Fatalf("stack root should not be text widget")
	}
	if len(resp.GetSetUi().GetRoot().GetChildren()) != 1 {
		t.Fatalf("children count = %d, want 1", len(resp.GetSetUi().GetRoot().GetChildren()))
	}
	if got := resp.GetSetUi().GetRoot().GetChildren()[0].GetText().GetValue(); got != "hello" {
		t.Fatalf("text value = %q, want %q", got, "hello")
	}

	envelope, err = adapter.FromInternal(ServerMessage{
		UpdateUI: &UIUpdate{
			ComponentID: "terminal_output",
			Node: ui.Descriptor{
				Type: "text",
				Props: map[string]string{
					"value": "patched",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("FromInternal() update_ui error = %v", err)
	}
	resp, ok = envelope.(*controlv1.ConnectResponse)
	if !ok {
		t.Fatalf("update_ui envelope type = %T, want *controlv1.ConnectResponse", envelope)
	}
	if resp.GetUpdateUi() == nil {
		t.Fatalf("expected update_ui payload")
	}
	if got := resp.GetUpdateUi().GetComponentId(); got != "terminal_output" {
		t.Fatalf("update_ui component_id = %q, want terminal_output", got)
	}
	if got := resp.GetUpdateUi().GetNode().GetText().GetValue(); got != "patched" {
		t.Fatalf("update_ui node text value = %q, want patched", got)
	}

	envelope, err = adapter.FromInternal(ServerMessage{
		StartStream: &StartStreamResponse{
			StreamID:       "stream-1",
			Kind:           "audio",
			SourceDeviceID: "d1",
			TargetDeviceID: "d2",
			Metadata:       map[string]string{"codec": "opus"},
		},
	})
	if err != nil {
		t.Fatalf("FromInternal() start_stream error = %v", err)
	}
	resp, ok = envelope.(*controlv1.ConnectResponse)
	if !ok {
		t.Fatalf("start_stream envelope type = %T, want *controlv1.ConnectResponse", envelope)
	}
	if resp.GetStartStream() == nil {
		t.Fatalf("expected start_stream payload")
	}
	if got := resp.GetStartStream().GetStreamId(); got != "stream-1" {
		t.Fatalf("start_stream stream_id = %q, want stream-1", got)
	}
	if got := resp.GetStartStream().GetKind(); got != "audio" {
		t.Fatalf("start_stream kind = %q, want audio", got)
	}
	if got := resp.GetStartStream().GetMetadata()["codec"]; got != "opus" {
		t.Fatalf("start_stream metadata codec = %q, want opus", got)
	}

	envelope, err = adapter.FromInternal(ServerMessage{
		StopStream: &StopStreamResponse{
			StreamID: "stream-1",
		},
	})
	if err != nil {
		t.Fatalf("FromInternal() stop_stream error = %v", err)
	}
	resp, ok = envelope.(*controlv1.ConnectResponse)
	if !ok {
		t.Fatalf("stop_stream envelope type = %T, want *controlv1.ConnectResponse", envelope)
	}
	if resp.GetStopStream() == nil {
		t.Fatalf("expected stop_stream payload")
	}
	if got := resp.GetStopStream().GetStreamId(); got != "stream-1" {
		t.Fatalf("stop_stream stream_id = %q, want stream-1", got)
	}

	envelope, err = adapter.FromInternal(ServerMessage{
		RouteStream: &RouteStreamResponse{
			StreamID:       "route:d1|d2|audio",
			SourceDeviceID: "d1",
			TargetDeviceID: "d2",
			Kind:           "audio",
		},
	})
	if err != nil {
		t.Fatalf("FromInternal() route_stream error = %v", err)
	}
	resp, ok = envelope.(*controlv1.ConnectResponse)
	if !ok {
		t.Fatalf("route_stream envelope type = %T, want *controlv1.ConnectResponse", envelope)
	}
	if resp.GetRouteStream() == nil {
		t.Fatalf("expected route_stream payload")
	}
	if got := resp.GetRouteStream().GetStreamId(); got != "route:d1|d2|audio" {
		t.Fatalf("route_stream stream_id = %q, want route:d1|d2|audio", got)
	}
	if got := resp.GetRouteStream().GetSourceDeviceId(); got != "d1" {
		t.Fatalf("route_stream source_device_id = %q, want d1", got)
	}
	if got := resp.GetRouteStream().GetTargetDeviceId(); got != "d2" {
		t.Fatalf("route_stream target_device_id = %q, want d2", got)
	}
	if got := resp.GetRouteStream().GetKind(); got != "audio" {
		t.Fatalf("route_stream kind = %q, want audio", got)
	}

	envelope, err = adapter.FromInternal(ServerMessage{
		WebRTCSignal: &WebRTCSignalResponse{
			StreamID:   "stream-1",
			SignalType: "answer",
			Payload:    "{\"sdp\":\"v=0-answer\"}",
		},
	})
	if err != nil {
		t.Fatalf("FromInternal() webrtc_signal error = %v", err)
	}
	resp, ok = envelope.(*controlv1.ConnectResponse)
	if !ok {
		t.Fatalf("webrtc_signal envelope type = %T, want *controlv1.ConnectResponse", envelope)
	}
	if resp.GetWebrtcSignal() == nil {
		t.Fatalf("expected webrtc_signal payload")
	}
	if got := resp.GetWebrtcSignal().GetStreamId(); got != "stream-1" {
		t.Fatalf("webrtc_signal stream_id = %q, want stream-1", got)
	}
	if got := resp.GetWebrtcSignal().GetSignalType(); got != "answer" {
		t.Fatalf("webrtc_signal signal_type = %q, want answer", got)
	}
	if got := resp.GetWebrtcSignal().GetPayload(); got != "{\"sdp\":\"v=0-answer\"}" {
		t.Fatalf("webrtc_signal payload = %q, want {\"sdp\":\"v=0-answer\"}", got)
	}

	envelope, err = adapter.FromInternal(ServerMessage{
		TransitionUI: &UITransition{
			Transition: "fade",
			DurationMS: 250,
		},
	})
	if err != nil {
		t.Fatalf("FromInternal() transition_ui error = %v", err)
	}
	resp, ok = envelope.(*controlv1.ConnectResponse)
	if !ok {
		t.Fatalf("transition_ui envelope type = %T, want *controlv1.ConnectResponse", envelope)
	}
	if resp.GetTransitionUi() == nil {
		t.Fatalf("expected transition_ui payload")
	}
	if got := resp.GetTransitionUi().GetTransition(); got != "fade" {
		t.Fatalf("transition_ui transition = %q, want fade", got)
	}
	if got := resp.GetTransitionUi().GetDurationMs(); got != 250 {
		t.Fatalf("transition_ui duration_ms = %d, want 250", got)
	}
}

func TestGeneratedProtoAdapterFromInternalRegisterAckMetadata(t *testing.T) {
	adapter := GeneratedProtoAdapter{}
	envelope, err := adapter.FromInternal(ServerMessage{
		RegisterAck: &RegisterResponse{
			ServerID: "srv-1",
			Message:  "registered",
			Metadata: map[string]string{
				"photo_frame_asset_base_url": "http://home.local:50052/photo-frame",
			},
		},
	})
	if err != nil {
		t.Fatalf("FromInternal(register ack) error = %v", err)
	}

	resp, ok := envelope.(*controlv1.ConnectResponse)
	if !ok {
		t.Fatalf("response envelope type = %T, want *controlv1.ConnectResponse", envelope)
	}
	ack := resp.GetRegisterAck()
	if ack == nil {
		t.Fatalf("expected register_ack payload")
	}
	if got := ack.GetMetadata()["photo_frame_asset_base_url"]; got != "http://home.local:50052/photo-frame" {
		t.Fatalf("register_ack metadata photo_frame_asset_base_url = %q, want configured value", got)
	}
}
