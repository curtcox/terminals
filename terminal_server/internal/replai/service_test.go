package replai

import (
	"context"
	"testing"
)

type memorySelections struct {
	state map[string][2]string
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
