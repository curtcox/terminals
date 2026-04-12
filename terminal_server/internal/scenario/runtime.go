package scenario

import (
	"context"
	"errors"
	"strconv"
	"time"
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

// HandleVoiceText parses spoken text and routes to HandleTrigger.
func (r *Runtime) HandleVoiceText(ctx context.Context, sourceID, spoken string, now time.Time) (string, error) {
	return r.HandleTrigger(ctx, ParseVoiceTrigger(sourceID, spoken, now))
}

// StopTrigger matches and stops a scenario for the selected devices.
func (r *Runtime) StopTrigger(_ context.Context, trigger Trigger) (string, error) {
	reg, ok := r.Engine.Match(trigger)
	if !ok {
		return "", ErrNoMatchingScenario
	}

	deviceIDs := targetDevices(r.Env, trigger)
	if err := r.Engine.Stop(reg.Scenario.Name(), deviceIDs); err != nil {
		return "", err
	}
	return reg.Scenario.Name(), nil
}

// StopVoiceText parses spoken text and routes to StopTrigger.
func (r *Runtime) StopVoiceText(ctx context.Context, sourceID, spoken string, now time.Time) (string, error) {
	return r.StopTrigger(ctx, ParseVoiceTrigger(sourceID, spoken, now))
}

// StatusData returns runtime-focused counters for control-plane system queries.
func (r *Runtime) StatusData() map[string]string {
	activeScenarios := 0
	registeredScenarios := 0
	if r != nil && r.Engine != nil {
		activeScenarios = len(r.Engine.ActiveSnapshot())
		registeredScenarios = len(r.Engine.RegistrySnapshot())
	}

	activeRoutes := 0
	if r != nil && r.Env != nil && r.Env.IO != nil {
		activeRoutes = r.Env.IO.RouteCount()
	}

	return map[string]string{
		"active_scenarios":     strconv.Itoa(activeScenarios),
		"active_routes":        strconv.Itoa(activeRoutes),
		"registered_scenarios": strconv.Itoa(registeredScenarios),
	}
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
