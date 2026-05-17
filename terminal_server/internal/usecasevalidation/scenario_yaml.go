package usecasevalidation

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	"github.com/curtcox/terminals/terminal_server/internal/transport"
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
	Connect          map[string]any    `yaml:"connect,omitempty"`
	Command          *CommandStep      `yaml:"command,omitempty"`
	Says             *SaysStep         `yaml:"says,omitempty"`
	ClockAdvance     *DurationStep     `yaml:"clock_advance,omitempty"`
	ClockAdvanceTo   *TimeStep         `yaml:"clock_advance_to,omitempty"`
	ProcessDueTimers *ProcessTimersStep `yaml:"process_due_timers,omitempty"`
	Expect           *ExpectStep       `yaml:"expect,omitempty"`
	Disconnect       *DisconnectStep   `yaml:"disconnect,omitempty"`
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
	ID                string `yaml:"id"`
	Description       string `yaml:"description,omitempty"`
	Terminal          string `yaml:"terminal,omitempty"`
	ScenarioStart     string `yaml:"scenario_start,omitempty"`
	RouteKind         string `yaml:"route_kind,omitempty"`
	BroadcastContains string `yaml:"broadcast_contains,omitempty"`
	TimersProcessed   *int   `yaml:"timers_processed,omitempty"`
}

// DisconnectStep closes one or all terminal sessions.
type DisconnectStep struct {
	Terminal string `yaml:"terminal,omitempty"`
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

func (h *Harness) runScenario(t *testing.T, spec *ScenarioFile) {
	t.Helper()
	if spec.Clock.Start != "" {
		start, err := time.Parse(time.RFC3339, spec.Clock.Start)
		if err != nil {
			t.Fatalf("scenario %s: clock.start: %v", spec.ID, err)
		}
		h.Clock().SetNow(start)
	}

	terminals := make(map[string]*SimTerminal)
	profiles := make(map[string]TerminalProfile, len(spec.Terminals))

	for alias, prof := range spec.Terminals {
		deviceID := prof.DeviceID
		if deviceID == "" {
			deviceID = alias
		}
		name := prof.Name
		if name == "" {
			name = deviceID
		}
		profiles[alias] = TerminalProfile{DeviceID: deviceID, Name: name}
	}

	h.StartServer()

	for i, step := range spec.Steps {
		switch {
		case step.Connect != nil:
			for alias, prof := range profiles {
				if _, ok := terminals[alias]; ok {
					continue
				}
				term := h.connectProfile(prof)
				if !term.WaitForAny(scenarioStepTimeout) {
					t.Fatalf("scenario %s step %d: terminal %s timed out connecting", spec.ID, i, alias)
				}
				terminals[alias] = term
			}
		case step.Command != nil:
			term := terminals[step.Command.Terminal]
			if term == nil {
				t.Fatalf("scenario %s step %d: unknown terminal %q", spec.ID, i, step.Command.Terminal)
			}
			args := copyStringMap(step.Command.Arguments)
			if args == nil {
				args = map[string]string{}
			}
			if _, hasFire := args["fire_unix_ms"]; !hasFire {
				if durStr, ok := args["duration_seconds"]; ok && durStr != "" {
					secs, err := strconv.Atoi(durStr)
					if err != nil {
						t.Fatalf("scenario %s step %d: duration_seconds: %v", spec.ID, i, err)
					}
					fire := h.Clock().Now().Add(time.Duration(secs) * time.Second)
					args["fire_unix_ms"] = strconv.FormatInt(fire.UnixMilli(), 10)
				}
			}
			term.Send(&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: fmt.Sprintf("%s-cmd-%d", spec.ID, i),
						DeviceId:  term.DeviceID,
						Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
						Intent:    step.Command.Intent,
						Arguments: args,
					},
				},
			})
		case step.Says != nil:
			term := terminals[step.Says.Terminal]
			if term == nil {
				t.Fatalf("scenario %s step %d: unknown terminal %q", spec.ID, i, step.Says.Terminal)
			}
			term.Send(&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: fmt.Sprintf("%s-says-%d", spec.ID, i),
						DeviceId:  term.DeviceID,
						Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
						Text:      step.Says.Text,
					},
				},
			})
		case step.ClockAdvance != nil:
			d, err := time.ParseDuration(step.ClockAdvance.Duration)
			if err != nil {
				t.Fatalf("scenario %s step %d: clock_advance.duration: %v", spec.ID, i, err)
			}
			h.Clock().Advance(d)
		case step.ClockAdvanceTo != nil:
			target, err := time.Parse(time.RFC3339, step.ClockAdvanceTo.Time)
			if err != nil {
				t.Fatalf("scenario %s step %d: clock_advance_to.time: %v", spec.ID, i, err)
			}
			h.Clock().AdvanceTo(target)
		case step.ProcessDueTimers != nil:
			processed, err := h.ProcessDueTimers(context.Background())
			assertID := step.ProcessDueTimers.AssertID
			if assertID == "" {
				assertID = "process-due-timers"
			}
			if step.ProcessDueTimers.ExpectProcessed != nil {
				want := *step.ProcessDueTimers.ExpectProcessed
				h.Assert(assertID, fmt.Sprintf("process due timers: want %d processed", want),
					err == nil && processed == want,
					fmt.Sprintf("processed=%d err=%v", processed, err))
			} else if err != nil {
				t.Fatalf("scenario %s step %d: ProcessDueTimers: %v", spec.ID, i, err)
			}
		case step.Expect != nil:
			h.runExpectStep(t, spec.ID, i, step.Expect, terminals)
		case step.Disconnect != nil:
			if step.Disconnect.Terminal == "" {
				for _, term := range terminals {
					_ = term.Disconnect()
				}
				terminals = make(map[string]*SimTerminal)
			} else {
				term := terminals[step.Disconnect.Terminal]
				if term == nil {
					t.Fatalf("scenario %s step %d: unknown terminal %q", spec.ID, i, step.Disconnect.Terminal)
				}
				_ = term.Disconnect()
				delete(terminals, step.Disconnect.Terminal)
			}
		default:
			t.Fatalf("scenario %s step %d: unsupported or empty step", spec.ID, i)
		}
	}
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

