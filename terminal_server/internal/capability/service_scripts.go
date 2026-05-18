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
		return s.executeStoreScriptCommand(sub, args)
	case "bus":
		return s.executeBusScriptCommand(sub, args)
	case "ui":
		return s.executeUIScriptCommand(sub, args)
	case "message":
		return s.executeMessageScriptCommand(sub, args)
	case "board":
		return s.executeBoardScriptCommand(sub, args)
	case "artifact":
		return s.executeArtifactScriptCommand(sub, args)
	case "canvas":
		return s.executeCanvasScriptCommand(sub, args)
	case "session":
		return s.executeSessionScriptCommand(sub, args)
	case "identity":
		return s.executeIdentityScriptCommand(sub, args)
	case "memory":
		return s.executeMemoryScriptCommand(sub, args)
	case "sim":
		return s.executeSimScriptCommand(sub, args)
	default:
		return fmt.Errorf("unsupported command group %q", group)
	}
}

func (s *Service) executeStoreScriptCommand(sub string, args []string) error {
	if sub != "put" {
		return fmt.Errorf("unsupported store command %q", sub)
	}
	if len(args) < 3 {
		return fmt.Errorf("usage: store put <namespace> <key> <value> [--ttl <duration>]")
	}
	value, ttl, err := parseStorePutValue(args[2:])
	if err != nil {
		return err
	}
	s.StorePut(args[0], args[1], value, ttl)
	return nil
}

func parseStorePutValue(rest []string) (string, time.Duration, error) {
	valueParts := make([]string, 0, len(rest))
	ttl := time.Duration(0)
	for i := 0; i < len(rest); i++ {
		token := rest[i]
		if token == "--ttl" {
			parsedTTL, err := parsePositiveDurationFlag(rest, &i, "--ttl")
			if err != nil {
				return "", 0, err
			}
			ttl = parsedTTL
			continue
		}
		if strings.HasPrefix(token, "--") {
			return "", 0, fmt.Errorf("unsupported flag %q", token)
		}
		valueParts = append(valueParts, token)
	}
	if len(valueParts) == 0 {
		return "", 0, fmt.Errorf("value is required")
	}
	return strings.Join(valueParts, " "), ttl, nil
}

func (s *Service) executeBusScriptCommand(sub string, args []string) error {
	if sub != "emit" {
		return fmt.Errorf("unsupported bus command %q", sub)
	}
	if len(args) < 2 {
		return fmt.Errorf("usage: bus emit <kind> <name> [payload]")
	}
	s.BusEmit(args[0], args[1], strings.Join(args[2:], " "))
	return nil
}

func (s *Service) executeUIScriptCommand(sub string, args []string) error {
	if sub != "push" {
		return fmt.Errorf("unsupported ui command %q", sub)
	}
	if len(args) < 2 {
		return fmt.Errorf("usage: ui push <device> <descriptor-expr> [--root <id>]")
	}
	descriptor, rootID, err := parseUIPushArgs(args[1:])
	if err != nil {
		return err
	}
	s.UIPush(args[0], descriptor, rootID)
	return nil
}

func parseUIPushArgs(rest []string) (string, string, error) {
	descriptorParts := make([]string, 0, len(rest))
	rootID := ""
	for i := 0; i < len(rest); i++ {
		token := rest[i]
		if token == "--root" {
			if i+1 >= len(rest) {
				return "", "", fmt.Errorf("--root requires an id")
			}
			rootID = rest[i+1]
			i++
			continue
		}
		if strings.HasPrefix(token, "--") {
			return "", "", fmt.Errorf("unsupported flag %q", token)
		}
		descriptorParts = append(descriptorParts, token)
	}
	if len(descriptorParts) == 0 {
		return "", "", fmt.Errorf("descriptor-expr is required")
	}
	return strings.Join(descriptorParts, " "), rootID, nil
}

