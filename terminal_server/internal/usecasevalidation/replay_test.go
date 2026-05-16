package usecasevalidation_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/usecasevalidation"
)

var bundleFlag = flag.String("bundle", "", "path to evidence bundle directory for replay")

// TestReplay reads a saved evidence bundle and reports its contents.
//
// Usage:
//
//	go test ./internal/usecasevalidation -run TestReplay -args -bundle artifacts/usecase-validation/<run-id>
func TestReplay(t *testing.T) {
	if *bundleFlag == "" {
		t.Skip("replay: no -bundle flag; run as: go test ./internal/usecasevalidation -run TestReplay -args -bundle <path>")
	}
	dir := *bundleFlag

	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		t.Fatalf("replay: bundle directory not found: %s", dir)
	}

	// Read manifest.json.
	manifestPath := filepath.Join(dir, "manifest.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("replay: could not read manifest.json: %v", err)
	}
	var m usecasevalidation.Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("replay: could not parse manifest.json: %v", err)
	}

	result := "PASS"
	if !m.Pass {
		result = "FAIL"
	}
	t.Logf("=== Bundle Replay ===")
	t.Logf("Use case:  %s", m.UseCaseID)
	t.Logf("Scenario:  %s", m.ScenarioName)
	t.Logf("Run ID:    %s", m.RunID)
	t.Logf("Result:    %s", result)
	t.Logf("Start:     %s", m.TimestampStart.Format("2006-01-02T15:04:05Z"))
	t.Logf("End:       %s", m.TimestampEnd.Format("2006-01-02T15:04:05Z"))
	if len(m.FailingAssertions) > 0 {
		t.Logf("Failing:   %v", m.FailingAssertions)
	}

	// Read assertions.jsonl if present.
	assertPath := filepath.Join(dir, "assertions.jsonl")
	if assertData, err := os.ReadFile(assertPath); err == nil {
		t.Logf("--- Assertions ---")
		dec := json.NewDecoder(bytes.NewReader(assertData))
		for dec.More() {
			var a usecasevalidation.AssertionRecord
			if err := dec.Decode(&a); err != nil {
				break
			}
			mark := "PASS"
			if !a.Pass {
				mark = "FAIL"
			}
			if a.Detail != "" {
				t.Logf("  [%s] %s: %s — %s", mark, a.ID, a.Description, a.Detail)
			} else {
				t.Logf("  [%s] %s: %s", mark, a.ID, a.Description)
			}
		}
	}

	// Print summary.md if present.
	summaryPath := filepath.Join(dir, "summary.md")
	if summaryData, err := os.ReadFile(summaryPath); err == nil {
		t.Logf("--- Summary ---\n%s", string(summaryData))
	}

	if !m.Pass {
		t.Errorf("replayed bundle reports failure: failing assertions: %v", m.FailingAssertions)
	}
}
