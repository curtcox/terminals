package ai

import (
	"bytes"
	"context"
	"image"
	"io"
)

// noopSentinel is the deterministic placeholder text used by every noop AI
// backend. Tests can match against it to assert that a noop implementation
// was wired in.
const noopSentinel = "ai backend not configured"

// NoopSpeechToText returns an immediately closed transcript channel. It
// satisfies SpeechToText so callers can wire a placeholder while a real
// engine is being integrated.
type NoopSpeechToText struct{}

// Transcribe drains the audio reader and returns a closed channel.
func (NoopSpeechToText) Transcribe(_ context.Context, audio io.Reader, _ STTOptions) (<-chan Transcript, error) {
	if audio != nil {
		_, _ = io.Copy(io.Discard, audio)
	}
	ch := make(chan Transcript)
	close(ch)
	return ch, nil
}

// NoopTextToSpeech returns an empty audio reader for any text input.
type NoopTextToSpeech struct{}

// Synthesize returns an empty reader; the noop backend produces no audio.
func (NoopTextToSpeech) Synthesize(_ context.Context, _ string, _ TTSOptions) (io.Reader, error) {
	return bytes.NewReader(nil), nil
}

// NoopLLM returns the deterministic placeholder string for every query.
type NoopLLM struct{}

// Query returns a fixed sentinel response so callers can detect that a
// real LLM has not been configured.
func (NoopLLM) Query(_ context.Context, _ []Message, _ LLMOptions) (*LLMResponse, error) {
	return &LLMResponse{Text: noopSentinel, FinishReason: "stop"}, nil
}

// NoopVisionAnalyzer returns an empty analysis for any frame.
type NoopVisionAnalyzer struct{}

// Analyze returns a deterministic empty analysis.
func (NoopVisionAnalyzer) Analyze(_ context.Context, _ image.Image, _ string) (*VisionAnalysis, error) {
	return &VisionAnalysis{Caption: noopSentinel}, nil
}

// NoopSoundClassifier returns an immediately closed event channel.
type NoopSoundClassifier struct{}

// Classify drains the audio reader and returns a closed channel.
func (NoopSoundClassifier) Classify(_ context.Context, audio io.Reader) (<-chan SoundEvent, error) {
	if audio != nil {
		_, _ = io.Copy(io.Discard, audio)
	}
	ch := make(chan SoundEvent)
	close(ch)
	return ch, nil
}

// NoopBackends bundles a noop implementation of every AI capability for
// callers (server main, integration tests) that want a complete placeholder
// set in one shot.
type NoopBackends struct {
	STT    SpeechToText
	TTS    TextToSpeech
	LLM    LLM
	Vision VisionAnalyzer
	Sound  SoundClassifier
}

// NewNoopBackends returns a complete set of noop AI backends.
func NewNoopBackends() NoopBackends {
	return NoopBackends{
		STT:    NoopSpeechToText{},
		TTS:    NoopTextToSpeech{},
		LLM:    NoopLLM{},
		Vision: NoopVisionAnalyzer{},
		Sound:  NoopSoundClassifier{},
	}
}
