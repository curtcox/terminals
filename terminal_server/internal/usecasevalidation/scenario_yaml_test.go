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
	runYAMLScenario(t, "t2-timer-reminder.yaml")
}

func TestYAMLScenarioT3T4SchoolMorning(t *testing.T) {
	runYAMLScenario(t, "t3-t4-school-morning.yaml")
}

func TestYAMLScenarioT3ActivityCancelsAlert(t *testing.T) {
	runYAMLScenario(t, "t3-activity-cancels-alert.yaml")
}

func TestYAMLScenarioAA1WebhookAnnounce(t *testing.T) {
	runYAMLScenario(t, "aa1-webhook-announce.yaml")
}

func TestYAMLScenarioAA4TimerCancel(t *testing.T) {
	runYAMLScenario(t, "aa4-timer-cancel.yaml")
}

func runYAMLScenario(t *testing.T, name string) {
	t.Helper()
	h := usecasevalidation.New(t)
	spec := h.RunScenarioFile(t, usecasevalidation.ScenarioFilePath(name))
	if len(spec.Usecases) == 0 {
		t.Fatal("scenario missing usecases")
	}
	label := spec.Usecases[0]
	if len(spec.Usecases) > 1 {
		label = spec.Usecases[0] + "/" + spec.Usecases[1]
	}
	h.Evidence(label)
}
