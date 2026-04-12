package device

import "time"

// CapabilitySet is a normalized map of declared device capabilities.
type CapabilitySet map[string]string

// Manifest represents the minimum registration payload for a device.
type Manifest struct {
	DeviceID     string
	DeviceName   string
	DeviceType   string
	Platform     string
	Capabilities CapabilitySet
}

// State represents server-observed lifecycle state for a device.
type State string

const (
	// StateConnected indicates the device has an active control connection.
	StateConnected State = "connected"
	// StateDisconnected indicates the device has no active control connection.
	StateDisconnected State = "disconnected"
)

// Device is the server-side record for a single registered terminal.
type Device struct {
	DeviceID      string
	DeviceName    string
	DeviceType    string
	Platform      string
	Capabilities  CapabilitySet
	State         State
	RegisteredAt  time.Time
	LastHeartbeat time.Time
}
