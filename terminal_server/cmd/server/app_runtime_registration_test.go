package main

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/appruntime"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
)

type mockAppActivation struct {
	lastTrigger appruntime.Trigger
}

type concurrentMockAppActivation struct {
	current int32
	maxSeen int32
}

func (m *mockAppActivation) ID() string { return "mock-id" }

func (m *mockAppActivation) DefinitionName() string { return "mock-def" }

func (m *mockAppActivation) Start(ctx context.Context, env *appruntime.Environment) error {
	_ = ctx
	_ = env
	return nil
}

func (m *mockAppActivation) Handle(ctx context.Context, env *appruntime.Environment, trigger appruntime.Trigger) error {
	_ = ctx
	_ = env
	m.lastTrigger = trigger
	return nil
}

func (m *mockAppActivation) Stop(ctx context.Context, env *appruntime.Environment) error {
	_ = ctx
	_ = env
	return nil
}

func (m *mockAppActivation) Suspend(ctx context.Context, env *appruntime.Environment) error {
	_ = ctx
	_ = env
	return nil
}

func (m *mockAppActivation) Resume(ctx context.Context, env *appruntime.Environment) error {
	_ = ctx
	_ = env
	return nil
}

func (m *concurrentMockAppActivation) ID() string { return "concurrent-id" }

func (m *concurrentMockAppActivation) DefinitionName() string { return "concurrent-def" }

func (m *concurrentMockAppActivation) Start(ctx context.Context, env *appruntime.Environment) error {
	_ = ctx
	_ = env
	return nil
}

func (m *concurrentMockAppActivation) Handle(ctx context.Context, env *appruntime.Environment, trigger appruntime.Trigger) error {
	_ = ctx
	_ = env
	_ = trigger
	now := atomic.AddInt32(&m.current, 1)
	for {
		seen := atomic.LoadInt32(&m.maxSeen)
		if now <= seen || atomic.CompareAndSwapInt32(&m.maxSeen, seen, now) {
			break
		}
	}
	time.Sleep(20 * time.Millisecond)
	atomic.AddInt32(&m.current, -1)
	return nil
}

func (m *concurrentMockAppActivation) Stop(ctx context.Context, env *appruntime.Environment) error {
	_ = ctx
	_ = env
	return nil
}

func (m *concurrentMockAppActivation) Suspend(ctx context.Context, env *appruntime.Environment) error {
	_ = ctx
	_ = env
	return nil
}

func (m *concurrentMockAppActivation) Resume(ctx context.Context, env *appruntime.Environment) error {
	_ = ctx
	_ = env
	return nil
}

func TestRegisterAppScenarioDefinitions(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "kitchen_watch")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(app) error = %v", err)
	}
	manifest := "name = \"kitchen_watch\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\nexports = [\"watch\"]\n"
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}

	runtime := appruntime.NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	engine := scenario.NewEngine()
	registerAppScenarioDefinitions(engine, runtime)

	snapshot := engine.RegistrySnapshot()
	if len(snapshot) != 1 {
		t.Fatalf("len(RegistrySnapshot()) = %d, want 1", len(snapshot))
	}
	if snapshot[0].Name != "app.kitchen_watch.watch" {
		t.Fatalf("registered name = %q, want app.kitchen_watch.watch", snapshot[0].Name)
	}

	match, ok := engine.MatchActivation(scenario.ActivationRequest{
		Trigger: scenario.Trigger{
			Kind:     scenario.TriggerManual,
			SourceID: "kitchen-1",
			Intent:   "watch",
		},
		RequestedAt: time.Now().UTC(),
	})
	if !ok {
		t.Fatalf("MatchActivation() ok = false, want true")
	}
	if match.Registration.Definition == nil || match.Registration.Definition.Name() != "app.kitchen_watch.watch" {
		t.Fatalf("matched registration definition = %+v, want app.kitchen_watch.watch", match.Registration.Definition)
	}

	activation, ok := match.Activation.(*appScenarioActivation)
	if !ok {
		t.Fatalf("activation type = %T, want *appScenarioActivation", match.Activation)
	}
	if got := activation.activation.ID(); got != "app:kitchen_watch:watch:kitchen-1:r1" {
		t.Fatalf("activation id = %q, want app:kitchen_watch:watch:kitchen-1:r1", got)
	}
}

func TestAppScenarioActivationHandleEventForwardsToAppActivation(t *testing.T) {
	mock := &mockAppActivation{}
	activation := &appScenarioActivation{
		name:       "app.kitchen_watch.watch",
		activation: mock,
	}
	occurredAt := time.Date(2026, 4, 15, 20, 30, 0, 0, time.UTC)
	err := activation.HandleEvent(context.Background(), nil, scenario.EventRecord{
		Kind:       "sound.classified",
		Subject:    "kitchen-1",
		Attributes: map[string]string{"label": "dishwasher_done"},
		OccurredAt: occurredAt,
	})
	if err != nil {
		t.Fatalf("HandleEvent() error = %v", err)
	}
	if mock.lastTrigger.Kind != "sound.classified" {
		t.Fatalf("trigger kind = %q, want sound.classified", mock.lastTrigger.Kind)
	}
	if mock.lastTrigger.Subject != "kitchen-1" {
		t.Fatalf("trigger subject = %q, want kitchen-1", mock.lastTrigger.Subject)
	}
	if mock.lastTrigger.Attributes["label"] != "dishwasher_done" {
		t.Fatalf("trigger attribute label = %q, want dishwasher_done", mock.lastTrigger.Attributes["label"])
	}
	if !mock.lastTrigger.OccurredAt.Equal(occurredAt) {
		t.Fatalf("trigger occurred_at = %s, want %s", mock.lastTrigger.OccurredAt, occurredAt)
	}
}

func TestAppScenarioActivationHandleEventSerializesConcurrentCalls(t *testing.T) {
	mock := &concurrentMockAppActivation{}
	activation := &appScenarioActivation{
		name:       "app.kitchen_watch.watch",
		activation: mock,
	}
	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			_ = activation.HandleEvent(context.Background(), nil, scenario.EventRecord{
				Kind:       "sound.classified",
				Subject:    "kitchen-1",
				OccurredAt: time.Now().UTC(),
			})
		}()
	}
	wg.Wait()

	if got := atomic.LoadInt32(&mock.maxSeen); got != 1 {
		t.Fatalf("max concurrent app Handle calls = %d, want 1", got)
	}
}
