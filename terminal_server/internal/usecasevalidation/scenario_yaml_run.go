package usecasevalidation

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	"github.com/curtcox/terminals/terminal_server/internal/transport"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

type scenarioRunState struct {
	spec          *ScenarioFile
	terminals     map[string]*SimTerminal
	profiles      map[string]TerminalProfile
	broadcastMark int
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

	state := scenarioRunState{
		spec:      spec,
		terminals: make(map[string]*SimTerminal),
		profiles:  make(map[string]TerminalProfile, len(spec.Terminals)),
	}
	for alias, prof := range spec.Terminals {
		deviceID := prof.DeviceID
		if deviceID == "" {
			deviceID = alias
		}
		name := prof.Name
		if name == "" {
			name = deviceID
		}
		state.profiles[alias] = TerminalProfile{DeviceID: deviceID, Name: name}
	}

	h.StartServer()

	for i, step := range spec.Steps {
		if !h.runScenarioStep(t, &state, i, step) {
			t.Fatalf("scenario %s step %d: unsupported or empty step", spec.ID, i)
		}
	}
}

func (h *Harness) runScenarioStep(t *testing.T, state *scenarioRunState, i int, step ScenarioStep) bool {
	t.Helper()
	spec := state.spec
	switch {
	case step.Connect != nil:
		h.runScenarioConnectStep(t, state, i)
	case step.Command != nil:
		h.runScenarioCommandStep(t, state, i, step.Command)
	case step.Says != nil:
		h.runScenarioSaysStep(t, state, i, step.Says)
	case step.Sensor != nil:
		h.runScenarioSensorStep(t, state, i, step.Sensor)
	case step.ClockAdvance != nil:
		h.runScenarioClockAdvanceStep(t, spec.ID, i, step.ClockAdvance)
	case step.ClockAdvanceTo != nil:
		h.runScenarioClockAdvanceToStep(t, spec.ID, i, step.ClockAdvanceTo)
	case step.ProcessDueTimers != nil:
		h.runScenarioProcessDueTimersStep(t, spec.ID, i, step.ProcessDueTimers)
	case step.MarkBroadcast != nil:
		state.broadcastMark = len(h.Broadcast.Events())
	case step.Yield != nil:
		h.runScenarioYieldStep(t, spec.ID, i, step.Yield)
	case step.Expect != nil:
		h.runExpectStep(t, spec.ID, i, step.Expect, state.terminals, state.broadcastMark)
	case step.Disconnect != nil:
		h.runScenarioDisconnectStep(t, state, i, step.Disconnect)
	default:
		return false
	}
	return true
}

func (h *Harness) runScenarioConnectStep(t *testing.T, state *scenarioRunState, i int) {
	t.Helper()
	spec := state.spec
	for alias, prof := range state.profiles {
		if _, ok := state.terminals[alias]; ok {
			continue
		}
		term := h.connectProfile(prof)
		if !term.WaitForAny(scenarioStepTimeout) {
			t.Fatalf("scenario %s step %d: terminal %s timed out connecting", spec.ID, i, alias)
		}
		state.terminals[alias] = term
	}
}

func (h *Harness) runScenarioCommandStep(t *testing.T, state *scenarioRunState, i int, cmd *CommandStep) {
	t.Helper()
	spec := state.spec
	term := state.terminals[cmd.Terminal]
	if term == nil {
		t.Fatalf("scenario %s step %d: unknown terminal %q", spec.ID, i, cmd.Terminal)
	}
	args := copyStringMap(cmd.Arguments)
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
				Intent:    cmd.Intent,
				Arguments: args,
			},
		},
	})
	h.RecordInteraction("command", describeCommandInteraction(cmd, args), cmd.Terminal)
}

func (h *Harness) runScenarioSaysStep(t *testing.T, state *scenarioRunState, i int, says *SaysStep) {
	t.Helper()
	spec := state.spec
	term := state.terminals[says.Terminal]
	if term == nil {
		t.Fatalf("scenario %s step %d: unknown terminal %q", spec.ID, i, says.Terminal)
	}
	term.Send(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: fmt.Sprintf("%s-says-%d", spec.ID, i),
				DeviceId:  term.DeviceID,
				Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
				Text:      says.Text,
			},
		},
	})
	h.RecordInteraction("voice", fmt.Sprintf("Say %q.", says.Text), says.Terminal)
}

