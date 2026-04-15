package io_test

import (
	"context"
	"testing"
	"time"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
)

type fakeAnalyzerRunner struct {
	started int
}

func (f *fakeAnalyzerRunner) StartAnalyzer(
	_ context.Context,
	sourceDeviceID string,
	_ string,
	emit func(iorouter.AnalyzerEvent),
) (func(), error) {
	f.started++
	emit(iorouter.AnalyzerEvent{
		Kind:       "sound.detected",
		Subject:    sourceDeviceID,
		Attributes: map[string]string{"label": "beep"},
		OccurredAt: time.Now().UTC(),
	})
	return func() {}, nil
}

func TestMediaPlannerApplyAndTear(t *testing.T) {
	router := iorouter.NewRouter()
	planner := router.MediaPlanner()

	handle, err := planner.Apply(context.Background(), iorouter.MediaPlan{
		Nodes: []iorouter.MediaNode{
			{ID: "mic", Kind: iorouter.NodeSourceMic, Args: map[string]string{"device_id": "d1"}},
			{ID: "speaker", Kind: iorouter.NodeSinkSpeaker, Args: map[string]string{"device_id": "d2"}},
		},
		Edges: []iorouter.MediaEdge{
			{From: "mic", To: "speaker"},
		},
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if router.RouteCount() != 1 {
		t.Fatalf("RouteCount() = %d, want 1", router.RouteCount())
	}

	if err := planner.Tear(context.Background(), handle); err != nil {
		t.Fatalf("Tear() error = %v", err)
	}
	if router.RouteCount() != 0 {
		t.Fatalf("RouteCount() after Tear = %d, want 0", router.RouteCount())
	}
}

func TestMediaPlannerForkAndAnalyzerEvent(t *testing.T) {
	router := iorouter.NewRouter()
	planner := router.MediaPlanner()

	runner := &fakeAnalyzerRunner{}
	planner.SetAnalyzerRunner(runner)

	events := make([]iorouter.AnalyzerEvent, 0)
	planner.SetAnalyzerSink(func(event iorouter.AnalyzerEvent) {
		events = append(events, event)
	})

	_, err := planner.Apply(context.Background(), iorouter.MediaPlan{
		Nodes: []iorouter.MediaNode{
			{ID: "mic", Kind: iorouter.NodeSourceMic, Args: map[string]string{"device_id": "d1"}},
			{ID: "fork", Kind: iorouter.NodeFork},
			{ID: "stt", Kind: iorouter.NodeSinkSTT, Args: map[string]string{"device_id": "server"}},
			{ID: "speaker", Kind: iorouter.NodeSinkSpeaker, Args: map[string]string{"device_id": "d2"}},
			{ID: "analyze", Kind: iorouter.NodeAnalyzer, Args: map[string]string{"name": "sound"}},
		},
		Edges: []iorouter.MediaEdge{
			{From: "mic", To: "fork"},
			{From: "fork", To: "stt"},
			{From: "fork", To: "speaker"},
			{From: "fork", To: "analyze"},
		},
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if runner.started != 1 {
		t.Fatalf("analyzer starts = %d, want 1", runner.started)
	}
	if len(events) != 1 || events[0].Kind != "sound.detected" {
		t.Fatalf("events = %+v, want one sound.detected", events)
	}
	if router.RouteCount() != 2 {
		t.Fatalf("RouteCount() = %d, want 2 (stt + speaker)", router.RouteCount())
	}
}
