package repl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// CONTENTS:
//   line  24  func (s *state) evalControlPlaneOps(ctx context.Context, group string, args []string, jsonOut bool) error
//   line  44  func (s *state) evalControlPlaneAppOps(ctx context.Context, sub string, args []string, jsonOut bool) error
//   line 124  func (s *state) evalControlPlaneAppsOps(ctx context.Context, sub string, args []string, jsonOut bool) error
//   line 527  func (s *state) evalControlPlaneConfigOps(ctx context.Context, sub string, jsonOut bool) error
//   line 542  func (s *state) evalControlPlaneDocsOps(sub string, args []string) error
//   line 617  func (s *state) evalControlPlaneLogsOps(ctx context.Context, group string, sub string, args []string) error
//   line 625  func (s *state) evalControlPlaneAIOps(ctx context.Context, sub string, args []string, jsonOut bool) error

func (s *state) evalControlPlaneOps(ctx context.Context, group string, args []string, jsonOut bool) error {
	sub := strings.ToLower(args[0])
	switch group {
	case "app":
		return s.evalControlPlaneOpsApp(ctx, sub, args, jsonOut)
	case "apps":
		return s.evalControlPlaneOpsApps(ctx, sub, args, jsonOut)
	case "config":
		return s.evalControlPlaneOpsConfig(ctx, sub, args, jsonOut)
	case "docs":
		return s.evalControlPlaneOpsDocs(ctx, sub, args, jsonOut)
	case "logs", "observe":
		return s.evalControlPlaneOpsLogs(ctx, group, sub, args, jsonOut)
	case "ai":
		return s.evalControlPlaneOpsAi(ctx, sub, args, jsonOut)
	default:
		return fmt.Errorf("unsupported command group: %s", group)
	}
}

func (s *state) evalControlPlaneAppOps(ctx context.Context, sub string, args []string, jsonOut bool) error {
	switch sub {
	case "ls":
		return s.evalControlPlaneAppOpsLs(ctx, sub, args, jsonOut)
	case "reload", "rollback":
		return s.evalControlPlaneAppOpsReload(ctx, sub, args, jsonOut)
	case "logs":
		return s.evalControlPlaneAppOpsLogs(ctx, sub, args, jsonOut)
	default:
		return fmt.Errorf("unknown command: app %s", sub)
	}
}

func (s *state) evalControlPlaneAppsOps(ctx context.Context, sub string, args []string, jsonOut bool) error {
	switch sub {
	case "migrate":
		return s.evalControlPlaneAppsOpsMigrate(ctx, sub, args, jsonOut)
	case "keys":
		return s.evalControlPlaneAppsOpsKeys(ctx, sub, args, jsonOut)
	default:
		return fmt.Errorf("unknown command: apps %s", sub)
	}
}

func (s *state) evalControlPlaneConfigOps(ctx context.Context, sub string, jsonOut bool) error {
	if sub != "show" {
		return fmt.Errorf("unknown command: config %s", sub)
	}
	body, err := s.fetchJSON(ctx, "/admin/api/status")
	if err != nil {
		return err
	}
	cfg := body["config"]
	if jsonOut {
		return writeJSON(s.out, cfg)
	}
	return writeJSON(s.out, cfg)
}

func (s *state) evalControlPlaneDocsOps(sub string, args []string) error {
	switch sub {
	case "ls":
		return s.evalControlPlaneDocsOpsLs(sub, args)
	case "search":
		return s.evalControlPlaneDocsOpsSearch(sub, args)
	case "open":
		return s.evalControlPlaneDocsOpsOpen(sub, args)
	case "examples":
		return s.evalControlPlaneDocsOpsExamples(sub, args)
	default:
		return fmt.Errorf("unknown command: docs %s", sub)
	}
}

func (s *state) evalControlPlaneLogsOps(ctx context.Context, group string, sub string, args []string) error {
	if sub != "tail" {
		return fmt.Errorf("unknown command: %s %s", group, sub)
	}
	query := strings.TrimSpace(strings.Join(args[1:], " "))
	return s.queryLogs(ctx, "", query)
}

