package scenario

import (
	"context"
	"errors"
	"strings"
	"sync"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
)

// IntercomScenario connects source device audio to all peer devices.
type IntercomScenario struct {
	trigger Trigger

	mu          sync.Mutex
	env         *Environment
	ownedRoutes []ioRoute
}

// Name returns the stable scenario identifier.
func (s *IntercomScenario) Name() string { return "intercom" }

// Match records trigger metadata when intercom mode is requested.
func (s *IntercomScenario) Match(trigger Trigger) bool {
	if !intentMatches(trigger.Intent, "intercom", "start intercom") {
		return false
	}
	s.trigger = trigger
	return true
}

// Start routes source microphone audio to other devices and announces activation.
func (s *IntercomScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil {
		return nil
	}
	targets := peerTargetDeviceIDs(env, s.trigger.SourceID, s.trigger.Arguments)
	ownedRoutes, err := connectBidirectionalSourceTargetsOwned(ctx, env, s.trigger.SourceID, targets, "audio")
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.env = env
	s.ownedRoutes = ownedRoutes
	s.mu.Unlock()
	if err := notifySource(ctx, env, s.trigger.SourceID, "Intercom active"); err != nil {
		return err
	}
	return nil
}

// Stop ends intercom mode and releases any owned routes.
func (s *IntercomScenario) Stop() error {
	s.mu.Lock()
	env := s.env
	routes := append([]ioRoute(nil), s.ownedRoutes...)
	s.env = nil
	s.ownedRoutes = nil
	s.mu.Unlock()
	return disconnectOwnedRoutes(env, routes)
}

// Suspend releases owned routes while preempted.
func (s *IntercomScenario) Suspend() error {
	s.mu.Lock()
	env := s.env
	routes := append([]ioRoute(nil), s.ownedRoutes...)
	s.mu.Unlock()
	return disconnectOwnedRoutes(env, routes)
}

// Resume reacquires owned routes after preemption.
func (s *IntercomScenario) Resume(_ context.Context, env *Environment) error {
	s.mu.Lock()
	routes := append([]ioRoute(nil), s.ownedRoutes...)
	s.env = env
	s.mu.Unlock()
	return reconnectOwnedRoutes(env, routes)
}

// InternalVideoCallScenario connects source and target with bidirectional audio/video.
type InternalVideoCallScenario struct {
	trigger Trigger
}

// Name returns the stable scenario identifier.
func (s *InternalVideoCallScenario) Name() string { return "internal_video_call" }

// Match records trigger metadata when an internal video call is requested.
func (s *InternalVideoCallScenario) Match(trigger Trigger) bool {
	if !intentMatches(trigger.Intent, "internal video call", "internal_video_call", "video call", "start video call") {
		return false
	}
	s.trigger = trigger
	return true
}

// Start routes bidirectional audio/video between source and target devices.
func (s *InternalVideoCallScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil {
		return nil
	}

	sourceID := strings.TrimSpace(s.trigger.SourceID)
	targetID := strings.TrimSpace(s.trigger.Arguments["target_device_id"])
	if targetID == "" {
		peers := nonSourceDeviceIDs(env, sourceID)
		if len(peers) > 0 {
			targetID = peers[0]
		}
	}
	if sourceID == "" || targetID == "" {
		return notifySource(ctx, env, sourceID, "Video call target unavailable")
	}

	if err := connectBidirectionalStreams(env, sourceID, targetID, "audio", "video"); err != nil {
		return err
	}

	if err := notifySource(ctx, env, sourceID, "Video call active: "+targetID); err != nil {
		return err
	}
	if env.Broadcast != nil {
		if err := env.Broadcast.Notify(ctx, []string{targetID}, "Incoming video call: "+sourceID); err != nil {
			return err
		}
	}
	return nil
}

// Stop ends internal video call mode and currently has no side effects.
func (s *InternalVideoCallScenario) Stop() error { return nil }

// PhoneCallScenario starts an external call through telephony bridge.
type PhoneCallScenario struct {
	trigger Trigger
}

// Name returns the stable scenario identifier.
func (s *PhoneCallScenario) Name() string { return "phone_call" }

// Match records trigger metadata when a phone call is requested.
func (s *PhoneCallScenario) Match(trigger Trigger) bool {
	if !intentMatches(trigger.Intent, "phone call", "phone_call", "call") {
		return false
	}
	s.trigger = trigger
	return true
}

