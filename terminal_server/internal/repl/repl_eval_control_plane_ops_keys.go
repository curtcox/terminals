package repl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

func (s *state) evalControlPlaneAppsOpsKeys(ctx context.Context, _ string, args []string, jsonOut bool) error {
	if len(args) == 0 {
		return errors.New("usage: apps keys <ls|show|add|confirm|revoke|archive|rotate|rotate-installer|rotations|verify|log>")
	}
	keySub := strings.TrimSpace(args[0])
	rest := args[1:]
	switch keySub {
	case "ls":
		return s.evalControlPlaneAppsOpsKeysLs(ctx, jsonOut)
	case "show":
		return s.evalControlPlaneAppsOpsKeysShow(ctx, rest)
	case "add":
		return s.evalControlPlaneAppsOpsKeysAdd(ctx, rest, jsonOut)
	case "confirm":
		return s.evalControlPlaneAppsOpsKeysConfirm(ctx, rest, jsonOut)
	case "revoke":
		return s.evalControlPlaneAppsOpsKeysRevoke(ctx, rest, jsonOut)
	case "archive":
		return s.evalControlPlaneAppsOpsKeysArchive(ctx, rest, jsonOut)
	case "verify":
		return s.evalControlPlaneAppsOpsKeysVerify(ctx, jsonOut)
	case "log":
		return s.evalControlPlaneAppsOpsKeysLog(ctx)
	case "rotations":
		return s.evalControlPlaneAppsOpsKeysRotations(ctx)
	case "rotate":
		return s.evalControlPlaneAppsOpsKeysRotate(ctx, rest, jsonOut)
	case "rotate-installer":
		return s.evalControlPlaneAppsOpsKeysRotateInstaller(ctx, jsonOut)
	default:
		return fmt.Errorf("unknown command: apps keys %s", keySub)
	}
}

func (s *state) evalControlPlaneAppsOpsKeysLs(ctx context.Context, jsonOut bool) error {
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
}

func (s *state) evalControlPlaneAppsOpsKeysShow(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return errors.New("usage: apps keys show <key_id>")
	}
	body, err := s.fetchJSON(ctx, "/admin/api/trust/keys")
	if err != nil {
		return err
	}
	want := strings.TrimSpace(args[0])
	keys, _ := body["keys"].([]any)
	for _, kAny := range keys {
		k, _ := kAny.(map[string]any)
		if k != nil && toString(k["key_id"]) == want {
			return writeJSON(s.out, k)
		}
	}
	return fmt.Errorf("key not found: %s", want)
}