func (s *state) evalControlPlaneAIOps(ctx context.Context, sub string, args []string, jsonOut bool) error {
	switch sub {
	case "providers":
		return s.evalControlPlaneAIOpsProviders(ctx, sub, args, jsonOut)
	case "models":
		return s.evalControlPlaneAIOpsModels(ctx, sub, args, jsonOut)
	case "use":
		return s.evalControlPlaneAIOpsUse(ctx, sub, args, jsonOut)
	case "status":
		return s.evalControlPlaneAIOpsStatus(ctx, sub, args, jsonOut)
	case "ask":
		return s.evalControlPlaneAIOpsAsk(ctx, sub, args, jsonOut)
	case "gen":
		return s.evalControlPlaneAIOpsGen(ctx, sub, args, jsonOut)
	case "run", "approve":
		return s.evalControlPlaneAIOpsRun(ctx, sub, args, jsonOut)
	case "reject":
		return s.evalControlPlaneAIOpsReject(ctx, sub, args, jsonOut)
	case "context":
		return s.evalControlPlaneAIOpsContext(ctx, sub, args, jsonOut)
	case "policy":
		return s.evalControlPlaneAIOpsPolicy(ctx, sub, args, jsonOut)
	case "history":
		return s.evalControlPlaneAIOpsHistory(ctx, sub, args, jsonOut)
	case "reset":
		return s.evalControlPlaneAIOpsReset(ctx, sub, args, jsonOut)
	default:
		return fmt.Errorf("unknown command: ai %s", sub)
	}
}

func (s *state) evalControlPlaneOpsApp(ctx context.Context, sub string, args []string, jsonOut bool) error {
	return s.evalControlPlaneAppOps(ctx, sub, args, jsonOut)
}

func (s *state) evalControlPlaneOpsApps(ctx context.Context, sub string, args []string, jsonOut bool) error {
	return s.evalControlPlaneAppsOps(ctx, sub, args, jsonOut)
}

func (s *state) evalControlPlaneOpsConfig(ctx context.Context, sub string, _ []string, jsonOut bool) error {
	return s.evalControlPlaneConfigOps(ctx, sub, jsonOut)
}

func (s *state) evalControlPlaneOpsDocs(_ context.Context, sub string, args []string, _ bool) error {
	return s.evalControlPlaneDocsOps(sub, args)
}

func (s *state) evalControlPlaneOpsLogs(ctx context.Context, group string, sub string, args []string, _ bool) error {
	return s.evalControlPlaneLogsOps(ctx, group, sub, args)
}

func (s *state) evalControlPlaneOpsAi(ctx context.Context, sub string, args []string, jsonOut bool) error {
	return s.evalControlPlaneAIOps(ctx, sub, args, jsonOut)
}

