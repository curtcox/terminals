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
	Initial  ui.Descriptor
}

// ControlService encapsulates control-plane operations.
type ControlService struct {
	serverID string
	devices  *device.Manager
	now      func() time.Time
	started  time.Time
}

// NewControlService creates a control service.
func NewControlService(serverID string, devices *device.Manager) *ControlService {
	started := time.Now().UTC()
	return &ControlService{
		serverID: serverID,
		devices:  devices,
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
		Initial:  initial,
	}, nil
}

// Heartbeat records a liveness pulse.
func (s *ControlService) Heartbeat(_ context.Context, deviceID string) error {
	return s.devices.Heartbeat(deviceID, s.now().UTC())
}

// UpdateCapabilities replaces capabilities for a registered device.
func (s *ControlService) UpdateCapabilities(_ context.Context, deviceID string, caps map[string]string) error {
	return s.devices.UpdateCapabilities(deviceID, caps)
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
	if timeout < 0 {
		timeout = 0
	}
	cutoff := s.now().UTC().Add(-timeout)
	return s.devices.MarkStaleDisconnected(cutoff)
}

// SetNowForTest overrides the service clock in tests.
func (s *ControlService) SetNowForTest(now func() time.Time) {
	if now == nil {
		return
	}
	s.now = now
}
