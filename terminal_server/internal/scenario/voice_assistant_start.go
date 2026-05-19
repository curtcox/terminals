package scenario

import (
	"context"
	"errors"
	"strings"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
)

func (s *VoiceAssistantScenario) requestVoiceAssistantClaims(ctx context.Context, env *Environment, activationID, deviceID string) error {
	routeIO, ok := env.IO.(interface{ Claims() *iorouter.ClaimManager })
	if !ok {
		return nil
	}
	claims := routeIO.Claims()
	if claims == nil {
		return nil
	}
	analyzeResource := resolveAudioInputAnalyzeResource(env, deviceID)
	speakerResource := resolveAudioOutResource(env, deviceID)
	overlayResource := resolveDisplayOverlayResource(env, deviceID)
	_, err := claims.Request(ctx, []iorouter.Claim{
		{
			ActivationID: activationID,
			DeviceID:     deviceID,
			Resource:     analyzeResource,
			Mode:         iorouter.ClaimShared,
			Priority:     int(PriorityNormal),
		},
		{
			ActivationID: activationID,
			DeviceID:     deviceID,
			Resource:     speakerResource,
			Mode:         iorouter.ClaimExclusive,
			Priority:     int(PriorityNormal),
		},
		{
			ActivationID: activationID,
			DeviceID:     deviceID,
			Resource:     overlayResource,
			Mode:         iorouter.ClaimShared,
			Priority:     int(PriorityNormal),
		},
	})
	if err != nil && !errors.Is(err, iorouter.ErrClaimConflict) {
		return err
	}
	return nil
}

func (s *VoiceAssistantScenario) applyVoiceAssistantMediaPlan(ctx context.Context, env *Environment, deviceID string) error {
	routeIO, ok := env.IO.(interface{ MediaPlanner() *iorouter.MediaPlanner })
	if !ok {
		return nil
	}
	planner := routeIO.MediaPlanner()
	if planner == nil {
		return nil
	}
	handle, err := planner.Apply(ctx, iorouter.MediaPlan{
		Nodes: []iorouter.MediaNode{
			{ID: "mic", Kind: iorouter.NodeSourceMic, Args: map[string]string{"device_id": deviceID}},
			{ID: "fork", Kind: iorouter.NodeFork},
			{ID: "stt", Kind: iorouter.NodeSinkSTT, Args: map[string]string{"device_id": "server"}},
			{ID: "rec", Kind: iorouter.NodeRecorder, Args: map[string]string{"device_id": "server"}},
			{ID: "tts", Kind: iorouter.NodeSourceTTS, Args: map[string]string{"device_id": "server"}},
			{ID: "speaker", Kind: iorouter.NodeSinkSpeaker, Args: map[string]string{"device_id": deviceID}},
		},
		Edges: []iorouter.MediaEdge{
			{From: "mic", To: "fork"},
			{From: "fork", To: "stt"},
			{From: "fork", To: "rec"},
			{From: "tts", To: "speaker"},
		},
	})
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.planHandle = handle
	s.mu.Unlock()
	return nil
}

func voiceAssistantQueryResponse(ctx context.Context, env *Environment, query string) (string, error) {
	response := "Voice assistant active"
	switch {
	case env.LLM != nil:
		out, err := env.LLM.Query(ctx, []LLMMessage{{Role: "user", Content: query}}, LLMOptions{})
		if err != nil {
			return "", err
		}
		if out != nil && strings.TrimSpace(out.Text) != "" {
			response = out.Text
		}
	case env.AI != nil:
		out, err := env.AI.Query(ctx, query)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(out) != "" {
			response = out
		}
	}
	return response, nil
}

func voiceAssistantQueryText(trigger Trigger) string {
	query := strings.TrimSpace(trigger.Arguments["query"])
	if query == "" {
		return "hello"
	}
	return query
}
