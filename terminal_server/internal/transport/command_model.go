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
)
