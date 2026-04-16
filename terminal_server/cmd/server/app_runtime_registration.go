package main

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/appruntime"
	"github.com/curtcox/terminals/terminal_server/internal/eventlog"
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
	mu         sync.Mutex
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
	a.mu.Lock()
	defer a.mu.Unlock()
	a.appEnv = &appruntime.Environment{}
	if err := a.activation.Start(ctx, a.appEnv); err != nil {
		return err
	}
	eventlog.Emit(ctx, "appruntime.op.emitted", slog.LevelInfo, "app runtime op emitted",
		slog.String("component", "appruntime"),
		slog.String("op_kind", "activation.start"),
		slog.String("activation_name", a.name),
		slog.String("activation_id", a.activation.ID()),
	)
	return nil
}

func (a *appScenarioActivation) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.activation.Stop(context.Background(), a.appEnv); err != nil {
		return err
	}
	eventlog.Emit(context.Background(), "appruntime.op.emitted", slog.LevelInfo, "app runtime op emitted",
		slog.String("component", "appruntime"),
		slog.String("op_kind", "activation.stop"),
		slog.String("activation_name", a.name),
		slog.String("activation_id", a.activation.ID()),
	)
	return nil
}

func (a *appScenarioActivation) Suspend() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.activation.Suspend(context.Background(), a.appEnv); err != nil {
		return err
	}
	eventlog.Emit(context.Background(), "appruntime.op.emitted", slog.LevelInfo, "app runtime op emitted",
		slog.String("component", "appruntime"),
		slog.String("op_kind", "activation.suspend"),
		slog.String("activation_name", a.name),
		slog.String("activation_id", a.activation.ID()),
	)
	return nil
}

func (a *appScenarioActivation) Resume(ctx context.Context, env *scenario.Environment) error {
	_ = env
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.activation.Resume(ctx, a.appEnv); err != nil {
		return err
	}
	eventlog.Emit(ctx, "appruntime.op.emitted", slog.LevelInfo, "app runtime op emitted",
		slog.String("component", "appruntime"),
		slog.String("op_kind", "activation.resume"),
		slog.String("activation_name", a.name),
		slog.String("activation_id", a.activation.ID()),
	)
	return nil
}

func (a *appScenarioActivation) HandleEvent(ctx context.Context, env *scenario.Environment, event scenario.EventRecord) error {
	_ = env
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.appEnv == nil {
		a.appEnv = &appruntime.Environment{}
	}
	occurredAt := event.OccurredAt.UTC()
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	if err := a.activation.Handle(ctx, a.appEnv, appruntime.Trigger{
		Kind:       strings.TrimSpace(event.Kind),
		Subject:    strings.TrimSpace(event.Subject),
		Attributes: copyStringMap(event.Attributes),
		OccurredAt: occurredAt,
	}); err != nil {
		return err
	}
	eventlog.Emit(ctx, "appruntime.op.emitted", slog.LevelInfo, "app runtime op emitted",
		slog.String("component", "appruntime"),
		slog.String("op_kind", "trigger.handle"),
		slog.String("activation_name", a.name),
		slog.String("activation_id", a.activation.ID()),
		slog.String("trigger_kind", strings.TrimSpace(event.Kind)),
		slog.String("trigger_subject", strings.TrimSpace(event.Subject)),
	)
	return nil
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
