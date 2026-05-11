package scenario

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
)

// AudioMonitorScenario stores monitor intent and confirms arming.
type AudioMonitorScenario struct {
	trigger Trigger

	mu           sync.Mutex
	stopFn       func()
	env          *Environment
	activationID string
	planHandle   iorouter.PlanHandle
}

// Name returns the stable scenario identifier.
func (s *AudioMonitorScenario) Name() string { return "audio_monitor" }

// Match records trigger metadata when audio monitor is requested.
func (s *AudioMonitorScenario) Match(trigger Trigger) bool {
	if !intentMatches(trigger.Intent, "audio monitor", "audio_monitor", "monitor audio") {
		return false
	}
	s.trigger = trigger
	return true
}

// Start persists the monitor target and notifies the source device.
func (s *AudioMonitorScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil {
		return nil
	}
	s.mu.Lock()
	s.env = env
	if s.activationID == "" {
		sourceID := strings.TrimSpace(s.trigger.SourceID)
		if sourceID == "" {
			sourceID = "unknown"
		}
		s.activationID = "audio_monitor:" + sourceID
	}
	s.mu.Unlock()
	target := strings.TrimSpace(s.trigger.Arguments["target"])
	if target == "" {
		target = "sound"
	}
	if env.Storage != nil && s.trigger.SourceID != "" {
		if err := env.Storage.Put(ctx, "audio_monitor:"+s.trigger.SourceID, target); err != nil {
			return err
		}
	}
	if err := notifySource(ctx, env, s.trigger.SourceID, "Audio monitor armed: "+target); err != nil {
		return err
	}
	return s.startMonitorLoop(ctx, env, target, strings.TrimSpace(s.trigger.SourceID))
}

// openAudioMonitorSource returns a live audio source for the monitored
// device, falling back to an immediate-EOF silence source when the runtime
// is not configured with a DeviceAudioSubscriber or no source device is set.
func openAudioMonitorSource(ctx context.Context, env *Environment, sourceID string) (AudioSource, func(), error) {
	if env != nil && env.DeviceAudio != nil && sourceID != "" {
		sub, err := env.DeviceAudio.SubscribeAudio(ctx, sourceID)
		if err != nil {
			return nil, nil, err
		}
		return sub, func() { _ = sub.Close() }, nil
	}
	return audioMonitorSilenceSource{}, func() {}, nil
}

// Stop ends monitor mode by canceling the active classifier goroutine and
// releasing any live audio subscription opened in Start. Safe to call when
// the scenario was never started or has already stopped.
func (s *AudioMonitorScenario) Stop() error {
	s.clearMonitorLoop()
	s.mu.Lock()
	env := s.env
	activationID := s.activationID
	planHandle := s.planHandle
	s.planHandle = ""
	s.env = nil
	s.mu.Unlock()
	if env != nil {
		if routeIO, ok := env.IO.(interface{ MediaPlanner() *iorouter.MediaPlanner }); ok {
			if planner := routeIO.MediaPlanner(); planner != nil && planHandle != "" {
				_ = planner.Tear(context.Background(), planHandle)
			}
		}
		if routeIO, ok := env.IO.(interface{ Claims() *iorouter.ClaimManager }); ok {
			if claims := routeIO.Claims(); claims != nil && activationID != "" {
				_ = claims.Release(context.Background(), activationID)
			}
		}
	}
	return nil
}

// Suspend releases live monitor resources while the scenario is preempted.
func (s *AudioMonitorScenario) Suspend() error {
	s.clearMonitorLoop()
	return nil
}

// Resume reacquires live monitor resources after preemption using the
// original trigger target and source device.
func (s *AudioMonitorScenario) Resume(ctx context.Context, env *Environment) error {
	target := strings.TrimSpace(s.trigger.Arguments["target"])
	if target == "" {
		target = "sound"
	}
	return s.startMonitorLoop(ctx, env, target, strings.TrimSpace(s.trigger.SourceID))
}

func (s *AudioMonitorScenario) clearMonitorLoop() {
	s.mu.Lock()
	stop := s.stopFn
	s.stopFn = nil
	s.mu.Unlock()
	if stop != nil {
		stop()
	}
}

