package scenario

import (
	"context"
	"errors"
)

var (
	// ErrNoMatchingScenario indicates no scenario handled the trigger.
	ErrNoMatchingScenario = errors.New("no matching scenario")
)

// Runtime coordinates trigger matching and activation.
type Runtime struct {
	Engine *Engine
	Env    *Environment
}

// NewRuntime creates a runtime with engine and environment.
func NewRuntime(engine *Engine, env *Environment) *Runtime {
	return &Runtime{
		Engine: engine,
		Env:    env,
	}
}

// HandleTrigger matches and activates a scenario for the selected devices.
func (r *Runtime) HandleTrigger(ctx context.Context, trigger Trigger) (string, error) {
	reg, ok := r.Engine.Match(trigger)
	if !ok {
		return "", ErrNoMatchingScenario
	}

	deviceIDs := targetDevices(r.Env, trigger)
	if err := r.Engine.Activate(ctx, r.Env, reg.Scenario.Name(), deviceIDs); err != nil {
		return "", err
	}
	return reg.Scenario.Name(), nil
}

func targetDevices(env *Environment, trigger Trigger) []string {
	if env == nil || env.Devices == nil {
		return nil
	}
	if explicit, ok := trigger.Arguments["device_id"]; ok && explicit != "" {
		return []string{explicit}
	}
	return env.Devices.ListDeviceIDs()
}
