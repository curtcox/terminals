package repl

import (
	"context"
	"errors"
	"fmt"
	"net/url"
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
	default:
		return s.evalControlPlaneMid(ctx, group, args, jsonOut)
	}
}
