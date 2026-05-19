package repl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func (s *state) evalControlPlaneMid(ctx context.Context, group string, args []string, jsonOut bool) error {
	sub := strings.ToLower(args[0])
	switch group {
	case "board":
		return s.evalControlPlaneMidBoard(ctx, sub, args, jsonOut)
	case "artifact":
		return s.evalControlPlaneMidArtifact(ctx, sub, args, jsonOut)
	case "canvas":
		return s.evalControlPlaneMidCanvas(ctx, sub, args, jsonOut)
	case "search":
		return s.evalControlPlaneMidSearch(ctx, sub, args, jsonOut)
	case "memory":
		return s.evalControlPlaneMidMemory(ctx, sub, args, jsonOut)
	case "bug":
		return s.evalControlPlaneMidBug(ctx, sub, args, jsonOut)
	case "placement":
		return s.evalControlPlaneMidPlacement(ctx, sub, args, jsonOut)
	case "cohort":
		return s.evalControlPlaneMidCohort(ctx, sub, args, jsonOut)
	case "ui":
		return s.evalControlPlaneMidUI(ctx, sub, args, jsonOut)
	case "recent":
		return s.evalControlPlaneMidRecent(ctx, sub, args, jsonOut)
	case "store":
		return s.evalControlPlaneMidStore(ctx, sub, args, jsonOut)
	default:
		return s.evalControlPlaneBus(ctx, group, args, jsonOut)
	}
}

func (s *state) evalControlPlaneMidBoard(ctx context.Context, sub string, args []string, jsonOut bool) error {
	switch sub {
	case "ls":
		return s.evalControlPlaneMidBoardLs(ctx, sub, args, jsonOut)
	case "pin":
		return s.evalControlPlaneMidBoardPin(ctx, sub, args, jsonOut)
	case "post":
		return s.evalControlPlaneMidBoardPost(ctx, sub, args, jsonOut)
	default:
		return fmt.Errorf("unknown command: board %s", sub)
	}
}

func (s *state) evalControlPlaneMidArtifact(ctx context.Context, sub string, args []string, jsonOut bool) error {
	switch sub {
	case "ls":
		return s.evalControlPlaneMidArtifactLs(ctx, sub, args, jsonOut)
	case "show":
		return s.evalControlPlaneMidArtifactShow(ctx, sub, args, jsonOut)
	case "history":
		return s.evalControlPlaneMidArtifactHistory(ctx, sub, args, jsonOut)
	case "create":
		return s.evalControlPlaneMidArtifactCreate(ctx, sub, args, jsonOut)
	case "patch":
		return s.evalControlPlaneMidArtifactPatch(ctx, sub, args, jsonOut)
	case "replace":
		return s.evalControlPlaneMidArtifactReplace(ctx, sub, args, jsonOut)
	case "template":
		return s.evalControlPlaneMidArtifactTemplate(ctx, sub, args, jsonOut)
	default:
		return fmt.Errorf("unknown command: artifact %s", sub)
	}
}

func (s *state) evalControlPlaneMidCanvas(ctx context.Context, sub string, args []string, jsonOut bool) error {
	switch sub {
	case "ls":
		query := url.Values{}
		plain := nonFlagArgs(args[1:])
		if len(plain) > 0 {
			query.Set("canvas", plain[0])
		}
		body, err := s.fetchJSONQuery(ctx, "/admin/api/canvas", query)
		if err != nil {
			return err
		}
		return writeJSON(s.out, body)
	case "annotate":
		plain := nonFlagArgs(args[1:])
		if len(plain) < 2 {
			return errors.New("usage: canvas annotate <canvas> <text>")
		}
		body, err := s.postFormJSON(ctx, "/admin/api/canvas/annotate", url.Values{
			"canvas": {plain[0]},
			"text":   {strings.Join(plain[1:], " ")},
		})
		if err != nil {
			return err
		}
		if jsonOut {
			return writeJSON(s.out, body)
		}
		annotationID := ""
		if itemMap, ok := body["annotation"].(map[string]any); ok {
			annotationID = toString(itemMap["id"])
		}
		_, err = fmt.Fprintf(s.out, "OK  annotation=%s\n", annotationID)
		return err
	default:
		return fmt.Errorf("unknown command: canvas %s", sub)
	}
}