func (s *state) evalControlPlaneAppsOpsKeysAdd(ctx context.Context, args []string, jsonOut bool) error {
	if len(args) < 2 {
		return errors.New("usage: apps keys add <key_id> <role[,role]>")
	}
	keyID := strings.TrimSpace(args[0])
	rolesStr := strings.TrimSpace(args[1])
	body, err := s.postJSON(ctx, "/admin/api/trust/keys", map[string]any{
		"key_id": keyID,
		"roles":  strings.Split(rolesStr, ","),
		"note":   strings.Join(args[2:], " "),
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  key_id=%s state=candidate\n", keyID)
	return err
}

func (s *state) evalControlPlaneAppsOpsKeysConfirm(ctx context.Context, args []string, jsonOut bool) error {
	if len(args) < 1 {
		return errors.New("usage: apps keys confirm <key_id>")
	}
	keyID := strings.TrimSpace(args[0])
	body, err := s.postJSON(ctx, "/admin/api/trust/keys/confirm", map[string]any{"key_id": keyID})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  key_id=%s state=active\n", keyID)
	return err
}

func (s *state) evalControlPlaneAppsOpsKeysRevoke(ctx context.Context, args []string, jsonOut bool) error {
	if len(args) < 1 {
		return errors.New("usage: apps keys revoke <key_id> [--reason <text>]")
	}
	keyID := strings.TrimSpace(args[0])
	reason := strings.Join(args[1:], " ")
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
}

func (s *state) evalControlPlaneAppsOpsKeysArchive(ctx context.Context, args []string, jsonOut bool) error {
	if len(args) < 1 {
		return errors.New("usage: apps keys archive <key_id>")
	}
	keyID := strings.TrimSpace(args[0])
	body, err := s.postJSON(ctx, "/admin/api/trust/keys/archive", map[string]any{"key_id": keyID})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  key_id=%s state=archived\n", keyID)
	return err
}

func (s *state) evalControlPlaneAppsOpsKeysVerify(ctx context.Context, jsonOut bool) error {
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
}

func (s *state) evalControlPlaneAppsOpsKeysLog(ctx context.Context) error {
	body, err := s.fetchJSON(ctx, "/admin/api/trust/log")
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalControlPlaneAppsOpsKeysRotations(ctx context.Context) error {
	body, err := s.fetchJSON(ctx, "/admin/api/trust/rotations")
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalControlPlaneAppsOpsKeysRotate(ctx context.Context, args []string, jsonOut bool) error {
	if len(args) < 1 {
		return errors.New("usage: apps keys rotate <--accept <json> | --rollback <seq> | --emit <old-key> <new-key> [names...]>")
	}
	switch strings.TrimSpace(args[0]) {
	case "--accept":
		return s.evalControlPlaneAppsOpsKeysRotateAccept(ctx, args[1:], jsonOut)
	case "--rollback":
		return s.evalControlPlaneAppsOpsKeysRotateRollback(ctx, args[1:], jsonOut)
	case "--emit":
		return s.evalControlPlaneAppsOpsKeysRotateEmit(args[1:])
	default:
		return fmt.Errorf("unknown flag for apps keys rotate: %s", args[0])
	}
}

func (s *state) evalControlPlaneAppsOpsKeysRotateAccept(ctx context.Context, args []string, jsonOut bool) error {
	if len(args) < 1 {
		return errors.New("usage: apps keys rotate --accept <rotation-json>")
	}
	rotJSON := strings.TrimSpace(args[0])
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
}

func (s *state) evalControlPlaneAppsOpsKeysRotateRollback(ctx context.Context, args []string, jsonOut bool) error {
	if len(args) < 1 {
		return errors.New("usage: apps keys rotate --rollback <accepted-seq>")
	}
	seqStr := strings.TrimSpace(args[0])
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
}

func (s *state) evalControlPlaneAppsOpsKeysRotateEmit(args []string) error {
	if len(args) < 2 {
		return errors.New("usage: apps keys rotate --emit <old-key-id> <new-key-id> [name ...]")
	}
	oldKey := strings.TrimSpace(args[0])
	newKey := strings.TrimSpace(args[1])
	names := args[2:]
	tmpl := map[string]any{
		"old_stmt": map[string]any{
			"schema": "rotation-stmt/1", "old_key": oldKey, "new_key": newKey,
			"proposed_at": "<unix-seconds>", "name_scope": names, "reason": "<optional>",
			"sig_old": "<base64: signature by old_key over canonical JSON of old_stmt fields>",
		},
		"new_stmt": map[string]any{
			"schema": "rotation-stmt/1", "old_key_stmt_digest": "<sha256 of serialised old_stmt payload>",
			"new_key": newKey, "accept_at": "<unix-seconds>",
			"sig_new": "<base64: signature by new_key over canonical JSON of new_stmt fields>",
		},
	}
	return writeJSON(s.out, tmpl)
}

func (s *state) evalControlPlaneAppsOpsKeysRotateInstaller(ctx context.Context, jsonOut bool) error {
	body, err := s.postJSON(ctx, "/admin/api/trust/keys/rotate-installer", map[string]any{})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  new_installer_key_id=%s\n", toString(body["new_installer_key_id"]))
	return err
}
