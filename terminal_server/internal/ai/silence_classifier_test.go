package ai

import (
	"bytes"
	"context"
	"encoding/binary"
	"math"
	"testing"
	"time"
)

// makePCM returns a little-endian 16-bit PCM buffer of the given duration
// filled with an alternating +amplitude/-amplitude square wave at the given
// normalized amplitude (0..1). Amplitude 0 produces a silent buffer.
func makePCM(sampleRate int, amplitude float64, d time.Duration) []byte {
	samples := int(int64(sampleRate) * int64(d) / int64(time.Second))
	if samples < 0 {
		samples = 0
	}
	buf := make([]byte, samples*2)
	sample := int16(math.Round(amplitude * float64(math.MaxInt16)))
	for i := 0; i < samples; i++ {
		v := sample
		if i%2 == 1 {
			v = -sample
		}
		binary.LittleEndian.PutUint16(buf[i*2:i*2+2], uint16(v))
	}
	return buf
}

func testConfig() SilenceClassifierConfig {
	return SilenceClassifierConfig{
		SampleRate:     8000,
		Channels:       1,
		WindowDuration: 10 * time.Millisecond,
		LoudThreshold:  0.3,
		QuietThreshold: 0.05,
		HoldDuration:   100 * time.Millisecond,
	}
}

func TestSilenceClassifierEmitsOnceOnLoudToQuietTransition(t *testing.T) {
	cfg := testConfig()
	cls := NewSilenceClassifier(cfg)

	loud := makePCM(cfg.SampleRate, 0.8, 200*time.Millisecond)
	quiet := makePCM(cfg.SampleRate, 0, 500*time.Millisecond)
	r := bytes.NewReader(append(loud, quiet...))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	stream, err := cls.Classify(ctx, r)
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}

	var events []SoundEvent //nolint:prealloc
	for ev := range stream {
		events = append(events, ev)
	}
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1; events = %+v", len(events), events)
	}
	if events[0].Label != SilenceAfterSoundLabel {
		t.Fatalf("event label = %q, want %q", events[0].Label, SilenceAfterSoundLabel)
	}
	// Event should fire some time after the loud segment begins winding
	// down into the quiet segment, so AtMS should be at least the hold
	// duration past the loud-to-quiet boundary.
	minAt := (200 + 100) // ms: end of loud + hold
	if events[0].AtMS < int64(minAt) {
		t.Fatalf("event AtMS = %d, want >= %d", events[0].AtMS, minAt)
	}
}

func TestSilenceClassifierDoesNotEmitForShortQuiet(t *testing.T) {
	cfg := testConfig()
	cls := NewSilenceClassifier(cfg)

	loud := makePCM(cfg.SampleRate, 0.8, 200*time.Millisecond)
	// Quiet segment shorter than HoldDuration (100ms).
	shortQuiet := makePCM(cfg.SampleRate, 0, 40*time.Millisecond)
	r := bytes.NewReader(append(loud, shortQuiet...))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	stream, err := cls.Classify(ctx, r)
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}

	var events []SoundEvent //nolint:prealloc
	for ev := range stream {
		events = append(events, ev)
	}
	if len(events) != 0 {
		t.Fatalf("event count = %d, want 0; events = %+v", len(events), events)
	}
}

func TestSilenceClassifierDoesNotEmitForPureQuiet(t *testing.T) {
	cfg := testConfig()
	cls := NewSilenceClassifier(cfg)

	// Sustained quiet with no prior loud segment should never emit, since
	// the state machine gates on a loud-to-quiet transition.
	quiet := makePCM(cfg.SampleRate, 0, 500*time.Millisecond)
	r := bytes.NewReader(quiet)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	stream, err := cls.Classify(ctx, r)
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}

	var events []SoundEvent //nolint:prealloc
	for ev := range stream {
		events = append(events, ev)
	}
	if len(events) != 0 {
		t.Fatalf("event count = %d, want 0; events = %+v", len(events), events)
	}
}

func TestSilenceClassifierDoesNotEmitForSustainedLoud(t *testing.T) {
	cfg := testConfig()
	cls := NewSilenceClassifier(cfg)

	loud := makePCM(cfg.SampleRate, 0.8, 500*time.Millisecond)
	r := bytes.NewReader(loud)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	stream, err := cls.Classify(ctx, r)
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}

	var events []SoundEvent //nolint:prealloc
	for ev := range stream {
		events = append(events, ev)
	}
	if len(events) != 0 {
		t.Fatalf("event count = %d, want 0; events = %+v", len(events), events)
	}
}

func TestSilenceClassifierAppliesDefaults(t *testing.T) {
	cls := NewSilenceClassifier(SilenceClassifierConfig{})
	if cls.cfg.SampleRate != 16000 {
		t.Fatalf("SampleRate = %d, want 16000", cls.cfg.SampleRate)
	}
	if cls.cfg.Channels != 1 {
		t.Fatalf("Channels = %d, want 1", cls.cfg.Channels)
	}
	if cls.cfg.WindowDuration != 20*time.Millisecond {
		t.Fatalf("WindowDuration = %v, want 20ms", cls.cfg.WindowDuration)
	}
	if cls.cfg.LoudThreshold <= 0 {
		t.Fatalf("LoudThreshold = %v, want > 0", cls.cfg.LoudThreshold)
	}
	if cls.cfg.QuietThreshold >= cls.cfg.LoudThreshold {
		t.Fatalf("QuietThreshold >= LoudThreshold (%v >= %v)", cls.cfg.QuietThreshold, cls.cfg.LoudThreshold)
	}
	if cls.cfg.HoldDuration != 2*time.Second {
		t.Fatalf("HoldDuration = %v, want 2s", cls.cfg.HoldDuration)
	}
	if cls.cfg.Label != SilenceAfterSoundLabel {
		t.Fatalf("Label = %q, want %q", cls.cfg.Label, SilenceAfterSoundLabel)
	}
}

func TestSilenceClassifierCanceledContextStopsImmediately(t *testing.T) {
	cfg := testConfig()
	cls := NewSilenceClassifier(cfg)

	// Use an already-canceled context to verify the goroutine exits.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	stream, err := cls.Classify(ctx, bytes.NewReader(makePCM(cfg.SampleRate, 0.8, 500*time.Millisecond)))
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}
	// Stream should close without emitting.
	done := make(chan struct{})
	drained := 0
	go func() {
		for range stream {
			drained++
		}
		close(done)
	}()
	select {
	case <-done:
		if drained != 0 {
			t.Fatalf("received %d events, want none", drained)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected stream to close on canceled context")
	}
}
