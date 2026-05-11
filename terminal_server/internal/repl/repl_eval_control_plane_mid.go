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
		switch sub {
		case "ls":
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
		case "pin":
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
		case "post":
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
		default:
			return fmt.Errorf("unknown command: board %s", sub)
		}
	case "artifact":
		switch sub {
		case "ls":
			body, err := s.fetchJSON(ctx, "/admin/api/artifact")
			if err != nil {
				return err
			}
			return writeJSON(s.out, body)
		case "show":
			plain := nonFlagArgs(args[1:])
			if len(plain) == 0 {
				return errors.New("usage: artifact show <artifact>")
			}
			body, err := s.fetchJSONQuery(ctx, "/admin/api/artifact/get", url.Values{"artifact_id": {plain[0]}})
			if err != nil {
				return err
			}
			return writeJSON(s.out, body)
		case "history":
			plain := nonFlagArgs(args[1:])
			if len(plain) == 0 {
				return errors.New("usage: artifact history <artifact>")
			}
			body, err := s.fetchJSONQuery(ctx, "/admin/api/artifact/history", url.Values{"artifact_id": {plain[0]}})
			if err != nil {
				return err
			}
			return writeJSON(s.out, body)
		case "create":
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
		case "patch":
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
		case "replace":
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
		case "template":
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
		default:
			return fmt.Errorf("unknown command: artifact %s", sub)
		}
	case "canvas":
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
	case "search":
		switch sub {
		case "query":
			plain := nonFlagArgs(args[1:])
			if len(plain) == 0 {
				return errors.New("usage: search query <text>")
			}
			body, err := s.fetchJSONQuery(ctx, "/admin/api/search", url.Values{"q": {strings.Join(plain, " ")}})
			if err != nil {
				return err
			}
			return writeJSON(s.out, body)
		case "timeline":
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
		case "related":
			plain := nonFlagArgs(args[1:])
			if len(plain) == 0 {
				return errors.New("usage: search related <subject-ref>")
			}
			body, err := s.fetchJSONQuery(ctx, "/admin/api/search/related", url.Values{"subject": {strings.Join(plain, " ")}})
			if err != nil {
				return err
			}
			return writeJSON(s.out, body)
		case "recent":
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
		default:
			return fmt.Errorf("unknown command: search %s", sub)
		}
	case "memory":
		switch sub {
		case "remember":
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
		case "recall":
			plain := nonFlagArgs(args[1:])
			if len(plain) == 0 {
				return errors.New("usage: memory recall <text>")
			}
			body, err := s.fetchJSONQuery(ctx, "/admin/api/memory", url.Values{"q": {strings.Join(plain, " ")}})
			if err != nil {
				return err
			}
			return writeJSON(s.out, body)
		case "stream":
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
		default:
			return fmt.Errorf("unknown command: memory %s", sub)
		}
	case "bug":
		switch sub {
		case "ls":
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
		case "show":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 1 {
				return errors.New("usage: bug show <report-id>")
			}
			body, err := s.fetchJSON(ctx, "/admin/api/bugs/"+url.PathEscape(plain[0]))
			if err != nil {
				return err
			}
			return writeJSON(s.out, body)
		case "file":
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
		case "confirm":
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
		case "tail":
			query := strings.TrimSpace(strings.Join(args[1:], " "))
			if query == "" {
				query = "bug.report"
			} else {
				query = "bug.report " + query
			}
			return s.queryLogs(ctx, "", query)
		default:
			return fmt.Errorf("unknown command: bug %s", sub)
		}
	case "placement":
		if sub != "ls" {
			return fmt.Errorf("unknown command: placement %s", sub)
		}
		body, err := s.fetchJSON(ctx, "/admin/api/placement")
		if err != nil {
			return err
		}
		return writeJSON(s.out, body)
	case "cohort":
		switch sub {
		case "ls":
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
		case "show":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 1 {
				return errors.New("usage: cohort show <name>")
			}
			body, err := s.fetchJSONQuery(ctx, "/admin/api/cohort", url.Values{"name": {plain[0]}})
			if err != nil {
				return err
			}
			return writeJSON(s.out, body)
		case "put":
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
		case "del":
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
		default:
			return fmt.Errorf("unknown command: cohort %s", sub)
		}
	case "ui":
		switch sub {
		case "push":
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
		case "patch":
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
		case "transition":
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
		case "broadcast":
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
		case "subscribe":
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
		case "snapshot":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 1 {
				return errors.New("usage: ui snapshot <device>")
			}
			body, err := s.fetchJSONQuery(ctx, "/admin/api/ui/snapshot", url.Values{"device_id": {plain[0]}})
			if err != nil {
				return err
			}
			return writeJSON(s.out, body)
		case "views":
			if len(args) < 2 {
				return errors.New("usage: ui views <ls|show|rm>")
			}
			viewSub := strings.ToLower(args[1])
			switch viewSub {
			case "ls":
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
			case "show":
				plain := nonFlagArgs(args[2:])
				if len(plain) < 1 {
					return errors.New("usage: ui views show <view-id>")
				}
				body, err := s.fetchJSONQuery(ctx, "/admin/api/ui/views", url.Values{"view_id": {plain[0]}})
				if err != nil {
					return err
				}
				return writeJSON(s.out, body)
			case "rm":
				plain := nonFlagArgs(args[2:])
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
			default:
				return fmt.Errorf("unknown command: ui views %s", viewSub)
			}
		default:
			return fmt.Errorf("unknown command: ui %s", sub)
		}
	case "recent":
		if sub != "ls" {
			return fmt.Errorf("unknown command: recent %s", sub)
		}
		body, err := s.fetchJSON(ctx, "/admin/api/recent")
		if err != nil {
			return err
		}
		return writeJSON(s.out, body)
	case "store":
		switch sub {
		case "ns":
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
		case "put":
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
		case "get":
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
		case "ls":
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
		case "del":
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
		case "watch":
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
		case "bind":
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
		default:
			return fmt.Errorf("unknown command: store %s", sub)
		}
	default:
		return s.evalControlPlaneBus(ctx, group, args, jsonOut)
	}
}