func (s *AudioMonitorScenario) startMonitorLoop(ctx context.Context, env *Environment, target, sourceID string) error {
	if env == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	s.clearMonitorLoop()

	activationID := ""
	s.mu.Lock()
	activationID = s.activationID
	s.mu.Unlock()
	if activationID == "" {
		activationID = "audio_monitor:" + sourceID
	}

	analyzerPlanActive := false
	if routeIO, ok := env.IO.(interface {
		MediaPlanner() *iorouter.MediaPlanner
		Claims() *iorouter.ClaimManager
	}); ok {
		if claims := routeIO.Claims(); claims != nil {
			analyzeResource := resolveAudioInputAnalyzeResource(env, sourceID)
			_, err := claims.Request(ctx, []iorouter.Claim{{
				ActivationID: activationID,
				DeviceID:     sourceID,
				Resource:     analyzeResource,
				Mode:         iorouter.ClaimShared,
				Priority:     int(PriorityNormal),
			}})
			if err != nil && !errors.Is(err, iorouter.ErrClaimConflict) {
				return err
			}
		}
		if planner := routeIO.MediaPlanner(); planner != nil {
			if planner.AnalyzerEnabled() {
				handle, err := planner.Apply(ctx, iorouter.MediaPlan{
					Nodes: []iorouter.MediaNode{
						{ID: "mic", Kind: iorouter.NodeSourceMic, Args: map[string]string{"device_id": sourceID}},
						{ID: "analyzer", Kind: iorouter.NodeAnalyzer, Args: map[string]string{"name": "sound"}},
					},
					Edges: []iorouter.MediaEdge{
						{From: "mic", To: "analyzer"},
					},
				})
				if err == nil {
					s.mu.Lock()
					s.planHandle = handle
					s.mu.Unlock()
					analyzerPlanActive = true
				}
			}
		}
	}

	// Prefer event-bus subscriptions emitted by analyzer nodes.
	if analyzerPlanActive && env.TriggerBus != nil {
		sub, cancel := env.TriggerBus.Subscribe(16)
		audioCtx, cancelAudio := context.WithCancel(ctx)
		stopFn := func() {
			cancelAudio()
			cancel()
		}
		s.mu.Lock()
		s.stopFn = stopFn
		s.mu.Unlock()

		go func() {
			defer stopFn()
			for {
				select {
				case <-audioCtx.Done():
					return
				case trigger, ok := <-sub:
					if !ok || trigger.EventV2 == nil {
						continue
					}
					if strings.TrimSpace(trigger.EventV2.Kind) != "sound.detected" {
						continue
					}
					if src := strings.TrimSpace(trigger.SourceID); src != "" && src != sourceID {
						continue
					}
					label := strings.TrimSpace(trigger.EventV2.Subject)
					if label == "" {
						label = strings.TrimSpace(trigger.EventV2.Attributes["label"])
					}
					if !audioMonitorEventMatchesTarget(target, label) {
						continue
					}
					if label == "" {
						label = target
					}
					_ = notifySource(ctx, env, sourceID, "Audio monitor detected: "+label)
					return
				}
			}
		}()
		return nil
	}

	// Fallback path for tests/contexts without an event bus analyzer runner.
	if env.Sound == nil {
		return nil
	}
	audioCtx, cancelAudio := context.WithCancel(ctx)
	audio, closeAudio, err := openAudioMonitorSource(audioCtx, env, sourceID)
	if err != nil {
		cancelAudio()
		return err
	}
	stream, err := env.Sound.Classify(audioCtx, audio)
	if err != nil {
		closeAudio()
		cancelAudio()
		return err
	}
	var stopOnce sync.Once
	stopFn := func() {
		stopOnce.Do(func() {
			cancelAudio()
			closeAudio()
		})
	}
	s.mu.Lock()
	s.stopFn = stopFn
	s.mu.Unlock()
	go func() {
		defer stopFn()
		for event := range stream {
			if !audioMonitorEventMatchesTarget(target, event.Label) {
				continue
			}
			messageLabel := strings.TrimSpace(event.Label)
			if messageLabel == "" {
				messageLabel = target
			}
			_ = notifySource(ctx, env, sourceID, "Audio monitor detected: "+messageLabel)
			return
		}
	}()
	return nil
}

type audioMonitorSilenceSource struct{}

func (audioMonitorSilenceSource) Read([]byte) (int, error) {
	return 0, io.EOF
}

func audioMonitorEventMatchesTarget(target, label string) bool {
	normalizedTarget := strings.TrimSpace(strings.ToLower(target))
	normalizedLabel := strings.TrimSpace(strings.ToLower(label))
	if normalizedLabel == "" {
		return false
	}
	if normalizedTarget == "" || normalizedTarget == "sound" {
		return true
	}
	return strings.Contains(normalizedLabel, normalizedTarget) || strings.Contains(normalizedTarget, normalizedLabel)
}

// PASystemScenario fans out source audio to peers with PA semantics.
type PASystemScenario struct {
	trigger Trigger

	recipe scenarioRecipeState
}