func (h *Harness) runExpectStep(t *testing.T, scenarioID string, stepIdx int, exp *ExpectStep, terminals map[string]*SimTerminal) {
	t.Helper()
	id := exp.ID
	if id == "" {
		id = fmt.Sprintf("%s-expect-%d", scenarioID, stepIdx)
	}
	desc := exp.Description
	if desc == "" {
		desc = id
	}

	pass := true
	var detail string

	if exp.Terminal != "" && exp.ScenarioStart != "" {
		term := terminals[exp.Terminal]
		if term == nil {
			t.Fatalf("scenario %s step %d: unknown terminal %q", scenarioID, stepIdx, exp.Terminal)
		}
		_, saw := term.WaitFor(func(env transport.ProtoServerEnvelope) bool {
			resp, ok := env.(*controlv1.ConnectResponse)
			return ok && resp.GetCommandResult() != nil &&
				resp.GetCommandResult().GetScenarioStart() == exp.ScenarioStart
		}, scenarioStepTimeout)
		if !saw {
			pass = false
			detail = fmt.Sprintf("terminal %s: scenario_start %q not seen (%d messages)",
				exp.Terminal, exp.ScenarioStart, len(term.Received()))
		}
	}

	if exp.Terminal != "" && exp.RouteKind != "" {
		term := terminals[exp.Terminal]
		_, saw := term.WaitFor(func(env transport.ProtoServerEnvelope) bool {
			resp, ok := env.(*controlv1.ConnectResponse)
			if !ok {
				return false
			}
			r := resp.GetRouteStream()
			return r != nil && r.GetKind() == exp.RouteKind
		}, scenarioStepTimeout)
		if !saw {
			pass = false
			if detail != "" {
				detail += "; "
			}
			detail += fmt.Sprintf("terminal %s: route_kind %q not seen", exp.Terminal, exp.RouteKind)
		}
	}

	if exp.BroadcastContains != "" {
		found := false
		for _, ev := range h.Broadcast.Events() {
			if ev.Message == exp.BroadcastContains {
				found = true
				break
			}
		}
		if !found {
			pass = false
			if detail != "" {
				detail += "; "
			}
			detail += fmt.Sprintf("broadcast missing %q (%d events)", exp.BroadcastContains, len(h.Broadcast.Events()))
		}
	}

	if exp.TimersProcessed != nil {
		processed, err := h.ProcessDueTimers(context.Background())
		want := *exp.TimersProcessed
		if err != nil || processed != want {
			pass = false
			if detail != "" {
				detail += "; "
			}
			detail += fmt.Sprintf("timers processed=%d want=%d err=%v", processed, want, err)
		}
	}

	h.Assert(id, desc, pass, detail)
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