func (s *state) evalControlPlaneMidSearch(ctx context.Context, sub string, args []string, jsonOut bool) error {
	switch sub {
	case "query":
		return s.evalControlPlaneMidSearchQuery(ctx, sub, args, jsonOut)
	case "timeline":
		return s.evalControlPlaneMidSearchTimeline(ctx, sub, args, jsonOut)
	case "related":
		return s.evalControlPlaneMidSearchRelated(ctx, sub, args, jsonOut)
	case "recent":
		return s.evalControlPlaneMidSearchRecent(ctx, sub, args, jsonOut)
	default:
		return fmt.Errorf("unknown command: search %s", sub)
	}
}

func (s *state) evalControlPlaneMidMemory(ctx context.Context, sub string, args []string, jsonOut bool) error {
	switch sub {
	case "remember":
		return s.evalControlPlaneMidMemoryRemember(ctx, sub, args, jsonOut)
	case "recall":
		return s.evalControlPlaneMidMemoryRecall(ctx, sub, args, jsonOut)
	case "stream":
		return s.evalControlPlaneMidMemoryStream(ctx, sub, args, jsonOut)
	default:
		return fmt.Errorf("unknown command: memory %s", sub)
	}
}

func (s *state) evalControlPlaneMidBug(ctx context.Context, sub string, args []string, jsonOut bool) error {
	switch sub {
	case "ls":
		return s.evalControlPlaneMidBugLs(ctx, sub, args, jsonOut)
	case "show":
		return s.evalControlPlaneMidBugShow(ctx, sub, args, jsonOut)
	case "file":
		return s.evalControlPlaneMidBugFile(ctx, sub, args, jsonOut)
	case "confirm":
		return s.evalControlPlaneMidBugConfirm(ctx, sub, args, jsonOut)
	case "tail":
		return s.evalControlPlaneMidBugTail(ctx, sub, args, jsonOut)
	default:
		return fmt.Errorf("unknown command: bug %s", sub)
	}
}