func (s *state) evalControlPlaneAppsOpsMigrate(ctx context.Context, _ string, args []string, jsonOut bool) error {
	if len(args) < 2 {
		return errors.New("usage: apps migrate <status|logs|retry|abort|drain-ready|reconcile>")
	}
	migrateSub := strings.TrimSpace(args[1])
	switch migrateSub {
	case "status":
		plain := nonFlagArgs(args[2:])
		if len(plain) < 1 {
			return errors.New("usage: apps migrate status <app>")
		}
		appName := strings.TrimSpace(plain[0])
		if appName == "" {
			return errors.New("usage: apps migrate status <app>")
		}
		body, err := s.fetchJSONQuery(ctx, "/admin/api/apps/migrate/status", url.Values{"app": {appName}})
		if err != nil {
			return err
		}
		if jsonOut {
			return writeJSON(s.out, body)
		}
		migration, _ := body["migration"].(map[string]any)
		recordSummary := migrationPendingRecordSummary(migration)
		_, err = fmt.Fprintf(
			s.out,
			"OK  app=%s verdict=%s steps=%v/%v last_step=%v pending_records=%s reconciliation_path=%s last_error=%q executor_ready=%v requires_drain=%v drain_ready=%v drain_timeout_seconds=%v drain_blocked_since=%s\n",
			appName,
			toString(migration["verdict"]),
			migration["steps_completed"],
			migration["steps_planned"],
			migration["last_step"],
			recordSummary,
			emptyAsNone(toString(migration["reconciliation_path"])),
			toString(migration["last_error"]),
			migration["executor_ready"],
			migration["requires_drain"],
			migration["drain_ready"],
			migration["drain_timeout_seconds"],
			emptyAsNone(toString(migration["drain_blocked_since"])),
		)
		return err
	case "logs":
		plain := nonFlagArgsSkippingFlagValues(args[2:], "--step")
		if len(plain) < 1 {
			return errors.New("usage: apps migrate logs <app> [--step <n>]")
		}
		appName := strings.TrimSpace(plain[0])
		if appName == "" {
			return errors.New("usage: apps migrate logs <app> [--step <n>]")
		}
		values := url.Values{"app": {appName}}
		if stepRaw := strings.TrimSpace(flagValue(args[2:], "--step")); stepRaw != "" {
			step, err := strconv.Atoi(stepRaw)
			if err != nil || step <= 0 {
				return errors.New("usage: apps migrate logs <app> [--step <n>]")
			}
			values.Set("step", strconv.Itoa(step))
		}
		body, err := s.fetchJSONQuery(ctx, "/admin/api/apps/migrate/logs", values)
		if err != nil {
			return err
		}
		if jsonOut {
			return writeJSON(s.out, body)
		}
		linesAny, _ := body["lines"].([]any)
		for _, line := range linesAny {
			if _, err := fmt.Fprintln(s.out, toString(line)); err != nil {
				return err
			}
		}
		_, err = fmt.Fprintf(
			s.out,
			"OK  app=%s lines=%d journal_exists=%v\n",
			appName,
			len(linesAny),
			body["journal_exists"],
		)
		return err
	case "retry", "abort":
		plain := nonFlagArgsSkippingFlagValues(args[2:], "--to")
		if len(plain) < 1 {
			if migrateSub == "abort" {
				return errors.New("usage: apps migrate abort <app> [--to <checkpoint|baseline>]")
			}
			return fmt.Errorf("usage: apps migrate %s <app>", migrateSub)
		}
		appName := strings.TrimSpace(plain[0])
		if appName == "" {
			if migrateSub == "abort" {
				return errors.New("usage: apps migrate abort <app> [--to <checkpoint|baseline>]")
			}
			return fmt.Errorf("usage: apps migrate %s <app>", migrateSub)
		}
		route := "/admin/api/apps/migrate/retry"
		values := url.Values{"app": {appName}}
		target := ""
		if migrateSub == "abort" {
			route = "/admin/api/apps/migrate/abort"
			target = strings.TrimSpace(flagValue(args[2:], "--to"))
			if target != "" {
				values.Set("to", target)
			}
		}
		body, err := s.postFormJSON(ctx, route, values)
		if err != nil {
			return err
		}
		if jsonOut {
			return writeJSON(s.out, body)
		}
		if migrateSub == "abort" {
			resolvedTarget := toString(body["to"])
			if resolvedTarget == "" {
				resolvedTarget = target
			}
			if resolvedTarget == "" {
				resolvedTarget = "checkpoint"
			}
			_, err = fmt.Fprintf(s.out, "OK  app=%s action=%s to=%s status=%s\n", appName, migrateSub, resolvedTarget, toString(body["status"]))
			return err
		}
		_, err = fmt.Fprintf(s.out, "OK  app=%s action=%s status=%s\n", appName, migrateSub, toString(body["status"]))
		return err
	case "drain-ready":
		plain := nonFlagArgs(args[2:])
		if len(plain) < 2 {
			return errors.New("usage: apps migrate drain-ready <app> <true|false>")
		}
		appName := strings.TrimSpace(plain[0])
		if appName == "" {
			return errors.New("usage: apps migrate drain-ready <app> <true|false>")
		}
		ready, err := strconv.ParseBool(strings.TrimSpace(plain[1]))
		if err != nil {
			return errors.New("usage: apps migrate drain-ready <app> <true|false>")
		}
		body, err := s.postFormJSON(ctx, "/admin/api/apps/migrate/drain-ready", url.Values{
			"app":   {appName},
			"ready": {strconv.FormatBool(ready)},
		})
		if err != nil {
			return err
		}
		if jsonOut {
			return writeJSON(s.out, body)
		}
		_, err = fmt.Fprintf(s.out, "OK  app=%s action=drain-ready ready=%t status=%s\n", appName, ready, toString(body["status"]))
		return err
	case "reconcile":
		plain := nonFlagArgs(args[2:])
		if len(plain) < 3 {
			return errors.New("usage: apps migrate reconcile <app> <record-id> <resolution>")
		}
		appName := strings.TrimSpace(plain[0])
		recordID := strings.TrimSpace(plain[1])
		resolution := strings.TrimSpace(plain[2])
		if appName == "" || recordID == "" || resolution == "" {
			return errors.New("usage: apps migrate reconcile <app> <record-id> <resolution>")
		}
		body, err := s.postFormJSON(ctx, "/admin/api/apps/migrate/reconcile", url.Values{
			"app":        {appName},
			"record_id":  {recordID},
			"resolution": {resolution},
		})
		if err != nil {
			return err
		}
		if jsonOut {
			return writeJSON(s.out, body)
		}
		_, err = fmt.Fprintf(s.out, "OK  app=%s action=reconcile status=%s\n", appName, toString(body["status"]))
		return err
	default:
		return fmt.Errorf("unknown command: apps migrate %s", migrateSub)
	}
}

