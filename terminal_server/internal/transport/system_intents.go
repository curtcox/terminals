package transport

import (
	"fmt"
	"strings"
)

// ParsedSystemIntent is a normalized system intent with optional argument.
type ParsedSystemIntent struct {
	Name string
	Arg  string
}

// SystemHelpIntentsString returns the user-facing system intent list.
func SystemHelpIntentsString() string {
	return strings.Join([]string{
		SystemIntentServerStatus,
		SystemIntentRuntimeStatus,
		SystemIntentScenarioRegistry,
		SystemIntentTransportMetrics,
		SystemIntentListDevices,
		SystemIntentActiveScenarios,
		SystemIntentPendingTimers,
		SystemIntentRecentCommands,
		SystemIntentDeviceStatus + " <device_id>",
		SystemIntentRunDueTimers,
		SystemIntentReconcileLiveness + " <seconds>",
		SystemIntentHelp,
	}, ",")
}

// ParseSystemIntent parses exact and parameterized system intents.
func ParseSystemIntent(raw string) (ParsedSystemIntent, error) {
	intent := strings.TrimSpace(raw)
	if intent == "" {
		return ParsedSystemIntent{}, ErrMissingCommandIntent
	}

	switch intent {
	case SystemIntentHelp,
		SystemIntentServerStatus,
		SystemIntentRuntimeStatus,
		SystemIntentScenarioRegistry,
		SystemIntentTransportMetrics,
		SystemIntentListDevices,
		SystemIntentActiveScenarios,
		SystemIntentPendingTimers,
		SystemIntentRecentCommands,
		SystemIntentRunDueTimers,
		SystemIntentReconcileLiveness:
		return ParsedSystemIntent{Name: intent}, nil
	}

	if strings.HasPrefix(intent, SystemIntentDeviceStatus+" ") {
		arg := strings.TrimSpace(strings.TrimPrefix(intent, SystemIntentDeviceStatus+" "))
		if arg == "" {
			return ParsedSystemIntent{}, fmt.Errorf("device_status requires device id")
		}
		return ParsedSystemIntent{Name: SystemIntentDeviceStatus, Arg: arg}, nil
	}
	if strings.HasPrefix(intent, SystemIntentReconcileLiveness+" ") {
		arg := strings.TrimSpace(strings.TrimPrefix(intent, SystemIntentReconcileLiveness+" "))
		if arg == "" {
			return ParsedSystemIntent{}, fmt.Errorf("invalid reconcile_liveness seconds: %s", arg)
		}
		return ParsedSystemIntent{Name: SystemIntentReconcileLiveness, Arg: arg}, nil
	}

	return ParsedSystemIntent{}, fmt.Errorf("unknown system intent: %s", intent)
}
