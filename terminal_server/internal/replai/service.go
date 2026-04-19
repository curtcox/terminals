package replai

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	// ErrMissingSessionID indicates a request did not include a session id.
	ErrMissingSessionID = errors.New("missing session id")
	// ErrMissingProvider indicates a request did not include a provider name.
	ErrMissingProvider = errors.New("missing provider")
	// ErrMissingModel indicates a request did not include a model name.
	ErrMissingModel = errors.New("missing model")
	// ErrProviderNotFound indicates a provider is unknown.
	ErrProviderNotFound = errors.New("provider not found")
)

// SessionSelectionStore persists sticky provider/model selection per session.
type SessionSelectionStore interface {
	GetSelection(sessionID string) (provider, model string, err error)
	SetSelection(sessionID, provider, model string) error
}

// ProviderConfig declares one configured AI provider.
type ProviderConfig struct {
	Name         string
	Models       []string
	DefaultModel string
}

// Config configures the AI selection service.
type Config struct {
	DefaultProvider string
	DefaultModel    string
	Providers       []ProviderConfig
}

// Provider is a runtime provider summary.
type Provider struct {
	Name         string   `json:"name"`
	DefaultModel string   `json:"default_model,omitempty"`
	Models       []string `json:"models,omitempty"`
}

// ListProvidersRequest lists configured providers.
type ListProvidersRequest struct{}

// ListProvidersResponse returns configured providers.
type ListProvidersResponse struct {
	Providers []Provider `json:"providers"`
}

// ListModelsRequest lists models for one provider.
type ListModelsRequest struct {
	Provider string
}

// ListModelsResponse returns provider model names.
type ListModelsResponse struct {
	Provider string   `json:"provider"`
	Models   []string `json:"models"`
}

// GetSelectionRequest returns sticky selection for one session.
type GetSelectionRequest struct {
	SessionID string
}

// GetSelectionResponse reports current provider/model for a session.
type GetSelectionResponse struct {
	SessionID string `json:"session_id"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
}

// SetSelectionRequest updates sticky selection for one session.
type SetSelectionRequest struct {
	SessionID string
	Provider  string
	Model     string
}

// SetSelectionResponse reports updated provider/model for a session.
type SetSelectionResponse struct {
	SessionID string `json:"session_id"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
}

// Service provides typed AI selection APIs used by REPL commands.
type Service struct {
	sessions        SessionSelectionStore
	providersByName map[string]Provider
	providerOrder   []string
	defaultProvider string
	defaultModel    string
}

// NewService constructs a typed AI selection service.
func NewService(sessions SessionSelectionStore, cfg Config) *Service {
	svc := &Service{
		sessions:        sessions,
		providersByName: map[string]Provider{},
		providerOrder:   []string{},
		defaultProvider: strings.TrimSpace(cfg.DefaultProvider),
		defaultModel:    strings.TrimSpace(cfg.DefaultModel),
	}
	for _, provider := range cfg.Providers {
		name := strings.TrimSpace(strings.ToLower(provider.Name))
		if name == "" {
			continue
		}
		models := dedupeSorted(provider.Models)
		p := Provider{
			Name:         name,
			DefaultModel: strings.TrimSpace(provider.DefaultModel),
			Models:       models,
		}
		if p.DefaultModel == "" && len(models) > 0 {
			p.DefaultModel = models[0]
		}
		svc.providersByName[name] = p
	}
	for name := range svc.providersByName {
		svc.providerOrder = append(svc.providerOrder, name)
	}
	sort.Strings(svc.providerOrder)
	if svc.defaultProvider == "" && len(svc.providerOrder) > 0 {
		svc.defaultProvider = svc.providerOrder[0]
	}
	if svc.defaultModel == "" {
		if provider, ok := svc.providersByName[svc.defaultProvider]; ok {
			svc.defaultModel = provider.DefaultModel
		}
	}
	return svc
}

// ListProviders returns configured providers.
func (s *Service) ListProviders(context.Context, ListProvidersRequest) (*ListProvidersResponse, error) {
	providers := make([]Provider, 0, len(s.providerOrder))
	for _, name := range s.providerOrder {
		p := s.providersByName[name]
		providers = append(providers, Provider{
			Name:         p.Name,
			DefaultModel: p.DefaultModel,
			Models:       append([]string(nil), p.Models...),
		})
	}
	return &ListProvidersResponse{Providers: providers}, nil
}

// ListModels returns models for a configured provider.
func (s *Service) ListModels(_ context.Context, req ListModelsRequest) (*ListModelsResponse, error) {
	provider, _, err := s.resolveProviderAndModel(req.Provider, "")
	if err != nil {
		return nil, err
	}
	p := s.providersByName[provider]
	return &ListModelsResponse{
		Provider: provider,
		Models:   append([]string(nil), p.Models...),
	}, nil
}

// GetSelection returns the sticky provider/model for a session.
func (s *Service) GetSelection(_ context.Context, req GetSelectionRequest) (*GetSelectionResponse, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return nil, ErrMissingSessionID
	}
	if s.sessions == nil {
		provider, model, err := s.resolveProviderAndModel("", "")
		if err != nil {
			return nil, err
		}
		return &GetSelectionResponse{SessionID: sessionID, Provider: provider, Model: model}, nil
	}
	provider, model, err := s.sessions.GetSelection(sessionID)
	if err != nil {
		return nil, err
	}
	provider, model, err = s.resolveProviderAndModel(provider, model)
	if err != nil {
		return nil, err
	}
	return &GetSelectionResponse{SessionID: sessionID, Provider: provider, Model: model}, nil
}

// SetSelection validates and updates sticky selection for a session.
func (s *Service) SetSelection(_ context.Context, req SetSelectionRequest) (*SetSelectionResponse, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return nil, ErrMissingSessionID
	}
	provider, model, err := s.resolveProviderAndModel(req.Provider, req.Model)
	if err != nil {
		return nil, err
	}
	if s.sessions != nil {
		if err := s.sessions.SetSelection(sessionID, provider, model); err != nil {
			return nil, err
		}
	}
	return &SetSelectionResponse{SessionID: sessionID, Provider: provider, Model: model}, nil
}

func (s *Service) resolveProviderAndModel(provider, model string) (string, string, error) {
	provider = strings.TrimSpace(strings.ToLower(provider))
	model = strings.TrimSpace(model)
	if provider == "" {
		provider = s.defaultProvider
	}
	if provider == "" {
		return "", "", ErrMissingProvider
	}
	p, ok := s.providersByName[provider]
	if !ok {
		return "", "", fmt.Errorf("%w: %s", ErrProviderNotFound, provider)
	}
	if model == "" {
		if p.DefaultModel != "" {
			model = p.DefaultModel
		} else if s.defaultModel != "" {
			model = s.defaultModel
		} else if len(p.Models) > 0 {
			model = p.Models[0]
		}
	}
	if model == "" {
		return "", "", ErrMissingModel
	}
	if len(p.Models) == 0 {
		return provider, model, nil
	}
	for _, candidate := range p.Models {
		if candidate == model {
			return provider, model, nil
		}
	}
	return "", "", fmt.Errorf("model %q is not configured for provider %q", model, provider)
}

func dedupeSorted(in []string) []string {
	uniq := map[string]struct{}{}
	for _, item := range in {
		name := strings.TrimSpace(item)
		if name == "" {
			continue
		}
		uniq[name] = struct{}{}
	}
	out := make([]string, 0, len(uniq))
	for name := range uniq {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
