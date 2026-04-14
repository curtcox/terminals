// Package transport implements control-plane message handling and adaptation.
package transport

// Command kinds.
const (
	CommandKindVoice  = "voice"
	CommandKindManual = "manual"
	CommandKindSystem = "system"
)

// Command actions.
const (
	CommandActionStart = "start"
	CommandActionStop  = "stop"
)

// System intents.
const (
	SystemIntentHelp              = "system_help"
	SystemIntentServerStatus      = "server_status"
	SystemIntentRuntimeStatus     = "runtime_status"
	SystemIntentScenarioRegistry  = "scenario_registry"
	SystemIntentTransportMetrics  = "transport_metrics"
	SystemIntentListDevices       = "list_devices"
	SystemIntentActiveScenarios   = "active_scenarios"
	SystemIntentPendingTimers     = "pending_timers"
	SystemIntentRecentCommands    = "recent_commands"
	SystemIntentRunDueTimers      = "run_due_timers"
	SystemIntentReconcileLiveness = "reconcile_liveness"
	SystemIntentDeviceStatus      = "device_status"
	SystemIntentTerminalRefresh   = "terminal_refresh"
	SystemIntentRecordingEvents   = "recording_events"
	SystemIntentListPlaybackFiles = "list_playback_artifacts"
)

// Manual command intents that are handled by transport scaffolding.
const (
	ManualIntentPlaybackMetadata = "playback_metadata"
)
