package repl

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

func (s *state) evalControlPlaneSession(ctx context.Context, sub string, args []string, jsonOut bool) error {
	switch sub {
	case "ls":
		return s.evalCollabSessionLs(ctx, jsonOut)
	case "create":
		return s.evalCollabSessionCreate(ctx, args, jsonOut)
	case "show":
		return s.evalCollabSessionShow(ctx, args, jsonOut)
	case "members":
		return s.evalCollabSessionMembers(ctx, args, jsonOut)
	case "join":
		return s.evalCollabSessionJoin(ctx, args, jsonOut)
	case "leave":
		return s.evalCollabSessionLeave(ctx, args, jsonOut)
	case "attach":
		return s.evalCollabSessionAttach(ctx, args, jsonOut)
	case "detach":
		return s.evalCollabSessionDetach(ctx, args, jsonOut)
	case "control":
		return s.evalCollabSessionControl(ctx, args, jsonOut)
	default:
		return fmt.Errorf("unknown command: session %s", sub)
	}
}

func (s *state) evalCollabSessionLs(ctx context.Context, jsonOut bool) error {
	body, err := s.fetchJSON(ctx, "/admin/api/session")
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
		rows = append(rows, []string{toString(row["id"]), toString(row["kind"]), toString(row["target"])})
	}
	return printTable(s.out, []string{"ID", "KIND", "TARGET"}, rows)
}

