package capability

// This file contains script parsing and execution helpers for the capability
// service. These helpers are separated from the main service implementation
// to keep the primary service file smaller and easier to navigate.

import (
	"fmt"
	"strings"
	"time"
)

// ScriptDryRun parses a script and returns a deterministic command summary.
func (s *Service) ScriptDryRun(path, script string) ScriptDryRunResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	commands, skippedCount := parseScriptCommands(script)
	result := ScriptDryRunResult{
		Path:         strings.TrimSpace(path),
		CommandCount: len(commands),
		SkippedCount: skippedCount,
		Commands:     commands,
		Issues:       []string{},
		CreatedAt:    s.now(),
	}
	s.appendRecentLocked("scripts", "dry-run "+result.Path)
	return result
}

// ScriptRun parses and executes a script against the current capability state.
func (s *Service) ScriptRun(path, script string) ScriptRunResult {
	commands, skippedCount := parseScriptCommands(script)
	result := ScriptRunResult{
		Path:          strings.TrimSpace(path),
		CommandCount:  len(commands),
		SkippedCount:  skippedCount,
		ExecutedCount: 0,
		FailedCount:   0,
		Commands:      commands,
		Issues:        []string{},
		CreatedAt:     s.now(),
	}
	for i, command := range commands {
		if err := s.executeScriptCommand(command); err != nil {
			result.FailedCount++
			result.Issues = append(result.Issues, fmt.Sprintf("command %d (%q): %v", i+1, command, err))
			continue
		}
		result.ExecutedCount++
	}
	s.mu.Lock()
	s.appendRecentLocked("scripts", "run "+result.Path)
	s.mu.Unlock()
	return result
}