func (s *state) evalControlPlaneAppsOpsKeys(ctx context.Context, _ string, args []string, jsonOut bool) error {
	if len(args) == 0 {
		return errors.New("usage: apps keys <ls|show|add|confirm|revoke|archive|rotate|rotate-installer|rotations|verify|log>")
	}
	keySub := strings.TrimSpace(args[0])
	switch keySub {
	case "ls":
		body, err := s.fetchJSON(ctx, "/admin/api/trust/keys")
		if err != nil {
			return err
		}
		if jsonOut {
			return writeJSON(s.out, body)
		}
		keys, _ := body["keys"].([]any)
		rows := make([][]string, 0, len(keys))
		for _, kAny := range keys {
			k, _ := kAny.(map[string]any)
			if k == nil {
				continue
			}
			rolesAny, _ := k["roles"].([]any)
			rolesStrs := make([]string, 0, len(rolesAny))
			for _, r := range rolesAny {
				if rs, ok := r.(string); ok {
					rolesStrs = append(rolesStrs, rs)
				}
			}
			rows = append(rows, []string{toString(k["key_id"]), strings.Join(rolesStrs, ","), toString(k["state"])})
		}
		return printTable(s.out, []string{"KEY_ID", "ROLES", "STATE"}, rows)
	case "show":
		if len(args) < 2 {
			return errors.New("usage: apps keys show <key_id>")
		}
		body, err := s.fetchJSON(ctx, "/admin/api/trust/keys")
		if err != nil {
			return err
		}
		want := strings.TrimSpace(args[1])
		keys, _ := body["keys"].([]any)
		for _, kAny := range keys {
			k, _ := kAny.(map[string]any)
			if k != nil && toString(k["key_id"]) == want {
				return writeJSON(s.out, k)
			}
		}
		return fmt.Errorf("key not found: %s", want)
	case "add":
		if len(args) < 3 {
			return errors.New("usage: apps keys add <key_id> <role[,role]>")
		}
		keyID := strings.TrimSpace(args[1])
		rolesStr := strings.TrimSpace(args[2])
		body, err := s.postJSON(ctx, "/admin/api/trust/keys", map[string]any{
			"key_id": keyID,
			"roles":  strings.Split(rolesStr, ","),
			"note":   strings.Join(args[3:], " "),
		})
		if err != nil {
			return err
		}
		if jsonOut {
			return writeJSON(s.out, body)
		}
		_, err = fmt.Fprintf(s.out, "OK  key_id=%s state=candidate\n", keyID)
		return err
	case "confirm":
		if len(args) < 2 {
			return errors.New("usage: apps keys confirm <key_id>")
		}
		keyID := strings.TrimSpace(args[1])
		body, err := s.postJSON(ctx, "/admin/api/trust/keys/confirm", map[string]any{"key_id": keyID})
		if err != nil {
			return err
		}
		if jsonOut {
			return writeJSON(s.out, body)
		}
		_, err = fmt.Fprintf(s.out, "OK  key_id=%s state=active\n", keyID)
		return err
	case "revoke":
		if len(args) < 2 {
			return errors.New("usage: apps keys revoke <key_id> [--reason <text>]")
		}
		keyID := strings.TrimSpace(args[1])
		reason := strings.Join(args[2:], " ")
		body, err := s.postJSON(ctx, "/admin/api/trust/keys/revoke", map[string]any{"key_id": keyID, "reason": reason})
		if err != nil {
			return err
		}
		if jsonOut {
			return writeJSON(s.out, body)
		}
		affected, _ := body["affected_apps"].([]any)
		_, err = fmt.Fprintf(s.out, "OK  key_id=%s state=revoked affected_apps=%d\n", keyID, len(affected))
		return err
	case "archive":
		if len(args) < 2 {
			return errors.New("usage: apps keys archive <key_id>")
		}
		keyID := strings.TrimSpace(args[1])
		body, err := s.postJSON(ctx, "/admin/api/trust/keys/archive", map[string]any{"key_id": keyID})
		if err != nil {
			return err
		}
		if jsonOut {
			return writeJSON(s.out, body)
		}
		_, err = fmt.Fprintf(s.out, "OK  key_id=%s state=archived\n", keyID)
		return err
	case "verify":
		body, err := s.fetchJSON(ctx, "/admin/api/trust/verify")
		if err != nil {
			return err
		}
		if jsonOut {
			return writeJSON(s.out, body)
		}
		_, err = fmt.Fprintf(s.out, "chain=%s entries=%v installer=%s\n",
			toString(body["chain_status"]), body["entry_count"], toString(body["installer_key"]))
		return err
	case "log":
		body, err := s.fetchJSON(ctx, "/admin/api/trust/log")
		if err != nil {
			return err
		}
		return writeJSON(s.out, body)
	case "rotations":
		body, err := s.fetchJSON(ctx, "/admin/api/trust/rotations")
		if err != nil {
			return err
		}
		return writeJSON(s.out, body)
	case "rotate":
		if len(args) < 2 {
			return errors.New("usage: apps keys rotate <--accept <json> | --rollback <seq> | --emit <old-key> <new-key> [names...]>")
		}
		flag := strings.TrimSpace(args[1])
		switch flag {
		case "--accept":
			if len(args) < 3 {
				return errors.New("usage: apps keys rotate --accept <rotation-json>")
			}
			rotJSON := strings.TrimSpace(args[2])
			var payload map[string]any
			if err := json.Unmarshal([]byte(rotJSON), &payload); err != nil {
				return fmt.Errorf("apps keys rotate --accept: invalid JSON: %w", err)
			}
			body, err := s.postJSON(ctx, "/admin/api/trust/keys/rotate", payload)
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(s.out, body)
			}
			_, err = fmt.Fprintf(s.out, "OK  old_key=%s new_key=%s accepted_seq=%v\n",
				toString(body["old_key"]), toString(body["new_key"]), body["accepted_seq"])
			return err
		case "--rollback":
			if len(args) < 3 {
				return errors.New("usage: apps keys rotate --rollback <accepted-seq>")
			}
			seqStr := strings.TrimSpace(args[2])
			var seq float64
			if _, err := fmt.Sscanf(seqStr, "%f", &seq); err != nil {
				return fmt.Errorf("apps keys rotate --rollback: invalid seq %q", seqStr)
			}
			body, err := s.postJSON(ctx, "/admin/api/trust/keys/rotate/rollback", map[string]any{"accepted_seq": int64(seq)})
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(s.out, body)
			}
			_, err = fmt.Fprintf(s.out, "OK  rolled_back_seq=%v\n", body["rolled_back_seq"])
			return err
		case "--emit":
			if len(args) < 4 {
				return errors.New("usage: apps keys rotate --emit <old-key-id> <new-key-id> [name ...]")
			}
			oldKey := strings.TrimSpace(args[2])
			newKey := strings.TrimSpace(args[3])
			names := args[4:]
			tmpl := map[string]any{
				"old_stmt": map[string]any{
					"schema":      "rotation-stmt/1",
					"old_key":     oldKey,
					"new_key":     newKey,
					"proposed_at": "<unix-seconds>",
					"name_scope":  names,
					"reason":      "<optional>",
					"sig_old":     "<base64: signature by old_key over canonical JSON of old_stmt fields>",
				},
				"new_stmt": map[string]any{
					"schema":              "rotation-stmt/1",
					"old_key_stmt_digest": "<sha256 of serialised old_stmt payload>",
					"new_key":             newKey,
					"accept_at":           "<unix-seconds>",
					"sig_new":             "<base64: signature by new_key over canonical JSON of new_stmt fields>",
				},
			}
			return writeJSON(s.out, tmpl)
		default:
			return fmt.Errorf("unknown flag for apps keys rotate: %s", flag)
		}
	case "rotate-installer":
		body, err := s.postJSON(ctx, "/admin/api/trust/keys/rotate-installer", map[string]any{})
		if err != nil {
			return err
		}
		if jsonOut {
			return writeJSON(s.out, body)
		}
		_, err = fmt.Fprintf(s.out, "OK  new_installer_key_id=%s\n", toString(body["new_installer_key_id"]))
		return err
	default:
		return fmt.Errorf("unknown command: apps keys %s", keySub)
	}
}

