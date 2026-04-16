package main

import (
	"context"
	"strings"

	"github.com/curtcox/terminals/terminal_server/internal/appruntime"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
)

func registerAppScenarioDefinitions(engine *scenario.Engine, runtime *appruntime.Runtime) {
	if engine == nil || runtime == nil {
		return
	}
	for _, definition := range runtime.Definitions() {
		engine.Register(scenario.Registration{
			Definition: appScenarioDefinition{definition: definition},
			Priority:   scenario.PriorityNormal,
		})
	}
}

type appScenarioDefinition struct {
	definition appruntime.AppDefinition
}

func (d appScenarioDefinition) Name() string {
	if d.definition == nil {
		return ""
	}
	return strings.TrimSpace(d.definition.Name())
}

func (d appScenarioDefinition) Match(req scenario.ActivationRequest) bool {
	if d.definition == nil {
		return false
	}
	return d.definition.Match(toAppActivationRequest(req))
}

func (d appScenarioDefinition) NewActivation(req scenario.ActivationRequest) (scenario.Scenario, error) {
	if d.definition == nil {
		return nil, nil
	}
	activation, err := d.definition.NewActivation(toAppActivationRequest(req))
	if err != nil || activation == nil {
		return nil, err
	}
	return &appScenarioActivation{
		name:       d.Name(),
		activation: activation,
	}, nil
}

type appScenarioActivation struct {
	name       string
	activation appruntime.AppActivation
	appEnv     *appruntime.Environment
}

func (a *appScenarioActivation) Name() string {
	return a.name
}

func (a *appScenarioActivation) Match(trigger scenario.Trigger) bool {
	_ = trigger
	return false
}

func (a *appScenarioActivation) Start(ctx context.Context, env *scenario.Environment) error {
	_ = env
	a.appEnv = &appruntime.Environment{}
	return a.activation.Start(ctx, a.appEnv)
}

func (a *appScenarioActivation) Stop() error {
	return a.activation.Stop(context.Background(), a.appEnv)
}

func (a *appScenarioActivation) Suspend() error {
	return a.activation.Suspend(context.Background(), a.appEnv)
}

func (a *appScenarioActivation) Resume(ctx context.Context, env *scenario.Environment) error {
	_ = env
	return a.activation.Resume(ctx, a.appEnv)
}

func toAppActivationRequest(req scenario.ActivationRequest) appruntime.ActivationRequest {
	deviceID := strings.TrimSpace(req.Trigger.SourceID)
	if deviceID == "" && len(req.Targets) > 0 {
		deviceID = strings.TrimSpace(req.Targets[0].DeviceID)
	}
	return appruntime.ActivationRequest{
		DeviceID: deviceID,
		Intent:   strings.TrimSpace(req.Trigger.Intent),
		Payload:  copyStringMap(req.Trigger.Arguments),
	}
}
