package repl

import (
	"context"
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
		var err error
		route, form, err = appRollbackRequest(appName, args[2:])
		if err != nil {
			return err
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

func appRollbackRequest(appName string, flagArgs []string) (string, url.Values, error) {
	form := url.Values{"app": {appName}}
	keepData := hasFlag(flagArgs, "--keep-data")
	archiveData := hasFlag(flagArgs, "--archive-data")
	purge := hasFlag(flagArgs, "--purge")
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
		return "", nil, errors.New("usage: app rollback <app> [--keep-data|--archive-data|--purge]")
	}
	return "/admin/api/apps/rollback", form, nil
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
