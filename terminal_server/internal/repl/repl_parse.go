package repl

import (
	"errors"
	"fmt"
	"strings"
)

func splitSegments(line string) []string {
	segments := make([]string, 0)
	var b strings.Builder
	inSingle := false
	inDouble := false
	for _, r := range line {
		switch r {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
			b.WriteRune(r)
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
			b.WriteRune(r)
		case ';':
			if inSingle || inDouble {
				b.WriteRune(r)
				continue
			}
			segment := strings.TrimSpace(b.String())
			if segment != "" {
				segments = append(segments, segment)
			}
			b.Reset()
		default:
			b.WriteRune(r)
		}
	}
	if tail := strings.TrimSpace(b.String()); tail != "" {
		segments = append(segments, tail)
	}
	return segments
}

func tokenize(line string) []string {
	if strings.TrimSpace(line) == "" {
		return nil
	}
	tokens := make([]string, 0)
	var b strings.Builder
	inSingle := false
	inDouble := false
	escaped := false
	flush := func() {
		if b.Len() == 0 {
			return
		}
		tokens = append(tokens, b.String())
		b.Reset()
	}
	for _, r := range line {
		switch {
		case escaped:
			b.WriteRune(r)
			escaped = false
		case r == '\\' && inDouble:
			escaped = true
		case r == '\'' && !inDouble:
			inSingle = !inSingle
		case r == '"' && !inSingle:
			inDouble = !inDouble
		case (r == ' ' || r == '\t') && !inSingle && !inDouble:
			flush()
		default:
			b.WriteRune(r)
		}
	}
	flush()
	for i := range tokens {
		tokens[i] = strings.TrimSpace(tokens[i])
	}
	return tokens
}

func hasFlag(args []string, name string) bool {
	for _, arg := range args {
		if strings.EqualFold(strings.TrimSpace(arg), name) {
			return true
		}
	}
	return false
}

func flagValue(args []string, name string) string {
	for i := range args {
		if !strings.EqualFold(strings.TrimSpace(args[i]), name) {
			continue
		}
		if i+1 >= len(args) {
			return ""
		}
		next := strings.TrimSpace(args[i+1])
		if strings.HasPrefix(next, "--") {
			return ""
		}
		return next
	}
	return ""
}

func nonFlagArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		trimmed := strings.TrimSpace(arg)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func nonFlagArgsSkippingFlagValues(args []string, valueFlags ...string) []string {
	skipValueFlags := make(map[string]struct{}, len(valueFlags))
	for _, flag := range valueFlags {
		skipValueFlags[strings.ToLower(strings.TrimSpace(flag))] = struct{}{}
	}
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		trimmed := strings.TrimSpace(args[i])
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "--") {
			if shouldSkipFollowingFlagValue(args, i, skipValueFlags) {
				i++
			}
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func shouldSkipFollowingFlagValue(args []string, i int, valueFlags map[string]struct{}) bool {
	if _, ok := valueFlags[strings.ToLower(strings.TrimSpace(args[i]))]; !ok {
		return false
	}
	if i+1 >= len(args) {
		return false
	}
	return !strings.HasPrefix(strings.TrimSpace(args[i+1]), "--")
}

func parseHandlersEmitValue(args []string) (kind string, name string, payload string) {
	for i := 0; i < len(args); i++ {
		if !strings.EqualFold(strings.TrimSpace(args[i]), "--emit") {
			continue
		}
		if i+1 >= len(args) {
			return "", "", ""
		}
		kind = strings.TrimSpace(args[i+1])
		if i+2 >= len(args) {
			return kind, "", ""
		}
		name = strings.TrimSpace(args[i+2])
		if i+3 >= len(args) {
			return kind, name, ""
		}
		payloadParts := make([]string, 0, len(args)-(i+3))
		for j := i + 3; j < len(args); j++ {
			part := strings.TrimSpace(args[j])
			if strings.HasPrefix(part, "--") {
				break
			}
			payloadParts = append(payloadParts, part)
		}
		return kind, name, strings.Join(payloadParts, " ")
	}
	return "", "", ""
}

func parseCSVValues(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

type scenarioDefineHook struct {
	kind    string
	command string
}

type scenarioDefineArgs struct {
	name         string
	matchIntents []string
	matchEvents  []string
	priority     string
	onStart      string
	onInput      string
	onEvents     []scenarioDefineHook
	onSuspend    string
	onResume     string
	onStop       string
}

func parseScenariosDefineArgs(args []string) (scenarioDefineArgs, error) {
	out := scenarioDefineArgs{}
	if len(args) == 0 {
		return out, errors.New("usage: scenarios define <name> [--match <intent|intent=x|event=y>]... [--priority <p>] [--on-start <command>] [--on-input <command>] [--on-event <kind> <command>]... [--on-suspend <command>] [--on-resume <command>] [--on-stop <command>]")
	}
	out.name = strings.TrimSpace(args[0])
	if out.name == "" || strings.HasPrefix(out.name, "--") {
		return out, errors.New("usage: scenarios define <name> [--match <intent|intent=x|event=y>]... [--priority <p>] [--on-start <command>] [--on-input <command>] [--on-event <kind> <command>]... [--on-suspend <command>] [--on-resume <command>] [--on-stop <command>]")
	}

	for i := 1; i < len(args); i++ {
		flag := strings.TrimSpace(args[i])
		if flag == "" || !strings.HasPrefix(flag, "--") {
			return out, fmt.Errorf("unexpected token in scenarios define: %s", args[i])
		}
		switch flag {
		case "--json":
			continue
		case "--match":
			if i+1 >= len(args) || strings.HasPrefix(strings.TrimSpace(args[i+1]), "--") {
				return out, errors.New("usage: scenarios define <name> ... --match <intent|intent=x|event=y>")
			}
			for _, token := range strings.Split(strings.TrimSpace(args[i+1]), ",") {
				token = strings.TrimSpace(token)
				if token == "" {
					continue
				}
				lower := strings.ToLower(token)
				switch {
				case strings.HasPrefix(lower, "event="):
					out.matchEvents = append(out.matchEvents, strings.TrimSpace(token[len("event="):]))
				case strings.HasPrefix(lower, "intent="):
					out.matchIntents = append(out.matchIntents, strings.TrimSpace(token[len("intent="):]))
				default:
					out.matchIntents = append(out.matchIntents, token)
				}
			}
			i++
		case "--priority":
			if i+1 >= len(args) || strings.HasPrefix(strings.TrimSpace(args[i+1]), "--") {
				return out, errors.New("usage: scenarios define <name> ... --priority <p>")
			}
			out.priority = strings.TrimSpace(args[i+1])
			i++
		case "--on-start":
			if i+1 >= len(args) || strings.HasPrefix(strings.TrimSpace(args[i+1]), "--") {
				return out, errors.New("usage: scenarios define <name> ... --on-start <command>")
			}
			out.onStart = strings.TrimSpace(args[i+1])
			i++
		case "--on-input":
			if i+1 >= len(args) || strings.HasPrefix(strings.TrimSpace(args[i+1]), "--") {
				return out, errors.New("usage: scenarios define <name> ... --on-input <command>")
			}
			out.onInput = strings.TrimSpace(args[i+1])
			i++
		case "--on-event":
			if i+2 >= len(args) {
				return out, errors.New("usage: scenarios define <name> ... --on-event <kind> <command>")
			}
			kind := strings.TrimSpace(args[i+1])
			if kind == "" || strings.HasPrefix(kind, "--") {
				return out, errors.New("usage: scenarios define <name> ... --on-event <kind> <command>")
			}
			j := i + 2
			parts := make([]string, 0, 2)
			for ; j < len(args); j++ {
				part := strings.TrimSpace(args[j])
				if strings.HasPrefix(part, "--") {
					break
				}
				parts = append(parts, part)
			}
			if len(parts) == 0 {
				return out, errors.New("usage: scenarios define <name> ... --on-event <kind> <command>")
			}
			out.onEvents = append(out.onEvents, scenarioDefineHook{kind: kind, command: strings.Join(parts, " ")})
			i = j - 1
		case "--on-suspend":
			if i+1 >= len(args) || strings.HasPrefix(strings.TrimSpace(args[i+1]), "--") {
				return out, errors.New("usage: scenarios define <name> ... --on-suspend <command>")
			}
			out.onSuspend = strings.TrimSpace(args[i+1])
			i++
		case "--on-resume":
			if i+1 >= len(args) || strings.HasPrefix(strings.TrimSpace(args[i+1]), "--") {
				return out, errors.New("usage: scenarios define <name> ... --on-resume <command>")
			}
			out.onResume = strings.TrimSpace(args[i+1])
			i++
		case "--on-stop":
			if i+1 >= len(args) || strings.HasPrefix(strings.TrimSpace(args[i+1]), "--") {
				return out, errors.New("usage: scenarios define <name> ... --on-stop <command>")
			}
			out.onStop = strings.TrimSpace(args[i+1])
			i++
		default:
			return out, fmt.Errorf("unknown flag for scenarios define: %s", flag)
		}
	}
	out.matchIntents = uniqueStrings(out.matchIntents)
	out.matchEvents = uniqueStrings(out.matchEvents)
	return out, nil
}

func normalizeBugSource(raw string) string {
	source := strings.TrimSpace(strings.ToUpper(raw))
	if source == "" {
		return "BUG_REPORT_SOURCE_ADMIN"
	}
	if !strings.HasPrefix(source, "BUG_REPORT_SOURCE_") {
		source = "BUG_REPORT_SOURCE_" + source
	}
	return source
}
