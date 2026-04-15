package scenario

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Priority defines scenario preemption order.
type Priority int

// Priority constants define scenario preemption order from lowest to highest.
const (
	// PriorityIdle is the lowest possible priority and never preempts others.
	PriorityIdle Priority = iota
	// PriorityLow is for background ambient scenarios.
	PriorityLow
	// PriorityNormal is the default scenario priority.
	PriorityNormal
	// PriorityHigh is for user-facing urgent scenarios.
	PriorityHigh
	// PriorityCritical is for emergency scenarios that preempt all others.
	PriorityCritical
)

// ErrScenarioNotFound indicates an unknown scenario name was referenced.
var ErrScenarioNotFound = errors.New("scenario not found")

// Registration wraps a scenario with runtime metadata.
type Registration struct {
	// Definition is the preferred registration model. It is stateless and
	// produces per-run activation instances.
	Definition ScenarioDefinition
	// Factory is a legacy-friendly path for registering per-activation
	// scenario constructors while the codebase migrates to Definition.
	Factory func() Scenario
	// Scenario is legacy registration support. When used directly, the same
	// scenario instance may be reused across activations.
	Scenario Scenario
	Priority Priority
}

// RegistrationInfo is a stable snapshot of a registered scenario.
type RegistrationInfo struct {
	Name     string
	Priority Priority
}

type activeScenario struct {
	name     string
	priority Priority
	scenario Scenario
}

// MatchResult contains the selected registration and the activation instance
// prepared for this specific request.
type MatchResult struct {
	Registration Registration
	Activation   Scenario
	Request      ActivationRequest
}

// Engine manages registration, matching, and activation with preemption.
type Engine struct {
	mu           sync.Mutex
	registry     map[string]Registration
	activeByDev  map[string]activeScenario
	suspendedDev map[string][]activeScenario
}

// NewEngine builds an empty scenario engine.
func NewEngine() *Engine {
	return &Engine{
		registry:     make(map[string]Registration),
		activeByDev:  make(map[string]activeScenario),
		suspendedDev: make(map[string][]activeScenario),
	}
}

// Register adds or replaces a scenario registration.
func (e *Engine) Register(reg Registration) {
	name := strings.TrimSpace(reg.name())
	if name == "" {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.registry[name] = reg
}

// Match returns the highest-priority matching scenario, if any.
func (e *Engine) Match(trigger Trigger) (Registration, bool) {
	result, ok := e.MatchActivation(ActivationRequest{
		Trigger:     trigger,
		RequestedAt: time.Now().UTC(),
	})
	if !ok {
		return Registration{}, false
	}
	return result.Registration, true
}

// MatchActivation returns the highest-priority matching definition and the
// prepared activation instance for this request.
func (e *Engine) MatchActivation(req ActivationRequest) (MatchResult, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	candidates := make([]MatchResult, 0, len(e.registry))
	for _, reg := range e.registry {
		if strings.TrimSpace(reg.name()) == "" {
			continue
		}
		activation, matched := reg.newActivation(req)
		if !matched || activation == nil {
			continue
		}
		candidates = append(candidates, MatchResult{
			Registration: reg,
			Activation:   activation,
			Request:      req,
		})
	}
	if len(candidates) == 0 {
		return MatchResult{}, false
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Registration.Priority > candidates[j].Registration.Priority
	})
	return candidates[0], true
}

// Activate starts the named scenario across target devices.
// If a lower-priority scenario is active on a device, it is suspended.
func (e *Engine) Activate(ctx context.Context, env *Environment, name string, deviceIDs []string) error {
	e.mu.Lock()
	reg, exists := e.registry[name]
	e.mu.Unlock()
	if !exists {
		return ErrScenarioNotFound
	}

	req := ActivationRequest{
		Trigger: Trigger{
			Kind:     TriggerManual,
			Intent:   name,
			SourceID: firstOrEmpty(deviceIDs),
			Arguments: map[string]string{
				"device_ids": joinCSV(deviceIDs),
			},
		},
		RequestedAt: time.Now().UTC(),
	}
	activation, matched := reg.newActivation(req)
	if !matched || activation == nil {
		return fmt.Errorf("scenario %q does not match activation request", name)
	}
	return e.ActivateMatched(ctx, env, MatchResult{
		Registration: reg,
		Activation:   activation,
		Request:      req,
	}, deviceIDs)
}

// ActivateMatched starts a pre-matched activation across target devices.
func (e *Engine) ActivateMatched(ctx context.Context, env *Environment, match MatchResult, deviceIDs []string) error {
	e.mu.Lock()
	reg := match.Registration
	name := strings.TrimSpace(reg.name())
	if name == "" || match.Activation == nil {
		e.mu.Unlock()
		return ErrScenarioNotFound
	}

	toSuspend := make(map[Scenario]struct{})
	for _, deviceID := range deviceIDs {
		if active, ok := e.activeByDev[deviceID]; ok {
			if active.priority > reg.Priority {
				// A higher-priority scenario already owns this device.
				continue
			}
			if active.priority < reg.Priority {
				e.suspendedDev[deviceID] = append(e.suspendedDev[deviceID], active)
				toSuspend[active.scenario] = struct{}{}
			}
		}
		e.activeByDev[deviceID] = activeScenario{
			name:     name,
			priority: reg.Priority,
			scenario: match.Activation,
		}
	}
	e.mu.Unlock()

	for suspended := range toSuspend {
		s, ok := suspended.(Suspendable)
		if !ok {
			continue
		}
		if err := s.Suspend(); err != nil {
			return err
		}
	}

	return match.Activation.Start(ctx, env)
}

