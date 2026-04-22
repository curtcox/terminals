package transport

import "sync/atomic"

// Metrics tracks control-plane message counters for observability.
type Metrics struct {
	registerReceived          atomic.Int64
	capabilityReceived        atomic.Int64
	heartbeatReceived         atomic.Int64
	sensorReceived            atomic.Int64
	streamReadyReceived       atomic.Int64
	webrtcSignalReceived      atomic.Int64
	commandReceived           atomic.Int64
	commandErrors             atomic.Int64
	protocolErrors            atomic.Int64
	dedupeHits                atomic.Int64
	voiceAudioReceived        atomic.Int64
	uiActionUnknownUnscoped   atomic.Int64
	uiActionUnknownActivation atomic.Int64
	uiActionUnknownStaleNode  atomic.Int64
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
		"voice_audio_received":   toString(m.voiceAudioReceived.Load()),
		`ui_action_unknown_component_total{reason="unscoped"}`:           toString(m.uiActionUnknownUnscoped.Load()),
		`ui_action_unknown_component_total{reason="unknown_activation"}`: toString(m.uiActionUnknownActivation.Load()),
		`ui_action_unknown_component_total{reason="stale_node"}`:         toString(m.uiActionUnknownStaleNode.Load()),
	}
}

// IncUnknownUIActionComponent increments the unknown-component counter keyed by reason.
func (m *Metrics) IncUnknownUIActionComponent(reason string) {
	switch reason {
	case "unscoped":
		m.uiActionUnknownUnscoped.Add(1)
	case "unknown_activation":
		m.uiActionUnknownActivation.Add(1)
	case "stale_node":
		m.uiActionUnknownStaleNode.Add(1)
	default:
		// Intentionally ignored: only canonical reasons are exported.
	}
}
