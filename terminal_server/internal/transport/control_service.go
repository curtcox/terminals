package transport

import (
	"context"
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
}

// NewControlService creates a control service.
func NewControlService(serverID string, devices *device.Manager) *ControlService {
	return &ControlService{
		serverID: serverID,
		devices:  devices,
		now:      time.Now,
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

	return RegisterResponse{
		ServerID: s.serverID,
		Message:  "registered",
		Initial:  ui.HelloWorld(registered.DeviceName),
	}, nil
}

// Heartbeat records a liveness pulse.
func (s *ControlService) Heartbeat(_ context.Context, deviceID string) error {
	return s.devices.Heartbeat(deviceID, s.now().UTC())
}
