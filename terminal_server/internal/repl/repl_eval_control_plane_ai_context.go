package repl

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

func (s *state) evalControlPlaneAIOpsContext(ctx context.Context, _ string, args []string, jsonOut bool) error {
	if strings.TrimSpace(s.session) == "" {
		return errors.New("ai context requires session id (TERMINALS_REPL_SESSION_ID)")
	}
	action := "show"
	if len(args) > 1 {
		action = strings.ToLower(strings.TrimSpace(args[1]))
	}
	switch action {
	case "show":
		return s.evalAIOpsContextShow(ctx, jsonOut)
	case "add":
		return s.evalAIOpsContextAdd(ctx, args, jsonOut)
	case "pin":
		return s.evalAIOpsContextPin(ctx, args, jsonOut)
	case "unpin":
		return s.evalAIOpsContextUnpin(ctx, args, jsonOut)
	case "clear":
		return s.evalAIOpsContextClear(ctx, jsonOut)
	default:
		return fmt.Errorf("unknown command: ai context %s", action)
	}
}

func (s *state) evalAIOpsContextShow(ctx context.Context, jsonOut bool) error {
	body, err := s.fetchJSONQuery(ctx, "/admin/api/repl/ai/context", url.Values{"session_id": {s.session}})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	pinned, _ := body["pinned"].([]any)
	if _, err := fmt.Fprintf(s.out, "session: %s\n", toString(body["session_id"])); err != nil {
		return err
	}
	if len(pinned) == 0 {
		_, err := fmt.Fprintln(s.out, "pinned: (none)")
		return err
	}
	if _, err := fmt.Fprintln(s.out, "pinned:"); err != nil {
		return err
	}
	for _, ref := range pinned {
		if _, err := fmt.Fprintf(s.out, "- %s\n", toString(ref)); err != nil {
			return err
		}
	}
	return nil
}

func (s *state) evalAIOpsContextAdd(ctx context.Context, args []string, jsonOut bool) error {
	if len(args) < 3 {
		return errors.New("usage: ai context add <ref>")
	}
	ref := strings.TrimSpace(args[2])
	body, err := s.postFormJSON(ctx, "/admin/api/repl/ai/context", url.Values{
		"session_id": {s.session},
		"ref":        {ref},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  added context ref for next turn: %s\n", toString(body["ref"]))
	return err
}

func (s *state) evalAIOpsContextPin(ctx context.Context, args []string, jsonOut bool) error {
	if len(args) < 3 {
		return errors.New("usage: ai context pin <ref>")
	}
	ref := strings.TrimSpace(args[2])
	body, err := s.postFormJSON(ctx, "/admin/api/repl/ai/context/pin", url.Values{
		"session_id": {s.session},
		"ref":        {ref},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  pinned context ref: %s\n", ref)
	return err
}

func (s *state) evalAIOpsContextUnpin(ctx context.Context, args []string, jsonOut bool) error {
	if len(args) < 3 {
		return errors.New("usage: ai context unpin <ref>")
	}
	ref := strings.TrimSpace(args[2])
	body, err := s.postFormJSON(ctx, "/admin/api/repl/ai/context/unpin", url.Values{
		"session_id": {s.session},
		"ref":        {ref},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  unpinned context ref: %s\n", ref)
	return err
}

func (s *state) evalAIOpsContextClear(ctx context.Context, jsonOut bool) error {
	body, err := s.postFormJSON(ctx, "/admin/api/repl/ai/context/clear", url.Values{
		"session_id": {s.session},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintln(s.out, "OK  cleared pinned context refs")
	return err
}
