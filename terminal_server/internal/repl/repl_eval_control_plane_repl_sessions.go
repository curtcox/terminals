package repl

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func (s *state) evalControlPlaneSessions(ctx context.Context, sub string, args []string, jsonOut bool) error {
	switch sub {
	case "ls":
		return s.evalReplSessionsLs(ctx, jsonOut)
	case "show":
		return s.evalReplSessionsShow(ctx, args)
	case "terminate":
		return s.evalReplSessionsTerminate(ctx, args, jsonOut)
	default:
		return fmt.Errorf("unknown command: sessions %s", sub)
	}
}

func (s *state) evalReplSessionsLs(ctx context.Context, jsonOut bool) error {
	body, err := s.fetchJSON(ctx, "/admin/api/repl/sessions")
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	items, _ := body["sessions"].([]any)
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		row, _ := item.(map[string]any)
		if row == nil {
			continue
		}
		attached := toAnySlice(lookupMapAny(row, "attached_devices", "AttachedDevices"))
		rows = append(rows, []string{
			toString(lookupMapAny(row, "id", "ID")),
			toString(lookupMapAny(row, "origin", "Origin")),
			toString(lookupMapAny(row, "agent_capability", "AgentCapability")),
			toString(lookupMapAny(row, "owner_activation_id", "OwnerActivationID")),
			strconv.Itoa(len(attached)),
			toString(lookupMapAny(row, "idle", "Idle")),
			formatUnixMillis(lookupMapAny(row, "created_at", "CreatedAt")),
		})
	}
	return printTable(s.out, []string{"ID", "ORIGIN", "CAPABILITY", "OWNER", "ATTACHED", "IDLE", "CREATED"}, rows)
}

func (s *state) evalReplSessionsShow(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return errors.New("usage: sessions show <session>")
	}
	sessionID := strings.TrimSpace(args[1])
	if sessionID == "" {
		return errors.New("usage: sessions show <session>")
	}
	body, err := s.fetchJSON(ctx, "/admin/api/repl/sessions/"+url.PathEscape(sessionID))
	if err != nil {
		return err
	}
	return writeJSON(s.out, body["session"])
}

func (s *state) evalReplSessionsTerminate(ctx context.Context, args []string, jsonOut bool) error {
	if len(args) < 2 {
		return errors.New("usage: sessions terminate <session>")
	}
	sessionID := strings.TrimSpace(args[1])
	if sessionID == "" {
		return errors.New("usage: sessions terminate <session>")
	}
	body, err := s.deleteJSON(ctx, "/admin/api/repl/sessions/"+url.PathEscape(sessionID))
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  terminated session %s\n", sessionID)
	return err
}
