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
		switch sub {
		case "emit":
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
		case "tail":
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
		case "replay":
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
		default:
			return fmt.Errorf("unknown command: bus %s", sub)
		}
	case "handlers":
		switch sub {
		case "ls":
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
		case "on":
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
		case "off":
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
		default:
			return fmt.Errorf("unknown command: handlers %s", sub)
		}
	case "scenarios":
		switch sub {
		case "ls":
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
		case "show":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 1 {
				return errors.New("usage: scenarios show <name>")
			}
			body, err := s.fetchJSONQuery(ctx, "/admin/api/scenarios/inline", url.Values{"name": {plain[0]}})
			if err != nil {
				return err
			}
			return writeJSON(s.out, body)
		case "define":
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
		case "undefine":
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
		default:
			return fmt.Errorf("unknown command: scenarios %s", sub)
		}
	case "sim":
		switch sub {
		case "device":
			if len(args) < 2 {
				return errors.New("usage: sim device <new|rm>")
			}
			deviceSub := strings.ToLower(strings.TrimSpace(args[1]))
			switch deviceSub {
			case "new":
				plain := nonFlagArgsSkippingFlagValues(args[2:], "--caps")
				if len(plain) < 1 {
					return errors.New("usage: sim device new <id> [--caps <cap[,cap...]>]")
				}
				form := url.Values{"device_id": {plain[0]}}
				if capsRaw := strings.TrimSpace(flagValue(args[2:], "--caps")); capsRaw != "" {
					for _, capValue := range parseCSVValues(capsRaw) {
						form.Add("caps", capValue)
					}
				}
				body, err := s.postFormJSON(ctx, "/admin/api/sim/devices/new", form)
				if err != nil {
					return err
				}
				if jsonOut {
					return writeJSON(s.out, body)
				}
				_, err = fmt.Fprintf(s.out, "OK  action=sim.device.new device=%s\n", plain[0])
				return err
			case "rm":
				plain := nonFlagArgs(args[2:])
				if len(plain) < 1 {
					return errors.New("usage: sim device rm <id>")
				}
				body, err := s.postFormJSON(ctx, "/admin/api/sim/devices/rm", url.Values{"device_id": {plain[0]}})
				if err != nil {
					return err
				}
				if jsonOut {
					return writeJSON(s.out, body)
				}
				_, err = fmt.Fprintf(s.out, "OK  action=sim.device.rm device=%s deleted=%s\n", plain[0], toString(body["deleted"]))
				return err
			default:
				return fmt.Errorf("unknown command: sim device %s", deviceSub)
			}
		case "input":
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
		case "ui":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 1 {
				return errors.New("usage: sim ui <id>")
			}
			body, err := s.fetchJSONQuery(ctx, "/admin/api/sim/ui", url.Values{"device_id": {plain[0]}})
			if err != nil {
				return err
			}
			return writeJSON(s.out, body)
		case "expect":
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
		case "record":
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
		default:
			return fmt.Errorf("unknown command: sim %s", sub)
		}
	case "scripts":
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
	case "activations":
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
	case "claims":
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
		if len(claimsByDevice) == 0 {
			_, err := fmt.Fprintln(s.out, "(no claims)")
			return err
		}
		deviceIDs := make([]string, 0, len(claimsByDevice))
		for deviceID := range claimsByDevice {
			deviceIDs = append(deviceIDs, deviceID)
		}
		sort.Strings(deviceIDs)
		for _, deviceID := range deviceIDs {
			if _, err := fmt.Fprintf(s.out, "%s\n", deviceID); err != nil {
				return err
			}
			claims, _ := claimsByDevice[deviceID].([]any)
			if len(claims) == 0 {
				if _, err := fmt.Fprintln(s.out, "  (none)"); err != nil {
					return err
				}
				continue
			}
			for _, claimAny := range claims {
				claim, _ := claimAny.(map[string]any)
				if claim == nil {
					continue
				}
				if _, err := fmt.Fprintf(s.out, "  - %s by %s\n", toString(claim["resource"]), toString(claim["activation_id"])); err != nil {
					return err
				}
			}
		}
		return nil
	default:
		return s.evalControlPlaneOps(ctx, group, args, jsonOut)
	}
}
