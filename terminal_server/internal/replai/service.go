// Package replai manages sticky AI provider and model selection for REPL sessions.
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
	// ErrMissingContextRef indicates a context ref was not provided.
	ErrMissingContextRef = errors.New("missing context ref")
	// ErrUnsupportedApprovalPolicy indicates the policy value is invalid.
	ErrUnsupportedApprovalPolicy = errors.New("unsupported approval policy")
)

const (
	// ApprovalPolicyPromptMutating prompts only for mutating calls.
	ApprovalPolicyPromptMutating = "prompt-mutating"
	// ApprovalPolicyPromptAll prompts for all tool calls.
	ApprovalPolicyPromptAll = "prompt-all"
	// ApprovalPolicyAutoReadOnly aliases prompt-mutating for readability.
	ApprovalPolicyAutoReadOnly = "auto-readonly"
)

// SessionSelectionStore persists sticky provider/model selection per session.
type SessionSelectionStore interface {
	GetSelection(sessionID string) (provider, model string, err error)
	SetSelection(sessionID, provider, model string) error
}

// SessionContextStore persists pinned context refs per session.
type SessionContextStore interface {
	GetPinnedContext(sessionID string) ([]string, error)
	SetPinnedContext(sessionID string, refs []string) error
}

// SessionPolicyStore persists approval policy per session.
type SessionPolicyStore interface {
	GetApprovalPolicy(sessionID string) (string, error)
	SetApprovalPolicy(sessionID, policy string) error
}

