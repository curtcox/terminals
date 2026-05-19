package repl

import (
	"errors"
	"fmt"
	"strings"
)

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

const scenarioDefineUsage = "usage: scenarios define <name> [--match <intent|intent=x|event=y>]... [--priority <p>] [--on-start <command>] [--on-input <command>] [--on-event <kind> <command>]... [--on-suspend <command>] [--on-resume <command>] [--on-stop <command>]"

func parseScenariosDefineArgs(args []string) (scenarioDefineArgs, error) {
	out := scenarioDefineArgs{}
	if len(args) == 0 {
		return out, errors.New(scenarioDefineUsage)
	}
	out.name = strings.TrimSpace(args[0])
	if out.name == "" || strings.HasPrefix(out.name, "--") {
		return out, errors.New(scenarioDefineUsage)
	}

	for i := 1; i < len(args); i++ {
		flag := strings.TrimSpace(args[i])
		if flag == "" || !strings.HasPrefix(flag, "--") {
			return out, fmt.Errorf("unexpected token in scenarios define: %s", args[i])
		}
		if flag == "--json" {
			continue
		}
		if err := parseScenarioDefineFlag(flag, args, &i, &out); err != nil {
			return out, err
		}
	}
	out.matchIntents = uniqueStrings(out.matchIntents)
	out.matchEvents = uniqueStrings(out.matchEvents)
	return out, nil
}

func parseScenarioDefineFlag(flag string, args []string, i *int, out *scenarioDefineArgs) error {
	switch flag {
	case "--match":
		return parseScenarioDefineMatch(args, i, out)
	case "--priority":
		return parseScenarioDefinePriority(args, i, out)
	case "--on-start":
		return parseScenarioDefineOnStart(args, i, out)
	case "--on-input":
		return parseScenarioDefineOnInput(args, i, out)
	case "--on-event":
		return parseScenarioDefineOnEvent(args, i, out)
	case "--on-suspend":
		return parseScenarioDefineOnSuspend(args, i, out)
	case "--on-resume":
		return parseScenarioDefineOnResume(args, i, out)
	case "--on-stop":
		return parseScenarioDefineOnStop(args, i, out)
	default:
		return fmt.Errorf("unknown flag for scenarios define: %s", flag)
	}
}

func parseScenarioDefineMatch(args []string, i *int, out *scenarioDefineArgs) error {
	if *i+1 >= len(args) || strings.HasPrefix(strings.TrimSpace(args[*i+1]), "--") {
		return errors.New("usage: scenarios define <name> ... --match <intent|intent=x|event=y>")
	}
	for _, token := range strings.Split(strings.TrimSpace(args[*i+1]), ",") {
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
	*i++
	return nil
}

func parseScenarioDefinePriority(args []string, i *int, out *scenarioDefineArgs) error {
	if *i+1 >= len(args) || strings.HasPrefix(strings.TrimSpace(args[*i+1]), "--") {
		return errors.New("usage: scenarios define <name> ... --priority <p>")
	}
	out.priority = strings.TrimSpace(args[*i+1])
	*i++
	return nil
}

func parseScenarioDefineOnStart(args []string, i *int, out *scenarioDefineArgs) error {
	if *i+1 >= len(args) || strings.HasPrefix(strings.TrimSpace(args[*i+1]), "--") {
		return errors.New("usage: scenarios define <name> ... --on-start <command>")
	}
	out.onStart = strings.TrimSpace(args[*i+1])
	*i++
	return nil
}

func parseScenarioDefineOnInput(args []string, i *int, out *scenarioDefineArgs) error {
	if *i+1 >= len(args) || strings.HasPrefix(strings.TrimSpace(args[*i+1]), "--") {
		return errors.New("usage: scenarios define <name> ... --on-input <command>")
	}
	out.onInput = strings.TrimSpace(args[*i+1])
	*i++
	return nil
}

func parseScenarioDefineOnEvent(args []string, i *int, out *scenarioDefineArgs) error {
	if *i+2 >= len(args) {
		return errors.New("usage: scenarios define <name> ... --on-event <kind> <command>")
	}
	kind := strings.TrimSpace(args[*i+1])
	if kind == "" || strings.HasPrefix(kind, "--") {
		return errors.New("usage: scenarios define <name> ... --on-event <kind> <command>")
	}
	j := *i + 2
	parts := make([]string, 0, 2)
	for ; j < len(args); j++ {
		part := strings.TrimSpace(args[j])
		if strings.HasPrefix(part, "--") {
			break
		}
		parts = append(parts, part)
	}
	if len(parts) == 0 {
		return errors.New("usage: scenarios define <name> ... --on-event <kind> <command>")
	}
	out.onEvents = append(out.onEvents, scenarioDefineHook{kind: kind, command: strings.Join(parts, " ")})
	*i = j - 1
	return nil
}

func parseScenarioDefineOnSuspend(args []string, i *int, out *scenarioDefineArgs) error {
	if *i+1 >= len(args) || strings.HasPrefix(strings.TrimSpace(args[*i+1]), "--") {
		return errors.New("usage: scenarios define <name> ... --on-suspend <command>")
	}
	out.onSuspend = strings.TrimSpace(args[*i+1])
	*i++
	return nil
}

func parseScenarioDefineOnResume(args []string, i *int, out *scenarioDefineArgs) error {
	if *i+1 >= len(args) || strings.HasPrefix(strings.TrimSpace(args[*i+1]), "--") {
		return errors.New("usage: scenarios define <name> ... --on-resume <command>")
	}
	out.onResume = strings.TrimSpace(args[*i+1])
	*i++
	return nil
}

func parseScenarioDefineOnStop(args []string, i *int, out *scenarioDefineArgs) error {
	if *i+1 >= len(args) || strings.HasPrefix(strings.TrimSpace(args[*i+1]), "--") {
		return errors.New("usage: scenarios define <name> ... --on-stop <command>")
	}
	out.onStop = strings.TrimSpace(args[*i+1])
	*i++
	return nil
}