// Name returns the stable scenario identifier.
func (s *PASystemScenario) Name() string { return "pa_system" }

// Match records trigger metadata when PA mode is requested.
func (s *PASystemScenario) Match(trigger Trigger) bool {
	if !intentMatches(trigger.Intent, "pa system", "pa_system", "pa mode") {
		return false
	}
	s.trigger = trigger
	return true
}

// Start routes source audio to peers and announces PA mode.
func (s *PASystemScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil {
		return nil
	}
	sourceID := strings.TrimSpace(s.trigger.SourceID)
	if sourceID == "" {
		sourceID = "unknown"
	}
	activationID := "pa_system:" + sourceID
	return s.recipe.start(ctx, env, ScenarioRecipe{
		ActivationID: activationID,
		Resolve: func(_ context.Context, env *Environment) []string {
			return peerTargetDeviceIDs(env, sourceID, s.trigger.Arguments)
		},
		Claims: func(targets []string) []iorouter.Claim {
			sourceResource := resolveAudioInputCaptureResource(env, sourceID)
			out := make([]iorouter.Claim, 0, len(targets)+1)
			out = append(out, iorouter.Claim{
				ActivationID: activationID,
				DeviceID:     sourceID,
				Resource:     sourceResource,
				Mode:         iorouter.ClaimExclusive,
				Priority:     int(PriorityHigh),
			})
			for _, targetID := range targets {
				targetResource := resolveAudioOutResource(env, targetID)
				out = append(out, iorouter.Claim{
					ActivationID: activationID,
					DeviceID:     targetID,
					Resource:     targetResource,
					Mode:         iorouter.ClaimExclusive,
					Priority:     int(PriorityHigh),
				})
			}
			return out
		},
		MediaPlan: func(targets []string) *iorouter.MediaPlan {
			sourceResource := resolveAudioInputCaptureResource(env, sourceID)
			nodes := make([]iorouter.MediaNode, 0, 2+len(targets))
			nodes = append(nodes,
				iorouter.MediaNode{ID: "mic", Kind: iorouter.NodeSourceMic, Args: map[string]string{"device_id": sourceID, "resource": sourceResource}},
				iorouter.MediaNode{ID: "fork", Kind: iorouter.NodeFork},
			)
			edges := make([]iorouter.MediaEdge, 0, 1+len(targets))
			edges = append(edges, iorouter.MediaEdge{From: "mic", To: "fork"})
			for idx, targetID := range targets {
				nodeID := fmt.Sprintf("speaker_%d", idx)
				targetResource := resolveAudioOutResource(env, targetID)
				nodes = append(nodes, iorouter.MediaNode{
					ID:   nodeID,
					Kind: iorouter.NodeSinkSpeaker,
					Args: map[string]string{"device_id": targetID, "stream_kind": "pa_audio", "resource": targetResource},
				})
				edges = append(edges, iorouter.MediaEdge{From: "fork", To: nodeID})
			}
			return &iorouter.MediaPlan{Nodes: nodes, Edges: edges}
		},
		OnStart: func(ctx context.Context, env *Environment, _ []string) error {
			if err := notifySource(ctx, env, sourceID, "PA system active"); err != nil {
				return err
			}
			if env.Broadcast == nil {
				return nil
			}
			peerIDs := nonSourceDeviceIDs(env, sourceID)
			if len(peerIDs) == 0 {
				return nil
			}
			return env.Broadcast.Notify(ctx, peerIDs, "PA from "+sourceID)
		},
	})
}

// Stop ends PA mode and releases any owned routes.
func (s *PASystemScenario) Stop() error {
	sourceID := strings.TrimSpace(s.trigger.SourceID)
	if sourceID == "" {
		sourceID = "unknown"
	}
	return s.recipe.stop(context.Background(), ScenarioRecipe{
		ActivationID: "pa_system:" + sourceID,
	})
}

// Suspend releases PA-owned routes while preempted.
func (s *PASystemScenario) Suspend() error {
	sourceID := strings.TrimSpace(s.trigger.SourceID)
	if sourceID == "" {
		sourceID = "unknown"
	}
	return s.recipe.stop(context.Background(), ScenarioRecipe{
		ActivationID: "pa_system:" + sourceID,
	})
}

// Resume reacquires PA-owned routes after preemption.
func (s *PASystemScenario) Resume(_ context.Context, env *Environment) error {
	return s.Start(context.Background(), env)
}

// AnnouncementScenario fans out source audio one-way for whole-house announcements.
type AnnouncementScenario struct {
	trigger Trigger

	recipe scenarioRecipeState
}

