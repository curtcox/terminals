package repl

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

func (s *state) evalControlPlaneIdentity(ctx context.Context, sub string, args []string, jsonOut bool) error {
	switch sub {
	case "ls":
		return s.evalIdentityLs(ctx, jsonOut)
	case "show":
		return s.evalIdentityShow(ctx, args)
	case "groups":
		return s.evalIdentityGroups(ctx, jsonOut)
	case "resolve":
		return s.evalIdentityResolve(ctx, args, jsonOut)
	case "prefs":
		return s.evalIdentityPrefs(ctx, args)
	case "ack":
		return s.evalIdentityAck(ctx, args, jsonOut)
	default:
		return fmt.Errorf("unknown command: identity %s", sub)
	}
}

func (s *state) evalIdentityLs(ctx context.Context, jsonOut bool) error {
	body, err := s.fetchJSON(ctx, "/admin/api/identity")
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	items, _ := body["identities"].([]any)
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		row, _ := item.(map[string]any)
		if row == nil {
			continue
		}
		rows = append(rows, []string{toString(row["id"]), toString(row["display_name"])})
	}
	return printTable(s.out, []string{"ID", "NAME"}, rows)
}

func (s *state) evalIdentityShow(ctx context.Context, args []string) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 1 {
		return errors.New("usage: identity show <identity>")
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/identity/show", url.Values{"identity": {plain[0]}})
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalIdentityGroups(ctx context.Context, jsonOut bool) error {
	body, err := s.fetchJSON(ctx, "/admin/api/identity/groups")
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	groups, _ := body["groups"].([]any)
	rows := make([][]string, 0, len(groups))
	for _, group := range groups {
		rows = append(rows, []string{toString(group)})
	}
	return printTable(s.out, []string{"GROUP"}, rows)
}

func (s *state) evalIdentityResolve(ctx context.Context, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 1 {
		return errors.New("usage: identity resolve <audience>")
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/identity/resolve", url.Values{"audience": {plain[0]}})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	audience := toString(body["audience"])
	if audience != "" {
		if _, err := fmt.Fprintf(s.out, "audience: %s\n", audience); err != nil {
			return err
		}
	}
	items, _ := body["identities"].([]any)
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		row, _ := item.(map[string]any)
		if row == nil {
			continue
		}
		rows = append(rows, []string{toString(row["id"]), toString(row["display_name"])})
	}
	return printTable(s.out, []string{"ID", "NAME"}, rows)
}

func (s *state) evalIdentityPrefs(ctx context.Context, args []string) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 1 {
		return errors.New("usage: identity prefs <identity>")
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/identity/prefs", url.Values{"identity": {plain[0]}})
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalIdentityAck(ctx context.Context, args []string, jsonOut bool) error {
	actionTokens := nonFlagArgs(args[1:])
	if len(actionTokens) == 0 {
		return errors.New("usage: identity ack <ls|show|record>")
	}
	action := strings.ToLower(strings.TrimSpace(actionTokens[0]))
	switch action {
	case "ls":
		return s.evalIdentityAckLs(ctx, actionTokens)
	case "show":
		return s.evalIdentityAckShow(ctx, actionTokens)
	case "record":
		return s.evalIdentityAckRecord(ctx, args, actionTokens, jsonOut)
	default:
		return fmt.Errorf("unknown command: identity ack %s", action)
	}
}

func (s *state) evalIdentityAckLs(ctx context.Context, actionTokens []string) error {
	query := url.Values{}
	if len(actionTokens) > 1 {
		query.Set("subject_ref", actionTokens[1])
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/identity/ack", query)
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalIdentityAckShow(ctx context.Context, actionTokens []string) error {
	if len(actionTokens) < 2 {
		return errors.New("usage: identity ack show <subject-ref>")
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/identity/ack", url.Values{"subject_ref": {actionTokens[1]}})
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalIdentityAckRecord(ctx context.Context, args []string, actionTokens []string, jsonOut bool) error {
	if len(actionTokens) < 2 {
		return errors.New("usage: identity ack record <subject-ref> --actor <actor-ref> [--mode <mode>]")
	}
	actor := flagValue(args[1:], "--actor")
	if strings.TrimSpace(actor) == "" {
		return errors.New("usage: identity ack record <subject-ref> --actor <actor-ref> [--mode <mode>]")
	}
	mode := defaultIfBlank(flagValue(args[1:], "--mode"), "read")
	body, err := s.postFormJSON(ctx, "/admin/api/identity/ack", url.Values{
		"subject_ref": {actionTokens[1]},
		"actor":       {actor},
		"mode":        {mode},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  subject=%s actor=%s mode=%s action=ack.record\n", actionTokens[1], actor, mode)
	return err
}
