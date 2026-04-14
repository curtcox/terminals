package scenario

// RegisterBuiltins installs core scenarios into the engine.
func RegisterBuiltins(engine *Engine) {
	engine.Register(Registration{
		Scenario: &IntercomScenario{},
		Priority: PriorityHigh,
	})
	engine.Register(Registration{
		Scenario: &PhoneCallScenario{},
		Priority: PriorityHigh,
	})
	engine.Register(Registration{
		Scenario: &InternalVideoCallScenario{},
		Priority: PriorityHigh,
	})
	engine.Register(Registration{
		Scenario: &VoiceAssistantScenario{},
		Priority: PriorityNormal,
	})
	engine.Register(Registration{
		Scenario: &AudioMonitorScenario{},
		Priority: PriorityNormal,
	})
	engine.Register(Registration{
		Scenario: &ScheduleMonitorScenario{},
		Priority: PriorityNormal,
	})
	engine.Register(Registration{
		Scenario: PhotoFrameScenario{},
		Priority: PriorityLow,
	})
	engine.Register(Registration{
		Scenario: &MultiWindowScenario{},
		Priority: PriorityNormal,
	})
	engine.Register(Registration{
		Scenario: &TimerReminderScenario{},
		Priority: PriorityNormal,
	})
	engine.Register(Registration{
		Scenario: &TerminalScenario{},
		Priority: PriorityNormal,
	})
	engine.Register(Registration{
		Scenario: &PASystemScenario{},
		Priority: PriorityHigh,
	})
	engine.Register(Registration{
		Scenario: &AnnouncementScenario{},
		Priority: PriorityHigh,
	})
	engine.Register(Registration{
		Scenario: AlertScenario{},
		Priority: PriorityCritical,
	})
}
