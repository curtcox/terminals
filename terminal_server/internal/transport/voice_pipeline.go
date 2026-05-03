package transport

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/recording"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

// VoicePipeline owns per-device voice audio buffering and the live audio
// publisher used by raw mic-audio handling.
type VoicePipeline struct {
	handler *StreamHandler

	mu          sync.Mutex
	buffers     map[string][]byte
	deviceAudio DeviceAudioPublisher
}

func NewVoicePipeline(handler *StreamHandler) *VoicePipeline {
	return &VoicePipeline{
		handler: handler,
		buffers: map[string][]byte{},
	}
}

func (p *VoicePipeline) SetDeviceAudioPublisher(pub DeviceAudioPublisher) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.deviceAudio = pub
}

// HandleAudio accumulates inbound mic audio per device and, on IsFinal, runs
// STT, wake-word handling, voice command dispatch, and optional TTS playback.
func (p *VoicePipeline) HandleAudio(ctx context.Context, va *VoiceAudioRequest) ([]ServerMessage, error) {
	if va == nil {
		return nil, ErrInvalidClientMessage
	}
	deviceID := strings.TrimSpace(va.DeviceID)
	if deviceID == "" {
		return nil, ErrInvalidClientMessage
	}
	h := p.handler
	if h == nil {
		return nil, errors.New("stream handler not configured")
	}
	if !h.deviceAllowsVoiceAudio(deviceID) {
		return nil, nil
	}

	buf, publisher := p.captureChunk(deviceID, va.Audio, va.IsFinal)
	recorder := h.currentRecordingManager()
	p.publishChunk(publisher, recorder, deviceID, va.Audio)
	if !va.IsFinal {
		return nil, nil
	}

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
	if h.runtime.Env.WakeWord != nil {
		detection, err := h.runtime.Env.WakeWord.Detect(ctx, spoken)
		if err != nil {
			return nil, err
		}
		if !detection.Detected {
			return nil, nil
		}
		if normalized := strings.TrimSpace(detection.Command); normalized != "" {
			spoken = normalized
		}
	}
	if h.wakeWordDedupe != nil {
		heardAt := time.Now().UTC()
		if h.control != nil {
			heardAt = h.control.now().UTC()
		}
		if !h.wakeWordDedupe.Allow(wakeWordCandidate{
			DeviceID:   deviceID,
			Spoken:     spoken,
			HeardAt:    heardAt,
			Confidence: spokenConfidence,
		}) {
			return nil, nil
		}
	}

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
	responseView := ui.VoiceAssistantResponsePatch(responseText)
	out = append(out, ServerMessage{
		UpdateUI: &UIUpdate{
			ComponentID: ui.GlobalOverlayComponentID,
			Node:        responseView,
		},
	})
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

func (p *VoicePipeline) captureChunk(deviceID string, chunk []byte, final bool) ([]byte, DeviceAudioPublisher) {
	p.mu.Lock()
	defer p.mu.Unlock()

	existing := p.buffers[deviceID]
	buf := make([]byte, 0, len(existing)+len(chunk))
	buf = append(buf, existing...)
	buf = append(buf, chunk...)
	if final {
		delete(p.buffers, deviceID)
	} else {
		p.buffers[deviceID] = buf
	}
	return buf, p.deviceAudio
}

func (p *VoicePipeline) publishChunk(
	publisher DeviceAudioPublisher,
	recorder recording.Manager,
	deviceID string,
	chunk []byte,
) {
	if len(chunk) == 0 {
		return
	}
	if publisher != nil {
		publisher.Publish(deviceID, chunk)
	}
	recordVoiceAudioChunk(recorder, deviceID, chunk)
}

func finalSpokenText(transcripts scenario.TranscriptStream) (string, float64) {
	var spoken string
	var confidence float64
	for tr := range transcripts {
		if tr.IsFinal && tr.Text != "" {
			spoken = tr.Text
			confidence = tr.Confidence
		} else if spoken == "" && tr.Text != "" {
			spoken = tr.Text
			confidence = tr.Confidence
		}
	}
	return strings.TrimSpace(spoken), confidence
}

func recordVoiceAudioChunk(recorder recording.Manager, deviceID string, chunk []byte) {
	writer, ok := recorder.(interface {
		WriteDeviceAudio(deviceID string, chunk []byte) error
	})
	if !ok {
		return
	}
	_ = writer.WriteDeviceAudio(deviceID, chunk)
}

func (h *StreamHandler) currentRecordingManager() recording.Manager {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.recording
}

// voiceAudioReader is a simple io.Reader over an accumulated voice buffer.
type voiceAudioReader struct {
	buf []byte
	off int
}

// Read consumes bytes from the buffered voice audio.
func (r *voiceAudioReader) Read(p []byte) (int, error) {
	if r.off >= len(r.buf) {
		return 0, io.EOF
	}
	n := copy(p, r.buf[r.off:])
	r.off += n
	return n, nil
}

// readAudioPlayback drains a scenario.AudioPlayback into a byte slice.
func readAudioPlayback(playback scenario.AudioPlayback) ([]byte, error) {
	if playback == nil {
		return nil, nil
	}
	buf := make([]byte, 0, 256)
	chunk := make([]byte, 256)
	for {
		n, err := playback.Read(chunk)
		if n > 0 {
			buf = append(buf, chunk[:n]...)
		}
		if err == io.EOF {
			return buf, nil
		}
		if err != nil {
			return nil, err
		}
	}
}
