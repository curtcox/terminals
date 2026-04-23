package scenario

import (
	"fmt"
	"path/filepath"
	"runtime"
	"time"
)

var (
	builtinAffordanceCoverageNow = func() time.Time {
		return time.Now().UTC()
	}
	builtinAffordanceOptOutAllowlistPath = defaultBuiltinAffordanceOptOutAllowlistPath
	// builtinMainLayerAffordanceOptOuts declares main-layer scenarios that
	// intentionally skip withCornerAffordance and therefore require allowlist entries.
	builtinMainLayerAffordanceOptOuts = map[string]struct{}{}
)

func defaultBuiltinAffordanceOptOutAllowlistPath() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "affordance_optouts.yaml"
	}
	return filepath.Join(filepath.Dir(file), "affordance_optouts.yaml")
}

func validateBuiltinAffordanceCoverage(
	registry []RegistrationInfo,
	configuredOptOuts map[string]struct{},
	allowlistPath string,
	now time.Time,
) error {
	allowlist, err := LoadAffordanceOptOutAllowlist(allowlistPath)
	if err != nil {
		return fmt.Errorf("load affordance opt-out allowlist %q: %w", allowlistPath, err)
	}
	if err := ValidateMainLayerAffordanceCoverage(registry, configuredOptOuts, allowlist, now); err != nil {
		return fmt.Errorf("validate builtin main-layer affordance coverage: %w", err)
	}
	return nil
}

// RegisterBuiltins installs core scenarios into the engine.
func RegisterBuiltins(engine *Engine) {
	engine.Register(Registration{
		Factory:  func() Scenario { return &IntercomScenario{} },
		Priority: PriorityHigh,
	})
	engine.Register(Registration{
		Factory:  func() Scenario { return &PhoneCallScenario{} },
		Priority: PriorityHigh,
	})
	engine.Register(Registration{
		Factory:  func() Scenario { return &InternalVideoCallScenario{} },
		Priority: PriorityHigh,
	})
	engine.Register(Registration{
		Factory:  func() Scenario { return &VoiceAssistantScenario{} },
		Priority: PriorityNormal,
	})
	engine.Register(Registration{
		Factory:  func() Scenario { return &AudioMonitorScenario{} },
		Priority: PriorityNormal,
	})
	engine.Register(Registration{
		Factory:  func() Scenario { return &ScheduleMonitorScenario{} },
		Priority: PriorityNormal,
	})
	engine.Register(Registration{
		Factory:  func() Scenario { return &RecentIMUAnomalyScenario{} },
		Priority: PriorityNormal,
	})
	engine.Register(Registration{
		Factory:  func() Scenario { return &SoundIdentificationScenario{} },
		Priority: PriorityNormal,
	})
	engine.Register(Registration{
		Factory:  func() Scenario { return &SoundLocalizationScenario{} },
		Priority: PriorityNormal,
	})
	engine.Register(Registration{
		Factory:  func() Scenario { return &PresenceQueryScenario{} },
		Priority: PriorityNormal,
	})
	engine.Register(Registration{
		Factory:  func() Scenario { return &BluetoothInventoryScenario{} },
		Priority: PriorityNormal,
	})
	engine.Register(Registration{
		Factory:  func() Scenario { return &TerminalVerificationScenario{} },
		Priority: PriorityNormal,
	})
	engine.Register(Registration{
		Factory:  func() Scenario { return PhotoFrameScenario{} },
		Priority: PriorityLow,
	})
	engine.Register(Registration{
		Factory:  func() Scenario { return &MultiWindowScenario{} },
		Priority: PriorityNormal,
	})
	engine.Register(Registration{
		Factory:  func() Scenario { return &TimerReminderScenario{} },
		Priority: PriorityNormal,
	})
	engine.Register(Registration{
		Factory:  func() Scenario { return &TerminalScenario{} },
		Priority: PriorityNormal,
	})
	engine.Register(Registration{
		Factory:  func() Scenario { return &ChatScenario{} },
		Priority: PriorityNormal,
	})
	engine.Register(Registration{
		Factory:  func() Scenario { return &BluetoothPassthroughScenario{} },
		Priority: PriorityNormal,
	})
	engine.Register(Registration{
		Factory:  func() Scenario { return &USBPassthroughScenario{} },
		Priority: PriorityNormal,
	})
	engine.Register(Registration{
		Factory:  func() Scenario { return &PASystemScenario{} },
		Priority: PriorityHigh,
	})
	engine.Register(Registration{
		Factory:  func() Scenario { return &AnnouncementScenario{} },
		Priority: PriorityHigh,
	})
	engine.Register(Registration{
		Factory:  func() Scenario { return AlertScenario{} },
		Priority: PriorityCritical,
	})

	if err := validateBuiltinAffordanceCoverage(
		engine.RegistrySnapshot(),
		builtinMainLayerAffordanceOptOuts,
		builtinAffordanceOptOutAllowlistPath(),
		builtinAffordanceCoverageNow(),
	); err != nil {
		panic(err)
	}
}
