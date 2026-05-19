package repl

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

func (s *state) evalControlPlane(ctx context.Context, group string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing subcommand for %s", group)
	}
	sub := strings.ToLower(args[0])
	jsonOut := hasFlag(args[1:], "--json")

	switch group {
	case "devices":
		return s.evalControlPlaneDevices(ctx, sub, args, jsonOut)
	case "sessions":
		return s.evalControlPlaneSessions(ctx, sub, args, jsonOut)
	case "identity":
		return s.evalControlPlaneIdentity(ctx, sub, args, jsonOut)
	case "session":
		return s.evalControlPlaneSession(ctx, sub, args, jsonOut)
	case "message":
		return s.evalControlPlaneMessage(ctx, sub, args, jsonOut)
	default:
		return s.evalControlPlaneMid(ctx, group, args, jsonOut)
	}
}

func (s *state) evalControlPlaneDevices(ctx context.Context, sub string, _ []string, jsonOut bool) error {
	if sub != "ls" {
		return fmt.Errorf("unknown command: devices %s", sub)
	}
	body, err := s.fetchJSON(ctx, "/admin/api/devices")
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	devices, _ := body["devices"].([]any)
	rows := make([][]string, 0, len(devices))
	for _, item := range devices {
		row, _ := item.(map[string]any)
		if row == nil {
			continue
		}
		caps := ""
		if m, ok := row["capabilities"].(map[string]any); ok {
			keys := make([]string, 0, len(m))
			for k := range m {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			caps = strings.Join(keys, ",")
		}
		rows = append(rows, []string{
			toString(row["device_id"]),
			toString(row["zone"]),
			caps,
			toString(row["state"]),
		})
	}
	return printTable(s.out, []string{"ID", "ZONE", "CAPS", "STATE"}, rows)
}