// Name returns the stable scenario identifier.
func (s *AnnouncementScenario) Name() string { return "announcement" }

// Match records trigger metadata when announcement mode is requested.
func (s *AnnouncementScenario) Match(trigger Trigger) bool {
	if !intentMatches(trigger.Intent, "announcement", "announce", "whole house announcement") {
		return false
	}
	s.trigger = trigger
	return true
}

// Start routes source audio to peers and announces announcement mode.
func (s *AnnouncementScenario) Start(ctx context.Context, env *Environment) error {
	if env == nil {
		return nil
	}
	sourceID := strings.TrimSpace(s.trigger.SourceID)
	if sourceID == "" {
		sourceID = "unknown"
	}
	activationID := "announcement:" + sourceID
	return s.recipe.start(ctx, env, ScenarioRecipe{
		ActivationID: activationID,
		Resolve: func(_ context.Context, env *Environment) []string {
			return peerTargetDeviceIDs(env, sourceID, s.trigger.Arguments)
		},
		Claims: func(targets []string) []iorouter.Claim {
			sourceResource := resolveAudioInputCaptureResource(env, sourceID)
			out := make([]iorouter.Claim, 0, len(targets)+1)
			out = append(out, iorouter.Claim{
				ActivationID: activationID,
				DeviceID:     sourceID,
				Resource:     sourceResource,
				Mode:         iorouter.ClaimExclusive,
				Priority:     int(PriorityHigh),
			})
			for _, targetID := range targets {
				targetResource := resolveAudioOutResource(env, targetID)
				out = append(out, iorouter.Claim{
					ActivationID: activationID,
					DeviceID:     targetID,
					Resource:     targetResource,
					Mode:         iorouter.ClaimExclusive,
					Priority:     int(PriorityHigh),
				})
			}
			return out
		},
		MediaPlan: func(targets []string) *iorouter.MediaPlan {
			sourceResource := resolveAudioInputCaptureResource(env, sourceID)
			nodes := make([]iorouter.MediaNode, 0, 2+len(targets))
			nodes = append(nodes,
				iorouter.MediaNode{ID: "mic", Kind: iorouter.NodeSourceMic, Args: map[string]string{"device_id": sourceID, "resource": sourceResource}},
				iorouter.MediaNode{ID: "fork", Kind: iorouter.NodeFork},
			)
			edges := make([]iorouter.MediaEdge, 0, 1+len(targets))
			edges = append(edges, iorouter.MediaEdge{From: "mic", To: "fork"})
			for idx, targetID := range targets {
				nodeID := fmt.Sprintf("speaker_%d", idx)
				targetResource := resolveAudioOutResource(env, targetID)
				nodes = append(nodes, iorouter.MediaNode{
					ID:   nodeID,
					Kind: iorouter.NodeSinkSpeaker,
					Args: map[string]string{"device_id": targetID, "stream_kind": "announcement_audio", "resource": targetResource},
				})
				edges = append(edges, iorouter.MediaEdge{From: "fork", To: nodeID})
			}
			return &iorouter.MediaPlan{Nodes: nodes, Edges: edges}
		},
		OnStart: func(ctx context.Context, env *Environment, _ []string) error {
			if err := notifySource(ctx, env, sourceID, "Announcement active"); err != nil {
				return err
			}
			if env.Broadcast == nil {
				return nil
			}
			peerIDs := nonSourceDeviceIDs(env, sourceID)
			if len(peerIDs) == 0 {
				return nil
			}
			return env.Broadcast.Notify(ctx, peerIDs, "Announcement from "+sourceID)
		},
	})
}

// Stop ends announcement mode and releases any owned routes.
func (s *AnnouncementScenario) Stop() error {
	sourceID := strings.TrimSpace(s.trigger.SourceID)
	if sourceID == "" {
		sourceID = "unknown"
	}
	return s.recipe.stop(context.Background(), ScenarioRecipe{
		ActivationID: "announcement:" + sourceID,
	})
}

// Suspend releases announcement-owned routes while preempted.
func (s *AnnouncementScenario) Suspend() error {
	sourceID := strings.TrimSpace(s.trigger.SourceID)
	if sourceID == "" {
		sourceID = "unknown"
	}
	return s.recipe.stop(context.Background(), ScenarioRecipe{
		ActivationID: "announcement:" + sourceID,
	})
}

// Resume reacquires announcement-owned routes after preemption.
func (s *AnnouncementScenario) Resume(_ context.Context, env *Environment) error {
	return s.Start(context.Background(), env)
}
