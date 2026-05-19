package repl

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

func (s *state) evalControlPlaneBus(ctx context.Context, group string, args []string, jsonOut bool) error {
	sub := strings.ToLower(args[0])
	switch group {
	case "bus":
		return s.evalControlPlaneBusBus(ctx, sub, args, jsonOut)
	case "handlers":
		return s.evalControlPlaneBusHandlers(ctx, sub, args, jsonOut)
	case "scenarios":
		return s.evalControlPlaneBusScenarios(ctx, sub, args, jsonOut)
	case "sim":
		return s.evalControlPlaneBusSim(ctx, sub, args, jsonOut)
	case "scripts":
		return s.evalControlPlaneBusScripts(ctx, sub, args, jsonOut)
	case "activations":
		return s.evalControlPlaneBusActivations(ctx, sub, args, jsonOut)
	case "claims":
		return s.evalControlPlaneBusClaims(ctx, sub, args, jsonOut)
	default:
		return s.evalControlPlaneOps(ctx, group, args, jsonOut)
	}
}

func (s *state) evalControlPlaneBusBus(ctx context.Context, sub string, args []string, jsonOut bool) error {
	switch sub {
	case "emit":
		return s.evalControlPlaneBusBusEmit(ctx, sub, args, jsonOut)
	case "tail":
		return s.evalControlPlaneBusBusTail(ctx, sub, args, jsonOut)
	case "replay":
		return s.evalControlPlaneBusBusReplay(ctx, sub, args, jsonOut)
	default:
		return fmt.Errorf("unknown command: bus %s", sub)
	}
}

func (s *state) evalControlPlaneBusHandlers(ctx context.Context, sub string, args []string, jsonOut bool) error {
	switch sub {
	case "ls":
		return s.evalControlPlaneBusHandlersLs(ctx, sub, args, jsonOut)
	case "on":
		return s.evalControlPlaneBusHandlersOn(ctx, sub, args, jsonOut)
	case "off":
		return s.evalControlPlaneBusHandlersOff(ctx, sub, args, jsonOut)
	default:
		return fmt.Errorf("unknown command: handlers %s", sub)
	}
}

func (s *state) evalControlPlaneBusScenarios(ctx context.Context, sub string, args []string, jsonOut bool) error {
	switch sub {
	case "ls":
		return s.evalControlPlaneBusScenariosLs(ctx, sub, args, jsonOut)
	case "show":
		return s.evalControlPlaneBusScenariosShow(ctx, sub, args, jsonOut)
	case "define":
		return s.evalControlPlaneBusScenariosDefine(ctx, sub, args, jsonOut)
	case "undefine":
		return s.evalControlPlaneBusScenariosUndefine(ctx, sub, args, jsonOut)
	default:
		return fmt.Errorf("unknown command: scenarios %s", sub)
	}
}

func (s *state) evalControlPlaneBusSim(ctx context.Context, sub string, args []string, jsonOut bool) error {
	switch sub {
	case "device":
		return s.evalControlPlaneBusSimDevice(ctx, sub, args, jsonOut)
	case "input":
		return s.evalControlPlaneBusSimInput(ctx, sub, args, jsonOut)
	case "ui":
		return s.evalControlPlaneBusSimUI(ctx, sub, args, jsonOut)
	case "expect":
		return s.evalControlPlaneBusSimExpect(ctx, sub, args, jsonOut)
	case "record":
		return s.evalControlPlaneBusSimRecord(ctx, sub, args, jsonOut)
	default:
		return fmt.Errorf("unknown command: sim %s", sub)
	}
}

func (s *state) evalControlPlaneBusScripts(ctx context.Context, sub string, args []string, jsonOut bool) error {
	switch sub {
	case "dry-run":
		plain := nonFlagArgs(args[1:])
		if len(plain) < 1 {
			return errors.New("usage: scripts dry-run <path>")
		}
		body, err := s.postFormJSON(ctx, "/admin/api/scripts/dry-run", url.Values{"path": {plain[0]}})
		if err != nil {
			return err
		}
		if jsonOut {
			return writeJSON(s.out, body)
		}
		result, _ := body["result"].(map[string]any)
		_, err = fmt.Fprintf(s.out, "OK  action=scripts.dry-run path=%s commands=%s skipped=%s\n", plain[0], toString(result["command_count"]), toString(result["skipped_count"]))
		return err
	case "run":
		plain := nonFlagArgs(args[1:])
		if len(plain) < 1 {
			return errors.New("usage: scripts run <path>")
		}
		body, err := s.postFormJSON(ctx, "/admin/api/scripts/run", url.Values{"path": {plain[0]}})
		if err != nil {
			return err
		}
		if jsonOut {
			return writeJSON(s.out, body)
		}
		result, _ := body["result"].(map[string]any)
		_, err = fmt.Fprintf(s.out, "OK  action=scripts.run path=%s commands=%s executed=%s failed=%s\n", plain[0], toString(result["command_count"]), toString(result["executed_count"]), toString(result["failed_count"]))
		return err
	default:
		return fmt.Errorf("unknown command: scripts %s", sub)
	}
}

