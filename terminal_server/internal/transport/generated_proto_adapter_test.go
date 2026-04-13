package transport

import (
	"testing"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	iov1 "github.com/curtcox/terminals/terminal_server/gen/go/io/v1"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
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