func (s *state) evalControlPlaneMidPlacement(ctx context.Context, sub string, _ []string, _ bool) error {
	if sub != "ls" {
		return fmt.Errorf("unknown command: placement %s", sub)
	}
	body, err := s.fetchJSON(ctx, "/admin/api/placement")
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalControlPlaneMidCohort(ctx context.Context, sub string, args []string, jsonOut bool) error {
	switch sub {
	case "ls":
		return s.evalControlPlaneMidCohortLs(ctx, sub, args, jsonOut)
	case "show":
		return s.evalControlPlaneMidCohortShow(ctx, sub, args, jsonOut)
	case "put":
		return s.evalControlPlaneMidCohortPut(ctx, sub, args, jsonOut)
	case "del":
		return s.evalControlPlaneMidCohortDel(ctx, sub, args, jsonOut)
	default:
		return fmt.Errorf("unknown command: cohort %s", sub)
	}
}

func (s *state) evalControlPlaneMidUI(ctx context.Context, sub string, args []string, jsonOut bool) error {
	switch sub {
	case "push":
		return s.evalControlPlaneMidUIPush(ctx, sub, args, jsonOut)
	case "patch":
		return s.evalControlPlaneMidUIPatch(ctx, sub, args, jsonOut)
	case "transition":
		return s.evalControlPlaneMidUITransition(ctx, sub, args, jsonOut)
	case "broadcast":
		return s.evalControlPlaneMidUIBroadcast(ctx, sub, args, jsonOut)
	case "subscribe":
		return s.evalControlPlaneMidUISubscribe(ctx, sub, args, jsonOut)
	case "snapshot":
		return s.evalControlPlaneMidUISnapshot(ctx, sub, args, jsonOut)
	case "views":
		return s.evalControlPlaneMidUIViews(ctx, sub, args, jsonOut)
	default:
		return fmt.Errorf("unknown command: ui %s", sub)
	}
}

func (s *state) evalControlPlaneMidRecent(ctx context.Context, sub string, _ []string, _ bool) error {
	if sub != "ls" {
		return fmt.Errorf("unknown command: recent %s", sub)
	}
	body, err := s.fetchJSON(ctx, "/admin/api/recent")
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalControlPlaneMidStore(ctx context.Context, sub string, args []string, jsonOut bool) error {
	switch sub {
	case "ns":
		return s.evalControlPlaneMidStoreNs(ctx, sub, args, jsonOut)
	case "put":
		return s.evalControlPlaneMidStorePut(ctx, sub, args, jsonOut)
	case "get":
		return s.evalControlPlaneMidStoreGet(ctx, sub, args, jsonOut)
	case "ls":
		return s.evalControlPlaneMidStoreLs(ctx, sub, args, jsonOut)
	case "del":
		return s.evalControlPlaneMidStoreDel(ctx, sub, args, jsonOut)
	case "watch":
		return s.evalControlPlaneMidStoreWatch(ctx, sub, args, jsonOut)
	case "bind":
		return s.evalControlPlaneMidStoreBind(ctx, sub, args, jsonOut)
	default:
		return fmt.Errorf("unknown command: store %s", sub)
	}
}

func (s *state) evalControlPlaneMidUIPush(ctx context.Context, _ string, args []string, jsonOut bool) error {
	plain := nonFlagArgsSkippingFlagValues(args[1:], "--root")
	if len(plain) < 2 {
		return errors.New("usage: ui push <device> <descriptor-expr> [--root <id>]")
	}
	form := url.Values{
		"device_id":  {plain[0]},
		"descriptor": {strings.Join(plain[1:], " ")},
	}
	if rootID := strings.TrimSpace(flagValue(args[1:], "--root")); rootID != "" {
		form.Set("root_id", rootID)
	}
	body, err := s.postFormJSON(ctx, "/admin/api/ui/push", form)
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  action=push device=%s\n", plain[0])
	return err
}

func (s *state) evalControlPlaneMidUIPatch(ctx context.Context, _ string, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 3 {
		return errors.New("usage: ui patch <device> <component-id> <descriptor-expr>")
	}
	body, err := s.postFormJSON(ctx, "/admin/api/ui/patch", url.Values{
		"device_id":    {plain[0]},
		"component_id": {plain[1]},
		"descriptor":   {strings.Join(plain[2:], " ")},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  action=patch device=%s component=%s\n", plain[0], plain[1])
	return err
}

func (s *state) evalControlPlaneMidUITransition(ctx context.Context, _ string, args []string, jsonOut bool) error {
	plain := nonFlagArgsSkippingFlagValues(args[1:], "--duration-ms")
	if len(plain) < 3 {
		return errors.New("usage: ui transition <device> <component-id> <transition> [--duration-ms <n>]")
	}
	form := url.Values{
		"device_id":    {plain[0]},
		"component_id": {plain[1]},
		"transition":   {plain[2]},
	}
	if duration := strings.TrimSpace(flagValue(args[1:], "--duration-ms")); duration != "" {
		form.Set("duration_ms", duration)
	}
	body, err := s.postFormJSON(ctx, "/admin/api/ui/transition", form)
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  action=transition device=%s transition=%s\n", plain[0], plain[2])
	return err
}

func (s *state) evalControlPlaneMidUIBroadcast(ctx context.Context, _ string, args []string, jsonOut bool) error {
	plain := nonFlagArgsSkippingFlagValues(args[1:], "--patch")
	if len(plain) < 2 {
		return errors.New("usage: ui broadcast <cohort> <descriptor-expr> [--patch <component-id>]")
	}
	form := url.Values{
		"cohort":     {plain[0]},
		"descriptor": {strings.Join(plain[1:], " ")},
	}
	if patchID := strings.TrimSpace(flagValue(args[1:], "--patch")); patchID != "" {
		form.Set("patch_id", patchID)
	}
	body, err := s.postFormJSON(ctx, "/admin/api/ui/broadcast", form)
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  action=broadcast cohort=%s\n", plain[0])
	return err
}

func (s *state) evalControlPlaneMidUISubscribe(ctx context.Context, _ string, args []string, jsonOut bool) error {
	plain := nonFlagArgsSkippingFlagValues(args[1:], "--to")
	if len(plain) < 1 {
		return errors.New("usage: ui subscribe <device> --to <activation|cohort>")
	}
	target := strings.TrimSpace(flagValue(args[1:], "--to"))
	if target == "" {
		return errors.New("usage: ui subscribe <device> --to <activation|cohort>")
	}
	body, err := s.postFormJSON(ctx, "/admin/api/ui/subscribe", url.Values{
		"device_id": {plain[0]},
		"to":        {target},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  action=subscribe device=%s to=%s\n", plain[0], target)
	return err
}

func (s *state) evalControlPlaneMidUISnapshot(ctx context.Context, _ string, args []string, _ bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 1 {
		return errors.New("usage: ui snapshot <device>")
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/ui/snapshot", url.Values{"device_id": {plain[0]}})
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalControlPlaneMidUIViews(ctx context.Context, _ string, args []string, jsonOut bool) error {
	if len(args) < 2 {
		return errors.New("usage: ui views <ls|show|rm>")
	}
	viewSub := strings.ToLower(args[1])
	switch viewSub {
	case "ls":
		return s.evalUIViewsLs(ctx, jsonOut)
	case "show":
		return s.evalUIViewsShow(ctx, args[2:])
	case "rm":
		return s.evalUIViewsRm(ctx, args[2:], jsonOut)
	default:
		return fmt.Errorf("unknown command: ui views %s", viewSub)
	}
}

func (s *state) evalControlPlaneMidArtifactLs(ctx context.Context, _ string, _ []string, _ bool) error {
	body, err := s.fetchJSON(ctx, "/admin/api/artifact")
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalControlPlaneMidArtifactShow(ctx context.Context, _ string, args []string, _ bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) == 0 {
		return errors.New("usage: artifact show <artifact>")
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/artifact/get", url.Values{"artifact_id": {plain[0]}})
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalControlPlaneMidArtifactHistory(ctx context.Context, _ string, args []string, _ bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) == 0 {
		return errors.New("usage: artifact history <artifact>")
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/artifact/history", url.Values{"artifact_id": {plain[0]}})
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalControlPlaneMidArtifactCreate(ctx context.Context, _ string, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 2 {
		return errors.New("usage: artifact create <kind> <title>")
	}
	body, err := s.postFormJSON(ctx, "/admin/api/artifact/create", url.Values{
		"kind":  {plain[0]},
		"title": {strings.Join(plain[1:], " ")},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	artifactID := ""
	if itemMap, ok := body["artifact"].(map[string]any); ok {
		artifactID = toString(itemMap["id"])
	}
	_, err = fmt.Fprintf(s.out, "OK  artifact=%s\n", artifactID)
	return err
}

func (s *state) evalControlPlaneMidArtifactPatch(ctx context.Context, _ string, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 2 {
		return errors.New("usage: artifact patch <artifact> <title>")
	}
	body, err := s.postFormJSON(ctx, "/admin/api/artifact/patch", url.Values{
		"artifact_id": {plain[0]},
		"title":       {strings.Join(plain[1:], " ")},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  artifact=%s action=patch\n", plain[0])
	return err
}

func (s *state) evalControlPlaneMidArtifactReplace(ctx context.Context, _ string, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 2 {
		return errors.New("usage: artifact replace <artifact> <title>")
	}
	body, err := s.postFormJSON(ctx, "/admin/api/artifact/replace", url.Values{
		"artifact_id": {plain[0]},
		"title":       {strings.Join(plain[1:], " ")},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  artifact=%s action=replace\n", plain[0])
	return err
}

func (s *state) evalControlPlaneMidArtifactTemplate(ctx context.Context, _ string, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 1 {
		return errors.New("usage: artifact template <save|apply> <args>")
	}
	action := strings.ToLower(strings.TrimSpace(plain[0]))
	switch action {
	case "save":
		if len(plain) < 3 {
			return errors.New("usage: artifact template save <name> <source-artifact>")
		}
		body, err := s.postFormJSON(ctx, "/admin/api/artifact/template/save", url.Values{
			"name":               {plain[1]},
			"source_artifact_id": {plain[2]},
		})
		if err != nil {
			return err
		}
		if jsonOut {
			return writeJSON(s.out, body)
		}
		_, err = fmt.Fprintf(s.out, "OK  template=%s source=%s action=save\n", plain[1], plain[2])
		return err
	case "apply":
		if len(plain) < 3 {
			return errors.New("usage: artifact template apply <name> <target-artifact>")
		}
		body, err := s.postFormJSON(ctx, "/admin/api/artifact/template/apply", url.Values{
			"name":               {plain[1]},
			"target_artifact_id": {plain[2]},
		})
		if err != nil {
			return err
		}
		if jsonOut {
			return writeJSON(s.out, body)
		}
		_, err = fmt.Fprintf(s.out, "OK  template=%s target=%s action=apply\n", plain[1], plain[2])
		return err
	default:
		return fmt.Errorf("unknown command: artifact template %s", action)
	}
}

func (s *state) evalControlPlaneMidStoreNs(ctx context.Context, _ string, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 1 || !strings.EqualFold(plain[0], "ls") {
		return errors.New("usage: store ns ls")
	}
	body, err := s.fetchJSON(ctx, "/admin/api/store/ns")
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	namespaces, _ := body["namespaces"].([]any)
	rows := make([][]string, 0, len(namespaces))
	for _, item := range namespaces {
		row, _ := item.(map[string]any)
		if row == nil {
			continue
		}
		rows = append(rows, []string{toString(row["name"]), toString(row["record_count"])})
	}
	return printTable(s.out, []string{"NAMESPACE", "RECORDS"}, rows)
}

func (s *state) evalControlPlaneMidStorePut(ctx context.Context, _ string, args []string, jsonOut bool) error {
	plain := nonFlagArgsSkippingFlagValues(args[1:], "--ttl")
	if len(plain) < 3 {
		return errors.New("usage: store put <namespace> <key> <value> [--ttl <duration>]")
	}
	form := url.Values{
		"namespace": {plain[0]},
		"key":       {plain[1]},
		"value":     {strings.Join(plain[2:], " ")},
	}
	if ttl := strings.TrimSpace(flagValue(args[1:], "--ttl")); ttl != "" {
		form.Set("ttl", ttl)
	}
	body, err := s.postFormJSON(ctx, "/admin/api/store/put", form)
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintln(s.out, "OK  stored")
	return err
}

func (s *state) evalControlPlaneMidStoreGet(ctx context.Context, _ string, args []string, _ bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 2 {
		return errors.New("usage: store get <namespace> <key>")
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/store/get", url.Values{
		"namespace": {plain[0]},
		"key":       {plain[1]},
	})
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalControlPlaneMidStoreLs(ctx context.Context, _ string, args []string, _ bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 1 {
		return errors.New("usage: store ls <namespace>")
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/store/ls", url.Values{
		"namespace": {plain[0]},
	})
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalControlPlaneMidStoreDel(ctx context.Context, _ string, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 2 {
		return errors.New("usage: store del <namespace> <key>")
	}
	body, err := s.postFormJSON(ctx, "/admin/api/store/del", url.Values{
		"namespace": {plain[0]},
		"key":       {plain[1]},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  deleted=%s namespace=%s key=%s\n", toString(body["deleted"]), plain[0], plain[1])
	return err
}

func (s *state) evalControlPlaneMidStoreWatch(ctx context.Context, _ string, args []string, _ bool) error {
	plain := nonFlagArgsSkippingFlagValues(args[1:], "--prefix")
	if len(plain) < 1 {
		return errors.New("usage: store watch <namespace> [--prefix <p>]")
	}
	query := url.Values{"namespace": {plain[0]}}
	if prefix := strings.TrimSpace(flagValue(args[1:], "--prefix")); prefix != "" {
		query.Set("prefix", prefix)
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/store/watch", query)
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalControlPlaneMidStoreBind(ctx context.Context, _ string, args []string, jsonOut bool) error {
	plain := nonFlagArgsSkippingFlagValues(args[1:], "--to")
	if len(plain) < 2 {
		return errors.New("usage: store bind <namespace> <key> --to <device>:<scenario>")
	}
	binding := strings.TrimSpace(flagValue(args[1:], "--to"))
	if binding == "" {
		return errors.New("usage: store bind <namespace> <key> --to <device>:<scenario>")
	}
	body, err := s.postFormJSON(ctx, "/admin/api/store/bind", url.Values{
		"namespace": {plain[0]},
		"key":       {plain[1]},
		"to":        {binding},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  namespace=%s key=%s bound_to=%s\n", plain[0], plain[1], binding)
	return err
}

func (s *state) evalControlPlaneMidCohortLs(ctx context.Context, _ string, _ []string, jsonOut bool) error {
	body, err := s.fetchJSON(ctx, "/admin/api/cohort")
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	items, _ := body["cohorts"].([]any)
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		row, _ := item.(map[string]any)
		if row == nil {
			continue
		}
		selectors := ""
		if values, ok := row["selectors"].([]any); ok {
			parts := make([]string, 0, len(values))
			for _, value := range values {
				parts = append(parts, toString(value))
			}
			selectors = strings.Join(parts, ",")
		}
		rows = append(rows, []string{toString(row["name"]), selectors})
	}
	return printTable(s.out, []string{"COHORT", "SELECTORS"}, rows)
}

func (s *state) evalControlPlaneMidCohortShow(ctx context.Context, _ string, args []string, _ bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 1 {
		return errors.New("usage: cohort show <name>")
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/cohort", url.Values{"name": {plain[0]}})
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalControlPlaneMidCohortPut(ctx context.Context, _ string, args []string, jsonOut bool) error {
	plain := nonFlagArgsSkippingFlagValues(args[1:], "--selectors")
	if len(plain) < 1 {
		return errors.New("usage: cohort put <name> --selectors <selector[,selector...]>")
	}
	selectors := strings.TrimSpace(flagValue(args[1:], "--selectors"))
	if selectors == "" {
		return errors.New("usage: cohort put <name> --selectors <selector[,selector...]>")
	}
	body, err := s.postFormJSON(ctx, "/admin/api/cohort/upsert", url.Values{
		"name":      {plain[0]},
		"selectors": {selectors},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  cohort=%s selectors=%s\n", plain[0], selectors)
	return err
}

func (s *state) evalControlPlaneMidCohortDel(ctx context.Context, _ string, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 1 {
		return errors.New("usage: cohort del <name>")
	}
	body, err := s.postFormJSON(ctx, "/admin/api/cohort/del", url.Values{"name": {plain[0]}})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  deleted=%s cohort=%s\n", toString(body["deleted"]), plain[0])
	return err
}

func (s *state) evalControlPlaneMidBugLs(ctx context.Context, _ string, _ []string, jsonOut bool) error {
	body, err := s.fetchJSON(ctx, "/admin/api/bugs")
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	items, _ := body["bugs"].([]any)
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		row, _ := item.(map[string]any)
		if row == nil {
			continue
		}
		rows = append(rows, []string{
			toString(row["report_id"]),
			toString(row["subject_device_id"]),
			toString(row["reporter_device_id"]),
			toString(row["source"]),
			toString(row["confirmed"]),
		})
	}
	return printTable(s.out, []string{"REPORT", "SUBJECT", "REPORTER", "SOURCE", "CONFIRMED"}, rows)
}

func (s *state) evalControlPlaneMidBugShow(ctx context.Context, _ string, args []string, _ bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 1 {
		return errors.New("usage: bug show <report-id>")
	}
	body, err := s.fetchJSON(ctx, "/admin/api/bugs/"+url.PathEscape(plain[0]))
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalControlPlaneMidBugFile(ctx context.Context, _ string, args []string, jsonOut bool) error {
	plain := nonFlagArgsSkippingFlagValues(args[1:], "--source", "--tags")
	if len(plain) < 3 {
		return errors.New("usage: bug file <reporter-device-id> <subject-device-id> <description> [--source <source>] [--tags <tag[,tag...]>]")
	}
	reporterDeviceID := plain[0]
	subjectDeviceID := plain[1]
	description := strings.Join(plain[2:], " ")
	source := normalizeBugSource(flagValue(args[1:], "--source"))
	tags := parseCSVValues(flagValue(args[1:], "--tags"))

	payload, err := json.Marshal(map[string]any{
		"reporterDeviceId": reporterDeviceID,
		"subjectDeviceId":  subjectDeviceID,
		"description":      description,
		"source":           source,
		"tags":             tags,
	})
	if err != nil {
		return err
	}
	body, err := s.doJSON(ctx, http.MethodPost, "/bug/intake", "application/json", strings.NewReader(string(payload)))
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	reportID := ""
	if ack, ok := body["ack"].(map[string]any); ok {
		reportID = toString(ack["report_id"])
	}
	_, err = fmt.Fprintf(s.out, "OK  report=%s subject=%s reporter=%s action=file\n", reportID, subjectDeviceID, reporterDeviceID)
	return err
}

func (s *state) evalControlPlaneMidBugConfirm(ctx context.Context, _ string, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 1 {
		return errors.New("usage: bug confirm <report-id>")
	}
	body, err := s.doJSON(ctx, http.MethodPost, "/admin/api/bugs/"+url.PathEscape(plain[0])+"/confirm", "", nil)
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  report=%s action=confirm\n", plain[0])
	return err
}

func (s *state) evalControlPlaneMidBugTail(ctx context.Context, _ string, args []string, _ bool) error {
	query := strings.TrimSpace(strings.Join(args[1:], " "))
	if query == "" {
		query = "bug.report"
	} else {
		query = "bug.report " + query
	}
	return s.queryLogs(ctx, "", query)
}

func (s *state) evalControlPlaneMidBoardLs(ctx context.Context, _ string, args []string, _ bool) error {
	query := url.Values{}
	plain := nonFlagArgs(args[1:])
	if len(plain) > 0 {
		query.Set("board", plain[0])
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/board", query)
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalControlPlaneMidBoardPin(ctx context.Context, _ string, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 2 {
		return errors.New("usage: board pin <board> <text>")
	}
	body, err := s.postFormJSON(ctx, "/admin/api/board/pin", url.Values{
		"board": {plain[0]},
		"text":  {strings.Join(plain[1:], " ")},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	itemID := ""
	if itemMap, ok := body["item"].(map[string]any); ok {
		itemID = toString(itemMap["id"])
	}
	_, err = fmt.Fprintf(s.out, "OK  board_item=%s\n", itemID)
	return err
}

func (s *state) evalControlPlaneMidBoardPost(ctx context.Context, _ string, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 2 {
		return errors.New("usage: board post <board> <text>")
	}
	body, err := s.postFormJSON(ctx, "/admin/api/board/post", url.Values{
		"board": {plain[0]},
		"text":  {strings.Join(plain[1:], " ")},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	itemID := ""
	if itemMap, ok := body["item"].(map[string]any); ok {
		itemID = toString(itemMap["id"])
	}
	_, err = fmt.Fprintf(s.out, "OK  board_item=%s action=post\n", itemID)
	return err
}

func (s *state) evalControlPlaneMidSearchQuery(ctx context.Context, _ string, args []string, _ bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) == 0 {
		return errors.New("usage: search query <text>")
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/search", url.Values{"q": {strings.Join(plain, " ")}})
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalControlPlaneMidSearchTimeline(ctx context.Context, _ string, args []string, _ bool) error {
	query := url.Values{}
	plain := nonFlagArgs(args[1:])
	if len(plain) > 0 {
		query.Set("scope", plain[0])
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/search/timeline", query)
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalControlPlaneMidSearchRelated(ctx context.Context, _ string, args []string, _ bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) == 0 {
		return errors.New("usage: search related <subject-ref>")
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/search/related", url.Values{"subject": {strings.Join(plain, " ")}})
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalControlPlaneMidSearchRecent(ctx context.Context, _ string, args []string, _ bool) error {
	query := url.Values{}
	plain := nonFlagArgs(args[1:])
	if len(plain) > 0 {
		query.Set("scope", plain[0])
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/search/recent", query)
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalControlPlaneMidMemoryRemember(ctx context.Context, _ string, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 2 {
		return errors.New("usage: memory remember <scope> <text>")
	}
	body, err := s.postFormJSON(ctx, "/admin/api/memory/remember", url.Values{
		"scope": {plain[0]},
		"text":  {strings.Join(plain[1:], " ")},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	memoryID := ""
	if itemMap, ok := body["memory"].(map[string]any); ok {
		memoryID = toString(itemMap["id"])
	}
	_, err = fmt.Fprintf(s.out, "OK  memory=%s\n", memoryID)
	return err
}

func (s *state) evalControlPlaneMidMemoryRecall(ctx context.Context, _ string, args []string, _ bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) == 0 {
		return errors.New("usage: memory recall <text>")
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/memory", url.Values{"q": {strings.Join(plain, " ")}})
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalControlPlaneMidMemoryStream(ctx context.Context, _ string, args []string, _ bool) error {
	query := url.Values{}
	plain := nonFlagArgs(args[1:])
	if len(plain) > 0 {
		query.Set("scope", plain[0])
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/memory/stream", query)
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}
