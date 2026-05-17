package usecasevalidation_test

import (
	"path/filepath"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/usecasevalidation"
)

func TestLoadScenarioFileT2(t *testing.T) {
	path := filepath.Join("testdata", "t2-timer-reminder.yaml")
	spec, err := usecasevalidation.LoadScenarioFile(path)
	if err != nil {
		t.Fatalf("LoadScenarioFile: %v", err)
	}
	if spec.ID != "t2-timer-reminder" {
		t.Fatalf("id = %q, want t2-timer-reminder", spec.ID)
	}
	if len(spec.Usecases) != 1 || spec.Usecases[0] != "T2" {
		t.Fatalf("usecases = %v, want [T2]", spec.Usecases)
	}
	if len(spec.Steps) < 5 {
		t.Fatalf("steps = %d, want at least 5", len(spec.Steps))
	}
}

// TestYAMLScenarioT2TimerReminder runs the T2 timer-reminder story from YAML.
// Phase 4 acceptance: at least one YAML scenario passes make usecase-validate.
func TestYAMLScenarioT2TimerReminder(t *testing.T) {
	h := usecasevalidation.New(t)
	spec := h.RunScenarioFile(t, usecasevalidation.ScenarioFilePath("t2-timer-reminder.yaml"))
	if len(spec.Usecases) == 0 {
		t.Fatal("scenario missing usecases")
	}
	h.Evidence(spec.Usecases[0])
}