func (s *state) evalControlPlaneBusActivations(ctx context.Context, sub string, _ []string, jsonOut bool) error {
	if sub != "ls" {
		return fmt.Errorf("unknown command: activations %s", sub)
	}
	body, err := s.fetchJSON(ctx, "/admin/api/activations")
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	active, _ := body["active_by_device"].(map[string]any)
	rows := make([][]string, 0, len(active))
	for deviceID, scenarioName := range active {
		rows = append(rows, []string{deviceID, toString(scenarioName)})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i][0] < rows[j][0] })
	return printTable(s.out, []string{"DEVICE", "ACTIVE"}, rows)
}

func (s *state) evalControlPlaneBusClaims(ctx context.Context, sub string, _ []string, jsonOut bool) error {
	if sub != "tree" {
		return fmt.Errorf("unknown command: claims %s", sub)
	}
	body, err := s.fetchJSON(ctx, "/admin/api/activations")
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	claimsByDevice, _ := body["claims_by_device"].(map[string]any)
	return printClaimsByDeviceTree(s.out, claimsByDevice)
}

func (s *state) evalControlPlaneBusSimDevice(ctx context.Context, _ string, args []string, jsonOut bool) error {
	if len(args) < 2 {
		return errors.New("usage: sim device <new|rm>")
	}
	deviceSub := strings.ToLower(strings.TrimSpace(args[1]))
	switch deviceSub {
	case "new":
		return s.evalSimDeviceNew(ctx, args[2:], jsonOut)
	case "rm":
		return s.evalSimDeviceRm(ctx, args[2:], jsonOut)
	default:
		return fmt.Errorf("unknown command: sim device %s", deviceSub)
	}
}