func (s *Service) executeMessageScriptCommand(sub string, args []string) error {
	switch sub {
	case "rooms":
		if len(args) != 0 {
			return fmt.Errorf("usage: message rooms")
		}
		_ = s.ListMessageRooms()
	case "post":
		if len(args) < 2 {
			return fmt.Errorf("usage: message post <room> <text>")
		}
		s.PostMessage(args[0], strings.Join(args[1:], " "))
	case "ls":
		room, err := optionalSingleArg(args, "usage: message ls [room]")
		if err != nil {
			return err
		}
		_ = s.ListMessages(room)
	default:
		return fmt.Errorf("unsupported message command %q", sub)
	}
	return nil
}

func (s *Service) executeBoardScriptCommand(sub string, args []string) error {
	switch sub {
	case "post":
		if len(args) < 2 {
			return fmt.Errorf("usage: board post <board> <text>")
		}
		s.PostBoard(args[0], strings.Join(args[1:], " "))
	case "ls":
		board, err := optionalSingleArg(args, "usage: board ls [board]")
		if err != nil {
			return err
		}
		_ = s.ListBoard(board)
	default:
		return fmt.Errorf("unsupported board command %q", sub)
	}
	return nil
}

func (s *Service) executeArtifactScriptCommand(sub string, args []string) error {
	switch sub {
	case "create":
		if len(args) < 2 {
			return fmt.Errorf("usage: artifact create <kind> <title>")
		}
		s.CreateArtifact(args[0], strings.Join(args[1:], " "))
	case "history":
		if len(args) != 1 {
			return fmt.Errorf("usage: artifact history <artifact>")
		}
		artifactID, err := s.resolveLatestArtifactID(args[0])
		if err != nil {
			return err
		}
		if _, ok := s.ArtifactHistory(artifactID); !ok {
			return fmt.Errorf("artifact not found")
		}
	default:
		return fmt.Errorf("unsupported artifact command %q", sub)
	}
	return nil
}

func (s *Service) executeCanvasScriptCommand(sub string, args []string) error {
	switch sub {
	case "annotate":
		if len(args) < 2 {
			return fmt.Errorf("usage: canvas annotate <canvas> <text>")
		}
		s.AnnotateCanvas(args[0], strings.Join(args[1:], " "))
	case "ls":
		canvas, err := optionalSingleArg(args, "usage: canvas ls [canvas]")
		if err != nil {
			return err
		}
		_ = s.ListCanvas(canvas)
	default:
		return fmt.Errorf("unsupported canvas command %q", sub)
	}
	return nil
}

func (s *Service) executeSessionScriptCommand(sub string, args []string) error {
	switch sub {
	case "create":
		if len(args) < 2 {
			return fmt.Errorf("usage: session create <kind> <target>")
		}
		s.CreateSession(args[0], args[1])
	case "join":
		if len(args) < 2 {
			return fmt.Errorf("usage: session join <session> <participant>")
		}
		sessionID, err := s.resolveLatestSessionID(args[0])
		if err != nil {
			return err
		}
		if _, ok := s.JoinSession(sessionID, args[1]); !ok {
			return fmt.Errorf("session not found")
		}
	case "members":
		if len(args) != 1 {
			return fmt.Errorf("usage: session members <session>")
		}
		sessionID, err := s.resolveLatestSessionID(args[0])
		if err != nil {
			return err
		}
		if _, ok := s.ListSessionParticipants(sessionID); !ok {
			return fmt.Errorf("session not found")
		}
	default:
		return fmt.Errorf("unsupported session command %q", sub)
	}
	return nil
}

func (s *Service) executeIdentityScriptCommand(sub string, args []string) error {
	if sub != "ack" {
		return fmt.Errorf("unsupported identity command %q", sub)
	}
	if len(args) == 0 {
		return fmt.Errorf("usage: identity ack <show|record>")
	}
	switch action := strings.ToLower(strings.TrimSpace(args[0])); action {
	case "record":
		return s.executeIdentityAckRecord(args[1:])
	case "show":
		return s.executeIdentityAckShow(args[1:])
	default:
		return fmt.Errorf("unsupported identity ack action %q", action)
	}
}

