package transport

import "sync/atomic"

// Metrics tracks control-plane message counters for observability.
type Metrics struct {
	registerReceived   atomic.Int64
	capabilityReceived atomic.Int64
	heartbeatReceived  atomic.Int64
	commandReceived    atomic.Int64
	commandErrors      atomic.Int64
	protocolErrors     atomic.Int64
}

// Snapshot returns a stable map of metric values.
func (m *Metrics) Snapshot() map[string]string {
	return map[string]string{
		"register_received":   toString(m.registerReceived.Load()),
		"capability_received": toString(m.capabilityReceived.Load()),
		"heartbeat_received":  toString(m.heartbeatReceived.Load()),
		"command_received":    toString(m.commandReceived.Load()),
		"command_errors":      toString(m.commandErrors.Load()),
		"protocol_errors":     toString(m.protocolErrors.Load()),
	}
}
