package transport

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func (p *VoicePipeline) finalizeVoiceAudio(
	ctx context.Context,
	h *StreamHandler,
	deviceID string,
	buf []byte,
) ([]ServerMessage, error) {
	if h.runtime == nil || h.runtime.Env == nil {
		return nil, errors.New("scenario runtime not configured")
	}
	if h.runtime.Env.STT == nil {
		return nil, errors.New("speech-to-text backend not configured")
	}

	source := &voiceAudioReader{buf: buf}
	transcripts, err := h.runtime.Env.STT.Transcribe(ctx, source)
	if err != nil {
		return nil, err
	}
	spoken, spokenConfidence := finalSpokenText(transcripts)
	if spoken == "" {
		return nil, ErrMissingCommandText
	}
	spoken, err = p.normalizeWakeWord(ctx, h, spoken)
	if err != nil {
		return nil, err
	}
	if spoken == "" {
		return nil, nil
	}
	if !p.allowWakeWordDedupe(h, deviceID, spoken, spokenConfidence) {
		return nil, nil
	}
	return p.dispatchVoiceCommand(ctx, h, deviceID, spoken)
}

func (p *VoicePipeline) normalizeWakeWord(ctx context.Context, h *StreamHandler, spoken string) (string, error) {
	if h.runtime.Env.WakeWord == nil {
		return spoken, nil
	}
	detection, err := h.runtime.Env.WakeWord.Detect(ctx, spoken)
	if err != nil {
		return "", err
	}
	if !detection.Detected {
		return "", nil
	}
	if normalized := strings.TrimSpace(detection.Command); normalized != "" {
		return normalized, nil
	}
	return spoken, nil
}

func (p *VoicePipeline) allowWakeWordDedupe(h *StreamHandler, deviceID, spoken string, confidence float64) bool {
	if h.wakeWordDedupe == nil {
		return true
	}
	heardAt := time.Now().UTC()
	if h.control != nil {
		heardAt = h.control.now().UTC()
	}
	return h.wakeWordDedupe.Allow(wakeWordCandidate{
		DeviceID:   deviceID,
		Spoken:     spoken,
		HeardAt:    heardAt,
		Confidence: confidence,
	})
}

func (p *VoicePipeline) dispatchVoiceCommand(
	ctx context.Context,
	h *StreamHandler,
	deviceID, spoken string,
) ([]ServerMessage, error) {
	beforeCount := h.broadcastEventCount()
	scenarioName, err := h.runtime.HandleVoiceText(ctx, deviceID, spoken, h.control.now().UTC())
	if err != nil {
		return nil, err
	}

	out := []ServerMessage{
		{ScenarioStart: scenarioName, Notification: "Scenario started: " + scenarioName},
	}
	responseText := h.latestBroadcastForDevice(deviceID, beforeCount)
	if responseText == "" {
		return out, nil
	}
	out = appendVoiceAssistantUI(out, responseText)
	return p.appendVoiceTTS(ctx, h, deviceID, responseText, out)
}

func appendVoiceAssistantUI(out []ServerMessage, responseText string) []ServerMessage {
	responseView := ui.VoiceAssistantResponsePatch(responseText)
	return append(out, ServerMessage{
		UpdateUI: &UIUpdate{
			ComponentID: ui.GlobalOverlayComponentID,
			Node:        responseView,
		},
	})
}

func (p *VoicePipeline) appendVoiceTTS(
	ctx context.Context,
	h *StreamHandler,
	deviceID, responseText string,
	out []ServerMessage,
) ([]ServerMessage, error) {
	if h.runtime.Env.TTS == nil {
		return out, nil
	}
	playback, err := h.runtime.Env.TTS.Synthesize(ctx, responseText, scenario.TTSOptions{
		Voice:  "default",
		Format: "pcm16",
	})
	if err != nil {
		return nil, err
	}
	audio, err := readAudioPlayback(playback)
	if err != nil {
		return nil, err
	}
	out = append(out, ServerMessage{
		PlayAudio: &PlayAudioResponse{
			DeviceID: deviceID,
			Audio:    audio,
			Format:   "pcm16",
		},
	})
	return out, nil
}
