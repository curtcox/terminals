package usecasevalidation

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/scenario"
)

// FakeTTS is a test double for scenario.TextToSpeech that records each synthesis
// call as a short silent PCM16 audio blob. The harness reads Captures() at
// Evidence time to write audio artifacts for the doc site.
type FakeTTS struct {
	mu       sync.Mutex
	captures []TTSCapture
}

// TTSCapture holds one recorded synthesis call.
type TTSCapture struct {
	Text      string
	PCM       []byte
	Timestamp time.Time
}

// Captures returns a snapshot of all recorded synthesis calls.
func (f *FakeTTS) Captures() []TTSCapture {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]TTSCapture, len(f.captures))
	copy(out, f.captures)
	return out
}

// Synthesize records the call and returns 0.1 s of 24 kHz mono silence (PCM16 LE).
func (f *FakeTTS) Synthesize(_ context.Context, text string, _ scenario.TTSOptions) (scenario.AudioPlayback, error) {
	const (
		sampleRate      = 24000
		durationSamples = sampleRate / 10 // 0.1 s
	)
	silent := make([]byte, durationSamples*2) // 2 bytes per sample (16-bit)
	f.mu.Lock()
	f.captures = append(f.captures, TTSCapture{
		Text:      text,
		PCM:       silent,
		Timestamp: time.Now().UTC(),
	})
	f.mu.Unlock()
	return bytes.NewReader(silent), nil
}