// SessionThreadStore persists LLM thread and exchange history per session.
type SessionThreadStore interface {
	GetThread(sessionID string) (string, error)
	SetThread(sessionID, thread string) error
	GetHistory(sessionID string) ([]string, error)
	SetHistory(sessionID string, history []string) error
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

// GetContextRequest returns pinned context for one session.
type GetContextRequest struct {
	SessionID string
}

// GetContextResponse reports pinned context refs.
type GetContextResponse struct {
	SessionID string   `json:"session_id"`
	Pinned    []string `json:"pinned"`
}

// AddContextRequest adds a one-shot context ref for next prompt.
type AddContextRequest struct {
	SessionID string
	Ref       string
}

// AddContextResponse reports accepted one-shot context ref.
type AddContextResponse struct {
	SessionID string `json:"session_id"`
	Ref       string `json:"ref"`
}

// PinContextRequest pins one context ref for future turns.
type PinContextRequest struct {
	SessionID string
	Ref       string
}

// PinContextResponse reports pinned context refs.
type PinContextResponse struct {
	SessionID string   `json:"session_id"`
	Pinned    []string `json:"pinned"`
}

// UnpinContextRequest removes one pinned context ref.
type UnpinContextRequest struct {
	SessionID string
	Ref       string
}

// UnpinContextResponse reports pinned context refs.
type UnpinContextResponse struct {
	SessionID string   `json:"session_id"`
	Pinned    []string `json:"pinned"`
}

// ClearContextRequest clears all pinned context refs.
type ClearContextRequest struct {
	SessionID string
}

// ClearContextResponse reports pinned context refs after clear.
type ClearContextResponse struct {
	SessionID string   `json:"session_id"`
	Pinned    []string `json:"pinned"`
}

// GetPolicyRequest returns approval policy for one session.
type GetPolicyRequest struct {
	SessionID string
}

// GetPolicyResponse reports current approval policy.
type GetPolicyResponse struct {
	SessionID string `json:"session_id"`
	Policy    string `json:"policy"`
}

// SetPolicyRequest sets approval policy for one session.
type SetPolicyRequest struct {
	SessionID string
	Policy    string
}

// SetPolicyResponse reports updated approval policy.
type SetPolicyResponse struct {
	SessionID string `json:"session_id"`
	Policy    string `json:"policy"`
}

// GetThreadRequest returns the current AI thread snapshot for one session.
type GetThreadRequest struct {
	SessionID string
}

// GetThreadResponse reports the current thread id and exchange history.
type GetThreadResponse struct {
	SessionID string   `json:"session_id"`
	Thread    string   `json:"thread"`
	History   []string `json:"history"`
}

// ResetThreadRequest clears thread state for one session.
type ResetThreadRequest struct {
	SessionID string
}

// ResetThreadResponse reports cleared thread state.
type ResetThreadResponse struct {
	SessionID string   `json:"session_id"`
	Thread    string   `json:"thread"`
	History   []string `json:"history"`
}

// Service provides typed AI selection APIs used by REPL commands.
type Service struct {
	sessions        SessionSelectionStore
	contexts        SessionContextStore
	policies        SessionPolicyStore
	threads         SessionThreadStore
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
	if store, ok := sessions.(SessionContextStore); ok {
		svc.contexts = store
	}
	if store, ok := sessions.(SessionPolicyStore); ok {
		svc.policies = store
	}
	if store, ok := sessions.(SessionThreadStore); ok {
		svc.threads = store
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

// GetContext returns pinned context refs for a session.
func (s *Service) GetContext(_ context.Context, req GetContextRequest) (*GetContextResponse, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return nil, ErrMissingSessionID
	}
	pinned, err := s.getPinnedContext(sessionID)
	if err != nil {
		return nil, err
	}
	return &GetContextResponse{SessionID: sessionID, Pinned: pinned}, nil
}

// AddContext validates one-shot context refs for upcoming prompts.
func (s *Service) AddContext(_ context.Context, req AddContextRequest) (*AddContextResponse, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return nil, ErrMissingSessionID
	}
	ref := strings.TrimSpace(req.Ref)
	if ref == "" {
		return nil, ErrMissingContextRef
	}
	return &AddContextResponse{SessionID: sessionID, Ref: ref}, nil
}

// PinContext adds a context ref to the pinned set.
func (s *Service) PinContext(_ context.Context, req PinContextRequest) (*PinContextResponse, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return nil, ErrMissingSessionID
	}
	ref := strings.TrimSpace(req.Ref)
	if ref == "" {
		return nil, ErrMissingContextRef
	}
	pinned, err := s.getPinnedContext(sessionID)
	if err != nil {
		return nil, err
	}
	updated := append([]string(nil), pinned...)
	if !containsString(updated, ref) {
		updated = append(updated, ref)
	}
	if err := s.setPinnedContext(sessionID, updated); err != nil {
		return nil, err
	}
	return &PinContextResponse{SessionID: sessionID, Pinned: updated}, nil
}

// UnpinContext removes one context ref from the pinned set.
func (s *Service) UnpinContext(_ context.Context, req UnpinContextRequest) (*UnpinContextResponse, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return nil, ErrMissingSessionID
	}
	ref := strings.TrimSpace(req.Ref)
	if ref == "" {
		return nil, ErrMissingContextRef
	}
	pinned, err := s.getPinnedContext(sessionID)
	if err != nil {
		return nil, err
	}
	updated := make([]string, 0, len(pinned))
	for _, candidate := range pinned {
		if candidate != ref {
			updated = append(updated, candidate)
		}
	}
	if err := s.setPinnedContext(sessionID, updated); err != nil {
		return nil, err
	}
	return &UnpinContextResponse{SessionID: sessionID, Pinned: updated}, nil
}

// ClearContext removes all pinned context refs for a session.
func (s *Service) ClearContext(_ context.Context, req ClearContextRequest) (*ClearContextResponse, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return nil, ErrMissingSessionID
	}
	if err := s.setPinnedContext(sessionID, nil); err != nil {
		return nil, err
	}
	return &ClearContextResponse{SessionID: sessionID, Pinned: []string{}}, nil
}

