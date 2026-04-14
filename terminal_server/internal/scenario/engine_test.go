package scenario

import (
	"context"
	"testing"
)

type stubScenario struct {
	name      string
	match     bool
	started   int
	stopped   int
	suspended int
	resumed   int
	startErr  error
	stopErr   error
}

func (s *stubScenario) Name() string                              { return s.name }
func (s *stubScenario) Match(Trigger) bool                        { return s.match }
func (s *stubScenario) Start(context.Context, *Environment) error { s.started++; return s.startErr }
func (s *stubScenario) Stop() error                               { s.stopped++; return s.stopErr }
func (s *stubScenario) Suspend() error                            { s.suspended++; return nil }
func (s *stubScenario) Resume(context.Context, *Environment) error {
	s.resumed++
	return nil
}

func TestMatchChoosesHighestPriority(t *testing.T) {
	e := NewEngine()
	low := &stubScenario{name: "low", match: true}
	high := &stubScenario{name: "high", match: true}
	e.Register(Registration{Scenario: low, Priority: PriorityLow})
	e.Register(Registration{Scenario: high, Priority: PriorityHigh})

	got, ok := e.Match(Trigger{Kind: TriggerManual})
	if !ok {
		t.Fatalf("Match() expected a match")
	}
	if got.Scenario.Name() != "high" {
		t.Fatalf("Match() = %q, want %q", got.Scenario.Name(), "high")
	}
}

func TestActivatePreemptsLowerPriority(t *testing.T) {
	e := NewEngine()
	normal := &stubScenario{name: "normal", match: true}
	critical := &stubScenario{name: "critical", match: true}
	e.Register(Registration{Scenario: normal, Priority: PriorityNormal})
	e.Register(Registration{Scenario: critical, Priority: PriorityCritical})

	if err := e.Activate(context.Background(), &Environment{}, "normal", []string{"device-1"}); err != nil {
		t.Fatalf("Activate(normal) error = %v", err)
	}
	if err := e.Activate(context.Background(), &Environment{}, "critical", []string{"device-1"}); err != nil {
		t.Fatalf("Activate(critical) error = %v", err)
	}

	active, ok := e.Active("device-1")
	if !ok {
		t.Fatalf("Active() expected device")
	}
	if active != "critical" {
		t.Fatalf("Active() = %q, want %q", active, "critical")
	}
}

func TestStopResumesSuspended(t *testing.T) {
	e := NewEngine()
	normal := &stubScenario{name: "normal", match: true}
	critical := &stubScenario{name: "critical", match: true}
	e.Register(Registration{Scenario: normal, Priority: PriorityNormal})
	e.Register(Registration{Scenario: critical, Priority: PriorityCritical})

	_ = e.Activate(context.Background(), &Environment{}, "normal", []string{"device-1"})
	_ = e.Activate(context.Background(), &Environment{}, "critical", []string{"device-1"})
	if normal.suspended != 1 {
		t.Fatalf("normal.Suspend calls = %d, want 1", normal.suspended)
	}
	if err := e.Stop(context.Background(), &Environment{}, "critical", []string{"device-1"}); err != nil {
		t.Fatalf("Stop(critical) error = %v", err)
	}
	if normal.resumed != 1 {
		t.Fatalf("normal.Resume calls = %d, want 1", normal.resumed)
	}

	active, ok := e.Active("device-1")
	if !ok {
		t.Fatalf("Active() expected resumed scenario")
	}
	if active != "normal" {
		t.Fatalf("Active() = %q, want %q", active, "normal")
	}
}

func TestActiveSnapshot(t *testing.T) {
	e := NewEngine()
	normal := &stubScenario{name: "normal", match: true}
	e.Register(Registration{Scenario: normal, Priority: PriorityNormal})

	if err := e.Activate(context.Background(), &Environment{}, "normal", []string{"device-1"}); err != nil {
		t.Fatalf("Activate() error = %v", err)
	}
	snap := e.ActiveSnapshot()
	if snap["device-1"] != "normal" {
		t.Fatalf("snapshot[device-1] = %q, want normal", snap["device-1"])
	}
	snap["device-1"] = "changed"
	verify := e.ActiveSnapshot()
	if verify["device-1"] != "normal" {
		t.Fatalf("snapshot should be copied")
	}
}

func TestRegistrySnapshot(t *testing.T) {
	e := NewEngine()
	e.Register(Registration{Scenario: &stubScenario{name: "b", match: false}, Priority: PriorityLow})
	e.Register(Registration{Scenario: &stubScenario{name: "a", match: false}, Priority: PriorityHigh})

	reg := e.RegistrySnapshot()
	if len(reg) != 2 {
		t.Fatalf("len(RegistrySnapshot()) = %d, want 2", len(reg))
	}
	if reg[0].Name != "a" || reg[1].Name != "b" {
		t.Fatalf("unexpected registry ordering: %+v", reg)
	}
	if reg[0].Priority != PriorityHigh {
		t.Fatalf("reg[0].Priority = %d, want %d", reg[0].Priority, PriorityHigh)
	}
}