// Start places a call and notifies the source device.
func (s *PhoneCallScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil {
		return nil
	}
	target := strings.TrimSpace(s.trigger.Arguments["target"])
	if target == "" {
		target = "unknown"
	}
	if env.Telephony != nil {
		if err := env.Telephony.Call(ctx, target); err != nil {
			return err
		}
	}
	return notifySource(ctx, env, s.trigger.SourceID, "Calling "+target)
}

// Stop ends phone call mode and currently has no side effects.
func (s *PhoneCallScenario) Stop() error { return nil }

// VoiceAssistantScenario proxies a prompt to AI backend and reports response.
type VoiceAssistantScenario struct {
	trigger Trigger

	mu           sync.Mutex
	env          *Environment
	activationID string
	planHandle   iorouter.PlanHandle
}

func (s *VoiceAssistantScenario) ensureActivationID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activationID != "" {
		return s.activationID
	}
	source := strings.TrimSpace(s.trigger.SourceID)
	if source == "" {
		source = "unknown"
	}
	s.activationID = "voice_assistant:" + source
	return s.activationID
}

// Name returns the stable scenario identifier.
func (s *VoiceAssistantScenario) Name() string { return "voice_assistant" }

// Match records trigger metadata when assistant mode is requested.
func (s *VoiceAssistantScenario) Match(trigger Trigger) bool {
	if !intentMatches(trigger.Intent, "voice assistant", "voice_assistant", "assistant") {
		return false
	}
	s.trigger = trigger
	return true
}

// Start queries the configured LLM (preferred) or legacy AIBackend and
// notifies the source device with the response.
func (s *VoiceAssistantScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil {
		return nil
	}
	s.mu.Lock()
	s.env = env
	s.mu.Unlock()

	activationID := s.ensureActivationID()
	deviceID := strings.TrimSpace(s.trigger.SourceID)
	if err := s.requestVoiceAssistantClaims(ctx, env, activationID, deviceID); err != nil {
		return err
	}
	if err := s.applyVoiceAssistantMediaPlan(ctx, env, deviceID); err != nil {
		return err
	}
	response, err := voiceAssistantQueryResponse(ctx, env, voiceAssistantQueryText(s.trigger))
	if err != nil {
		return err
	}
	return notifySource(ctx, env, s.trigger.SourceID, response)
}

// Stop ends assistant mode and releases claims/media resources.
func (s *VoiceAssistantScenario) Stop() error {
	s.mu.Lock()
	env := s.env
	activationID := s.activationID
	planHandle := s.planHandle
	s.planHandle = ""
	s.env = nil
	s.mu.Unlock()

	if env == nil {
		return nil
	}
	if err := tearVoiceAssistantMediaPlan(env, planHandle); err != nil {
		return err
	}
	return releaseVoiceAssistantClaims(env, activationID)
}

func tearVoiceAssistantMediaPlan(env *Environment, planHandle iorouter.PlanHandle) error {
	if planHandle == "" {
		return nil
	}
	routeIO, ok := env.IO.(interface{ MediaPlanner() *iorouter.MediaPlanner })
	if !ok {
		return nil
	}
	planner := routeIO.MediaPlanner()
	if planner == nil {
		return nil
	}
	return planner.Tear(context.Background(), planHandle)
}

func releaseVoiceAssistantClaims(env *Environment, activationID string) error {
	if activationID == "" {
		return nil
	}
	routeIO, ok := env.IO.(interface{ Claims() *iorouter.ClaimManager })
	if !ok {
		return nil
	}
	claims := routeIO.Claims()
	if claims == nil {
		return nil
	}
	return claims.Release(context.Background(), activationID)
}

// Suspend releases assistant-owned media resources while preempted.
func (s *VoiceAssistantScenario) Suspend() error {
	return s.Stop()
}

// Resume restores assistant media resources and claims after preemption.
func (s *VoiceAssistantScenario) Resume(ctx context.Context, env *Environment) error {
	return s.Start(ctx, env)
}

func connectBidirectionalStreams(env *Environment, sourceID, targetID string, kinds ...string) error {
	if env == nil || env.IO == nil {
		return nil
	}
	for _, streamKind := range kinds {
		if err := env.IO.Connect(sourceID, targetID, streamKind); err != nil && !errors.Is(err, iorouter.ErrRouteExists) {
			return err
		}
		if err := env.IO.Connect(targetID, sourceID, streamKind); err != nil && !errors.Is(err, iorouter.ErrRouteExists) {
			return err
		}
	}
	return nil
}