func (s *Service) executeScriptCommand(command string) error {
	tokens := strings.Fields(strings.TrimSpace(command))
	if len(tokens) == 0 {
		return nil
	}
	if len(tokens) < 2 {
		return fmt.Errorf("invalid command")
	}
	group := strings.ToLower(tokens[0])
	sub := strings.ToLower(tokens[1])
	args := tokens[2:]

	switch group {
	case "store":
		if sub != "put" {
			return fmt.Errorf("unsupported store command %q", sub)
		}
		if len(args) < 3 {
			return fmt.Errorf("usage: store put <namespace> <key> <value> [--ttl <duration>]")
		}
		namespace := args[0]
		key := args[1]
		rest := args[2:]
		valueParts := make([]string, 0, len(rest))
		ttl := time.Duration(0)
		for i := 0; i < len(rest); i++ {
			token := rest[i]
			if token == "--ttl" {
				if i+1 >= len(rest) {
					return fmt.Errorf("--ttl requires a duration")
				}
				parsedTTL, err := time.ParseDuration(rest[i+1])
				if err != nil || parsedTTL <= 0 {
					return fmt.Errorf("invalid --ttl duration %q", rest[i+1])
				}
				ttl = parsedTTL
				i++
				continue
			}
			if strings.HasPrefix(token, "--") {
				return fmt.Errorf("unsupported flag %q", token)
			}
			valueParts = append(valueParts, token)
		}
		if len(valueParts) == 0 {
			return fmt.Errorf("value is required")
		}
		s.StorePut(namespace, key, strings.Join(valueParts, " "), ttl)
		return nil
	case "bus":
		if sub != "emit" {
			return fmt.Errorf("unsupported bus command %q", sub)
		}
		if len(args) < 2 {
			return fmt.Errorf("usage: bus emit <kind> <name> [payload]")
		}
		kind := args[0]
		name := args[1]
		payload := ""
		if len(args) > 2 {
			payload = strings.Join(args[2:], " ")
		}
		s.BusEmit(kind, name, payload)
		return nil
	case "ui":
		if sub != "push" {
			return fmt.Errorf("unsupported ui command %q", sub)
		}
		if len(args) < 2 {
			return fmt.Errorf("usage: ui push <device> <descriptor-expr> [--root <id>]")
		}
		deviceID := args[0]
		rest := args[1:]
		descriptorParts := make([]string, 0, len(rest))
		rootID := ""
		for i := 0; i < len(rest); i++ {
			token := rest[i]
			if token == "--root" {
				if i+1 >= len(rest) {
					return fmt.Errorf("--root requires an id")
				}
				rootID = rest[i+1]
				i++
				continue
			}
			if strings.HasPrefix(token, "--") {
				return fmt.Errorf("unsupported flag %q", token)
			}
			descriptorParts = append(descriptorParts, token)
		}
		if len(descriptorParts) == 0 {
			return fmt.Errorf("descriptor-expr is required")
		}
		s.UIPush(deviceID, strings.Join(descriptorParts, " "), rootID)
		return nil
	case "message":
		switch sub {
		case "rooms":
			if len(args) != 0 {
				return fmt.Errorf("usage: message rooms")
			}
			_ = s.ListMessageRooms()
			return nil
		case "post":
			if len(args) < 2 {
				return fmt.Errorf("usage: message post <room> <text>")
			}
			room := args[0]
			text := strings.Join(args[1:], " ")
			s.PostMessage(room, text)
			return nil
		case "ls":
			if len(args) > 1 {
				return fmt.Errorf("usage: message ls [room]")
			}
			room := ""
			if len(args) == 1 {
				room = args[0]
			}
			_ = s.ListMessages(room)
			return nil
		default:
			return fmt.Errorf("unsupported message command %q", sub)
		}
	case "board":
		switch sub {
		case "post":
			if len(args) < 2 {
				return fmt.Errorf("usage: board post <board> <text>")
			}
			board := args[0]
			text := strings.Join(args[1:], " ")
			s.PostBoard(board, text)
			return nil
		case "ls":
			if len(args) > 1 {
				return fmt.Errorf("usage: board ls [board]")
			}
			board := ""
			if len(args) == 1 {
				board = args[0]
			}
			_ = s.ListBoard(board)
			return nil
		default:
			return fmt.Errorf("unsupported board command %q", sub)
		}
	case "artifact":
		switch sub {
		case "create":
			if len(args) < 2 {
				return fmt.Errorf("usage: artifact create <kind> <title>")
			}
			kind := args[0]
			title := strings.Join(args[1:], " ")
			s.CreateArtifact(kind, title)
			return nil
		case "history":
			if len(args) != 1 {
				return fmt.Errorf("usage: artifact history <artifact>")
			}
			artifactID := strings.TrimSpace(args[0])
			if strings.EqualFold(artifactID, "latest") {
				artifacts := s.ListArtifacts()
				if len(artifacts) == 0 {
					return fmt.Errorf("artifact not found")
				}
				artifactID = artifacts[len(artifacts)-1].ID
			}
			if _, ok := s.ArtifactHistory(artifactID); !ok {
				return fmt.Errorf("artifact not found")
			}
			return nil
		default:
			return fmt.Errorf("unsupported artifact command %q", sub)
		}
	case "canvas":
		switch sub {
		case "annotate":
			if len(args) < 2 {
				return fmt.Errorf("usage: canvas annotate <canvas> <text>")
			}
			canvas := args[0]
			text := strings.Join(args[1:], " ")
			s.AnnotateCanvas(canvas, text)
			return nil
		case "ls":
			if len(args) > 1 {
				return fmt.Errorf("usage: canvas ls [canvas]")
			}
			canvas := ""
			if len(args) == 1 {
				canvas = args[0]
			}
			_ = s.ListCanvas(canvas)
			return nil
		default:
			return fmt.Errorf("unsupported canvas command %q", sub)
		}
	case "session":
		switch sub {
		case "create":
			if len(args) < 2 {
				return fmt.Errorf("usage: session create <kind> <target>")
			}
			s.CreateSession(args[0], args[1])
			return nil
		case "join":
			if len(args) < 2 {
				return fmt.Errorf("usage: session join <session> <participant>")
			}
			sessionID := strings.TrimSpace(args[0])
			if strings.EqualFold(sessionID, "latest") {
				sessions := s.ListSessions()
				if len(sessions) == 0 {
					return fmt.Errorf("session not found")
				}
				sessionID = sessions[len(sessions)-1].ID
			}
			if _, ok := s.JoinSession(sessionID, args[1]); !ok {
				return fmt.Errorf("session not found")
			}
			return nil
		case "members":
			if len(args) != 1 {
				return fmt.Errorf("usage: session members <session>")
			}
			sessionID := strings.TrimSpace(args[0])
			if strings.EqualFold(sessionID, "latest") {
				sessions := s.ListSessions()
				if len(sessions) == 0 {
					return fmt.Errorf("session not found")
				}
				sessionID = sessions[len(sessions)-1].ID
			}
			if _, ok := s.ListSessionParticipants(sessionID); !ok {
				return fmt.Errorf("session not found")
			}
			return nil
		default:
			return fmt.Errorf("unsupported session command %q", sub)
		}
	case "identity":
		if sub != "ack" {
			return fmt.Errorf("unsupported identity command %q", sub)
		}
		if len(args) == 0 {
			return fmt.Errorf("usage: identity ack <show|record>")
		}
		action := strings.ToLower(strings.TrimSpace(args[0]))
		switch action {
		case "record":
			if len(args) < 2 {
				return fmt.Errorf("usage: identity ack record <subject-ref> --actor <actor-ref> [--mode <mode>]")
			}
			subjectRef := strings.TrimSpace(args[1])
			if subjectRef == "" {
				return fmt.Errorf("subject-ref is required")
			}
			actor := ""
			mode := "read"
			rest := args[2:]
			for i := 0; i < len(rest); i++ {
				token := rest[i]
				switch token {
				case "--actor":
					if i+1 >= len(rest) {
						return fmt.Errorf("--actor requires an actor-ref")
					}
					actor = strings.TrimSpace(rest[i+1])
					i++
				case "--mode":
					if i+1 >= len(rest) {
						return fmt.Errorf("--mode requires a mode")
					}
					mode = strings.TrimSpace(rest[i+1])
					i++
				default:
					return fmt.Errorf("unsupported flag %q", token)
				}
			}
			if actor == "" {
				return fmt.Errorf("--actor is required")
			}
			if _, ok := s.RecordAcknowledgement(subjectRef, actor, mode); !ok {
				return fmt.Errorf("invalid acknowledgement")
			}
			return nil
		case "show":
			if len(args) != 2 {
				return fmt.Errorf("usage: identity ack show <subject-ref>")
			}
			subjectRef := strings.TrimSpace(args[1])
			if subjectRef == "" {
				return fmt.Errorf("subject-ref is required")
			}
			_ = s.GetAcknowledgements(subjectRef)
			return nil
		default:
			return fmt.Errorf("unsupported identity ack action %q", action)
		}
	case "memory":
		switch sub {
		case "remember":
			if len(args) < 2 {
				return fmt.Errorf("usage: memory remember <scope> <text>")
			}
			scope := args[0]
			text := strings.Join(args[1:], " ")
			s.Remember(scope, text)
			return nil
		case "recall":
			if len(args) < 1 {
				return fmt.Errorf("usage: memory recall <text>")
			}
			_ = s.Recall(strings.Join(args, " "))
			return nil
		default:
			return fmt.Errorf("unsupported memory command %q", sub)
		}
	case "sim":
		switch sub {
		case "device":
			if len(args) < 2 {
				return fmt.Errorf("usage: sim device <new|rm> <id>")
			}
			action := strings.ToLower(args[0])
			deviceID := args[1]
			switch action {
			case "new":
				caps := []string{}
				if len(args) > 2 {
					rest := args[2:]
					for i := 0; i < len(rest); i++ {
						if rest[i] != "--caps" {
							return fmt.Errorf("unsupported sim device flag %q", rest[i])
						}
						if i+1 >= len(rest) {
							return fmt.Errorf("--caps requires comma-separated values")
						}
						caps = strings.Split(rest[i+1], ",")
						i++
					}
				}
				s.SimDeviceUpsert(deviceID, caps)
				return nil
			case "rm":
				if ok := s.SimDeviceDelete(deviceID); !ok {
					return fmt.Errorf("sim device not found")
				}
				return nil
			default:
				return fmt.Errorf("unsupported sim device action %q", action)
			}
		case "input":
			if len(args) < 3 {
				return fmt.Errorf("usage: sim input <id> <component-id> <action> [<value>]")
			}
			deviceID := args[0]
			componentID := args[1]
			action := args[2]
			value := ""
			if len(args) > 3 {
				value = strings.Join(args[3:], " ")
			}
			if _, ok := s.SimRecordInput(deviceID, componentID, action, value); !ok {
				return fmt.Errorf("sim device not found")
			}
			return nil
		case "expect":
			if len(args) < 3 {
				return fmt.Errorf("usage: sim expect <id> <ui|message> <selector> [--within <duration>]")
			}
			deviceID := args[0]
			kind := args[1]
			rest := args[2:]
			selectorParts := make([]string, 0, len(rest))
			within := time.Duration(0)
			for i := 0; i < len(rest); i++ {
				token := rest[i]
				if token == "--within" {
					if i+1 >= len(rest) {
						return fmt.Errorf("--within requires a duration")
					}
					dur, err := time.ParseDuration(rest[i+1])
					if err != nil || dur <= 0 {
						return fmt.Errorf("invalid --within duration %q", rest[i+1])
					}
					within = dur
					i++
					continue
				}
				if strings.HasPrefix(token, "--") {
					return fmt.Errorf("unsupported flag %q", token)
				}
				selectorParts = append(selectorParts, token)
			}
			if len(selectorParts) == 0 {
				return fmt.Errorf("selector is required")
			}
			result, ok := s.SimExpect(deviceID, kind, strings.Join(selectorParts, " "), within)
			if !ok {
				return fmt.Errorf("sim device not found")
			}
			if !result.Matched {
				return fmt.Errorf("expectation not matched")
			}
			return nil
		case "record":
			if len(args) < 1 {
				return fmt.Errorf("usage: sim record <id> [--duration <duration>]")
			}
			deviceID := args[0]
			duration := time.Duration(0)
			if len(args) > 1 {
				rest := args[1:]
				for i := 0; i < len(rest); i++ {
					if rest[i] != "--duration" {
						return fmt.Errorf("unsupported flag %q", rest[i])
					}
					if i+1 >= len(rest) {
						return fmt.Errorf("--duration requires a duration")
					}
					parsed, err := time.ParseDuration(rest[i+1])
					if err != nil || parsed <= 0 {
						return fmt.Errorf("invalid --duration %q", rest[i+1])
					}
					duration = parsed
					i++
				}
			}
			if _, ok := s.SimRecord(deviceID, duration); !ok {
				return fmt.Errorf("sim device not found")
			}
			return nil
		default:
			return fmt.Errorf("unsupported sim command %q", sub)
		}
	default:
		return fmt.Errorf("unsupported command group %q", group)
	}
}

func parseScriptCommands(script string) ([]string, int) {
	commands := make([]string, 0)
	skippedCount := 0
	for _, rawLine := range strings.Split(script, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			skippedCount++
			continue
		}
		commands = append(commands, line)
	}
	return commands, skippedCount
}
