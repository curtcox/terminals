package transport

import (
	"context"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/device"
)

func TestHandleMessageWebRTCSignalProducesRelayToPeer(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		SessionDeviceID: "device-1",
		WebRTCSignal: &WebRTCSignalRequest{
			StreamID:   "route:device-1|device-2|audio",
			SignalType: "offer",
			Payload:    "{\"sdp\":\"v=0-offer\"}",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(webrtc_signal) error = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if out[0].RelayToDeviceID != "device-2" {
		t.Fatalf("RelayToDeviceID = %q, want device-2", out[0].RelayToDeviceID)
	}
	if out[0].WebRTCSignal == nil {
		t.Fatalf("expected webrtc signal payload")
	}
	if out[0].WebRTCSignal.StreamID != "route:device-1|device-2|audio" {
		t.Fatalf("stream_id = %q, want route stream id", out[0].WebRTCSignal.StreamID)
	}
	if out[0].WebRTCSignal.SignalType != "offer" {
		t.Fatalf("signal_type = %q, want offer", out[0].WebRTCSignal.SignalType)
	}
	if out[0].WebRTCSignal.Payload != "{\"sdp\":\"v=0-offer\"}" {
		t.Fatalf("payload = %q, want offer payload", out[0].WebRTCSignal.Payload)
	}

	replyOut, replyErr := handler.HandleMessage(context.Background(), ClientMessage{
		SessionDeviceID: "device-2",
		WebRTCSignal: &WebRTCSignalRequest{
			StreamID:   "route:device-1|device-2|audio",
			SignalType: "answer",
			Payload:    "{\"sdp\":\"v=0-answer\"}",
		},
	})
	if replyErr != nil {
		t.Fatalf("HandleMessage(webrtc answer) error = %v", replyErr)
	}
	if len(replyOut) != 1 {
		t.Fatalf("len(replyOut) = %d, want 1", len(replyOut))
	}
	if replyOut[0].RelayToDeviceID != "device-1" {
		t.Fatalf("reply RelayToDeviceID = %q, want device-1", replyOut[0].RelayToDeviceID)
	}
}

type fakeWebRTCSignalEngine struct {
	responses   []WebRTCSignalEngineResponse
	err         error
	lastRequest WebRTCSignalEngineRequest
	removeCalls []string
}

func (f *fakeWebRTCSignalEngine) HandleSignal(_ context.Context, req WebRTCSignalEngineRequest) ([]WebRTCSignalEngineResponse, error) {
	f.lastRequest = req
	if f.err != nil {
		return nil, f.err
	}
	out := append([]WebRTCSignalEngineResponse(nil), f.responses...)
	return out, nil
}

func (f *fakeWebRTCSignalEngine) RemoveStream(streamID string) {
	f.removeCalls = append(f.removeCalls, streamID)
}

func TestHandleMessageWebRTCSignalUsesServerManagedEngine(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)
	engine := &fakeWebRTCSignalEngine{
		responses: []WebRTCSignalEngineResponse{
			{
				TargetDeviceID: "device-1",
				Signal: WebRTCSignalResponse{
					StreamID:   "route:device-1|device-2|audio",
					SignalType: "answer",
					Payload:    "{\"sdp\":\"v=0-answer\"}",
				},
			},
		},
	}
	handler.SetWebRTCSignalEngine(engine)
	handler.registerMediaStream(StartStreamResponse{
		StreamID:       "route:device-1|device-2|audio",
		Kind:           "audio",
		SourceDeviceID: "device-1",
		TargetDeviceID: "device-2",
		Metadata: map[string]string{
			"webrtc_mode": "server_managed",
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		SessionDeviceID: "device-1",
		WebRTCSignal: &WebRTCSignalRequest{
			StreamID:   "route:device-1|device-2|audio",
			SignalType: "offer",
			Payload:    "{\"sdp\":\"v=0-offer\"}",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(server-managed webrtc signal) error = %v", err)
	}
	if engine.lastRequest.DeviceID != "device-1" {
		t.Fatalf("engine request device = %q, want device-1", engine.lastRequest.DeviceID)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if out[0].RelayToDeviceID != "" {
		t.Fatalf("RelayToDeviceID = %q, want empty (back to session device)", out[0].RelayToDeviceID)
	}
	if out[0].WebRTCSignal == nil || out[0].WebRTCSignal.SignalType != "answer" {
		t.Fatalf("expected answer signal from engine, got %+v", out[0].WebRTCSignal)
	}
}

func TestHandleMessageWebRTCSignalServerManagedFallsBackToRelayOnEngineError(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)
	engine := &fakeWebRTCSignalEngine{err: ErrInvalidClientMessage}
	handler.SetWebRTCSignalEngine(engine)
	handler.registerMediaStream(StartStreamResponse{
		StreamID:       "route:device-1|device-2|audio",
		Kind:           "audio",
		SourceDeviceID: "device-1",
		TargetDeviceID: "device-2",
		Metadata: map[string]string{
			"webrtc_mode": "server_managed",
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		SessionDeviceID: "device-1",
		WebRTCSignal: &WebRTCSignalRequest{
			StreamID:   "route:device-1|device-2|audio",
			SignalType: "offer",
			Payload:    "{\"sdp\":\"v=0-offer\"}",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(server-managed fallback) error = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if out[0].RelayToDeviceID != "device-2" {
		t.Fatalf("RelayToDeviceID = %q, want device-2 fallback relay", out[0].RelayToDeviceID)
	}
	if out[0].WebRTCSignal == nil || out[0].WebRTCSignal.SignalType != "offer" {
		t.Fatalf("expected fallback offer relay, got %+v", out[0].WebRTCSignal)
	}
}

func TestUnregisterMediaStreamRemovesServerManagedWebRTCStream(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)
	engine := &fakeWebRTCSignalEngine{}
	handler.SetWebRTCSignalEngine(engine)

	streamID := "route:device-1|device-2|audio"
	handler.registerMediaStream(StartStreamResponse{
		StreamID:       streamID,
		Kind:           "audio",
		SourceDeviceID: "device-1",
		TargetDeviceID: "device-2",
		Metadata: map[string]string{
			"webrtc_mode": "server_managed",
		},
	})
	handler.unregisterMediaStream(streamID)

	if len(engine.removeCalls) != 1 || engine.removeCalls[0] != streamID {
		t.Fatalf("removeCalls = %+v, want [%s]", engine.removeCalls, streamID)
	}
}