func (h *Harness) runScenarioSensorStep(t *testing.T, state *scenarioRunState, i int, sensor *SensorStep) {
	t.Helper()
	spec := state.spec
	term := state.terminals[sensor.Terminal]
	if term == nil {
		t.Fatalf("scenario %s step %d: unknown terminal %q", spec.ID, i, sensor.Terminal)
	}
	if len(sensor.Values) == 0 {
		t.Fatalf("scenario %s step %d: sensor.values required", spec.ID, i)
	}
	term.Send(SensorDataRequest(term.DeviceID, h.Clock().Now().UnixMilli(), sensor.Values))
	h.RecordInteraction("sensor", describeSensorInteraction(sensor), sensor.Terminal)
}

func (h *Harness) runScenarioClockAdvanceStep(t *testing.T, scenarioID string, i int, step *DurationStep) {
	t.Helper()
	d, err := time.ParseDuration(step.Duration)
	if err != nil {
		t.Fatalf("scenario %s step %d: clock_advance.duration: %v", scenarioID, i, err)
	}
	h.Clock().Advance(d)
}

func (h *Harness) runScenarioClockAdvanceToStep(t *testing.T, scenarioID string, i int, step *TimeStep) {
	t.Helper()
	target, err := time.Parse(time.RFC3339, step.Time)
	if err != nil {
		t.Fatalf("scenario %s step %d: clock_advance_to.time: %v", scenarioID, i, err)
	}
	h.Clock().AdvanceTo(target)
}

func (h *Harness) runScenarioProcessDueTimersStep(t *testing.T, scenarioID string, i int, step *ProcessTimersStep) {
	t.Helper()
	processed, err := h.ProcessDueTimers(context.Background())
	assertID := step.AssertID
	if assertID == "" {
		assertID = "process-due-timers"
	}
	if step.ExpectProcessed != nil {
		want := *step.ExpectProcessed
		h.Assert(assertID, fmt.Sprintf("process due timers: want %d processed", want),
			err == nil && processed == want,
			fmt.Sprintf("processed=%d err=%v", processed, err))
	} else if err != nil {
		t.Fatalf("scenario %s step %d: ProcessDueTimers: %v", scenarioID, i, err)
	}
}

func (h *Harness) runScenarioYieldStep(t *testing.T, scenarioID string, i int, step *YieldStep) {
	t.Helper()
	d := 50 * time.Millisecond
	if step.Duration != "" {
		parsed, err := time.ParseDuration(step.Duration)
		if err != nil {
			t.Fatalf("scenario %s step %d: yield.duration: %v", scenarioID, i, err)
		}
		d = parsed
	}
	time.Sleep(d)
}

func (h *Harness) runScenarioDisconnectStep(t *testing.T, state *scenarioRunState, i int, step *DisconnectStep) {
	t.Helper()
	spec := state.spec
	if step.Terminal == "" {
		for _, term := range state.terminals {
			_ = term.Disconnect()
		}
		state.terminals = make(map[string]*SimTerminal)
		return
	}
	term := state.terminals[step.Terminal]
	if term == nil {
		t.Fatalf("scenario %s step %d: unknown terminal %q", spec.ID, i, step.Terminal)
	}
	_ = term.Disconnect()
	delete(state.terminals, step.Terminal)
}

func (h *Harness) runExpectStep(t *testing.T, scenarioID string, stepIdx int, exp *ExpectStep, terminals map[string]*SimTerminal, broadcastMark int) {
	t.Helper()
	id, desc := expectStepMeta(scenarioID, stepIdx, exp)
	pass, detail := true, ""
	pass, detail = h.expectScenarioStart(t, scenarioID, stepIdx, exp, terminals, pass, detail)
	pass, detail = h.expectRouteKind(t, scenarioID, stepIdx, exp, terminals, pass, detail)
	events := h.expectBroadcastEvents(exp, broadcastMark)
	pass, detail = h.expectBroadcastContains(exp, events, pass, detail)
	pass, detail = h.expectBroadcastNotContains(exp, events, pass, detail)
	pass, detail = h.expectTimersProcessed(exp, pass, detail)

	h.Assert(id, desc, pass, detail)
	if exp.Terminal != "" {
		if term := terminals[exp.Terminal]; term != nil {
			msgs := term.Received()
			h.CaptureFrame(id, exp.Terminal, msgs)
			h.CaptureAudio(id, exp.Terminal, msgs)
		}
	}
}

