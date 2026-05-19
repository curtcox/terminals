package scenario

import (
	"context"
	"strings"
)

type activationPlan struct {
	toSuspend map[Scenario]struct{}
}

func planActivation(e *Engine, reg Registration, activation Scenario, deviceIDs []string) (activationPlan, error) {
	plan := activationPlan{toSuspend: make(map[Scenario]struct{})}
	name := strings.TrimSpace(reg.name())
	if name == "" || activation == nil {
		return plan, ErrScenarioNotFound
	}
	for _, deviceID := range deviceIDs {
		if active, ok := e.activeByDev[deviceID]; ok {
			if active.priority > reg.Priority {
				continue
			}
			if active.priority < reg.Priority {
				e.suspendedDev[deviceID] = append(e.suspendedDev[deviceID], active)
				plan.toSuspend[active.scenario] = struct{}{}
			}
		}
		e.activeByDev[deviceID] = activeScenario{
			name:     name,
			priority: reg.Priority,
			scenario: activation,
		}
	}
	return plan, nil
}

func suspendScenarios(plan activationPlan) error {
	for suspended := range plan.toSuspend {
		s, ok := suspended.(Suspendable)
		if !ok {
			continue
		}
		if err := s.Suspend(); err != nil {
			return err
		}
	}
	return nil
}

func startMatchedActivation(ctx context.Context, env *Environment, match MatchResult) error {
	if resultScenario, ok := match.Activation.(ResultScenario); ok {
		result, err := resultScenario.StartResult(ctx, env)
		if err != nil {
			return err
		}
		if err := ExecuteOperations(ctx, env, result.Ops, match.Request.RequestedAt); err != nil {
			return err
		}
		for _, trigger := range result.Emit {
			if env != nil && env.TriggerBus != nil {
				env.TriggerBus.Publish(trigger)
			}
		}
		return nil
	}
	return match.Activation.Start(ctx, env)
}