func (s *state) evalCollabSessionCreate(ctx context.Context, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 2 {
		return errors.New("usage: session create <kind> <target>")
	}
	body, err := s.postFormJSON(ctx, "/admin/api/session/create", url.Values{
		"kind":   {plain[0]},
		"target": {plain[1]},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	sessionID := ""
	if sessionMap, ok := body["session"].(map[string]any); ok {
		sessionID = toString(sessionMap["id"])
	}
	_, err = fmt.Fprintf(s.out, "OK  session=%s\n", sessionID)
	return err
}

func (s *state) evalCollabSessionShow(ctx context.Context, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 1 {
		return errors.New("usage: session show <session>")
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/session/show", url.Values{"session_id": {plain[0]}})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	sessionMap, _ := body["session"].(map[string]any)
	if sessionMap == nil {
		return writeJSON(s.out, body)
	}
	rows := [][]string{
		{"session", toString(sessionMap["id"])},
		{"kind", toString(sessionMap["kind"])},
		{"target", toString(sessionMap["target"])},
	}
	if err := printTable(s.out, []string{"FIELD", "VALUE"}, rows); err != nil {
		return err
	}
	participants, _ := sessionMap["participants"].([]any)
	memberRows := make([][]string, 0, len(participants))
	for _, item := range participants {
		member, _ := item.(map[string]any)
		if member == nil {
			continue
		}
		memberRows = append(memberRows, []string{
			toString(member["identity_id"]),
			toString(member["joined_at"]),
		})
	}
	return printTable(s.out, []string{"PARTICIPANT", "JOINED_AT"}, memberRows)
}

func (s *state) evalCollabSessionMembers(ctx context.Context, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 1 {
		return errors.New("usage: session members <session>")
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/session/members", url.Values{"session_id": {plain[0]}})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	participants, _ := body["participants"].([]any)
	rows := make([][]string, 0, len(participants))
	for _, item := range participants {
		member, _ := item.(map[string]any)
		if member == nil {
			continue
		}
		rows = append(rows, []string{
			toString(member["identity_id"]),
			toString(member["joined_at"]),
		})
	}
	return printTable(s.out, []string{"PARTICIPANT", "JOINED_AT"}, rows)
}

func (s *state) evalCollabSessionJoin(ctx context.Context, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 2 {
		return errors.New("usage: session join <session> <participant>")
	}
	body, err := s.postFormJSON(ctx, "/admin/api/session/join", url.Values{
		"session_id":  {plain[0]},
		"participant": {plain[1]},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	sessionID := plain[0]
	if sessionMap, ok := body["session"].(map[string]any); ok {
		if id := toString(sessionMap["id"]); id != "" {
			sessionID = id
		}
	}
	_, err = fmt.Fprintf(s.out, "OK  session=%s participant=%s action=join\n", sessionID, plain[1])
	return err
}

func (s *state) evalCollabSessionLeave(ctx context.Context, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 2 {
		return errors.New("usage: session leave <session> <participant>")
	}
	body, err := s.postFormJSON(ctx, "/admin/api/session/leave", url.Values{
		"session_id":  {plain[0]},
		"participant": {plain[1]},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	sessionID := plain[0]
	if sessionMap, ok := body["session"].(map[string]any); ok {
		if id := toString(sessionMap["id"]); id != "" {
			sessionID = id
		}
	}
	_, err = fmt.Fprintf(s.out, "OK  session=%s participant=%s action=leave\n", sessionID, plain[1])
	return err
}

func (s *state) evalCollabSessionAttach(ctx context.Context, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 2 {
		return errors.New("usage: session attach <session> <device-ref>")
	}
	body, err := s.postFormJSON(ctx, "/admin/api/session/attach", url.Values{
		"session_id": {plain[0]},
		"device_ref": {plain[1]},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  session=%s device=%s action=attach\n", plain[0], plain[1])
	return err
}

func (s *state) evalCollabSessionDetach(ctx context.Context, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 2 {
		return errors.New("usage: session detach <session> <device-ref>")
	}
	body, err := s.postFormJSON(ctx, "/admin/api/session/detach", url.Values{
		"session_id": {plain[0]},
		"device_ref": {plain[1]},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  session=%s device=%s action=detach\n", plain[0], plain[1])
	return err
}

func (s *state) evalCollabSessionControl(ctx context.Context, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 1 {
		return errors.New("usage: session control <request|grant|revoke>")
	}
	action := strings.ToLower(plain[0])
	switch action {
	case "request":
		return s.evalCollabSessionControlRequest(ctx, plain, jsonOut)
	case "grant":
		return s.evalCollabSessionControlGrant(ctx, plain, jsonOut)
	case "revoke":
		return s.evalCollabSessionControlRevoke(ctx, plain, jsonOut)
	default:
		return fmt.Errorf("unknown command: session control %s", action)
	}
}

func (s *state) evalCollabSessionControlRequest(ctx context.Context, plain []string, jsonOut bool) error {
	if len(plain) < 3 {
		return errors.New("usage: session control request <session> <participant> [control-type]")
	}
	controlType := ""
	if len(plain) > 3 {
		controlType = plain[3]
	}
	body, err := s.postFormJSON(ctx, "/admin/api/session/control/request", url.Values{
		"session_id":   {plain[1]},
		"participant":  {plain[2]},
		"control_type": {controlType},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  session=%s participant=%s action=control.request type=%s\n", plain[1], plain[2], defaultIfBlank(controlType, "interactive"))
	return err
}

func (s *state) evalCollabSessionControlGrant(ctx context.Context, plain []string, jsonOut bool) error {
	if len(plain) < 3 {
		return errors.New("usage: session control grant <session> <participant> [granted-by] [control-type]")
	}
	grantedBy := ""
	if len(plain) > 3 {
		grantedBy = plain[3]
	}
	controlType := ""
	if len(plain) > 4 {
		controlType = plain[4]
	}
	body, err := s.postFormJSON(ctx, "/admin/api/session/control/grant", url.Values{
		"session_id":   {plain[1]},
		"participant":  {plain[2]},
		"granted_by":   {grantedBy},
		"control_type": {controlType},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  session=%s participant=%s action=control.grant by=%s type=%s\n", plain[1], plain[2], defaultIfBlank(grantedBy, "system"), defaultIfBlank(controlType, "interactive"))
	return err
}

func (s *state) evalCollabSessionControlRevoke(ctx context.Context, plain []string, jsonOut bool) error {
	if len(plain) < 3 {
		return errors.New("usage: session control revoke <session> <participant> [revoked-by]")
	}
	revokedBy := ""
	if len(plain) > 3 {
		revokedBy = plain[3]
	}
	body, err := s.postFormJSON(ctx, "/admin/api/session/control/revoke", url.Values{
		"session_id":  {plain[1]},
		"participant": {plain[2]},
		"revoked_by":  {revokedBy},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  session=%s participant=%s action=control.revoke by=%s\n", plain[1], plain[2], defaultIfBlank(revokedBy, "system"))
	return err
}
