package scenario

import (
	"context"
	"errors"
	"strings"
	"sync"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
)

func (s *AudioMonitorScenario) audioMonitorActivationID(sourceID string) string {
	s.mu.Lock()
	activationID := s.activationID
	s.mu.Unlock()
	if activationID != "" {
		return activationID
	}
	return "audio_monitor:" + sourceID
}

func (s *AudioMonitorScenario) requestAudioMonitorAnalyzeClaim(
	ctx context.Context,
	env *Environment,
	activationID, sourceID string,
) error {
	routeIO, ok := env.IO.(interface{ Claims() *iorouter.ClaimManager })
	if !ok {
		return nil
	}
	claims := routeIO.Claims()
	if claims == nil {
		return nil
	}
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
	return nil
}

func (s *AudioMonitorScenario) applyAudioMonitorAnalyzerPlan(
	ctx context.Context,
	env *Environment,
	sourceID string,
) (bool, error) {
	routeIO, ok := env.IO.(interface{ MediaPlanner() *iorouter.MediaPlanner })
	if !ok {
		return false, nil
	}
	planner := routeIO.MediaPlanner()
	if planner == nil || !planner.AnalyzerEnabled() {
		return false, nil
	}
	handle, err := planner.Apply(ctx, iorouter.MediaPlan{
		Nodes: []iorouter.MediaNode{
			{ID: "mic", Kind: iorouter.NodeSourceMic, Args: map[string]string{"device_id": sourceID}},
			{ID: "analyzer", Kind: iorouter.NodeAnalyzer, Args: map[string]string{"name": "sound"}},
		},
		Edges: []iorouter.MediaEdge{
			{From: "mic", To: "analyzer"},
		},
	})
	if err != nil {
		return false, err
	}
	s.mu.Lock()
	s.planHandle = handle
	s.mu.Unlock()
	return true, nil
}

func (s *AudioMonitorScenario) runAudioMonitorTriggerBusLoop(
	ctx context.Context,
	env *Environment,
	target, sourceID string,
) bool {
	if env.TriggerBus == nil {
		return false
	}
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
				if audioMonitorTriggerBusEventHandled(ctx, env, target, sourceID, trigger, ok) {
					return
				}
			}
		}
	}()
	return true
}

func audioMonitorTriggerBusEventHandled(
	ctx context.Context,
	env *Environment,
	target, sourceID string,
	trigger Trigger,
	ok bool,
) bool {
	if !ok || trigger.EventV2 == nil {
		return false
	}
	if strings.TrimSpace(trigger.EventV2.Kind) != "sound.detected" {
		return false
	}
	if src := strings.TrimSpace(trigger.SourceID); src != "" && src != sourceID {
		return false
	}
	label := strings.TrimSpace(trigger.EventV2.Subject)
	if label == "" {
		label = strings.TrimSpace(trigger.EventV2.Attributes["label"])
	}
	if !audioMonitorEventMatchesTarget(target, label) {
		return false
	}
	if label == "" {
		label = target
	}
	_ = notifySource(ctx, env, sourceID, "Audio monitor detected: "+label)
	return true
}

func (s *AudioMonitorScenario) runAudioMonitorClassifierLoop(
	ctx context.Context,
	env *Environment,
	target, sourceID string,
) error {
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
