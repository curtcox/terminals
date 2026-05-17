package usecasevalidation

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
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
	broadcastMark := 0

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
			h.RecordInteraction("command", describeCommandInteraction(step.Command, args), step.Command.Terminal)
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
			h.RecordInteraction("voice", fmt.Sprintf("Say %q.", step.Says.Text), step.Says.Terminal)
		case step.Sensor != nil:
			term := terminals[step.Sensor.Terminal]
			if term == nil {
				t.Fatalf("scenario %s step %d: unknown terminal %q", spec.ID, i, step.Sensor.Terminal)
			}
			if len(step.Sensor.Values) == 0 {
				t.Fatalf("scenario %s step %d: sensor.values required", spec.ID, i)
			}
			term.Send(SensorDataRequest(term.DeviceID, h.Clock().Now().UnixMilli(), step.Sensor.Values))
			h.RecordInteraction("sensor", describeSensorInteraction(step.Sensor), step.Sensor.Terminal)
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
		case step.MarkBroadcast != nil:
			broadcastMark = len(h.Broadcast.Events())
		case step.Yield != nil:
			d := 50 * time.Millisecond
			if step.Yield.Duration != "" {
				parsed, err := time.ParseDuration(step.Yield.Duration)
				if err != nil {
					t.Fatalf("scenario %s step %d: yield.duration: %v", spec.ID, i, err)
				}
				d = parsed
			}
			time.Sleep(d)
		case step.Expect != nil:
			h.runExpectStep(t, spec.ID, i, step.Expect, terminals, broadcastMark)
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

func (h *Harness) runExpectStep(t *testing.T, scenarioID string, stepIdx int, exp *ExpectStep, terminals map[string]*SimTerminal, broadcastMark int) {
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

	events := h.Broadcast.Events()
	if exp.BroadcastSinceMark {
		events = events[broadcastMark:]
	}

	if exp.BroadcastContains != "" {
		found := false
		for _, ev := range events {
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
			detail += fmt.Sprintf("broadcast missing %q (%d events)", exp.BroadcastContains, len(events))
		}
	}

	if exp.BroadcastNotContains != "" {
		for _, ev := range events {
			if ev.Message == exp.BroadcastNotContains {
				pass = false
				if detail != "" {
					detail += "; "
				}
				detail += fmt.Sprintf("broadcast unexpectedly contains %q", exp.BroadcastNotContains)
				break
			}
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
