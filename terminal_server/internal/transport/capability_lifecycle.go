package transport

import (
	"context"
	"errors"

	"github.com/curtcox/terminals/terminal_server/internal/device"
)

// CapabilityLifecycle owns hello/register/snapshot/delta handling for the
// control stream. It produces the ack messages and surfaces the before/after
// capability snapshots its caller needs to drive UI replay, route replay, and
// scenario capability-change effects. Those concerns intentionally remain in
// StreamHandler; this collaborator stays focused on the lifecycle itself.
type CapabilityLifecycle struct {
	control *ControlService
}

// NewCapabilityLifecycle binds a lifecycle collaborator to a ControlService.
func NewCapabilityLifecycle(control *ControlService) *CapabilityLifecycle {
	return &CapabilityLifecycle{control: control}
}

// CapabilityResult is the output of snapshot/delta handling. Messages contains
// the ack messages StreamHandler should emit (CapabilityAck for delta;
// CapabilityAck + RegisterAck for snapshot). BeforeCaps/AfterCaps drive
// downstream capability-change effects, and RegisterAck (when present) is the
// same pointer as the RegisterAck in Messages so callers can attach an initial
// UI descriptor when no prior UI exists.
type CapabilityResult struct {
	DeviceID          string
	Messages          []ServerMessage
	BeforeCaps        map[string]string
	AfterCaps         map[string]string
	AfterDeviceName   string
	IsInitialBaseline bool
	HadPriorDevice    bool
	RegisterAck       *RegisterResponse
}

// HandleHello processes a Hello request and returns the corresponding
// HelloAck server message. Errors are returned as-is so the caller can map
// them to error responses and bump protocol metrics.
func (c *CapabilityLifecycle) HandleHello(ctx context.Context, req HelloRequest) ([]ServerMessage, error) {
	resp, err := c.control.Hello(ctx, req)
	if err != nil {
		return nil, err
	}
	return []ServerMessage{{HelloAck: &resp}}, nil
}

// HandleRegister processes a deprecated Register request and returns the
// RegisterAck. The caller is responsible for UI replay and route replay.
func (c *CapabilityLifecycle) HandleRegister(ctx context.Context, req RegisterRequest) (RegisterResponse, error) {
	return c.control.Register(ctx, req)
}

// HandleSnapshot applies a capability snapshot, falling back to an implicit
// Hello when the device record does not yet exist (preserving existing
// compatibility behavior). It returns the CapabilityAck + RegisterAck
// messages along with the before/after capability maps.
func (c *CapabilityLifecycle) HandleSnapshot(ctx context.Context, req CapabilitySnapshotRequest) (CapabilityResult, error) {
	before, _ := c.control.devices.Get(req.DeviceID)
	ack, err := c.control.ApplyCapabilitySnapshot(ctx, req.DeviceID, req.Generation, req.Capabilities)
	if err != nil && errors.Is(err, device.ErrDeviceNotFound) {
		_, helloErr := c.control.Hello(ctx, HelloRequest{
			DeviceID:      req.DeviceID,
			DeviceName:    req.Capabilities["device_name"],
			DeviceType:    req.Capabilities["device_type"],
			Platform:      req.Capabilities["platform"],
			ClientVersion: "",
		})
		if helloErr != nil {
			return CapabilityResult{}, helloErr
		}
		ack, err = c.control.ApplyCapabilitySnapshot(ctx, req.DeviceID, req.Generation, req.Capabilities)
	}
	if err != nil {
		return CapabilityResult{}, err
	}
	after, _ := c.control.devices.Get(req.DeviceID)
	ack.Invalidations = capabilityInvalidations(before.Capabilities, after.Capabilities)
	registerAck := &RegisterResponse{
		ServerID: c.control.serverID,
		Message:  "registered",
		Metadata: cloneStringMap(c.control.metadata),
	}
	messages := []ServerMessage{
		{CapabilityAck: &ack},
		{RegisterAck: registerAck},
	}
	return CapabilityResult{
		DeviceID:          req.DeviceID,
		Messages:          messages,
		BeforeCaps:        before.Capabilities,
		AfterCaps:         after.Capabilities,
		AfterDeviceName:   after.DeviceName,
		IsInitialBaseline: before.Generation == 0 && len(before.Capabilities) == 0,
		HadPriorDevice:    before.Generation != 0,
		RegisterAck:       registerAck,
	}, nil
}

// HandleDelta applies a capability delta and returns the resulting
// CapabilityAck along with the before/after capability maps.
func (c *CapabilityLifecycle) HandleDelta(ctx context.Context, req CapabilityDeltaRequest) (CapabilityResult, error) {
	before, _ := c.control.devices.Get(req.DeviceID)
	ack, err := c.control.ApplyCapabilityDelta(ctx, req.DeviceID, req.Generation, req.Capabilities)
	if err != nil {
		return CapabilityResult{}, err
	}
	after, _ := c.control.devices.Get(req.DeviceID)
	ack.Invalidations = capabilityInvalidations(before.Capabilities, after.Capabilities)
	return CapabilityResult{
		DeviceID:        req.DeviceID,
		Messages:        []ServerMessage{{CapabilityAck: &ack}},
		BeforeCaps:      before.Capabilities,
		AfterCaps:       after.Capabilities,
		AfterDeviceName: after.DeviceName,
		HadPriorDevice:  before.Generation != 0,
	}, nil
}

// HandleUpdateCapabilities processes the deprecated Capability message,
// which has no ack response on success.
func (c *CapabilityLifecycle) HandleUpdateCapabilities(ctx context.Context, req CapabilityUpdateRequest) error {
	return c.control.UpdateCapabilities(ctx, req.DeviceID, req.Capabilities)
}