func expectStepMeta(scenarioID string, stepIdx int, exp *ExpectStep) (id, desc string) {
	id = exp.ID
	if id == "" {
		id = fmt.Sprintf("%s-expect-%d", scenarioID, stepIdx)
	}
	desc = exp.Description
	if desc == "" {
		desc = id
	}
	return id, desc
}

func (h *Harness) expectScenarioStart(t *testing.T, scenarioID string, stepIdx int, exp *ExpectStep, terminals map[string]*SimTerminal, pass bool, detail string) (bool, string) {
	t.Helper()
	if exp.Terminal == "" || exp.ScenarioStart == "" {
		return pass, detail
	}
	term := terminals[exp.Terminal]
	if term == nil {
		t.Fatalf("scenario %s step %d: unknown terminal %q", scenarioID, stepIdx, exp.Terminal)
	}
	_, saw := term.WaitFor(func(env transport.ProtoServerEnvelope) bool {
		resp, ok := env.(*controlv1.ConnectResponse)
		return ok && resp.GetCommandResult() != nil &&
			resp.GetCommandResult().GetScenarioStart() == exp.ScenarioStart
	}, scenarioStepTimeout)
	if saw {
		return pass, detail
	}
	return false, fmt.Sprintf("terminal %s: scenario_start %q not seen (%d messages)",
		exp.Terminal, exp.ScenarioStart, len(term.Received()))
}

func (h *Harness) expectRouteKind(t *testing.T, scenarioID string, stepIdx int, exp *ExpectStep, terminals map[string]*SimTerminal, pass bool, detail string) (bool, string) {
	if exp.Terminal == "" || exp.RouteKind == "" {
		return pass, detail
	}
	term := terminals[exp.Terminal]
	if term == nil {
		t.Fatalf("scenario %s step %d: unknown terminal %q", scenarioID, stepIdx, exp.Terminal)
	}
	_, saw := term.WaitFor(func(env transport.ProtoServerEnvelope) bool {
		resp, ok := env.(*controlv1.ConnectResponse)
		if !ok {
			return false
		}
		r := resp.GetRouteStream()
		return r != nil && r.GetKind() == exp.RouteKind
	}, scenarioStepTimeout)
	if saw {
		return pass, detail
	}
	msg := fmt.Sprintf("terminal %s: route_kind %q not seen", exp.Terminal, exp.RouteKind)
	return false, appendExpectDetail(detail, msg)
}

func (h *Harness) expectBroadcastEvents(exp *ExpectStep, broadcastMark int) []ui.BroadcastEvent {
	events := h.Broadcast.Events()
	if exp.BroadcastSinceMark {
		events = events[broadcastMark:]
	}
	return events
}

func (h *Harness) expectBroadcastContains(exp *ExpectStep, events []ui.BroadcastEvent, pass bool, detail string) (bool, string) {
	if exp.BroadcastContains == "" {
		return pass, detail
	}
	for _, ev := range events {
		if ev.Message == exp.BroadcastContains {
			return pass, detail
		}
	}
	msg := fmt.Sprintf("broadcast missing %q (%d events)", exp.BroadcastContains, len(events))
	return false, appendExpectDetail(detail, msg)
}

func (h *Harness) expectBroadcastNotContains(exp *ExpectStep, events []ui.BroadcastEvent, pass bool, detail string) (bool, string) {
	if exp.BroadcastNotContains == "" {
		return pass, detail
	}
	for _, ev := range events {
		if ev.Message == exp.BroadcastNotContains {
			msg := fmt.Sprintf("broadcast unexpectedly contains %q", exp.BroadcastNotContains)
			return false, appendExpectDetail(detail, msg)
		}
	}
	return pass, detail
}

func (h *Harness) expectTimersProcessed(exp *ExpectStep, pass bool, detail string) (bool, string) {
	if exp.TimersProcessed == nil {
		return pass, detail
	}
	processed, err := h.ProcessDueTimers(context.Background())
	want := *exp.TimersProcessed
	if err == nil && processed == want {
		return pass, detail
	}
	msg := fmt.Sprintf("timers processed=%d want=%d err=%v", processed, want, err)
	return false, appendExpectDetail(detail, msg)
}

func appendExpectDetail(detail, msg string) string {
	if detail == "" {
		return msg
	}
	return detail + "; " + msg
}
