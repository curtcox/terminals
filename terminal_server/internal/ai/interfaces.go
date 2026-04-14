package ai

import (
	"context"
	"image"
	"io"
)

// Transcript is a single speech recognition result emitted by SpeechToText.
type Transcript struct {
	Text       string
	Confidence float64
	IsFinal    bool
}

// STTOptions configures a speech-to-text request.
type STTOptions struct {
	Language   string
	SampleRate int
	Channels   int
}

// SpeechToText converts an audio stream into transcripts.
type SpeechToText interface {
	Transcribe(ctx context.Context, audio io.Reader, opts STTOptions) (<-chan Transcript, error)
}

// TTSOptions configures a text-to-speech synthesis request.
type TTSOptions struct {
	Voice  string
	Format string
}

// TextToSpeech synthesizes spoken audio from text.
type TextToSpeech interface {
	Synthesize(ctx context.Context, text string, opts TTSOptions) (io.Reader, error)
}

// Message is a single LLM conversation entry.
type Message struct {
	Role    string
	Content string
}

// LLMOptions configures a large-language-model query.
type LLMOptions struct {
	Model       string
	MaxTokens   int
	Temperature float64
}

// LLMResponse is the result of an LLM query.
type LLMResponse struct {
	Text         string
	FinishReason string
}

// LLM exposes a chat-style large language model.
type LLM interface {
	Query(ctx context.Context, messages []Message, opts LLMOptions) (*LLMResponse, error)
}

// VisionAnalysis describes the result of analyzing a single frame.
type VisionAnalysis struct {
	Caption string
	Labels  []string
}

// VisionAnalyzer interprets images.
type VisionAnalyzer interface {
	Analyze(ctx context.Context, frame image.Image, prompt string) (*VisionAnalysis, error)
}

// SoundEvent describes a classified audio event.
type SoundEvent struct {
	Label      string
	Confidence float64
	AtMS       int64
}

// SoundClassifier streams classified events from audio input.
type SoundClassifier interface {
	Classify(ctx context.Context, audio io.Reader) (<-chan SoundEvent, error)
}
