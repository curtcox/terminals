package scenario

import (
	"context"
	"errors"
	"sort"
	"sync"
)

// Priority defines scenario preemption order.
type Priority int

const (
	PriorityIdle Priority = iota
	PriorityLow
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

var (
	// ErrScenarioNotFound indicates an unknown scenario name was referenced.
	ErrScenarioNotFound = errors.New("scenario not found")
)

// Registration wraps a scenario with runtime metadata.
type Registration struct {
	Scenario Scenario
	Priority Priority
}

type activeScenario struct {
	name     string
	priority Priority
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
	e.mu.Lock()
	defer e.mu.Unlock()
	e.registry[reg.Scenario.Name()] = reg
}

// Match returns the highest-priority matching scenario, if any.
func (e *Engine) Match(trigger Trigger) (Registration, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	candidates := make([]Registration, 0, len(e.registry))
	for _, reg := range e.registry {
		if reg.Scenario.Match(trigger) {
			candidates = append(candidates, reg)
		}
	}
	if len(candidates) == 0 {
		return Registration{}, false
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Priority > candidates[j].Priority
	})
	return candidates[0], true
}

// Activate starts the named scenario across target devices.
// If a lower-priority scenario is active on a device, it is suspended.
func (e *Engine) Activate(ctx context.Context, env *Environment, name string, deviceIDs []string) error {
	e.mu.Lock()
	reg, exists := e.registry[name]
	if !exists {
		e.mu.Unlock()
		return ErrScenarioNotFound
	}

	for _, deviceID := range deviceIDs {
		if active, ok := e.activeByDev[deviceID]; ok {
			if active.priority > reg.Priority {
				// A higher-priority scenario already owns this device.
				continue
			}
			if active.priority < reg.Priority {
				e.suspendedDev[deviceID] = append(e.suspendedDev[deviceID], active)
			}
		}
		e.activeByDev[deviceID] = activeScenario{name: name, priority: reg.Priority}
	}
	e.mu.Unlock()

	return reg.Scenario.Start(ctx, env)
}

// Stop deactivates the named scenario and resumes any suspended scenario.
func (e *Engine) Stop(name string, deviceIDs []string) error {
	e.mu.Lock()
	reg, exists := e.registry[name]
	if !exists {
		e.mu.Unlock()
		return ErrScenarioNotFound
	}

	for _, deviceID := range deviceIDs {
		active, ok := e.activeByDev[deviceID]
		if !ok || active.name != name {
			continue
		}
		delete(e.activeByDev, deviceID)

		stack := e.suspendedDev[deviceID]
		if len(stack) == 0 {
			continue
		}
		resume := stack[len(stack)-1]
		e.suspendedDev[deviceID] = stack[:len(stack)-1]
		e.activeByDev[deviceID] = resume
	}
	e.mu.Unlock()

	return reg.Scenario.Stop()
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
