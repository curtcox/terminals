package replai

import (
	"context"
	"testing"
)

type memorySelections struct {
	state   map[string][2]string
	context map[string][]string
	policy  map[string]string
	thread  map[string]string
	history map[string][]string
}

func (m *memorySelections) GetSelection(sessionID string) (string, string, error) {
	pair := m.state[sessionID]
	return pair[0], pair[1], nil
}

func (m *memorySelections) SetSelection(sessionID, provider, model string) error {
	if m.state == nil {
		m.state = map[string][2]string{}
	}
	m.state[sessionID] = [2]string{provider, model}
	return nil
}

func (m *memorySelections) GetPinnedContext(sessionID string) ([]string, error) {
	if m.context == nil {
		return nil, nil
	}
	return append([]string(nil), m.context[sessionID]...), nil
}

func (m *memorySelections) SetPinnedContext(sessionID string, refs []string) error {
	if m.context == nil {
		m.context = map[string][]string{}
	}
	m.context[sessionID] = append([]string(nil), refs...)
	return nil
}

func (m *memorySelections) GetApprovalPolicy(sessionID string) (string, error) {
	if m.policy == nil {
		return "", nil
	}
	return m.policy[sessionID], nil
}

func (m *memorySelections) SetApprovalPolicy(sessionID, policy string) error {
	if m.policy == nil {
		m.policy = map[string]string{}
	}
	m.policy[sessionID] = policy
	return nil
}

func (m *memorySelections) GetThread(sessionID string) (string, error) {
	if m.thread == nil {
		return "", nil
	}
	return m.thread[sessionID], nil
}

func (m *memorySelections) SetThread(sessionID, thread string) error {
	if m.thread == nil {
		m.thread = map[string]string{}
	}
	m.thread[sessionID] = thread
	return nil
}

func (m *memorySelections) GetHistory(sessionID string) ([]string, error) {
	if m.history == nil {
		return nil, nil
	}
	return append([]string(nil), m.history[sessionID]...), nil
}

func (m *memorySelections) SetHistory(sessionID string, history []string) error {
	if m.history == nil {
		m.history = map[string][]string{}
	}
	m.history[sessionID] = append([]string(nil), history...)
	return nil
}

func TestServiceListsProvidersAndModels(t *testing.T) {
	svc := NewService(nil, Config{
		DefaultProvider: "ollama",
		DefaultModel:    "llama3.1",
		Providers: []ProviderConfig{
			{Name: "ollama", Models: []string{"llama3.1", "qwen3"}, DefaultModel: "llama3.1"},
			{Name: "openrouter", Models: []string{"anthropic/claude-sonnet-4-6"}},
		},
	})
	providers, err := svc.ListProviders(context.Background(), ListProvidersRequest{})
	if err != nil {
		t.Fatalf("ListProviders() error = %v", err)
	}
	if len(providers.Providers) != 2 {
		t.Fatalf("len(providers) = %d, want 2", len(providers.Providers))
	}
	if providers.Providers[0].Name != "ollama" {
		t.Fatalf("providers[0].Name = %q, want ollama", providers.Providers[0].Name)
	}
	models, err := svc.ListModels(context.Background(), ListModelsRequest{Provider: "openrouter"})
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}
	if len(models.Models) != 1 || models.Models[0] != "anthropic/claude-sonnet-4-6" {
		t.Fatalf("models = %+v, want anthropic/claude-sonnet-4-6", models.Models)
	}
}

func TestServiceGetAndSetSelection(t *testing.T) {
	store := &memorySelections{}
	svc := NewService(store, Config{
		DefaultProvider: "ollama",
		DefaultModel:    "llama3.1",
		Providers: []ProviderConfig{
			{Name: "ollama", Models: []string{"llama3.1"}},
			{Name: "openrouter", Models: []string{"anthropic/claude-sonnet-4-6"}},
		},
	})
	initial, err := svc.GetSelection(context.Background(), GetSelectionRequest{SessionID: "repl-1"})
	if err != nil {
		t.Fatalf("GetSelection(default) error = %v", err)
	}
	if initial.Provider != "ollama" || initial.Model != "llama3.1" {
		t.Fatalf("initial selection = %+v, want ollama/llama3.1", initial)
	}
	if _, err := svc.SetSelection(context.Background(), SetSelectionRequest{
		SessionID: "repl-1",
		Provider:  "openrouter",
		Model:     "anthropic/claude-sonnet-4-6",
	}); err != nil {
		t.Fatalf("SetSelection() error = %v", err)
	}
	updated, err := svc.GetSelection(context.Background(), GetSelectionRequest{SessionID: "repl-1"})
	if err != nil {
		t.Fatalf("GetSelection(updated) error = %v", err)
	}
	if updated.Provider != "openrouter" || updated.Model != "anthropic/claude-sonnet-4-6" {
		t.Fatalf("updated selection = %+v, want openrouter/anthropic/claude-sonnet-4-6", updated)
	}
}

func TestServiceRejectsUnknownProviderOrModel(t *testing.T) {
	svc := NewService(nil, Config{
		DefaultProvider: "ollama",
		DefaultModel:    "llama3.1",
		Providers: []ProviderConfig{
			{Name: "ollama", Models: []string{"llama3.1"}},
		},
	})
	if _, err := svc.ListModels(context.Background(), ListModelsRequest{Provider: "missing"}); err == nil {
		t.Fatalf("ListModels(missing provider) expected error")
	}
	if _, err := svc.SetSelection(context.Background(), SetSelectionRequest{
		SessionID: "repl-1",
		Provider:  "ollama",
		Model:     "bad-model",
	}); err == nil {
		t.Fatalf("SetSelection(bad model) expected error")
	}
}