func (s *Service) executeIdentityAckRecord(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: identity ack record <subject-ref> --actor <actor-ref> [--mode <mode>]")
	}
	subjectRef := strings.TrimSpace(args[0])
	if subjectRef == "" {
		return fmt.Errorf("subject-ref is required")
	}
	actor, mode, err := parseIdentityAckRecordFlags(args[1:])
	if err != nil {
		return err
	}
	if _, ok := s.RecordAcknowledgement(subjectRef, actor, mode); !ok {
		return fmt.Errorf("invalid acknowledgement")
	}
	return nil
}

func parseIdentityAckRecordFlags(rest []string) (string, string, error) {
	actor := ""
	mode := "read"
	for i := 0; i < len(rest); i++ {
		token := rest[i]
		switch token {
		case "--actor":
			if i+1 >= len(rest) {
				return "", "", fmt.Errorf("--actor requires an actor-ref")
			}
			actor = strings.TrimSpace(rest[i+1])
			i++
		case "--mode":
			if i+1 >= len(rest) {
				return "", "", fmt.Errorf("--mode requires a mode")
			}
			mode = strings.TrimSpace(rest[i+1])
			i++
		default:
			return "", "", fmt.Errorf("unsupported flag %q", token)
		}
	}
	if actor == "" {
		return "", "", fmt.Errorf("--actor is required")
	}
	return actor, mode, nil
}

func (s *Service) executeIdentityAckShow(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: identity ack show <subject-ref>")
	}
	subjectRef := strings.TrimSpace(args[0])
	if subjectRef == "" {
		return fmt.Errorf("subject-ref is required")
	}
	_ = s.GetAcknowledgements(subjectRef)
	return nil
}

func (s *Service) executeMemoryScriptCommand(sub string, args []string) error {
	switch sub {
	case "remember":
		if len(args) < 2 {
			return fmt.Errorf("usage: memory remember <scope> <text>")
		}
		s.Remember(args[0], strings.Join(args[1:], " "))
	case "recall":
		if len(args) < 1 {
			return fmt.Errorf("usage: memory recall <text>")
		}
		_ = s.Recall(strings.Join(args, " "))
	default:
		return fmt.Errorf("unsupported memory command %q", sub)
	}
	return nil
}

func (s *Service) executeSimScriptCommand(sub string, args []string) error {
	switch sub {
	case "device":
		return s.executeSimDeviceScriptCommand(args)
	case "input":
		return s.executeSimInputScriptCommand(args)
	case "expect":
		return s.executeSimExpectScriptCommand(args)
	case "record":
		return s.executeSimRecordScriptCommand(args)
	default:
		return fmt.Errorf("unsupported sim command %q", sub)
	}
}

func (s *Service) executeSimDeviceScriptCommand(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: sim device <new|rm> <id>")
	}
	action := strings.ToLower(args[0])
	deviceID := args[1]
	switch action {
	case "new":
		caps, err := parseSimDeviceCaps(args[2:])
		if err != nil {
			return err
		}
		s.SimDeviceUpsert(deviceID, caps)
	case "rm":
		if ok := s.SimDeviceDelete(deviceID); !ok {
			return fmt.Errorf("sim device not found")
		}
	default:
		return fmt.Errorf("unsupported sim device action %q", action)
	}
	return nil
}

func parseSimDeviceCaps(rest []string) ([]string, error) {
	caps := []string{}
	for i := 0; i < len(rest); i++ {
		if rest[i] != "--caps" {
			return nil, fmt.Errorf("unsupported sim device flag %q", rest[i])
		}
		if i+1 >= len(rest) {
			return nil, fmt.Errorf("--caps requires comma-separated values")
		}
		caps = strings.Split(rest[i+1], ",")
		i++
	}
	return caps, nil
}