func (s *state) evalControlPlaneAIOpsProviders(ctx context.Context, _ string, _ []string, jsonOut bool) error {
	body, err := s.fetchJSON(ctx, "/admin/api/repl/ai/providers")
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	items, _ := body["providers"].([]any)
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		row, _ := item.(map[string]any)
		if row == nil {
			continue
		}
		models, _ := row["models"].([]any)
		rows = append(rows, []string{
			toString(row["name"]),
			toString(row["default_model"]),
			strconv.Itoa(len(models)),
		})
	}
	return printTable(s.out, []string{"PROVIDER", "DEFAULT", "MODELS"}, rows)
}

func (s *state) evalControlPlaneAIOpsModels(ctx context.Context, _ string, args []string, jsonOut bool) error {
	provider := ""
	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "--") {
			continue
		}
		provider = strings.TrimSpace(arg)
		break
	}
	query := url.Values{}
	if provider != "" {
		query.Set("provider", provider)
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/repl/ai/models", query)
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	models, _ := body["models"].([]any)
	for _, model := range models {
		if _, err := fmt.Fprintln(s.out, toString(model)); err != nil {
			return err
		}
	}
	return nil
}

func (s *state) evalControlPlaneAIOpsUse(ctx context.Context, _ string, args []string, jsonOut bool) error {
	if len(args) < 3 {
		return errors.New("usage: ai use <provider> <model>")
	}
	if strings.TrimSpace(s.session) == "" {
		return errors.New("ai session selection requires session id (TERMINALS_REPL_SESSION_ID)")
	}
	provider := strings.TrimSpace(args[1])
	model := strings.TrimSpace(args[2])
	body, err := s.postFormJSON(ctx, "/admin/api/repl/ai/selection", url.Values{
		"session_id": {s.session},
		"provider":   {provider},
		"model":      {model},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "provider: %s  model: %s (sticky for %s)\n", toString(body["provider"]), toString(body["model"]), s.session)
	return err
}

func (s *state) evalControlPlaneAIOpsStatus(ctx context.Context, _ string, _ []string, jsonOut bool) error {
	if strings.TrimSpace(s.session) == "" {
		return errors.New("ai status requires session id (TERMINALS_REPL_SESSION_ID)")
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/repl/ai/selection", url.Values{"session_id": {s.session}})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "session: %s\nprovider: %s\nmodel: %s\n", toString(body["session_id"]), toString(body["provider"]), toString(body["model"]))
	return err
}

func (s *state) evalControlPlaneAIOpsAsk(ctx context.Context, _ string, args []string, jsonOut bool) error {
	if strings.TrimSpace(s.session) == "" {
		return errors.New("ai ask requires session id (TERMINALS_REPL_SESSION_ID)")
	}
	plain := nonFlagArgs(args[1:])
	if len(plain) == 0 {
		return errors.New("usage: ai ask <prompt>")
	}
	prompt := strings.TrimSpace(strings.Join(plain, " "))
	body, err := s.postFormJSON(ctx, "/admin/api/repl/ai/ask", url.Values{
		"session_id": {s.session},
		"prompt":     {prompt},
	})
	if err != nil {
		return err
	}
	s.capturePendingAIProposal(body)
	if jsonOut {
		return writeJSON(s.out, body)
	}
	if _, err := fmt.Fprintf(s.out, "session: %s\nprovider: %s\nmodel: %s\nthread: %s\n", toString(body["session_id"]), toString(body["provider"]), toString(body["model"]), toString(body["thread"])); err != nil {
		return err
	}
	_, err = fmt.Fprintf(s.out, "answer:\n%s\n", toString(body["answer"]))
	return err
}

func (s *state) evalControlPlaneAIOpsGen(ctx context.Context, _ string, args []string, jsonOut bool) error {
	if strings.TrimSpace(s.session) == "" {
		return errors.New("ai gen requires session id (TERMINALS_REPL_SESSION_ID)")
	}
	plain := nonFlagArgs(args[1:])
	if len(plain) == 0 {
		return errors.New("usage: ai gen <description>")
	}
	description := strings.TrimSpace(strings.Join(plain, " "))
	body, err := s.postFormJSON(ctx, "/admin/api/repl/ai/gen", url.Values{
		"session_id":  {s.session},
		"description": {description},
	})
	if err != nil {
		return err
	}
	s.capturePendingAIProposal(body)
	if jsonOut {
		return writeJSON(s.out, body)
	}
	if _, err := fmt.Fprintf(s.out, "session: %s\nprovider: %s\nmodel: %s\nthread: %s\n", toString(body["session_id"]), toString(body["provider"]), toString(body["model"]), toString(body["thread"])); err != nil {
		return err
	}
	_, err = fmt.Fprintf(s.out, "generated:\n%s\n", toString(body["output"]))
	return err
}

func (s *state) evalControlPlaneAIOpsRun(ctx context.Context, _ string, _ []string, jsonOut bool) error {
	pending := s.pending
	if pending == nil || strings.TrimSpace(pending.Command) == "" {
		return errors.New("no pending AI proposal (run ai ask/ai gen first)")
	}
	command := strings.TrimSpace(pending.Command)
	s.pending = nil
	if jsonOut {
		if err := writeJSON(s.out, map[string]any{"status": "approved", "command": command}); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(s.out, "OK  approved pending command: %s\n", command); err != nil {
		return err
	}
	exit, err := s.eval(ctx, command)
	if err != nil {
		return err
	}
	if exit {
		_, err = fmt.Fprintln(s.out, "warning: approved command requested REPL exit and was ignored")
		return err
	}
	return nil
}

func (s *state) evalControlPlaneAIOpsReject(_ context.Context, _ string, _ []string, jsonOut bool) error {
	pending := s.pending
	if pending == nil || strings.TrimSpace(pending.Command) == "" {
		return errors.New("no pending AI proposal (run ai ask/ai gen first)")
	}
	s.pending = nil
	if jsonOut {
		if err := writeJSON(s.out, map[string]any{"status": "rejected", "command": pending.Command}); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(s.out, "OK  rejected pending command: %s\n", pending.Command)
	return err
}

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
	case "add":
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
	case "pin":
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
	case "unpin":
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
	case "clear":
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
	default:
		return fmt.Errorf("unknown command: ai context %s", action)
	}
}

func (s *state) evalControlPlaneAIOpsPolicy(ctx context.Context, _ string, args []string, jsonOut bool) error {
	if strings.TrimSpace(s.session) == "" {
		return errors.New("ai policy requires session id (TERMINALS_REPL_SESSION_ID)")
	}
	action := "show"
	if len(args) > 1 {
		action = strings.ToLower(strings.TrimSpace(args[1]))
	}
	switch action {
	case "show":
		body, err := s.fetchJSONQuery(ctx, "/admin/api/repl/ai/policy", url.Values{"session_id": {s.session}})
		if err != nil {
			return err
		}
		if jsonOut {
			return writeJSON(s.out, body)
		}
		_, err = fmt.Fprintf(s.out, "session: %s\npolicy: %s\n", toString(body["session_id"]), toString(body["policy"]))
		return err
	case "set":
		if len(args) < 3 {
			return errors.New("usage: ai policy set <auto-readonly|prompt-all|prompt-mutating>")
		}
		policy := strings.TrimSpace(args[2])
		body, err := s.postFormJSON(ctx, "/admin/api/repl/ai/policy", url.Values{
			"session_id": {s.session},
			"policy":     {policy},
		})
		if err != nil {
			return err
		}
		if jsonOut {
			return writeJSON(s.out, body)
		}
		_, err = fmt.Fprintf(s.out, "OK  policy set to %s\n", toString(body["policy"]))
		return err
	default:
		return fmt.Errorf("unknown command: ai policy %s", action)
	}
}

func (s *state) evalControlPlaneAIOpsHistory(ctx context.Context, _ string, _ []string, jsonOut bool) error {
	if strings.TrimSpace(s.session) == "" {
		return errors.New("ai history requires session id (TERMINALS_REPL_SESSION_ID)")
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/repl/ai/history", url.Values{"session_id": {s.session}})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	if _, err := fmt.Fprintf(s.out, "session: %s\n", toString(body["session_id"])); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(s.out, "thread: %s\n", toString(body["thread"])); err != nil {
		return err
	}
	history, _ := body["history"].([]any)
	if len(history) == 0 {
		_, err := fmt.Fprintln(s.out, "history: (empty)")
		return err
	}
	if _, err := fmt.Fprintln(s.out, "history:"); err != nil {
		return err
	}
	for _, line := range history {
		if _, err := fmt.Fprintf(s.out, "- %s\n", toString(line)); err != nil {
			return err
		}
	}
	return nil
}

func (s *state) evalControlPlaneAIOpsReset(ctx context.Context, _ string, _ []string, jsonOut bool) error {
	if strings.TrimSpace(s.session) == "" {
		return errors.New("ai reset requires session id (TERMINALS_REPL_SESSION_ID)")
	}
	body, err := s.postFormJSON(ctx, "/admin/api/repl/ai/reset", url.Values{
		"session_id": {s.session},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintln(s.out, "OK  cleared AI thread and exchange history")
	return err
}

func (s *state) evalControlPlaneAppOpsLs(ctx context.Context, _ string, _ []string, jsonOut bool) error {
	body, err := s.fetchJSON(ctx, "/admin/api/apps")
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	apps, _ := body["apps"].([]any)
	rows := make([][]string, 0, len(apps))
	for _, appAny := range apps {
		app, _ := appAny.(map[string]any)
		if app == nil {
			continue
		}
		rows = append(rows, []string{toString(app["name"]), toString(app["version"])})
	}
	return printTable(s.out, []string{"APP", "VERSION"}, rows)
}

func (s *state) evalControlPlaneAppOpsReload(ctx context.Context, sub string, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 1 {
		if sub == "rollback" {
			return errors.New("usage: app rollback <app> [--keep-data|--archive-data|--purge]")
		}
		return fmt.Errorf("usage: app %s <app>", sub)
	}
	appName := strings.TrimSpace(plain[0])
	if appName == "" {
		if sub == "rollback" {
			return errors.New("usage: app rollback <app> [--keep-data|--archive-data|--purge]")
		}
		return fmt.Errorf("usage: app %s <app>", sub)
	}
	route := "/admin/api/apps/reload"
	form := url.Values{"app": {appName}}
	if sub == "rollback" {
		route = "/admin/api/apps/rollback"
		keepData := hasFlag(args[2:], "--keep-data")
		archiveData := hasFlag(args[2:], "--archive-data")
		purge := hasFlag(args[2:], "--purge")
		selected := 0
		if keepData {
			selected++
			form.Set("mode", "keep_data")
		}
		if archiveData {
			selected++
			form.Set("mode", "archive_data")
		}
		if purge {
			selected++
			form.Set("mode", "purge")
		}
		if selected > 1 {
			return errors.New("usage: app rollback <app> [--keep-data|--archive-data|--purge]")
		}
	}
	body, err := s.postFormJSON(ctx, route, form)
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  app=%s action=%s version=%s\n", appName, sub, toString(body["version"]))
	return err
}

func (s *state) evalControlPlaneAppOpsLogs(ctx context.Context, _ string, args []string, _ bool) error {
	if len(args) < 2 {
		return errors.New("usage: app logs <app> [query]")
	}
	appName := strings.TrimSpace(args[1])
	query := strings.TrimSpace(strings.Join(args[2:], " "))
	return s.queryLogs(ctx, appName, query)
}

func (s *state) evalControlPlaneDocsOpsLs(_ string, _ []string) error {
	topics, err := listDocTopics(s.docsRoot)
	if err != nil {
		return err
	}
	for _, topic := range topics {
		if _, err := fmt.Fprintln(s.out, topic); err != nil {
			return err
		}
	}
	return nil
}

func (s *state) evalControlPlaneDocsOpsSearch(_ string, args []string) error {
	if len(args) < 2 {
		return errors.New("usage: docs search <query>")
	}
	query := strings.ToLower(strings.TrimSpace(strings.Join(args[1:], " ")))
	matches, err := searchDocTopics(s.docsRoot, query)
	if err != nil {
		return err
	}
	if len(matches) == 0 {
		_, err := fmt.Fprintln(s.out, "(no matches)")
		return err
	}
	if s.docsMode == DocsRenderModeTerminal {
		if _, err := fmt.Fprintf(s.out, "search results for %q\n", strings.Join(args[1:], " ")); err != nil {
			return err
		}
	}
	for _, topic := range matches {
		line := "- " + topic
		if s.docsMode == DocsRenderModeMarkdown {
			line = "- `" + topic + "`"
		}
		if _, err := fmt.Fprintln(s.out, line); err != nil {
			return err
		}
	}
	return nil
}

func (s *state) evalControlPlaneDocsOpsOpen(_ string, args []string) error {
	if len(args) < 2 {
		return errors.New("usage: docs open <topic>")
	}
	topic := strings.TrimSpace(strings.Join(args[1:], " "))
	path := resolveDocTopicPath(s.docsRoot, topic)
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(s.out, string(content))
	return err
}

func (s *state) evalControlPlaneDocsOpsExamples(_ string, args []string) error {
	filter := ""
	if len(args) > 1 {
		filter = strings.ToLower(strings.Join(args[1:], " "))
	}
	topics, err := listDocTopics(filepath.Join(s.docsRoot, "examples"))
	if err != nil {
		return err
	}
	for _, topic := range topics {
		if filter == "" || strings.Contains(strings.ToLower(topic), filter) {
			if _, err := fmt.Fprintln(s.out, topic); err != nil {
				return err
			}
		}
	}
	return nil
}
