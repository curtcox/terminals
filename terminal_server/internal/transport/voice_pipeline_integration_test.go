package transport

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/telephony"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

// pipelineSTT, pipelineLLM, and pipelineTTS are minimal scenario-shaped fakes
// used to drive the STT → LLM → TTS pipeline through the control stream.

type pipelineSTT struct {
	transcribed string
	captured    []byte
}

func (s *pipelineSTT) Transcribe(_ context.Context, audio scenario.AudioSource) (scenario.TranscriptStream, error) {
	if audio != nil {
		s.captured, _ = io.ReadAll(audioReader{audio})
	}
	out := make(chan scenario.Transcript, 1)
	out <- scenario.Transcript{Text: s.transcribed, IsFinal: true, Confidence: 1.0}
	close(out)
	return out, nil
}

type audioReader struct {
	src scenario.AudioSource
}

func (a audioReader) Read(p []byte) (int, error) {
	if a.src == nil {
		return 0, io.EOF
	}
	return a.src.Read(p)
}

type pipelineLLM struct {
	response string
	queries  [][]scenario.LLMMessage
	failWith error
}

func (l *pipelineLLM) Query(
	_ context.Context,
	messages []scenario.LLMMessage,
	_ scenario.LLMOptions,
) (*scenario.LLMResponse, error) {
	if l.failWith != nil {
		return nil, l.failWith
	}
	copyMsgs := make([]scenario.LLMMessage, len(messages))
	copy(copyMsgs, messages)
	l.queries = append(l.queries, copyMsgs)
	return &scenario.LLMResponse{Text: l.response, FinishReason: "stop"}, nil
}

type pipelineTTS struct {
	calls []string
	audio []byte
}

func (t *pipelineTTS) Synthesize(
	_ context.Context,
	text string,
	_ scenario.TTSOptions,
) (scenario.AudioPlayback, error) {
	t.calls = append(t.calls, text)
	return playback{r: bytes.NewReader(t.audio)}, nil
}

type playback struct {
	r io.Reader
}

func (p playback) Read(buf []byte) (int, error) {
	return p.r.Read(buf)
}