// Stop deactivates the named scenario and resumes any suspended scenario.
func (e *Engine) Stop(ctx context.Context, env *Environment, name string, deviceIDs []string) error {
	e.mu.Lock()
	_, exists := e.registry[name]
	if !exists {
		e.mu.Unlock()
		return ErrScenarioNotFound
	}

	toStop := make(map[Scenario]struct{})
	toResume := make(map[Scenario]struct{})
	for _, deviceID := range deviceIDs {
		active, ok := e.activeByDev[deviceID]
		if !ok || active.name != name {
			continue
		}
		delete(e.activeByDev, deviceID)
		toStop[active.scenario] = struct{}{}

		stack := e.suspendedDev[deviceID]
		if len(stack) == 0 {
			continue
		}
		resumed := stack[len(stack)-1]
		e.suspendedDev[deviceID] = stack[:len(stack)-1]
		e.activeByDev[deviceID] = resumed
		toResume[resumed.scenario] = struct{}{}
	}
	e.mu.Unlock()

	for activation := range toStop {
		if err := activation.Stop(); err != nil {
			return err
		}
	}
	for resumed := range toResume {
		r, ok := resumed.(Resumable)
		if !ok {
			continue
		}
		if err := r.Resume(ctx, env); err != nil {
			return err
		}
	}
	return nil
}

// Active returns the active scenario name for a device.
func (e *Engine) Active(deviceID string) (string, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	active, ok := e.activeByDev[deviceID]
	if !ok {
		return "", false
	}
	return active.name, true
}

// ActiveScenario returns the active scenario instance for a device.
func (e *Engine) ActiveScenario(deviceID string) (Scenario, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	active, ok := e.activeByDev[deviceID]
	if !ok || active.scenario == nil {
		return nil, false
	}
	return active.scenario, true
}

// ActiveSnapshot returns a copy of active scenario names keyed by device id.
func (e *Engine) ActiveSnapshot() map[string]string {
	e.mu.Lock()
	defer e.mu.Unlock()

	out := make(map[string]string, len(e.activeByDev))
	for deviceID, active := range e.activeByDev {
		out[deviceID] = active.name
	}
	return out
}

// SuspendedSnapshot returns a copy of suspended scenario names per device,
// ordered from oldest to newest suspension for each device.
func (e *Engine) SuspendedSnapshot() map[string][]string {
	e.mu.Lock()
	defer e.mu.Unlock()

	out := make(map[string][]string, len(e.suspendedDev))
	for deviceID, stack := range e.suspendedDev {
		names := make([]string, 0, len(stack))
		for _, suspended := range stack {
			names = append(names, suspended.name)
		}
		out[deviceID] = names
	}
	return out
}

// RegistrySnapshot returns all registered scenarios sorted by name.
func (e *Engine) RegistrySnapshot() []RegistrationInfo {
	e.mu.Lock()
	defer e.mu.Unlock()

	out := make([]RegistrationInfo, 0, len(e.registry))
	for _, reg := range e.registry {
		name := strings.TrimSpace(reg.name())
		if name == "" {
			continue
		}
		out = append(out, RegistrationInfo{
			Name:     name,
			Priority: reg.Priority,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func (r Registration) name() string {
	if r.Definition != nil {
		return r.Definition.Name()
	}
	if r.Factory != nil {
		if scenario := r.Factory(); scenario != nil {
			return scenario.Name()
		}
	}
	if r.Scenario != nil {
		return r.Scenario.Name()
	}
	return ""
}

func (r Registration) newActivation(req ActivationRequest) (Scenario, bool) {
	if r.Definition != nil {
		if !r.Definition.Match(req) {
			return nil, false
		}
		activation, err := r.Definition.NewActivation(req)
		if err != nil {
			return nil, false
		}
		return activation, activation != nil
	}
	if r.Factory != nil {
		scenario := r.Factory()
		if scenario == nil {
			return nil, false
		}
		if !scenario.Match(req.Trigger) {
			return nil, false
		}
		return scenario, true
	}
	if r.Scenario != nil && r.Scenario.Match(req.Trigger) {
		return r.Scenario, true
	}
	return nil, false
}

func firstOrEmpty(in []string) string {
	if len(in) == 0 {
		return ""
	}
	return strings.TrimSpace(in[0])
}

func joinCSV(items []string) string {
	if len(items) == 0 {
		return ""
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		out = append(out, item)
	}
	return strings.Join(out, ",")
}
