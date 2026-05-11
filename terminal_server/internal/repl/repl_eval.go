package repl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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
	case "sessions":
		switch sub {
		case "ls":
			body, err := s.fetchJSON(ctx, "/admin/api/repl/sessions")
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(s.out, body)
			}
			items, _ := body["sessions"].([]any)
			rows := make([][]string, 0, len(items))
			for _, item := range items {
				row, _ := item.(map[string]any)
				if row == nil {
					continue
				}
				attached := toAnySlice(lookupMapAny(row, "attached_devices", "AttachedDevices"))
				rows = append(rows, []string{
					toString(lookupMapAny(row, "id", "ID")),
					toString(lookupMapAny(row, "origin", "Origin")),
					toString(lookupMapAny(row, "agent_capability", "AgentCapability")),
					toString(lookupMapAny(row, "owner_activation_id", "OwnerActivationID")),
					strconv.Itoa(len(attached)),
					toString(lookupMapAny(row, "idle", "Idle")),
					formatUnixMillis(lookupMapAny(row, "created_at", "CreatedAt")),
				})
			}
			return printTable(s.out, []string{"ID", "ORIGIN", "CAPABILITY", "OWNER", "ATTACHED", "IDLE", "CREATED"}, rows)
		case "show":
			if len(args) < 2 {
				return errors.New("usage: sessions show <session>")
			}
			sessionID := strings.TrimSpace(args[1])
			if sessionID == "" {
				return errors.New("usage: sessions show <session>")
			}
			body, err := s.fetchJSON(ctx, "/admin/api/repl/sessions/"+url.PathEscape(sessionID))
			if err != nil {
				return err
			}
			session := body["session"]
			return writeJSON(s.out, session)
		case "terminate":
			if len(args) < 2 {
				return errors.New("usage: sessions terminate <session>")
			}
			sessionID := strings.TrimSpace(args[1])
			if sessionID == "" {
				return errors.New("usage: sessions terminate <session>")
			}
			body, err := s.deleteJSON(ctx, "/admin/api/repl/sessions/"+url.PathEscape(sessionID))
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(s.out, body)
			}
			_, err = fmt.Fprintf(s.out, "OK  terminated session %s\n", sessionID)
			return err
		default:
			return fmt.Errorf("unknown command: sessions %s", sub)
		}
	case "identity":
		switch sub {
		case "ls":
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
		case "show":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 1 {
				return errors.New("usage: identity show <identity>")
			}
			body, err := s.fetchJSONQuery(ctx, "/admin/api/identity/show", url.Values{"identity": {plain[0]}})
			if err != nil {
				return err
			}
			return writeJSON(s.out, body)
		case "groups":
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
		case "resolve":
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
		case "prefs":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 1 {
				return errors.New("usage: identity prefs <identity>")
			}
			body, err := s.fetchJSONQuery(ctx, "/admin/api/identity/prefs", url.Values{"identity": {plain[0]}})
			if err != nil {
				return err
			}
			return writeJSON(s.out, body)
		case "ack":
			actionTokens := nonFlagArgs(args[1:])
			if len(actionTokens) == 0 {
				return errors.New("usage: identity ack <ls|show|record>")
			}
			action := strings.ToLower(strings.TrimSpace(actionTokens[0]))
			switch action {
			case "ls":
				query := url.Values{}
				if len(actionTokens) > 1 {
					query.Set("subject_ref", actionTokens[1])
				}
				body, err := s.fetchJSONQuery(ctx, "/admin/api/identity/ack", query)
				if err != nil {
					return err
				}
				return writeJSON(s.out, body)
			case "show":
				if len(actionTokens) < 2 {
					return errors.New("usage: identity ack show <subject-ref>")
				}
				body, err := s.fetchJSONQuery(ctx, "/admin/api/identity/ack", url.Values{"subject_ref": {actionTokens[1]}})
				if err != nil {
					return err
				}
				return writeJSON(s.out, body)
			case "record":
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
			default:
				return fmt.Errorf("unknown command: identity ack %s", action)
			}
		default:
			return fmt.Errorf("unknown command: identity %s", sub)
		}
	case "session":
		switch sub {
		case "ls":
			body, err := s.fetchJSON(ctx, "/admin/api/session")
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(s.out, body)
			}
			items, _ := body["sessions"].([]any)
			rows := make([][]string, 0, len(items))
			for _, item := range items {
				row, _ := item.(map[string]any)
				if row == nil {
					continue
				}
				rows = append(rows, []string{toString(row["id"]), toString(row["kind"]), toString(row["target"])})
			}
			return printTable(s.out, []string{"ID", "KIND", "TARGET"}, rows)
		case "create":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 2 {
				return errors.New("usage: session create <kind> <target>")
			}
			body, err := s.postFormJSON(ctx, "/admin/api/session/create", url.Values{
				"kind":   {plain[0]},
				"target": {plain[1]},
			})
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(s.out, body)
			}
			sessionID := ""
			if sessionMap, ok := body["session"].(map[string]any); ok {
				sessionID = toString(sessionMap["id"])
			}
			_, err = fmt.Fprintf(s.out, "OK  session=%s\n", sessionID)
			return err
		case "show":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 1 {
				return errors.New("usage: session show <session>")
			}
			body, err := s.fetchJSONQuery(ctx, "/admin/api/session/show", url.Values{"session_id": {plain[0]}})
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(s.out, body)
			}
			sessionMap, _ := body["session"].(map[string]any)
			if sessionMap == nil {
				return writeJSON(s.out, body)
			}
			rows := [][]string{
				{"session", toString(sessionMap["id"])},
				{"kind", toString(sessionMap["kind"])},
				{"target", toString(sessionMap["target"])},
			}
			if err := printTable(s.out, []string{"FIELD", "VALUE"}, rows); err != nil {
				return err
			}
			participants, _ := sessionMap["participants"].([]any)
			memberRows := make([][]string, 0, len(participants))
			for _, item := range participants {
				member, _ := item.(map[string]any)
				if member == nil {
					continue
				}
				memberRows = append(memberRows, []string{
					toString(member["identity_id"]),
					toString(member["joined_at"]),
				})
			}
			return printTable(s.out, []string{"PARTICIPANT", "JOINED_AT"}, memberRows)
		case "members":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 1 {
				return errors.New("usage: session members <session>")
			}
			body, err := s.fetchJSONQuery(ctx, "/admin/api/session/members", url.Values{"session_id": {plain[0]}})
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(s.out, body)
			}
			participants, _ := body["participants"].([]any)
			rows := make([][]string, 0, len(participants))
			for _, item := range participants {
				member, _ := item.(map[string]any)
				if member == nil {
					continue
				}
				rows = append(rows, []string{
					toString(member["identity_id"]),
					toString(member["joined_at"]),
				})
			}
			return printTable(s.out, []string{"PARTICIPANT", "JOINED_AT"}, rows)
		case "join":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 2 {
				return errors.New("usage: session join <session> <participant>")
			}
			body, err := s.postFormJSON(ctx, "/admin/api/session/join", url.Values{
				"session_id":  {plain[0]},
				"participant": {plain[1]},
			})
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(s.out, body)
			}
			sessionID := plain[0]
			if sessionMap, ok := body["session"].(map[string]any); ok {
				if id := toString(sessionMap["id"]); id != "" {
					sessionID = id
				}
			}
			_, err = fmt.Fprintf(s.out, "OK  session=%s participant=%s action=join\n", sessionID, plain[1])
			return err
		case "leave":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 2 {
				return errors.New("usage: session leave <session> <participant>")
			}
			body, err := s.postFormJSON(ctx, "/admin/api/session/leave", url.Values{
				"session_id":  {plain[0]},
				"participant": {plain[1]},
			})
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(s.out, body)
			}
			sessionID := plain[0]
			if sessionMap, ok := body["session"].(map[string]any); ok {
				if id := toString(sessionMap["id"]); id != "" {
					sessionID = id
				}
			}
			_, err = fmt.Fprintf(s.out, "OK  session=%s participant=%s action=leave\n", sessionID, plain[1])
			return err
		case "attach":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 2 {
				return errors.New("usage: session attach <session> <device-ref>")
			}
			body, err := s.postFormJSON(ctx, "/admin/api/session/attach", url.Values{
				"session_id": {plain[0]},
				"device_ref": {plain[1]},
			})
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(s.out, body)
			}
			_, err = fmt.Fprintf(s.out, "OK  session=%s device=%s action=attach\n", plain[0], plain[1])
			return err
		case "detach":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 2 {
				return errors.New("usage: session detach <session> <device-ref>")
			}
			body, err := s.postFormJSON(ctx, "/admin/api/session/detach", url.Values{
				"session_id": {plain[0]},
				"device_ref": {plain[1]},
			})
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(s.out, body)
			}
			_, err = fmt.Fprintf(s.out, "OK  session=%s device=%s action=detach\n", plain[0], plain[1])
			return err
		case "control":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 1 {
				return errors.New("usage: session control <request|grant|revoke>")
			}
			action := strings.ToLower(plain[0])
			switch action {
			case "request":
				if len(plain) < 3 {
					return errors.New("usage: session control request <session> <participant> [control-type]")
				}
				controlType := ""
				if len(plain) > 3 {
					controlType = plain[3]
				}
				body, err := s.postFormJSON(ctx, "/admin/api/session/control/request", url.Values{
					"session_id":   {plain[1]},
					"participant":  {plain[2]},
					"control_type": {controlType},
				})
				if err != nil {
					return err
				}
				if jsonOut {
					return writeJSON(s.out, body)
				}
				_, err = fmt.Fprintf(s.out, "OK  session=%s participant=%s action=control.request type=%s\n", plain[1], plain[2], defaultIfBlank(controlType, "interactive"))
				return err
			case "grant":
				if len(plain) < 3 {
					return errors.New("usage: session control grant <session> <participant> [granted-by] [control-type]")
				}
				grantedBy := ""
				if len(plain) > 3 {
					grantedBy = plain[3]
				}
				controlType := ""
				if len(plain) > 4 {
					controlType = plain[4]
				}
				body, err := s.postFormJSON(ctx, "/admin/api/session/control/grant", url.Values{
					"session_id":   {plain[1]},
					"participant":  {plain[2]},
					"granted_by":   {grantedBy},
					"control_type": {controlType},
				})
				if err != nil {
					return err
				}
				if jsonOut {
					return writeJSON(s.out, body)
				}
				_, err = fmt.Fprintf(s.out, "OK  session=%s participant=%s action=control.grant by=%s type=%s\n", plain[1], plain[2], defaultIfBlank(grantedBy, "system"), defaultIfBlank(controlType, "interactive"))
				return err
			case "revoke":
				if len(plain) < 3 {
					return errors.New("usage: session control revoke <session> <participant> [revoked-by]")
				}
				revokedBy := ""
				if len(plain) > 3 {
					revokedBy = plain[3]
				}
				body, err := s.postFormJSON(ctx, "/admin/api/session/control/revoke", url.Values{
					"session_id":  {plain[1]},
					"participant": {plain[2]},
					"revoked_by":  {revokedBy},
				})
				if err != nil {
					return err
				}
				if jsonOut {
					return writeJSON(s.out, body)
				}
				_, err = fmt.Fprintf(s.out, "OK  session=%s participant=%s action=control.revoke by=%s\n", plain[1], plain[2], defaultIfBlank(revokedBy, "system"))
				return err
			default:
				return fmt.Errorf("unknown command: session control %s", action)
			}
		default:
			return fmt.Errorf("unknown command: session %s", sub)
		}
	case "message":
		switch sub {
		case "rooms":
			body, err := s.fetchJSON(ctx, "/admin/api/message/rooms")
			if err != nil {
				return err
			}
			return writeJSON(s.out, body)
		case "room":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 1 {
				return errors.New("usage: message room <new|show>")
			}
			action := strings.ToLower(plain[0])
			switch action {
			case "new":
				if len(plain) < 2 {
					return errors.New("usage: message room new <name>")
				}
				body, err := s.postFormJSON(ctx, "/admin/api/message/room", url.Values{"name": {strings.Join(plain[1:], " ")}})
				if err != nil {
					return err
				}
				if jsonOut {
					return writeJSON(s.out, body)
				}
				roomID := ""
				roomName := strings.Join(plain[1:], " ")
				if roomMap, ok := body["room"].(map[string]any); ok {
					roomID = toString(roomMap["id"])
					if name := toString(roomMap["name"]); name != "" {
						roomName = name
					}
				}
				_, err = fmt.Fprintf(s.out, "OK  room=%s name=%s action=create\n", roomID, roomName)
				return err
			case "show":
				if len(plain) < 2 {
					return errors.New("usage: message room show <room>")
				}
				body, err := s.fetchJSONQuery(ctx, "/admin/api/message/room", url.Values{"room": {plain[1]}})
				if err != nil {
					return err
				}
				return writeJSON(s.out, body)
			default:
				return fmt.Errorf("unknown command: message room %s", action)
			}
		case "ls":
			query := url.Values{}
			plain := nonFlagArgs(args[1:])
			if len(plain) > 0 {
				query.Set("room", plain[0])
			}
			body, err := s.fetchJSONQuery(ctx, "/admin/api/message", query)
			if err != nil {
				return err
			}
			return writeJSON(s.out, body)
		case "get":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 1 {
				return errors.New("usage: message get <message>")
			}
			body, err := s.fetchJSONQuery(ctx, "/admin/api/message/get", url.Values{"message_id": {plain[0]}})
			if err != nil {
				return err
			}
			return writeJSON(s.out, body)
		case "unread":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 1 {
				return errors.New("usage: message unread <identity> [room]")
			}
			query := url.Values{"identity_id": {plain[0]}}
			if len(plain) > 1 {
				query.Set("room", plain[1])
			}
			body, err := s.fetchJSONQuery(ctx, "/admin/api/message/unread", query)
			if err != nil {
				return err
			}
			return writeJSON(s.out, body)
		case "post":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 2 {
				return errors.New("usage: message post <room> <text>")
			}
			body, err := s.postFormJSON(ctx, "/admin/api/message/post", url.Values{
				"room": {plain[0]},
				"text": {strings.Join(plain[1:], " ")},
			})
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(s.out, body)
			}
			messageID := ""
			if msgMap, ok := body["message"].(map[string]any); ok {
				messageID = toString(msgMap["id"])
			}
			_, err = fmt.Fprintf(s.out, "OK  message=%s\n", messageID)
			return err
		case "dm":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 2 {
				return errors.New("usage: message dm <target> <text>")
			}
			body, err := s.postFormJSON(ctx, "/admin/api/message/dm", url.Values{
				"target_ref": {plain[0]},
				"text":       {strings.Join(plain[1:], " ")},
			})
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(s.out, body)
			}
			messageID := ""
			if msgMap, ok := body["message"].(map[string]any); ok {
				messageID = toString(msgMap["id"])
			}
			_, err = fmt.Fprintf(s.out, "OK  message=%s target=%s action=dm\n", messageID, plain[0])
			return err
		case "thread":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 2 {
				return errors.New("usage: message thread <root-message> <text>")
			}
			body, err := s.postFormJSON(ctx, "/admin/api/message/thread", url.Values{
				"root_ref": {plain[0]},
				"text":     {strings.Join(plain[1:], " ")},
			})
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(s.out, body)
			}
			messageID := ""
			if msgMap, ok := body["message"].(map[string]any); ok {
				messageID = toString(msgMap["id"])
			}
			_, err = fmt.Fprintf(s.out, "OK  message=%s root=%s action=thread\n", messageID, plain[0])
			return err
		case "ack":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 2 {
				return errors.New("usage: message ack <identity> <message>")
			}
			body, err := s.postFormJSON(ctx, "/admin/api/message/ack", url.Values{
				"identity_id": {plain[0]},
				"message_id":  {plain[1]},
			})
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(s.out, body)
			}
			_, err = fmt.Fprintf(s.out, "OK  identity=%s message=%s action=ack\n", plain[0], plain[1])
			return err
		default:
			return fmt.Errorf("unknown command: message %s", sub)
		}
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
	case "bus":
		switch sub {
		case "emit":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 2 {
				return errors.New("usage: bus emit <kind> <name> [payload]")
			}
			payload := ""
			if len(plain) > 2 {
				payload = strings.Join(plain[2:], " ")
			}
			body, err := s.postFormJSON(ctx, "/admin/api/bus/emit", url.Values{
				"kind":    {plain[0]},
				"name":    {plain[1]},
				"payload": {payload},
			})
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(s.out, body)
			}
			eventID := ""
			if itemMap, ok := body["event"].(map[string]any); ok {
				eventID = toString(itemMap["id"])
			}
			_, err = fmt.Fprintf(s.out, "OK  event=%s\n", eventID)
			return err
		case "tail":
			query := url.Values{}
			if kind := strings.TrimSpace(flagValue(args[1:], "--kind")); kind != "" {
				query.Set("kind", kind)
			}
			if name := strings.TrimSpace(flagValue(args[1:], "--name")); name != "" {
				query.Set("name", name)
			}
			if limitRaw := strings.TrimSpace(flagValue(args[1:], "--limit")); limitRaw != "" {
				if limit, err := strconv.Atoi(limitRaw); err != nil || limit <= 0 {
					return errors.New("usage: bus tail [--kind <kind>] [--name <name>] [--limit <n>]")
				}
				query.Set("limit", limitRaw)
			}
			body, err := s.fetchJSONQuery(ctx, "/admin/api/bus", query)
			if err != nil {
				return err
			}
			return writeJSON(s.out, body)
		case "replay":
			plain := nonFlagArgsSkippingFlagValues(args[1:], "--kind", "--name", "--limit")
			if len(plain) < 2 {
				return errors.New("usage: bus replay <from-id> <to-id> [--kind <kind>] [--name <name>] [--limit <n>]")
			}
			query := url.Values{
				"from": {plain[0]},
				"to":   {plain[1]},
			}
			if kind := strings.TrimSpace(flagValue(args[1:], "--kind")); kind != "" {
				query.Set("kind", kind)
			}
			if name := strings.TrimSpace(flagValue(args[1:], "--name")); name != "" {
				query.Set("name", name)
			}
			if limitRaw := strings.TrimSpace(flagValue(args[1:], "--limit")); limitRaw != "" {
				if limit, err := strconv.Atoi(limitRaw); err != nil || limit <= 0 {
					return errors.New("usage: bus replay <from-id> <to-id> [--kind <kind>] [--name <name>] [--limit <n>]")
				}
				query.Set("limit", limitRaw)
			}
			body, err := s.fetchJSONQuery(ctx, "/admin/api/bus/replay", query)
			if err != nil {
				return err
			}
			return writeJSON(s.out, body)
		default:
			return fmt.Errorf("unknown command: bus %s", sub)
		}
	case "handlers":
		switch sub {
		case "ls":
			body, err := s.fetchJSON(ctx, "/admin/api/handlers")
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(s.out, body)
			}
			items, _ := body["handlers"].([]any)
			rows := make([][]string, 0, len(items))
			for _, item := range items {
				row, _ := item.(map[string]any)
				if row == nil {
					continue
				}
				target := toString(row["run_command"])
				if target == "" {
					emitKind := toString(row["emit_kind"])
					emitName := toString(row["emit_name"])
					emitPayload := toString(row["emit_payload"])
					target = strings.TrimSpace("emit " + emitKind + " " + emitName + " " + emitPayload)
				}
				rows = append(rows, []string{toString(row["id"]), toString(row["selector"]), toString(row["action"]), target})
			}
			return printTable(s.out, []string{"HANDLER", "SELECTOR", "ACTION", "TARGET"}, rows)
		case "on":
			plain := nonFlagArgsSkippingFlagValues(args[1:], "--run")
			if len(plain) < 2 {
				return errors.New("usage: handlers on <selector> <action> (--run <command> | --emit <kind> <name> [payload])")
			}
			selector := plain[0]
			action := plain[1]
			runCommand := strings.TrimSpace(flagValue(args[1:], "--run"))
			emitKind, emitName, emitPayload := parseHandlersEmitValue(args[1:])
			hasRun := runCommand != ""
			hasEmit := emitKind != "" || emitName != "" || emitPayload != ""
			if hasRun == hasEmit {
				return errors.New("usage: handlers on <selector> <action> (--run <command> | --emit <kind> <name> [payload])")
			}

			form := url.Values{
				"selector": {selector},
				"action":   {action},
			}
			if hasRun {
				form.Set("run", runCommand)
			} else {
				if emitName == "" {
					return errors.New("usage: handlers on <selector> <action> --emit <kind> <name> [payload]")
				}
				form.Set("emit_kind", emitKind)
				form.Set("emit_name", emitName)
				if emitPayload != "" {
					form.Set("emit_payload", emitPayload)
				}
			}
			body, err := s.postFormJSON(ctx, "/admin/api/handlers/on", form)
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(s.out, body)
			}
			handlerID := ""
			if itemMap, ok := body["handler"].(map[string]any); ok {
				handlerID = toString(itemMap["id"])
			}
			_, err = fmt.Fprintf(s.out, "OK  handler=%s selector=%s action=%s\n", handlerID, selector, action)
			return err
		case "off":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 1 {
				return errors.New("usage: handlers off <handler-id>")
			}
			body, err := s.postFormJSON(ctx, "/admin/api/handlers/off", url.Values{"handler_id": {plain[0]}})
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(s.out, body)
			}
			_, err = fmt.Fprintf(s.out, "OK  deleted=%s handler=%s\n", toString(body["deleted"]), plain[0])
			return err
		default:
			return fmt.Errorf("unknown command: handlers %s", sub)
		}
	case "scenarios":
		switch sub {
		case "ls":
			body, err := s.fetchJSON(ctx, "/admin/api/scenarios/inline")
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(s.out, body)
			}
			items, _ := body["scenarios"].([]any)
			rows := make([][]string, 0, len(items))
			for _, item := range items {
				row, _ := item.(map[string]any)
				if row == nil {
					continue
				}
				intents := joinAnyStrings(row["match_intents"], ",")
				events := joinAnyStrings(row["match_events"], ",")
				rows = append(rows, []string{toString(row["name"]), toString(row["priority"]), intents, events})
			}
			return printTable(s.out, []string{"SCENARIO", "PRIORITY", "INTENTS", "EVENTS"}, rows)
		case "show":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 1 {
				return errors.New("usage: scenarios show <name>")
			}
			body, err := s.fetchJSONQuery(ctx, "/admin/api/scenarios/inline", url.Values{"name": {plain[0]}})
			if err != nil {
				return err
			}
			return writeJSON(s.out, body)
		case "define":
			def, err := parseScenariosDefineArgs(args[1:])
			if err != nil {
				return err
			}
			form := url.Values{"name": {def.name}}
			for _, intent := range def.matchIntents {
				form.Add("match_intent", intent)
			}
			for _, event := range def.matchEvents {
				form.Add("match_event", event)
			}
			if def.priority != "" {
				form.Set("priority", def.priority)
			}
			if def.onStart != "" {
				form.Set("on_start", def.onStart)
			}
			if def.onInput != "" {
				form.Set("on_input", def.onInput)
			}
			if def.onSuspend != "" {
				form.Set("on_suspend", def.onSuspend)
			}
			if def.onResume != "" {
				form.Set("on_resume", def.onResume)
			}
			if def.onStop != "" {
				form.Set("on_stop", def.onStop)
			}
			for _, hook := range def.onEvents {
				form.Add("on_event_kind", hook.kind)
				form.Add("on_event_command", hook.command)
			}
			body, err := s.postFormJSON(ctx, "/admin/api/scenarios/inline/define", form)
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(s.out, body)
			}
			_, err = fmt.Fprintf(s.out, "OK  action=define scenario=%s\n", def.name)
			return err
		case "undefine":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 1 {
				return errors.New("usage: scenarios undefine <name>")
			}
			body, err := s.postFormJSON(ctx, "/admin/api/scenarios/inline/undefine", url.Values{"name": {plain[0]}})
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(s.out, body)
			}
			_, err = fmt.Fprintf(s.out, "OK  deleted=%s scenario=%s\n", toString(body["deleted"]), plain[0])
			return err
		default:
			return fmt.Errorf("unknown command: scenarios %s", sub)
		}
	case "sim":
		switch sub {
		case "device":
			if len(args) < 2 {
				return errors.New("usage: sim device <new|rm>")
			}
			deviceSub := strings.ToLower(strings.TrimSpace(args[1]))
			switch deviceSub {
			case "new":
				plain := nonFlagArgsSkippingFlagValues(args[2:], "--caps")
				if len(plain) < 1 {
					return errors.New("usage: sim device new <id> [--caps <cap[,cap...]>]")
				}
				form := url.Values{"device_id": {plain[0]}}
				if capsRaw := strings.TrimSpace(flagValue(args[2:], "--caps")); capsRaw != "" {
					for _, capValue := range parseCSVValues(capsRaw) {
						form.Add("caps", capValue)
					}
				}
				body, err := s.postFormJSON(ctx, "/admin/api/sim/devices/new", form)
				if err != nil {
					return err
				}
				if jsonOut {
					return writeJSON(s.out, body)
				}
				_, err = fmt.Fprintf(s.out, "OK  action=sim.device.new device=%s\n", plain[0])
				return err
			case "rm":
				plain := nonFlagArgs(args[2:])
				if len(plain) < 1 {
					return errors.New("usage: sim device rm <id>")
				}
				body, err := s.postFormJSON(ctx, "/admin/api/sim/devices/rm", url.Values{"device_id": {plain[0]}})
				if err != nil {
					return err
				}
				if jsonOut {
					return writeJSON(s.out, body)
				}
				_, err = fmt.Fprintf(s.out, "OK  action=sim.device.rm device=%s deleted=%s\n", plain[0], toString(body["deleted"]))
				return err
			default:
				return fmt.Errorf("unknown command: sim device %s", deviceSub)
			}
		case "input":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 3 {
				return errors.New("usage: sim input <id> <component-id> <action> [<value>]")
			}
			value := ""
			if len(plain) > 3 {
				value = strings.Join(plain[3:], " ")
			}
			body, err := s.postFormJSON(ctx, "/admin/api/sim/input", url.Values{
				"device_id":    {plain[0]},
				"component_id": {plain[1]},
				"action":       {plain[2]},
				"value":        {value},
			})
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(s.out, body)
			}
			_, err = fmt.Fprintf(s.out, "OK  action=sim.input device=%s component=%s event=%s\n", plain[0], plain[1], plain[2])
			return err
		case "ui":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 1 {
				return errors.New("usage: sim ui <id>")
			}
			body, err := s.fetchJSONQuery(ctx, "/admin/api/sim/ui", url.Values{"device_id": {plain[0]}})
			if err != nil {
				return err
			}
			return writeJSON(s.out, body)
		case "expect":
			plain := nonFlagArgsSkippingFlagValues(args[1:], "--within")
			if len(plain) < 3 {
				return errors.New("usage: sim expect <id> <ui|message> <selector> [--within <duration>]")
			}
			form := url.Values{
				"device_id": {plain[0]},
				"kind":      {plain[1]},
				"selector":  {strings.Join(plain[2:], " ")},
			}
			if within := strings.TrimSpace(flagValue(args[1:], "--within")); within != "" {
				form.Set("within", within)
			}
			body, err := s.postFormJSON(ctx, "/admin/api/sim/expect", form)
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(s.out, body)
			}
			result, _ := body["result"].(map[string]any)
			_, err = fmt.Fprintf(s.out, "OK  action=sim.expect device=%s kind=%s matched=%s\n", plain[0], plain[1], toString(result["matched"]))
			return err
		case "record":
			plain := nonFlagArgsSkippingFlagValues(args[1:], "--duration")
			if len(plain) < 1 {
				return errors.New("usage: sim record <id> [--duration <duration>]")
			}
			form := url.Values{"device_id": {plain[0]}}
			if duration := strings.TrimSpace(flagValue(args[1:], "--duration")); duration != "" {
				form.Set("duration", duration)
			}
			body, err := s.postFormJSON(ctx, "/admin/api/sim/record", form)
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(s.out, body)
			}
			result, _ := body["result"].(map[string]any)
			inputs := toAnySlice(result["inputs"])
			messages := toAnySlice(result["messages"])
			_, err = fmt.Fprintf(s.out, "OK  action=sim.record device=%s inputs=%d messages=%d\n", plain[0], len(inputs), len(messages))
			return err
		default:
			return fmt.Errorf("unknown command: sim %s", sub)
		}
	case "scripts":
		switch sub {
		case "dry-run":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 1 {
				return errors.New("usage: scripts dry-run <path>")
			}
			body, err := s.postFormJSON(ctx, "/admin/api/scripts/dry-run", url.Values{"path": {plain[0]}})
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(s.out, body)
			}
			result, _ := body["result"].(map[string]any)
			_, err = fmt.Fprintf(s.out, "OK  action=scripts.dry-run path=%s commands=%s skipped=%s\n", plain[0], toString(result["command_count"]), toString(result["skipped_count"]))
			return err
		case "run":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 1 {
				return errors.New("usage: scripts run <path>")
			}
			body, err := s.postFormJSON(ctx, "/admin/api/scripts/run", url.Values{"path": {plain[0]}})
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(s.out, body)
			}
			result, _ := body["result"].(map[string]any)
			_, err = fmt.Fprintf(s.out, "OK  action=scripts.run path=%s commands=%s executed=%s failed=%s\n", plain[0], toString(result["command_count"]), toString(result["executed_count"]), toString(result["failed_count"]))
			return err
		default:
			return fmt.Errorf("unknown command: scripts %s", sub)
		}
	case "activations":
		if sub != "ls" {
			return fmt.Errorf("unknown command: activations %s", sub)
		}
		body, err := s.fetchJSON(ctx, "/admin/api/activations")
		if err != nil {
			return err
		}
		if jsonOut {
			return writeJSON(s.out, body)
		}
		active, _ := body["active_by_device"].(map[string]any)
		rows := make([][]string, 0, len(active))
		for deviceID, scenarioName := range active {
			rows = append(rows, []string{deviceID, toString(scenarioName)})
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i][0] < rows[j][0] })
		return printTable(s.out, []string{"DEVICE", "ACTIVE"}, rows)
	case "claims":
		if sub != "tree" {
			return fmt.Errorf("unknown command: claims %s", sub)
		}
		body, err := s.fetchJSON(ctx, "/admin/api/activations")
		if err != nil {
			return err
		}
		if jsonOut {
			return writeJSON(s.out, body)
		}
		claimsByDevice, _ := body["claims_by_device"].(map[string]any)
		if len(claimsByDevice) == 0 {
			_, err := fmt.Fprintln(s.out, "(no claims)")
			return err
		}
		deviceIDs := make([]string, 0, len(claimsByDevice))
		for deviceID := range claimsByDevice {
			deviceIDs = append(deviceIDs, deviceID)
		}
		sort.Strings(deviceIDs)
		for _, deviceID := range deviceIDs {
			if _, err := fmt.Fprintf(s.out, "%s\n", deviceID); err != nil {
				return err
			}
			claims, _ := claimsByDevice[deviceID].([]any)
			if len(claims) == 0 {
				if _, err := fmt.Fprintln(s.out, "  (none)"); err != nil {
					return err
				}
				continue
			}
			for _, claimAny := range claims {
				claim, _ := claimAny.(map[string]any)
				if claim == nil {
					continue
				}
				if _, err := fmt.Fprintf(s.out, "  - %s by %s\n", toString(claim["resource"]), toString(claim["activation_id"])); err != nil {
					return err
				}
			}
		}
		return nil
	case "app":
		switch sub {
		case "ls":
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
		case "reload", "rollback":
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
		case "logs":
			if len(args) < 2 {
				return errors.New("usage: app logs <app> [query]")
			}
			appName := strings.TrimSpace(args[1])
			query := strings.TrimSpace(strings.Join(args[2:], " "))
			return s.queryLogs(ctx, appName, query)
		default:
			return fmt.Errorf("unknown command: app %s", sub)
		}
	case "apps":
		switch sub {
		case "migrate":
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
		case "keys":
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
		default:
			return fmt.Errorf("unknown command: apps %s", sub)
		}
	case "config":
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
	case "docs":
		switch sub {
		case "ls":
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
		case "search":
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
		case "open":
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
		case "examples":
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
		default:
			return fmt.Errorf("unknown command: docs %s", sub)
		}
	case "logs":
		if sub != "tail" {
			return fmt.Errorf("unknown command: logs %s", sub)
		}
		query := strings.TrimSpace(strings.Join(args[1:], " "))
		return s.queryLogs(ctx, "", query)
	case "observe":
		if sub != "tail" {
			return fmt.Errorf("unknown command: observe %s", sub)
		}
		query := strings.TrimSpace(strings.Join(args[1:], " "))
		return s.queryLogs(ctx, "", query)
	case "ai":
		switch sub {
		case "providers":
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
		case "models":
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
		case "use":
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
		case "status":
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
		case "ask":
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
		case "gen":
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
		case "run", "approve":
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
		case "reject":
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
		case "context":
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
		case "policy":
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
		case "history":
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
		case "reset":
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
		default:
			return fmt.Errorf("unknown command: ai %s", sub)
		}
	default:
		return fmt.Errorf("unsupported command group: %s", group)
	}
}

