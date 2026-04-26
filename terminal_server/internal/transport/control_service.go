package transport

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

// RegisterRequest is the transport-neutral register payload.
type RegisterRequest struct {
	DeviceID     string
	DeviceName   string
	DeviceType   string
	Platform     string
	Capabilities map[string]string
}

// RegisterResponse is the transport-neutral register response payload.
type RegisterResponse struct {
	ServerID string
	Message  string
	Metadata map[string]string
	Initial  ui.Descriptor
}

// HelloRequest is the transport-neutral hello handshake payload.
type HelloRequest struct {
	DeviceID      string
	DeviceName    string
	DeviceType    string
	Platform      string
	ClientVersion string
}

// HelloResponse is sent after a successful hello message.
type HelloResponse struct {
	ServerID            string
	SessionID           string
	HeartbeatIntervalMS int64
}

// CapabilityInvalidation describes one resource invalidated by a capability change.
type CapabilityInvalidation struct {
	Resource string
	Reason   string
}

// CapabilityLifecycleAck reports accepted capability generation.
type CapabilityLifecycleAck struct {
	DeviceID           string
	AcceptedGeneration uint64
	SnapshotApplied    bool
	Invalidations      []CapabilityInvalidation
}

// ControlService encapsulates control-plane operations.
type ControlService struct {
	serverID string
	devices  *device.Manager
	metadata map[string]string
	now      func() time.Time
	started  time.Time
}

// NewControlService creates a control service.
func NewControlService(serverID string, devices *device.Manager) *ControlService {
	started := time.Now().UTC()
	return &ControlService{
		serverID: serverID,
		devices:  devices,
		metadata: map[string]string{},
		now:      time.Now,
		started:  started,
	}
}

// Register registers or refreshes a device record and returns initial UI.
func (s *ControlService) Register(_ context.Context, req RegisterRequest) (RegisterResponse, error) {
	registered, err := s.devices.Register(device.Manifest{
		DeviceID:     req.DeviceID,
		DeviceName:   req.DeviceName,
		DeviceType:   req.DeviceType,
		Platform:     req.Platform,
		Capabilities: req.Capabilities,
	})
	if err != nil {
		return RegisterResponse{}, err
	}

	initial := ui.HelloWorld(registered.DeviceName)
	if err := ui.Validate(initial); err != nil {
		return RegisterResponse{}, fmt.Errorf("validate initial ui: %w", err)
	}

	return RegisterResponse{
		ServerID: s.serverID,
		Message:  "registered",
		Metadata: cloneStringMap(s.metadata),
		Initial:  initial,
	}, nil
}

// Hello ensures a device identity record exists before capability sync.
func (s *ControlService) Hello(_ context.Context, req HelloRequest) (HelloResponse, error) {
	_, err := s.devices.Register(device.Manifest{
		DeviceID:     req.DeviceID,
		DeviceName:   req.DeviceName,
		DeviceType:   req.DeviceType,
		Platform:     req.Platform,
		Capabilities: map[string]string{},
	})
	if err != nil {
		return HelloResponse{}, err
	}
	return HelloResponse{
		ServerID:            s.serverID,
		SessionID:           req.DeviceID + ":" + strconv.FormatInt(s.now().UTC().UnixMilli(), 10),
		HeartbeatIntervalMS: 5000,
	}, nil
}

// SetRegisterMetadata configures metadata included with each RegisterAck.
func (s *ControlService) SetRegisterMetadata(metadata map[string]string) {
	s.metadata = cloneStringMap(metadata)
}

// Heartbeat records a liveness pulse.
func (s *ControlService) Heartbeat(_ context.Context, deviceID string) error {
	return s.devices.Heartbeat(deviceID, s.now().UTC())
}

// UpdateCapabilities replaces capabilities for a registered device.
func (s *ControlService) UpdateCapabilities(_ context.Context, deviceID string, caps map[string]string) error {
	return s.devices.UpdateCapabilities(deviceID, caps)
}

// ApplyCapabilitySnapshot applies a full capability baseline.
func (s *ControlService) ApplyCapabilitySnapshot(_ context.Context, deviceID string, generation uint64, caps map[string]string) (CapabilityLifecycleAck, error) {
	if err := s.devices.ApplyCapabilitySnapshot(deviceID, generation, caps); err != nil {
		return CapabilityLifecycleAck{}, err
	}
	return CapabilityLifecycleAck{
		DeviceID:           deviceID,
		AcceptedGeneration: generation,
		SnapshotApplied:    true,
	}, nil
}

// ApplyCapabilityDelta applies a generation-ordered capability update.
func (s *ControlService) ApplyCapabilityDelta(_ context.Context, deviceID string, generation uint64, caps map[string]string) (CapabilityLifecycleAck, error) {
	if err := s.devices.ApplyCapabilityDelta(deviceID, generation, caps); err != nil {
		return CapabilityLifecycleAck{}, err
	}
	return CapabilityLifecycleAck{
		DeviceID:           deviceID,
		AcceptedGeneration: generation,
		SnapshotApplied:    false,
	}, nil
}

// Disconnect marks a device as disconnected when a control stream ends.
func (s *ControlService) Disconnect(_ context.Context, deviceID string) error {
	return s.devices.MarkDisconnected(deviceID)
}

// StatusData returns a stable map representation for system status responses.
func (s *ControlService) StatusData() map[string]string {
	now := s.now().UTC()
	connected := 0
	disconnected := 0
	for _, d := range s.devices.List() {
		if d.State == device.StateConnected {
			connected++
		} else {
			disconnected++
		}
	}

	uptime := now.Sub(s.started)
	if uptime < 0 {
		uptime = 0
	}

	return map[string]string{
		"server_id":            s.serverID,
		"uptime_seconds":       strconv.FormatInt(int64(uptime.Seconds()), 10),
		"devices_total":        strconv.Itoa(connected + disconnected),
		"devices_connected":    strconv.Itoa(connected),
		"devices_disconnected": strconv.Itoa(disconnected),
	}
}

// ReconcileLiveness marks stale devices as disconnected based on a heartbeat timeout.
func (s *ControlService) ReconcileLiveness(timeout time.Duration) int {
	return len(s.ReconcileLivenessDeviceIDs(timeout))
}

// ReconcileLivenessDeviceIDs marks stale devices as disconnected and returns
// the IDs that changed state.
func (s *ControlService) ReconcileLivenessDeviceIDs(timeout time.Duration) []string {
	if timeout < 0 {
		timeout = 0
	}
	cutoff := s.now().UTC().Add(-timeout)
	return s.devices.MarkStaleDisconnectedDevices(cutoff)
}

// SetNowForTest overrides the service clock in tests.
func (s *ControlService) SetNowForTest(now func() time.Time) {
	if now == nil {
		return
	}
	s.now = now
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
