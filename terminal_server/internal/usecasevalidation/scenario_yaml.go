package usecasevalidation

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	"gopkg.in/yaml.v3"
)

const scenarioStepTimeout = 2 * time.Second

// TerminalProfile describes a simulated terminal in a YAML scenario.
type TerminalProfile struct {
	DeviceID string `yaml:"device_id"`
	Name     string `yaml:"name"`
}

// ScenarioFile is the parsed YAML scenario specification.
type ScenarioFile struct {
	ID        string                     `yaml:"id"`
	Usecases  []string                   `yaml:"usecases"`
	Clock     ScenarioClock              `yaml:"clock"`
	Terminals map[string]TerminalProfile `yaml:"terminals"`
	Steps     []ScenarioStep             `yaml:"steps"`
}

// ScenarioClock configures synthetic time for the scenario.
type ScenarioClock struct {
	Start string `yaml:"start"`
}

// ScenarioStep is one ordered step in a YAML scenario.
type ScenarioStep struct {
	Connect          map[string]any     `yaml:"connect,omitempty"`
	Command          *CommandStep       `yaml:"command,omitempty"`
	Says             *SaysStep          `yaml:"says,omitempty"`
	Sensor           *SensorStep        `yaml:"sensor,omitempty"`
	ClockAdvance     *DurationStep      `yaml:"clock_advance,omitempty"`
	ClockAdvanceTo   *TimeStep          `yaml:"clock_advance_to,omitempty"`
	ProcessDueTimers *ProcessTimersStep `yaml:"process_due_timers,omitempty"`
	MarkBroadcast    map[string]any     `yaml:"mark_broadcast,omitempty"`
	Yield            *YieldStep         `yaml:"yield,omitempty"`
	Expect           *ExpectStep        `yaml:"expect,omitempty"`
	Disconnect       *DisconnectStep    `yaml:"disconnect,omitempty"`
}

// CommandStep sends COMMAND_KIND_MANUAL with an intent and optional arguments.
type CommandStep struct {
	Terminal  string            `yaml:"terminal"`
	Intent    string            `yaml:"intent"`
	Arguments map[string]string `yaml:"arguments,omitempty"`
}

// SaysStep sends COMMAND_KIND_VOICE with the given transcript text.
type SaysStep struct {
	Terminal string `yaml:"terminal"`
	Text     string `yaml:"text"`
}

// SensorStep injects a sensor reading from a terminal at the current synthetic time.
type SensorStep struct {
	Terminal string             `yaml:"terminal"`
	Values   map[string]float64 `yaml:"values"`
}

// DurationStep advances the fake clock by a Go duration string.
type DurationStep struct {
	Duration string `yaml:"duration"`
}

// TimeStep advances the fake clock to an absolute RFC3339 time.
type TimeStep struct {
	Time string `yaml:"time"`
}

// ProcessTimersStep runs ProcessDueTimers at the current synthetic time.
type ProcessTimersStep struct {
	ExpectProcessed *int   `yaml:"expect_processed,omitempty"`
	AssertID        string `yaml:"assert_id,omitempty"`
}

// ExpectStep records one or more harness assertions.
type ExpectStep struct {
	ID                   string `yaml:"id"`
	Description          string `yaml:"description,omitempty"`
	Terminal             string `yaml:"terminal,omitempty"`
	ScenarioStart        string `yaml:"scenario_start,omitempty"`
	RouteKind            string `yaml:"route_kind,omitempty"`
	BroadcastContains    string `yaml:"broadcast_contains,omitempty"`
	BroadcastNotContains string `yaml:"broadcast_not_contains,omitempty"`
	BroadcastSinceMark   bool   `yaml:"broadcast_since_mark,omitempty"`
	TimersProcessed      *int   `yaml:"timers_processed,omitempty"`
}

// DisconnectStep closes one or all terminal sessions.
type DisconnectStep struct {
	Terminal string `yaml:"terminal,omitempty"`
}

// YieldStep waits briefly so async session handlers can settle. Prefer short
// durations; scenario timing should still be driven by the fake clock.
type YieldStep struct {
	Duration string `yaml:"duration,omitempty"`
}

// LoadScenarioFile reads and parses a YAML scenario from path.
func LoadScenarioFile(path string) (*ScenarioFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var spec ScenarioFile
	if err := yaml.Unmarshal(raw, &spec); err != nil {
		return nil, err
	}
	if spec.ID == "" {
		return nil, fmt.Errorf("scenario %s: missing id", path)
	}
	if len(spec.Terminals) == 0 {
		return nil, fmt.Errorf("scenario %s: missing terminals", spec.ID)
	}
	if len(spec.Steps) == 0 {
		return nil, fmt.Errorf("scenario %s: missing steps", spec.ID)
	}
	return &spec, nil
}

// RunScenarioFile executes a YAML scenario file against the harness.
func (h *Harness) RunScenarioFile(t *testing.T, path string) *ScenarioFile {
	t.Helper()
	spec, err := LoadScenarioFile(path)
	if err != nil {
		t.Fatalf("load scenario %s: %v", path, err)
	}
	h.runScenario(t, spec)
	return spec
}

func (h *Harness) connectProfile(prof TerminalProfile) *SimTerminal {
	return h.ConnectTerminal(prof.DeviceID, &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: prof.DeviceID,
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: prof.Name},
				},
			},
		},
	})
}

// ScenarioFilePath resolves a path under testdata/ next to this package.
func ScenarioFilePath(name string) string {
	return filepath.Join("testdata", name)
}

func copyStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func describeCommandInteraction(step *CommandStep, args map[string]string) string {
	if step == nil {
		return ""
	}
	parts := []string{fmt.Sprintf("Run %q", step.Intent)}
	keys := make([]string, 0, len(args))
	for key := range args {
		if key == "fire_unix_ms" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) > 0 {
		values := make([]string, 0, len(keys))
		for _, key := range keys {
			values = append(values, fmt.Sprintf("%s=%q", key, args[key]))
		}
		parts = append(parts, "with "+strings.Join(values, ", "))
	}
	return strings.Join(parts, " ") + "."
}

func describeSensorInteraction(step *SensorStep) string {
	if step == nil {
		return ""
	}
	keys := make([]string, 0, len(step.Values))
	for key := range step.Values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		values = append(values, fmt.Sprintf("%s=%g", key, step.Values[key]))
	}
	return "Trigger sensor input: " + strings.Join(values, ", ") + "."
}
