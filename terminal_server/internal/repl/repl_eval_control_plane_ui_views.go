package repl

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

func (s *state) evalUIViewsLs(ctx context.Context, jsonOut bool) error {
	body, err := s.fetchJSON(ctx, "/admin/api/ui/views")
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	items, _ := body["views"].([]any)
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		row, _ := item.(map[string]any)
		if row == nil {
			continue
		}
		rows = append(rows, []string{toString(row["view_id"]), toString(row["root_id"])})
	}
	return printTable(s.out, []string{"VIEW", "ROOT"}, rows)
}

func (s *state) evalUIViewsShow(ctx context.Context, args []string) error {
	plain := nonFlagArgs(args)
	if len(plain) < 1 {
		return errors.New("usage: ui views show <view-id>")
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/ui/views", url.Values{"view_id": {plain[0]}})
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalUIViewsRm(ctx context.Context, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args)
	if len(plain) < 1 {
		return errors.New("usage: ui views rm <view-id>")
	}
	body, err := s.postFormJSON(ctx, "/admin/api/ui/views/del", url.Values{"view_id": {plain[0]}})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  deleted=%s view=%s\n", toString(body["deleted"]), plain[0])
	return err
}
