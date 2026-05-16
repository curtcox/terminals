package usecasevalidation

import (
	"context"
	"image"
	"sync"

	"github.com/curtcox/terminals/terminal_server/internal/scenario"
)

// FakeVisionAnalyzer is a test double for scenario.VisionAnalyzer that returns
// a pre-configured analysis for every call. The image passed to Analyze is
// ignored; results are returned immediately. Use this to inject deterministic
// vision analysis results into harness-based scenario tests.
type FakeVisionAnalyzer struct {
	Caption string
	Labels  []string

	mu    sync.Mutex
	calls int
}

// Analyze returns the configured caption and labels and records the call.
func (f *FakeVisionAnalyzer) Analyze(_ context.Context, _ image.Image, _ string) (*scenario.VisionAnalysis, error) {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()
	labels := make([]string, len(f.Labels))
	copy(labels, f.Labels)
	return &scenario.VisionAnalysis{Caption: f.Caption, Labels: labels}, nil
}

// Calls returns the number of times Analyze was called.
func (f *FakeVisionAnalyzer) Calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}