func TestServiceContextAndPolicyLifecycle(t *testing.T) {
	store := &memorySelections{}
	svc := NewService(store, Config{
		DefaultProvider: "ollama",
		DefaultModel:    "llama3.1",
		Providers: []ProviderConfig{
			{Name: "ollama", Models: []string{"llama3.1"}},
		},
	})

	contextResp, err := svc.GetContext(context.Background(), GetContextRequest{SessionID: "repl-1"})
	if err != nil {
		t.Fatalf("GetContext(default) error = %v", err)
	}
	if len(contextResp.Pinned) != 0 {
		t.Fatalf("default pinned context = %#v, want empty", contextResp.Pinned)
	}

	if _, err := svc.PinContext(context.Background(), PinContextRequest{SessionID: "repl-1", Ref: "devices:ls"}); err != nil {
		t.Fatalf("PinContext(devices:ls) error = %v", err)
	}
	if _, err := svc.PinContext(context.Background(), PinContextRequest{SessionID: "repl-1", Ref: "claims:tree"}); err != nil {
		t.Fatalf("PinContext(claims:tree) error = %v", err)
	}
	pinned, err := svc.GetContext(context.Background(), GetContextRequest{SessionID: "repl-1"})
	if err != nil {
		t.Fatalf("GetContext(pinned) error = %v", err)
	}
	if len(pinned.Pinned) != 2 {
		t.Fatalf("len(pinned) = %d, want 2", len(pinned.Pinned))
	}

	if _, err := svc.UnpinContext(context.Background(), UnpinContextRequest{SessionID: "repl-1", Ref: "devices:ls"}); err != nil {
		t.Fatalf("UnpinContext() error = %v", err)
	}
	unpinned, err := svc.GetContext(context.Background(), GetContextRequest{SessionID: "repl-1"})
	if err != nil {
		t.Fatalf("GetContext(unpinned) error = %v", err)
	}
	if len(unpinned.Pinned) != 1 || unpinned.Pinned[0] != "claims:tree" {
		t.Fatalf("unpinned state = %#v, want [claims:tree]", unpinned.Pinned)
	}

	if _, err := svc.ClearContext(context.Background(), ClearContextRequest{SessionID: "repl-1"}); err != nil {
		t.Fatalf("ClearContext() error = %v", err)
	}
	cleared, err := svc.GetContext(context.Background(), GetContextRequest{SessionID: "repl-1"})
	if err != nil {
		t.Fatalf("GetContext(cleared) error = %v", err)
	}
	if len(cleared.Pinned) != 0 {
		t.Fatalf("cleared pinned context = %#v, want empty", cleared.Pinned)
	}

	defaultPolicy, err := svc.GetPolicy(context.Background(), GetPolicyRequest{SessionID: "repl-1"})
	if err != nil {
		t.Fatalf("GetPolicy(default) error = %v", err)
	}
	if defaultPolicy.Policy != ApprovalPolicyPromptMutating {
		t.Fatalf("default policy = %q, want %q", defaultPolicy.Policy, ApprovalPolicyPromptMutating)
	}

	setPolicy, err := svc.SetPolicy(context.Background(), SetPolicyRequest{SessionID: "repl-1", Policy: "auto-readonly"})
	if err != nil {
		t.Fatalf("SetPolicy(auto-readonly) error = %v", err)
	}
	if setPolicy.Policy != ApprovalPolicyPromptMutating {
		t.Fatalf("set policy = %q, want %q", setPolicy.Policy, ApprovalPolicyPromptMutating)
	}

	if _, err := svc.SetPolicy(context.Background(), SetPolicyRequest{SessionID: "repl-1", Policy: "invalid"}); err == nil {
		t.Fatalf("SetPolicy(invalid) expected error")
	}

	if _, err := svc.AddContext(context.Background(), AddContextRequest{SessionID: "repl-1", Ref: ""}); err == nil {
		t.Fatalf("AddContext(empty ref) expected error")
	}

	if err := store.SetThread("repl-1", "thread-42"); err != nil {
		t.Fatalf("store.SetThread() error = %v", err)
	}
	if err := store.SetHistory("repl-1", []string{"user: why suspended?", "assistant: preempted by red_alert"}); err != nil {
		t.Fatalf("store.SetHistory() error = %v", err)
	}
	thread, err := svc.GetThread(context.Background(), GetThreadRequest{SessionID: "repl-1"})
	if err != nil {
		t.Fatalf("GetThread() error = %v", err)
	}
	if thread.Thread != "thread-42" || len(thread.History) != 2 {
		t.Fatalf("thread snapshot = %#v, want thread-42 + 2 history entries", thread)
	}

	reset, err := svc.ResetThread(context.Background(), ResetThreadRequest{SessionID: "repl-1"})
	if err != nil {
		t.Fatalf("ResetThread() error = %v", err)
	}
	if reset.Thread != "" || len(reset.History) != 0 {
		t.Fatalf("reset snapshot = %#v, want empty thread/history", reset)
	}
	postReset, err := svc.GetThread(context.Background(), GetThreadRequest{SessionID: "repl-1"})
	if err != nil {
		t.Fatalf("GetThread(post-reset) error = %v", err)
	}
	if postReset.Thread != "" || len(postReset.History) != 0 {
		t.Fatalf("post-reset thread snapshot = %#v, want empty thread/history", postReset)
	}
}
