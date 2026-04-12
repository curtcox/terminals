package scenario

// RegisterBuiltins installs core scenarios into the engine.
func RegisterBuiltins(engine *Engine) {
	engine.Register(Registration{
		Scenario: PhotoFrameScenario{},
		Priority: PriorityLow,
	})
	engine.Register(Registration{
		Scenario: AlertScenario{},
		Priority: PriorityCritical,
	})
}
