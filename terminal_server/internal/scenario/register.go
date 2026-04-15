package scenario

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
}