// TestControlStreamVoiceAudioPipeline drives the full Phase-5 voice pipeline
// through the control stream's voice_audio path:
//
//	VoiceAudio → STT → Runtime.HandleVoiceText → VoiceAssistant scenario
//	          → LLM → Broadcaster → TTS → PlayAudio reply.
//
// The test asserts that:
//   - The STT fake consumed the buffered mic audio.
//   - The LLM fake received the transcribed user prompt.
//   - The TTS fake synthesized the LLM response.
//   - The server emitted a PlayAudio response targeting the source device
//     with the TTS-generated audio bytes.
func TestControlStreamVoiceAudioPipeline(t *testing.T) {
	stt := &pipelineSTT{transcribed: "assistant what is the weather"}
	llm := &pipelineLLM{response: "It is sunny in Test City"}
	tts := &pipelineTTS{audio: []byte{0x01, 0x02, 0x03, 0x04}}

	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	broadcaster := ui.NewMemoryBroadcaster()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        iorouter.NewRouter(),
		LLM:       llm,
		STT:       stt,
		TTS:       tts,
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: broadcaster,
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{
			DeviceID:   "device-1",
			DeviceName: "Kitchen Chromebook",
		},
	}); err != nil {
		t.Fatalf("register error = %v", err)
	}

	// Non-final chunk is buffered and produces no response.
	partial, err := handler.HandleMessage(context.Background(), ClientMessage{
		VoiceAudio: &VoiceAudioRequest{
			DeviceID:   "device-1",
			Audio:      []byte("opus-"),
			SampleRate: 16000,
			IsFinal:    false,
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(non-final) error = %v", err)
	}
	if len(partial) != 0 {
		t.Fatalf("non-final voice_audio produced responses: %+v", partial)
	}

	// Final chunk runs STT → LLM → TTS and yields a PlayAudio reply.
	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		VoiceAudio: &VoiceAudioRequest{
			DeviceID:   "device-1",
			Audio:      []byte("encoded-mic-audio"),
			SampleRate: 16000,
			IsFinal:    true,
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(final voice_audio) error = %v", err)
	}

	// STT must have consumed both the buffered and the final chunk together.
	wantCaptured := "opus-encoded-mic-audio"
	if string(stt.captured) != wantCaptured {
		t.Fatalf("STT captured = %q, want %q", string(stt.captured), wantCaptured)
	}

	// LLM must have received the transcribed query without the wake word.
	if len(llm.queries) != 1 {
		t.Fatalf("LLM query count = %d, want 1", len(llm.queries))
	}
	if len(llm.queries[0]) != 1 ||
		llm.queries[0][0].Role != "user" ||
		!strings.Contains(llm.queries[0][0].Content, "what is the weather") {
		t.Fatalf("LLM query payload = %+v, want user turn with the weather prompt", llm.queries[0])
	}

	// Broadcaster must have recorded the voice assistant's reply.
	events := broadcaster.Events()
	if len(events) == 0 {
		t.Fatalf("expected broadcast event for LLM response")
	}
	last := events[len(events)-1]
	if last.Message != "It is sunny in Test City" {
		t.Fatalf("broadcast message = %q, want LLM response", last.Message)
	}

	// TTS must have been called exactly once with the LLM response.
	if len(tts.calls) != 1 || tts.calls[0] != "It is sunny in Test City" {
		t.Fatalf("TTS calls = %+v, want single synthesis of LLM response", tts.calls)
	}

	// The control stream must reply with the TTS audio as a PlayAudio message.
	var play *PlayAudioResponse
	sawAssistantStart := false
	for _, msg := range out {
		if msg.ScenarioStart == "voice_assistant" {
			sawAssistantStart = true
		}
		if msg.PlayAudio != nil {
			play = msg.PlayAudio
		}
	}
	if !sawAssistantStart {
		t.Fatalf("expected voice_assistant scenario start, got %+v", out)
	}
	if play == nil {
		t.Fatalf("expected PlayAudio reply, got %+v", out)
	}
	if play.DeviceID != "device-1" {
		t.Fatalf("PlayAudio.DeviceID = %q, want device-1", play.DeviceID)
	}
	if !bytes.Equal(play.Audio, []byte{0x01, 0x02, 0x03, 0x04}) {
		t.Fatalf("PlayAudio.Audio = % x, want % x", play.Audio, []byte{0x01, 0x02, 0x03, 0x04})
	}
	if play.Format == "" {
		t.Fatalf("PlayAudio.Format is empty")
	}
}

// TestControlStreamVoiceCommandTextPath confirms the text-only Command{Kind:voice}
// path still works for tests that do not exercise raw mic audio.
func TestControlStreamVoiceCommandTextPath(t *testing.T) {
	llm := &pipelineLLM{response: "It is sunny in Test City"}

	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	broadcaster := ui.NewMemoryBroadcaster()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        iorouter.NewRouter(),
		LLM:       llm,
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: broadcaster,
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{
			DeviceID:   "device-1",
			DeviceName: "Kitchen Chromebook",
		},
	}); err != nil {
		t.Fatalf("register error = %v", err)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-voice-text",
			DeviceID:  "device-1",
			Kind:      "voice",
			Text:      "assistant what is the weather",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(voice command) error = %v", err)
	}

	sawAssistantStart := false
	for _, msg := range out {
		if msg.ScenarioStart == "voice_assistant" {
			sawAssistantStart = true
			break
		}
	}
	if !sawAssistantStart {
		t.Fatalf("expected voice_assistant scenario start, got %+v", out)
	}
	if len(llm.queries) != 1 {
		t.Fatalf("LLM query count = %d, want 1", len(llm.queries))
	}
}

// TestControlStreamVoiceAudioSurfacesLLMError confirms an LLM failure during
// the voice_audio pipeline is surfaced as an error to the caller.
func TestControlStreamVoiceAudioSurfacesLLMError(t *testing.T) {
	stt := &pipelineSTT{transcribed: "assistant tell me a joke"}
	llm := &pipelineLLM{failWith: errors.New("llm offline")}
	tts := &pipelineTTS{audio: []byte{0x01}}

	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	broadcaster := ui.NewMemoryBroadcaster()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        iorouter.NewRouter(),
		LLM:       llm,
		STT:       stt,
		TTS:       tts,
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: broadcaster,
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{
			DeviceID:   "device-1",
			DeviceName: "Kitchen Chromebook",
		},
	}); err != nil {
		t.Fatalf("register error = %v", err)
	}

	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		VoiceAudio: &VoiceAudioRequest{
			DeviceID:   "device-1",
			Audio:      []byte("mic-audio"),
			SampleRate: 16000,
			IsFinal:    true,
		},
	})
	if err == nil {
		t.Fatalf("expected error when LLM fails")
	}
	if !strings.Contains(err.Error(), "llm offline") {
		t.Fatalf("error = %v, want contains llm offline", err)
	}
}
