package repl

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func (s *state) evalControlPlaneAppsOpsMigrate(ctx context.Context, _ string, args []string, jsonOut bool) error {
	if len(args) < 2 {
		return errors.New("usage: apps migrate <status|logs|retry|abort|drain-ready|reconcile>")
	}
	migrateSub := strings.TrimSpace(args[1])
	rest := args[2:]
	switch migrateSub {
	case "status":
		return s.evalControlPlaneAppsOpsMigrateStatus(ctx, rest, jsonOut)
	case "logs":
		return s.evalControlPlaneAppsOpsMigrateLogs(ctx, rest, jsonOut)
	case "retry":
		return s.evalControlPlaneAppsOpsMigrateRetry(ctx, rest, jsonOut)
	case "abort":
		return s.evalControlPlaneAppsOpsMigrateAbort(ctx, rest, jsonOut)
	case "drain-ready":
		return s.evalControlPlaneAppsOpsMigrateDrainReady(ctx, rest, jsonOut)
	case "reconcile":
		return s.evalControlPlaneAppsOpsMigrateReconcile(ctx, rest, jsonOut)
	default:
		return fmt.Errorf("unknown command: apps migrate %s", migrateSub)
	}
}

func (s *state) evalControlPlaneAppsOpsMigrateStatus(ctx context.Context, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args)
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
}

func (s *state) evalControlPlaneAppsOpsMigrateLogs(ctx context.Context, args []string, jsonOut bool) error {
	plain := nonFlagArgsSkippingFlagValues(args, "--step")
	if len(plain) < 1 {
		return errors.New("usage: apps migrate logs <app> [--step <n>]")
	}
	appName := strings.TrimSpace(plain[0])
	if appName == "" {
		return errors.New("usage: apps migrate logs <app> [--step <n>]")
	}
	values := url.Values{"app": {appName}}
	if stepRaw := strings.TrimSpace(flagValue(args, "--step")); stepRaw != "" {
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
	_, err = fmt.Fprintf(s.out, "OK  app=%s lines=%d journal_exists=%v\n", appName, len(linesAny), body["journal_exists"])
	return err
}

func (s *state) evalControlPlaneAppsOpsMigrateRetry(ctx context.Context, args []string, jsonOut bool) error {
	appName, err := requiredMigrateAppName(args, "usage: apps migrate retry <app>")
	if err != nil {
		return err
	}
	body, err := s.postFormJSON(ctx, "/admin/api/apps/migrate/retry", url.Values{"app": {appName}})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  app=%s action=retry status=%s\n", appName, toString(body["status"]))
	return err
}

func (s *state) evalControlPlaneAppsOpsMigrateAbort(ctx context.Context, args []string, jsonOut bool) error {
	plain := nonFlagArgsSkippingFlagValues(args, "--to")
	appName, err := requiredMigrateAppName(plain, "usage: apps migrate abort <app> [--to <checkpoint|baseline>]")
	if err != nil {
		return err
	}
	target := strings.TrimSpace(flagValue(args, "--to"))
	values := url.Values{"app": {appName}}
	if target != "" {
		values.Set("to", target)
	}
	body, err := s.postFormJSON(ctx, "/admin/api/apps/migrate/abort", values)
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	resolvedTarget := toString(body["to"])
	if resolvedTarget == "" {
		resolvedTarget = target
	}
	if resolvedTarget == "" {
		resolvedTarget = "checkpoint"
	}
	_, err = fmt.Fprintf(s.out, "OK  app=%s action=abort to=%s status=%s\n", appName, resolvedTarget, toString(body["status"]))
	return err
}

func requiredMigrateAppName(plain []string, usage string) (string, error) {
	if len(plain) < 1 {
		return "", errors.New(usage)
	}
	appName := strings.TrimSpace(plain[0])
	if appName == "" {
		return "", errors.New(usage)
	}
	return appName, nil
}

func (s *state) evalControlPlaneAppsOpsMigrateDrainReady(ctx context.Context, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args)
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
}

func (s *state) evalControlPlaneAppsOpsMigrateReconcile(ctx context.Context, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args)
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
}