// GetPolicy returns the approval policy for a session.
func (s *Service) GetPolicy(_ context.Context, req GetPolicyRequest) (*GetPolicyResponse, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return nil, ErrMissingSessionID
	}
	policy, err := s.getPolicy(sessionID)
	if err != nil {
		return nil, err
	}
	if policy == "" {
		policy = ApprovalPolicyPromptMutating
	}
	return &GetPolicyResponse{SessionID: sessionID, Policy: policy}, nil
}

// SetPolicy validates and persists approval policy for a session.
func (s *Service) SetPolicy(_ context.Context, req SetPolicyRequest) (*SetPolicyResponse, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return nil, ErrMissingSessionID
	}
	policy, err := normalizePolicy(req.Policy)
	if err != nil {
		return nil, err
	}
	if err := s.setPolicy(sessionID, policy); err != nil {
		return nil, err
	}
	return &SetPolicyResponse{SessionID: sessionID, Policy: policy}, nil
}

// GetThread returns thread id and exchange history for one session.
func (s *Service) GetThread(_ context.Context, req GetThreadRequest) (*GetThreadResponse, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return nil, ErrMissingSessionID
	}
	thread, history, err := s.getThread(sessionID)
	if err != nil {
		return nil, err
	}
	return &GetThreadResponse{SessionID: sessionID, Thread: thread, History: history}, nil
}

// ResetThread clears thread id and exchange history for one session.
func (s *Service) ResetThread(_ context.Context, req ResetThreadRequest) (*ResetThreadResponse, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return nil, ErrMissingSessionID
	}
	if err := s.resetThread(sessionID); err != nil {
		return nil, err
	}
	return &ResetThreadResponse{SessionID: sessionID, Thread: "", History: []string{}}, nil
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
		switch {
		case p.DefaultModel != "":
			model = p.DefaultModel
		case s.defaultModel != "":
			model = s.defaultModel
		case len(p.Models) > 0:
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

func (s *Service) getPinnedContext(sessionID string) ([]string, error) {
	if s.contexts == nil {
		return []string{}, nil
	}
	pinned, err := s.contexts.GetPinnedContext(sessionID)
	if err != nil {
		return nil, err
	}
	return append([]string(nil), pinned...), nil
}

func (s *Service) setPinnedContext(sessionID string, refs []string) error {
	if s.contexts == nil {
		return nil
	}
	return s.contexts.SetPinnedContext(sessionID, refs)
}

func (s *Service) getPolicy(sessionID string) (string, error) {
	if s.policies == nil {
		return ApprovalPolicyPromptMutating, nil
	}
	policy, err := s.policies.GetApprovalPolicy(sessionID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(policy) == "" {
		return ApprovalPolicyPromptMutating, nil
	}
	return normalizePolicy(policy)
}

func (s *Service) setPolicy(sessionID, policy string) error {
	if s.policies == nil {
		return nil
	}
	return s.policies.SetApprovalPolicy(sessionID, policy)
}

func (s *Service) getThread(sessionID string) (string, []string, error) {
	if s.threads == nil {
		return "", []string{}, nil
	}
	thread, err := s.threads.GetThread(sessionID)
	if err != nil {
		return "", nil, err
	}
	history, err := s.threads.GetHistory(sessionID)
	if err != nil {
		return "", nil, err
	}
	return strings.TrimSpace(thread), append([]string(nil), history...), nil
}

func (s *Service) resetThread(sessionID string) error {
	if s.threads == nil {
		return nil
	}
	if err := s.threads.SetThread(sessionID, ""); err != nil {
		return err
	}
	return s.threads.SetHistory(sessionID, nil)
}

func normalizePolicy(policy string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(policy))
	switch value {
	case ApprovalPolicyPromptMutating, ApprovalPolicyPromptAll:
		return value, nil
	case ApprovalPolicyAutoReadOnly:
		return ApprovalPolicyPromptMutating, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedApprovalPolicy, policy)
	}
}

func containsString(list []string, value string) bool {
	for _, candidate := range list {
		if candidate == value {
			return true
		}
	}
	return false
}
