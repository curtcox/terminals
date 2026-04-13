package transport

import "sync/atomic"

// Metrics tracks control-plane message counters for observability.
type Metrics struct {
	registerReceived     atomic.Int64
	capabilityReceived   atomic.Int64
	heartbeatReceived    atomic.Int64
	sensorReceived       atomic.Int64
	streamReadyReceived  atomic.Int64
	webrtcSignalReceived atomic.Int64
	commandReceived      atomic.Int64
	commandErrors        atomic.Int64
	protocolErrors       atomic.Int64
	dedupeHits           atomic.Int64
}

// Snapshot returns a stable map of metric values.
func (m *Metrics) Snapshot() map[string]string {
	return map[string]string{
		"register_received":      toString(m.registerReceived.Load()),
		"capability_received":    toString(m.capabilityReceived.Load()),
		"heartbeat_received":     toString(m.heartbeatReceived.Load()),
		"sensor_received":        toString(m.sensorReceived.Load()),
		"stream_ready_received":  toString(m.streamReadyReceived.Load()),
		"webrtc_signal_received": toString(m.webrtcSignalReceived.Load()),
		"command_received":       toString(m.commandReceived.Load()),
		"command_errors":         toString(m.commandErrors.Load()),
		"protocol_errors":        toString(m.protocolErrors.Load()),
		"dedupe_hits":            toString(m.dedupeHits.Load()),
	}
}
