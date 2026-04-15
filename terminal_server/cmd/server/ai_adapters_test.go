package main

import (
	"bytes"
	"context"
	"errors"
	"image"
	"io"
	"reflect"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/ai"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
)

type fakeVisionBackend struct {
	response *ai.VisionAnalysis
	err      error
	prompt   string
}

func (f *fakeVisionBackend) Analyze(_ context.Context, _ image.Image, prompt string) (*ai.VisionAnalysis, error) {
	f.prompt = prompt
	if f.err != nil {
		return nil, f.err
	}
	return f.response, nil
}

type fakeSoundBackend struct {
	events  []ai.SoundEvent
	err     error
	captured []byte
}

func (f *fakeSoundBackend) Classify(_ context.Context, audio io.Reader) (<-chan ai.SoundEvent, error) {
	if audio != nil {
		f.captured, _ = io.ReadAll(audio)
	}
	if f.err != nil {
		return nil, f.err
	}
	out := make(chan ai.SoundEvent, len(f.events))
	for _, event := range f.events {
		out <- event
	}
	close(out)
	return out, nil
}

func TestScenarioVisionAnalyzerAnalyzeMapsResponse(t *testing.T) {
	backend := &fakeVisionBackend{response: &ai.VisionAnalysis{Caption: "kitchen", Labels: []string{"sink", "window"}}}
	adapter := scenarioVisionAnalyzer{backend: backend}

	frame := image.NewRGBA(image.Rect(0, 0, 2, 2))
	got, err := adapter.Analyze(context.Background(), frame, "describe")
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if backend.prompt != "describe" {
		t.Fatalf("prompt = %q, want describe", backend.prompt)
	}
	if got == nil {
		t.Fatalf("Analyze() response = nil, want non-nil")
	}
	if got.Caption != "kitchen" {
		t.Fatalf("Caption = %q, want kitchen", got.Caption)
	}
	if !reflect.DeepEqual(got.Labels, []string{"sink", "window"}) {
		t.Fatalf("Labels = %+v, want [sink window]", got.Labels)
	}
}

func TestScenarioVisionAnalyzerAnalyzePropagatesError(t *testing.T) {
	adapter := scenarioVisionAnalyzer{backend: &fakeVisionBackend{err: errors.New("vision unavailable")}}
	_, err := adapter.Analyze(context.Background(), image.NewRGBA(image.Rect(0, 0, 1, 1)), "describe")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestScenarioSoundClassifierClassifyMapsStream(t *testing.T) {
	backend := &fakeSoundBackend{events: []ai.SoundEvent{{Label: "beep", Confidence: 0.9, AtMS: 1200}}}
	adapter := scenarioSoundClassifier{backend: backend}

	stream, err := adapter.Classify(context.Background(), bytes.NewBufferString("pcm"))
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}
	if string(backend.captured) != "pcm" {
		t.Fatalf("captured audio = %q, want pcm", string(backend.captured))
	}

	var events []scenario.SoundEvent //nolint:prealloc
	for event := range stream {
		events = append(events, event)
	}
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1", len(events))
	}
	if events[0].Label != "beep" || events[0].AtMS != 1200 {
		t.Fatalf("event = %+v, want mapped beep event", events[0])
	}
}

func TestScenarioSoundClassifierClassifyPropagatesError(t *testing.T) {
	adapter := scenarioSoundClassifier{backend: &fakeSoundBackend{err: errors.New("sound unavailable")}}
	_, err := adapter.Classify(context.Background(), bytes.NewBufferString("pcm"))
	if err == nil {
		t.Fatalf("expected error")
	}
}