func (s *state) evalControlPlaneBusSimInput(ctx context.Context, _ string, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 3 {
		return errors.New("usage: sim input <id> <component-id> <action> [<value>]")
	}
	value := ""
	if len(plain) > 3 {
		value = strings.Join(plain[3:], " ")
	}
	body, err := s.postFormJSON(ctx, "/admin/api/sim/input", url.Values{
		"device_id":    {plain[0]},
		"component_id": {plain[1]},
		"action":       {plain[2]},
		"value":        {value},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  action=sim.input device=%s component=%s event=%s\n", plain[0], plain[1], plain[2])
	return err
}

func (s *state) evalControlPlaneBusSimUI(ctx context.Context, _ string, args []string, _ bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 1 {
		return errors.New("usage: sim ui <id>")
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/sim/ui", url.Values{"device_id": {plain[0]}})
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalControlPlaneBusSimExpect(ctx context.Context, _ string, args []string, jsonOut bool) error {
	plain := nonFlagArgsSkippingFlagValues(args[1:], "--within")
	if len(plain) < 3 {
		return errors.New("usage: sim expect <id> <ui|message> <selector> [--within <duration>]")
	}
	form := url.Values{
		"device_id": {plain[0]},
		"kind":      {plain[1]},
		"selector":  {strings.Join(plain[2:], " ")},
	}
	if within := strings.TrimSpace(flagValue(args[1:], "--within")); within != "" {
		form.Set("within", within)
	}
	body, err := s.postFormJSON(ctx, "/admin/api/sim/expect", form)
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	result, _ := body["result"].(map[string]any)
	_, err = fmt.Fprintf(s.out, "OK  action=sim.expect device=%s kind=%s matched=%s\n", plain[0], plain[1], toString(result["matched"]))
	return err
}

func (s *state) evalControlPlaneBusSimRecord(ctx context.Context, _ string, args []string, jsonOut bool) error {
	plain := nonFlagArgsSkippingFlagValues(args[1:], "--duration")
	if len(plain) < 1 {
		return errors.New("usage: sim record <id> [--duration <duration>]")
	}
	form := url.Values{"device_id": {plain[0]}}
	if duration := strings.TrimSpace(flagValue(args[1:], "--duration")); duration != "" {
		form.Set("duration", duration)
	}
	body, err := s.postFormJSON(ctx, "/admin/api/sim/record", form)
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	result, _ := body["result"].(map[string]any)
	inputs := toAnySlice(result["inputs"])
	messages := toAnySlice(result["messages"])
	_, err = fmt.Fprintf(s.out, "OK  action=sim.record device=%s inputs=%d messages=%d\n", plain[0], len(inputs), len(messages))
	return err
}

func (s *state) evalControlPlaneBusScenariosLs(ctx context.Context, _ string, _ []string, jsonOut bool) error {
	body, err := s.fetchJSON(ctx, "/admin/api/scenarios/inline")
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	items, _ := body["scenarios"].([]any)
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		row, _ := item.(map[string]any)
		if row == nil {
			continue
		}
		intents := joinAnyStrings(row["match_intents"], ",")
		events := joinAnyStrings(row["match_events"], ",")
		rows = append(rows, []string{toString(row["name"]), toString(row["priority"]), intents, events})
	}
	return printTable(s.out, []string{"SCENARIO", "PRIORITY", "INTENTS", "EVENTS"}, rows)
}

func (s *state) evalControlPlaneBusScenariosShow(ctx context.Context, _ string, args []string, _ bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 1 {
		return errors.New("usage: scenarios show <name>")
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/scenarios/inline", url.Values{"name": {plain[0]}})
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalControlPlaneBusScenariosDefine(ctx context.Context, _ string, args []string, jsonOut bool) error {
	def, err := parseScenariosDefineArgs(args[1:])
	if err != nil {
		return err
	}
	form := url.Values{"name": {def.name}}
	for _, intent := range def.matchIntents {
		form.Add("match_intent", intent)
	}
	for _, event := range def.matchEvents {
		form.Add("match_event", event)
	}
	if def.priority != "" {
		form.Set("priority", def.priority)
	}
	if def.onStart != "" {
		form.Set("on_start", def.onStart)
	}
	if def.onInput != "" {
		form.Set("on_input", def.onInput)
	}
	if def.onSuspend != "" {
		form.Set("on_suspend", def.onSuspend)
	}
	if def.onResume != "" {
		form.Set("on_resume", def.onResume)
	}
	if def.onStop != "" {
		form.Set("on_stop", def.onStop)
	}
	for _, hook := range def.onEvents {
		form.Add("on_event_kind", hook.kind)
		form.Add("on_event_command", hook.command)
	}
	body, err := s.postFormJSON(ctx, "/admin/api/scenarios/inline/define", form)
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  action=define scenario=%s\n", def.name)
	return err
}

func (s *state) evalControlPlaneBusScenariosUndefine(ctx context.Context, _ string, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 1 {
		return errors.New("usage: scenarios undefine <name>")
	}
	body, err := s.postFormJSON(ctx, "/admin/api/scenarios/inline/undefine", url.Values{"name": {plain[0]}})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  deleted=%s scenario=%s\n", toString(body["deleted"]), plain[0])
	return err
}

func (s *state) evalControlPlaneBusBusEmit(ctx context.Context, _ string, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 2 {
		return errors.New("usage: bus emit <kind> <name> [payload]")
	}
	payload := ""
	if len(plain) > 2 {
		payload = strings.Join(plain[2:], " ")
	}
	body, err := s.postFormJSON(ctx, "/admin/api/bus/emit", url.Values{
		"kind":    {plain[0]},
		"name":    {plain[1]},
		"payload": {payload},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	eventID := ""
	if itemMap, ok := body["event"].(map[string]any); ok {
		eventID = toString(itemMap["id"])
	}
	_, err = fmt.Fprintf(s.out, "OK  event=%s\n", eventID)
	return err
}

func (s *state) evalControlPlaneBusBusTail(ctx context.Context, _ string, args []string, _ bool) error {
	query := url.Values{}
	if kind := strings.TrimSpace(flagValue(args[1:], "--kind")); kind != "" {
		query.Set("kind", kind)
	}
	if name := strings.TrimSpace(flagValue(args[1:], "--name")); name != "" {
		query.Set("name", name)
	}
	if limitRaw := strings.TrimSpace(flagValue(args[1:], "--limit")); limitRaw != "" {
		if limit, err := strconv.Atoi(limitRaw); err != nil || limit <= 0 {
			return errors.New("usage: bus tail [--kind <kind>] [--name <name>] [--limit <n>]")
		}
		query.Set("limit", limitRaw)
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/bus", query)
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalControlPlaneBusBusReplay(ctx context.Context, _ string, args []string, _ bool) error {
	plain := nonFlagArgsSkippingFlagValues(args[1:], "--kind", "--name", "--limit")
	if len(plain) < 2 {
		return errors.New("usage: bus replay <from-id> <to-id> [--kind <kind>] [--name <name>] [--limit <n>]")
	}
	query := url.Values{
		"from": {plain[0]},
		"to":   {plain[1]},
	}
	if kind := strings.TrimSpace(flagValue(args[1:], "--kind")); kind != "" {
		query.Set("kind", kind)
	}
	if name := strings.TrimSpace(flagValue(args[1:], "--name")); name != "" {
		query.Set("name", name)
	}
	if limitRaw := strings.TrimSpace(flagValue(args[1:], "--limit")); limitRaw != "" {
		if limit, err := strconv.Atoi(limitRaw); err != nil || limit <= 0 {
			return errors.New("usage: bus replay <from-id> <to-id> [--kind <kind>] [--name <name>] [--limit <n>]")
		}
		query.Set("limit", limitRaw)
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/bus/replay", query)
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalControlPlaneBusHandlersLs(ctx context.Context, _ string, _ []string, jsonOut bool) error {
	body, err := s.fetchJSON(ctx, "/admin/api/handlers")
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	items, _ := body["handlers"].([]any)
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		row, _ := item.(map[string]any)
		if row == nil {
			continue
		}
		target := toString(row["run_command"])
		if target == "" {
			emitKind := toString(row["emit_kind"])
			emitName := toString(row["emit_name"])
			emitPayload := toString(row["emit_payload"])
			target = strings.TrimSpace("emit " + emitKind + " " + emitName + " " + emitPayload)
		}
		rows = append(rows, []string{toString(row["id"]), toString(row["selector"]), toString(row["action"]), target})
	}
	return printTable(s.out, []string{"HANDLER", "SELECTOR", "ACTION", "TARGET"}, rows)
}

func (s *state) evalControlPlaneBusHandlersOn(ctx context.Context, _ string, args []string, jsonOut bool) error {
	plain := nonFlagArgsSkippingFlagValues(args[1:], "--run")
	if len(plain) < 2 {
		return errors.New("usage: handlers on <selector> <action> (--run <command> | --emit <kind> <name> [payload])")
	}
	selector := plain[0]
	action := plain[1]
	runCommand := strings.TrimSpace(flagValue(args[1:], "--run"))
	emitKind, emitName, emitPayload := parseHandlersEmitValue(args[1:])
	hasRun := runCommand != ""
	hasEmit := emitKind != "" || emitName != "" || emitPayload != ""
	if hasRun == hasEmit {
		return errors.New("usage: handlers on <selector> <action> (--run <command> | --emit <kind> <name> [payload])")
	}

	form := url.Values{
		"selector": {selector},
		"action":   {action},
	}
	if hasRun {
		form.Set("run", runCommand)
	} else {
		if emitName == "" {
			return errors.New("usage: handlers on <selector> <action> --emit <kind> <name> [payload]")
		}
		form.Set("emit_kind", emitKind)
		form.Set("emit_name", emitName)
		if emitPayload != "" {
			form.Set("emit_payload", emitPayload)
		}
	}
	body, err := s.postFormJSON(ctx, "/admin/api/handlers/on", form)
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	handlerID := ""
	if itemMap, ok := body["handler"].(map[string]any); ok {
		handlerID = toString(itemMap["id"])
	}
	_, err = fmt.Fprintf(s.out, "OK  handler=%s selector=%s action=%s\n", handlerID, selector, action)
	return err
}

func (s *state) evalControlPlaneBusHandlersOff(ctx context.Context, _ string, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 1 {
		return errors.New("usage: handlers off <handler-id>")
	}
	body, err := s.postFormJSON(ctx, "/admin/api/handlers/off", url.Values{"handler_id": {plain[0]}})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  deleted=%s handler=%s\n", toString(body["deleted"]), plain[0])
	return err
}
