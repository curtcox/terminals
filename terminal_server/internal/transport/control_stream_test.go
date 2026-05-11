package transport

import (
	"context"
	"encoding/binary"
	"sync"
	"testing"
	"time"

	diagnosticsv1 "github.com/curtcox/terminals/terminal_server/gen/go/diagnostics/v1"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/recording"
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