func (s *Service) executeSimInputScriptCommand(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: sim input <id> <component-id> <action> [<value>]")
	}
	value := strings.Join(args[3:], " ")
	if _, ok := s.SimRecordInput(args[0], args[1], args[2], value); !ok {
		return fmt.Errorf("sim device not found")
	}
	return nil
}

func (s *Service) executeSimExpectScriptCommand(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: sim expect <id> <ui|message> <selector> [--within <duration>]")
	}
	selector, within, err := parseSimExpectSelector(args[2:])
	if err != nil {
		return err
	}
	result, ok := s.SimExpect(args[0], args[1], selector, within)
	if !ok {
		return fmt.Errorf("sim device not found")
	}
	if !result.Matched {
		return fmt.Errorf("expectation not matched")
	}
	return nil
}

func parseSimExpectSelector(rest []string) (string, time.Duration, error) {
	selectorParts := make([]string, 0, len(rest))
	within := time.Duration(0)
	for i := 0; i < len(rest); i++ {
		token := rest[i]
		if token == "--within" {
			parsed, err := parsePositiveDurationFlag(rest, &i, "--within")
			if err != nil {
				return "", 0, err
			}
			within = parsed
			continue
		}
		if strings.HasPrefix(token, "--") {
			return "", 0, fmt.Errorf("unsupported flag %q", token)
		}
		selectorParts = append(selectorParts, token)
	}
	if len(selectorParts) == 0 {
		return "", 0, fmt.Errorf("selector is required")
	}
	return strings.Join(selectorParts, " "), within, nil
}

func (s *Service) executeSimRecordScriptCommand(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: sim record <id> [--duration <duration>]")
	}
	duration, err := parseSimRecordDuration(args[1:])
	if err != nil {
		return err
	}
	if _, ok := s.SimRecord(args[0], duration); !ok {
		return fmt.Errorf("sim device not found")
	}
	return nil
}

func parseSimRecordDuration(rest []string) (time.Duration, error) {
	duration := time.Duration(0)
	for i := 0; i < len(rest); i++ {
		if rest[i] != "--duration" {
			return 0, fmt.Errorf("unsupported flag %q", rest[i])
		}
		parsed, err := parsePositiveDurationFlag(rest, &i, "--duration")
		if err != nil {
			return 0, err
		}
		duration = parsed
	}
	return duration, nil
}

func parsePositiveDurationFlag(tokens []string, index *int, flag string) (time.Duration, error) {
	if *index+1 >= len(tokens) {
		return 0, fmt.Errorf("%s requires a duration", flag)
	}
	raw := tokens[*index+1]
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("invalid %s duration %q", flag, raw)
	}
	(*index)++
	return parsed, nil
}

func optionalSingleArg(args []string, usage string) (string, error) {
	if len(args) > 1 {
		return "", fmt.Errorf("%s", usage)
	}
	if len(args) == 1 {
		return args[0], nil
	}
	return "", nil
}

func (s *Service) resolveLatestArtifactID(raw string) (string, error) {
	artifactID := strings.TrimSpace(raw)
	if !strings.EqualFold(artifactID, "latest") {
		return artifactID, nil
	}
	artifacts := s.ListArtifacts()
	if len(artifacts) == 0 {
		return "", fmt.Errorf("artifact not found")
	}
	return artifacts[len(artifacts)-1].ID, nil
}

func (s *Service) resolveLatestSessionID(raw string) (string, error) {
	sessionID := strings.TrimSpace(raw)
	if !strings.EqualFold(sessionID, "latest") {
		return sessionID, nil
	}
	sessions := s.ListSessions()
	if len(sessions) == 0 {
		return "", fmt.Errorf("session not found")
	}
	return sessions[len(sessions)-1].ID, nil
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
