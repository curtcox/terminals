// Package repl implements the control-plane REPL used by terminal sessions.
package repl

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Options configures a REPL run.
type Options struct {
	Prompt       string
	AdminBaseURL string
	SessionID    string
	DocsMode     DocsRenderMode
}

type commandClassification string

const (
	commandReadOnly    commandClassification = "read_only"
	commandOperational commandClassification = "operational"
	commandMutating    commandClassification = "mutating"
)

type commandSpec struct {
	Name                 string
	Usage                string
	Summary              string
	Classification       commandClassification
	Examples             []string
	RelatedDocs          []string
	DiscouragedForAgents bool
}

func replCommandSpecs() []commandSpec {
	return []commandSpec{
		{Name: "help", Usage: "help [command]", Summary: "Show REPL help or help for one command", Classification: commandReadOnly, Examples: []string{"help", "help app reload"}},
		{Name: "describe", Usage: "describe <command>", Summary: "Show a detailed command description", Classification: commandReadOnly, Examples: []string{"describe sessions terminate"}},
		{Name: "complete", Usage: "complete <prefix>", Summary: "List command completions for a prefix", Classification: commandReadOnly, Examples: []string{"complete app r"}},
		{Name: "echo", Usage: "echo <text>", Summary: "Print text", Classification: commandReadOnly},
		{Name: "sleep", Usage: "sleep <seconds>", Summary: "Sleep for N seconds", Classification: commandOperational},
		{Name: "printf", Usage: "printf <text>", Summary: "Print text without newline (supports \\xNN escapes)", Classification: commandReadOnly},
		{Name: "clear", Usage: "clear", Summary: "Clear terminal display", Classification: commandReadOnly},
		{Name: "exit", Usage: "exit", Summary: "Exit REPL", Classification: commandReadOnly},
		{Name: "devices ls", Usage: "devices ls [--json]", Summary: "List devices", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/devices"}},
		{Name: "sessions ls", Usage: "sessions ls [--json]", Summary: "List REPL sessions", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/sessions"}},
		{Name: "sessions show", Usage: "sessions show <session>", Summary: "Show one REPL session", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/sessions"}},
		{Name: "sessions terminate", Usage: "sessions terminate <session>", Summary: "Terminate one REPL session", Classification: commandMutating, RelatedDocs: []string{"repl/commands/sessions"}},
		{Name: "identity ls", Usage: "identity ls [--json]", Summary: "List identities and audiences", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/identity"}},
		{Name: "identity resolve", Usage: "identity resolve <audience> [--json]", Summary: "Resolve an audience to identities", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/identity"}},
		{Name: "session ls", Usage: "session ls [--json]", Summary: "List interactive sessions", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/session"}},
		{Name: "session create", Usage: "session create <kind> <target> [--json]", Summary: "Create a generalized interactive session", Classification: commandMutating, RelatedDocs: []string{"repl/commands/session"}},
		{Name: "message ls", Usage: "message ls [room] [--json]", Summary: "List messages", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/message"}},
		{Name: "message post", Usage: "message post <room> <text> [--json]", Summary: "Post a room/direct message", Classification: commandMutating, RelatedDocs: []string{"repl/commands/message"}},
		{Name: "board ls", Usage: "board ls [board] [--json]", Summary: "List board or bulletin entries", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/board"}},
		{Name: "board pin", Usage: "board pin <board> <text> [--json]", Summary: "Pin a bulletin entry", Classification: commandMutating, RelatedDocs: []string{"repl/commands/board"}},
		{Name: "artifact ls", Usage: "artifact ls [--json]", Summary: "List shared artifacts", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/artifact"}},
		{Name: "artifact create", Usage: "artifact create <kind> <title> [--json]", Summary: "Create a shared artifact", Classification: commandMutating, RelatedDocs: []string{"repl/commands/artifact"}},
		{Name: "canvas ls", Usage: "canvas ls [canvas] [--json]", Summary: "List canvas annotations", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/canvas"}},
		{Name: "canvas annotate", Usage: "canvas annotate <canvas> <text> [--json]", Summary: "Annotate a shared canvas", Classification: commandMutating, RelatedDocs: []string{"repl/commands/canvas"}},
		{Name: "search query", Usage: "search query <text> [--json]", Summary: "Run unified search", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/search"}},
		{Name: "memory remember", Usage: "memory remember <scope> <text> [--json]", Summary: "Store a memory entry", Classification: commandMutating, RelatedDocs: []string{"repl/commands/memory"}},
		{Name: "memory recall", Usage: "memory recall <text> [--json]", Summary: "Recall memory entries", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/memory"}},
		{Name: "placement ls", Usage: "placement ls [--json]", Summary: "List placement metadata", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/placement"}},
		{Name: "recent ls", Usage: "recent ls [--json]", Summary: "List recent activity", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/recent"}},
		{Name: "store put", Usage: "store put <namespace> <key> <value> [--json]", Summary: "Write typed key-value state", Classification: commandMutating, RelatedDocs: []string{"repl/commands/store"}},
		{Name: "store get", Usage: "store get <namespace> <key> [--json]", Summary: "Read typed key-value state", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/store"}},
		{Name: "store ls", Usage: "store ls <namespace> [--json]", Summary: "List typed key-value state", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/store"}},
		{Name: "bus emit", Usage: "bus emit <kind> <name> [payload] [--json]", Summary: "Emit typed bus events or intents", Classification: commandMutating, RelatedDocs: []string{"repl/commands/bus"}},
		{Name: "bus tail", Usage: "bus tail [--json]", Summary: "Tail recent bus events", Classification: commandOperational, RelatedDocs: []string{"repl/commands/bus"}},
		{Name: "activations ls", Usage: "activations ls [--json]", Summary: "List active scenario by device", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/activations"}},
		{Name: "claims tree", Usage: "claims tree [--json]", Summary: "Show claims grouped by device", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/claims"}},
		{Name: "app ls", Usage: "app ls [--json]", Summary: "List loaded apps", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/app"}},
		{Name: "app logs", Usage: "app logs <app> [<query>]", Summary: "Query app-related logs", Classification: commandOperational, RelatedDocs: []string{"repl/commands/app"}},
		{Name: "app reload", Usage: "app reload <app> [--json]", Summary: "Reload an app package", Classification: commandMutating, RelatedDocs: []string{"repl/commands/app"}},
		{Name: "app rollback", Usage: "app rollback <app> [--json]", Summary: "Rollback an app package", Classification: commandMutating, RelatedDocs: []string{"repl/commands/app"}},
		{Name: "config show", Usage: "config show [--json]", Summary: "Show effective config", Classification: commandReadOnly},
		{Name: "logs tail", Usage: "logs tail [<query>]", Summary: "Query recent server logs", Classification: commandOperational, RelatedDocs: []string{"repl/commands/logs"}},
		{Name: "observe tail", Usage: "observe tail [<query>]", Summary: "Alias for logs tail", Classification: commandOperational, RelatedDocs: []string{"repl/commands/logs"}},
		{Name: "docs ls", Usage: "docs ls", Summary: "List documentation topics", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/docs"}},
		{Name: "docs search", Usage: "docs search <query>", Summary: "Search documentation topics", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/docs"}},
		{Name: "docs open", Usage: "docs open <topic>", Summary: "Open one documentation topic", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/docs"}},
		{Name: "docs examples", Usage: "docs examples [filter]", Summary: "List example topics", Classification: commandReadOnly, RelatedDocs: []string{"repl/examples/app-dev-loop"}},
		{Name: "ai providers", Usage: "ai providers [--json]", Summary: "List configured AI providers", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/ai"}, DiscouragedForAgents: true},
		{Name: "ai models", Usage: "ai models [provider] [--json]", Summary: "List models for a provider", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/ai"}, DiscouragedForAgents: true},
		{Name: "ai use", Usage: "ai use <provider> <model> [--json]", Summary: "Set sticky provider/model selection for this session", Classification: commandMutating, RelatedDocs: []string{"repl/commands/ai"}, DiscouragedForAgents: true},
		{Name: "ai status", Usage: "ai status [--json]", Summary: "Show current provider/model selection for this session", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/ai"}, DiscouragedForAgents: true},
	}
}

func replCommandSpec(name string) (commandSpec, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, spec := range replCommandSpecs() {
		if strings.EqualFold(spec.Name, name) {
			return spec, true
		}
	}
	return commandSpec{}, false
}

// Run executes the Terminals control-plane REPL over stdin/stdout.
func Run(ctx context.Context, in io.Reader, out io.Writer, opts Options) error {
	if in == nil {
		return errors.New("nil input")
	}
	if out == nil {
		return errors.New("nil output")
	}
	prompt := strings.TrimSpace(opts.Prompt)
	if prompt == "" {
		prompt = "repl>"
	}
	prompt += " "

	state := newStateWithDocsMode(out, opts.AdminBaseURL, opts.SessionID, opts.DocsMode)
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 1024), 1024*1024)

	if _, err := fmt.Fprintf(out, "Terminals REPL (control-plane only). Type 'help' for commands.\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprint(out, prompt); err != nil {
		return err
	}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			if _, err := fmt.Fprint(out, prompt); err != nil {
				return err
			}
			continue
		}
		exit, err := state.eval(ctx, line)
		if err != nil {
			if _, writeErr := fmt.Fprintf(out, "error: %v\n", err); writeErr != nil {
				return writeErr
			}
		}
		if exit {
			return nil
		}
		if _, err := fmt.Fprint(out, prompt); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

type state struct {
	out      io.Writer
	adminURL string
	session  string
	client   *http.Client
	docsMode DocsRenderMode
	docsRoot string
}

func newState(out io.Writer, adminBaseURL, sessionID string) *state {
	return newStateWithDocsMode(out, adminBaseURL, sessionID, DocsRenderModeTerminal)
}

func newStateWithDocsMode(out io.Writer, adminBaseURL, sessionID string, docsMode DocsRenderMode) *state {
	adminBaseURL = strings.TrimSpace(adminBaseURL)
	if adminBaseURL == "" {
		adminBaseURL = strings.TrimSpace(os.Getenv("TERMINALS_REPL_ADMIN_URL"))
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		sessionID = strings.TrimSpace(os.Getenv("TERMINALS_REPL_SESSION_ID"))
	}
	if adminBaseURL == "" {
		adminBaseURL = "http://127.0.0.1:50053"
	}
	adminBaseURL = strings.TrimSuffix(adminBaseURL, "/")
	return &state{
		out:      out,
		adminURL: adminBaseURL,
		session:  sessionID,
		client:   &http.Client{Timeout: 3 * time.Second},
		docsMode: normalizeDocsRenderMode(docsMode),
		docsRoot: resolveDocsRoot(),
	}
}

func (s *state) eval(ctx context.Context, line string) (bool, error) {
	segments := splitSegments(line)
	for _, segment := range segments {
		tokens := tokenize(segment)
		if len(tokens) == 0 {
			continue
		}
		exit, err := s.evalOne(ctx, tokens)
		if err != nil {
			return false, err
		}
		if exit {
			return true, nil
		}
	}
	return false, nil
}

func (s *state) evalOne(ctx context.Context, tokens []string) (bool, error) {
	cmd := strings.ToLower(tokens[0])
	switch cmd {
	case "help":
		return false, s.printHelp(tokens[1:])
	case "describe":
		if err := s.describeCommand(tokens[1:]); err != nil {
			return false, err
		}
		return false, nil
	case "complete":
		if err := s.completeCommand(tokens[1:]); err != nil {
			return false, err
		}
		return false, nil
	case "echo":
		_, err := fmt.Fprintln(s.out, strings.Join(tokens[1:], " "))
		return false, err
	case "sleep":
		if len(tokens) < 2 {
			return false, errors.New("usage: sleep <seconds>")
		}
		secs, err := strconv.ParseFloat(tokens[1], 64)
		if err != nil || secs < 0 {
			return false, fmt.Errorf("invalid sleep duration: %s", tokens[1])
		}
		t := time.NewTimer(time.Duration(secs * float64(time.Second)))
		defer t.Stop()
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-t.C:
			return false, nil
		}
	case "printf":
		if len(tokens) < 2 {
			return false, errors.New("usage: printf <text>")
		}
		text := decodeEscapes(strings.Join(tokens[1:], " "))
		_, err := fmt.Fprint(s.out, text)
		return false, err
	case "clear":
		_, err := fmt.Fprint(s.out, "\033[2J\033[H")
		return false, err
	case "exit", "quit":
		_, err := fmt.Fprintln(s.out, "bye")
		return true, err
	case "devices", "sessions", "identity", "session", "message", "board", "artifact", "canvas", "search", "memory", "placement", "recent", "store", "bus", "activations", "claims", "app", "config", "docs", "logs", "observe", "ai":
		return false, s.evalControlPlane(ctx, cmd, tokens[1:])
	default:
		input := strings.ToLower(strings.TrimSpace(strings.Join(tokens, " ")))
		suggestions := suggestApproxCommands(input, 3)
		if len(suggestions) == 0 {
			return false, fmt.Errorf("unknown command: %s", tokens[0])
		}
		return false, fmt.Errorf("unknown command: %s (try: %s)", tokens[0], strings.Join(suggestions, ", "))
	}
}

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
		default:
			return fmt.Errorf("unknown command: session %s", sub)
		}
	case "message":
		switch sub {
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
		if sub != "query" {
			return fmt.Errorf("unknown command: search %s", sub)
		}
		plain := nonFlagArgs(args[1:])
		if len(plain) == 0 {
			return errors.New("usage: search query <text>")
		}
		body, err := s.fetchJSONQuery(ctx, "/admin/api/search", url.Values{"q": {strings.Join(plain, " ")}})
		if err != nil {
			return err
		}
		return writeJSON(s.out, body)
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
		default:
			return fmt.Errorf("unknown command: memory %s", sub)
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
		case "put":
			plain := nonFlagArgs(args[1:])
			if len(plain) < 3 {
				return errors.New("usage: store put <namespace> <key> <value>")
			}
			body, err := s.postFormJSON(ctx, "/admin/api/store/put", url.Values{
				"namespace": {plain[0]},
				"key":       {plain[1]},
				"value":     {strings.Join(plain[2:], " ")},
			})
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
			body, err := s.fetchJSON(ctx, "/admin/api/bus")
			if err != nil {
				return err
			}
			return writeJSON(s.out, body)
		default:
			return fmt.Errorf("unknown command: bus %s", sub)
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
			if len(args) < 2 {
				return fmt.Errorf("usage: app %s <app>", sub)
			}
			appName := strings.TrimSpace(args[1])
			if appName == "" {
				return fmt.Errorf("usage: app %s <app>", sub)
			}
			route := "/admin/api/apps/reload"
			if sub == "rollback" {
				route = "/admin/api/apps/rollback"
			}
			body, err := s.postFormJSON(ctx, route, url.Values{"app": {appName}})
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
		default:
			return fmt.Errorf("unknown command: ai %s", sub)
		}
	default:
		return fmt.Errorf("unsupported command group: %s", group)
	}
}

func (s *state) fetchJSON(ctx context.Context, route string) (map[string]any, error) {
	return s.doJSON(ctx, http.MethodGet, route, "", nil)
}

func (s *state) fetchJSONQuery(ctx context.Context, route string, query url.Values) (map[string]any, error) {
	base, err := url.JoinPath(s.adminURL, route)
	if err != nil {
		return nil, err
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	parsed.RawQuery = query.Encode()
	return s.doJSON(ctx, http.MethodGet, parsed.String(), "", nil)
}

func (s *state) deleteJSON(ctx context.Context, route string) (map[string]any, error) {
	return s.doJSON(ctx, http.MethodDelete, route, "", nil)
}

func (s *state) fetchTextQuery(ctx context.Context, route string, query url.Values) (string, error) {
	base, err := url.JoinPath(s.adminURL, route)
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	parsed.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return "", err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("admin request failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (s *state) postFormJSON(ctx context.Context, route string, form url.Values) (map[string]any, error) {
	if form == nil {
		form = url.Values{}
	}
	return s.doJSON(ctx, http.MethodPost, route, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
}

func (s *state) doJSON(ctx context.Context, method, route, contentType string, body io.Reader) (map[string]any, error) {
	u := strings.TrimSpace(route)
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		var err error
		u, err = url.JoinPath(s.adminURL, route)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(contentType) != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("admin request failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (s *state) printHelp(args []string) error {
	query := strings.TrimSpace(strings.ToLower(strings.Join(args, " ")))
	if query != "" {
		spec, ok := replCommandSpec(query)
		if !ok {
			_, err := fmt.Fprintf(s.out, "unknown command %q\n", query)
			return err
		}
		return s.renderCommandSpec(spec)
	}

	rows := make([][]string, 0, len(replCommandSpecs()))
	specs := replCommandSpecs()
	sort.Slice(specs, func(i, j int) bool { return specs[i].Name < specs[j].Name })
	for _, spec := range specs {
		rows = append(rows, []string{spec.Usage, string(spec.Classification), spec.Summary})
	}
	if err := printTable(s.out, []string{"COMMAND", "CLASS", "SUMMARY"}, rows); err != nil {
		return err
	}
	_, err := fmt.Fprintln(s.out, "Run `help <command>` or `describe <command>` for details.")
	return err
}

func (s *state) describeCommand(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: describe <command>")
	}
	query := strings.TrimSpace(strings.ToLower(strings.Join(args, " ")))
	spec, ok := replCommandSpec(query)
	if !ok {
		return fmt.Errorf("unknown command: %s", query)
	}
	return s.renderCommandSpec(spec)
}

func (s *state) completeCommand(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: complete <prefix>")
	}
	prefix := strings.ToLower(strings.Join(args, " "))
	matches := completeCommands(prefix, 32)
	if len(matches) == 0 {
		_, err := fmt.Fprintln(s.out, "(no completions)")
		return err
	}
	for _, match := range matches {
		if _, err := fmt.Fprintln(s.out, match); err != nil {
			return err
		}
	}
	return nil
}

func (s *state) renderCommandSpec(spec commandSpec) error {
	if _, err := fmt.Fprintf(s.out, "%s\n", spec.Usage); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(s.out, "classification: %s\n", spec.Classification); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(s.out, spec.Summary); err != nil {
		return err
	}
	for _, ex := range spec.Examples {
		if _, err := fmt.Fprintf(s.out, "example: %s\n", ex); err != nil {
			return err
		}
	}
	for _, ref := range spec.RelatedDocs {
		if _, err := fmt.Fprintf(s.out, "docs: %s\n", ref); err != nil {
			return err
		}
	}
	if spec.DiscouragedForAgents {
		if _, err := fmt.Fprintln(s.out, "discouraged_for_agents: true"); err != nil {
			return err
		}
	}
	return nil
}

func (s *state) queryLogs(ctx context.Context, appName, query string) error {
	params := url.Values{}
	if strings.TrimSpace(appName) != "" {
		params.Set("app", strings.TrimSpace(appName))
	}
	if strings.TrimSpace(query) != "" {
		params.Set("q", strings.TrimSpace(query))
	}
	body, err := s.fetchTextQuery(ctx, "/admin/logs.jsonl", params)
	if err != nil {
		return err
	}
	body = strings.TrimSpace(body)
	if body == "" {
		_, err := fmt.Fprintln(s.out, "(no log records)")
		return err
	}
	_, err = fmt.Fprintln(s.out, body)
	return err
}

func formatUnixMillis(raw any) string {
	switch typed := raw.(type) {
	case float64:
		if typed <= 0 {
			return ""
		}
		return time.UnixMilli(int64(typed)).UTC().Format(time.RFC3339)
	case int64:
		if typed <= 0 {
			return ""
		}
		return time.UnixMilli(typed).UTC().Format(time.RFC3339)
	case json.Number:
		n, err := typed.Int64()
		if err != nil || n <= 0 {
			return ""
		}
		return time.UnixMilli(n).UTC().Format(time.RFC3339)
	case string:
		if strings.TrimSpace(typed) == "" {
			return ""
		}
		if parsed, err := time.Parse(time.RFC3339Nano, typed); err == nil {
			return parsed.UTC().Format(time.RFC3339)
		}
		if parsed, err := time.Parse(time.RFC3339, typed); err == nil {
			return parsed.UTC().Format(time.RFC3339)
		}
		return typed
	default:
		return ""
	}
}

func listDocTopics(root string) ([]string, error) {
	out := make([]string, 0, 32)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = strings.TrimSuffix(filepath.ToSlash(rel), ".md")
		if rel == "index" || rel == "." {
			out = append(out, "repl/index")
			return nil
		}
		out = append(out, "repl/"+rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

func searchDocTopics(root, query string) ([]string, error) {
	if query == "" {
		return listDocTopics(root)
	}
	out := make([]string, 0, 16)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		topic := "repl/" + strings.TrimSuffix(filepath.ToSlash(rel), ".md")
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(strings.ToLower(topic), query) || strings.Contains(strings.ToLower(string(content)), query) {
			out = append(out, topic)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

func resolveDocTopicPath(root, topic string) string {
	topic = strings.TrimSpace(topic)
	topic = strings.TrimPrefix(topic, "repl/")
	topic = strings.TrimSuffix(topic, ".md")
	if topic == "" || topic == "repl" {
		topic = "index"
	}
	return filepath.Join(root, filepath.FromSlash(topic)+".md")
}

func splitSegments(line string) []string {
	segments := make([]string, 0)
	var b strings.Builder
	inSingle := false
	inDouble := false
	for _, r := range line {
		switch r {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
			b.WriteRune(r)
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
			b.WriteRune(r)
		case ';':
			if inSingle || inDouble {
				b.WriteRune(r)
				continue
			}
			segment := strings.TrimSpace(b.String())
			if segment != "" {
				segments = append(segments, segment)
			}
			b.Reset()
		default:
			b.WriteRune(r)
		}
	}
	if tail := strings.TrimSpace(b.String()); tail != "" {
		segments = append(segments, tail)
	}
	return segments
}

func tokenize(line string) []string {
	if strings.TrimSpace(line) == "" {
		return nil
	}
	tokens := make([]string, 0)
	var b strings.Builder
	inSingle := false
	inDouble := false
	escaped := false
	flush := func() {
		if b.Len() == 0 {
			return
		}
		tokens = append(tokens, b.String())
		b.Reset()
	}
	for _, r := range line {
		switch {
		case escaped:
			b.WriteRune(r)
			escaped = false
		case r == '\\' && inDouble:
			escaped = true
		case r == '\'' && !inDouble:
			inSingle = !inSingle
		case r == '"' && !inSingle:
			inDouble = !inDouble
		case (r == ' ' || r == '\t') && !inSingle && !inDouble:
			flush()
		default:
			b.WriteRune(r)
		}
	}
	flush()
	for i := range tokens {
		tokens[i] = strings.TrimSpace(tokens[i])
	}
	return tokens
}

func hasFlag(args []string, name string) bool {
	for _, arg := range args {
		if strings.EqualFold(strings.TrimSpace(arg), name) {
			return true
		}
	}
	return false
}

func nonFlagArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		trimmed := strings.TrimSpace(arg)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func decodeEscapes(in string) string {
	quoted := strconv.Quote(in)
	decoded, err := strconv.Unquote(quoted)
	if err != nil {
		return in
	}
	decoded, err = strconv.Unquote("\"" + strings.ReplaceAll(decoded, "\"", "\\\"") + "\"")
	if err != nil {
		return decoded
	}
	return decoded
}

func writeJSON(out io.Writer, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(b))
	return err
}

func toString(v any) string {
	switch typed := v.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func lookupMapAny(m map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			return value
		}
	}
	return nil
}

func toAnySlice(v any) []any {
	switch typed := v.(type) {
	case []any:
		return typed
	case []string:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, item)
		}
		return out
	default:
		return nil
	}
}

func printTable(out io.Writer, headers []string, rows [][]string) error {
	if len(headers) == 0 {
		return nil
	}
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i := range headers {
			if i >= len(row) {
				continue
			}
			if len(row[i]) > widths[i] {
				widths[i] = len(row[i])
			}
		}
	}
	var line bytes.Buffer
	for i, h := range headers {
		if i > 0 {
			line.WriteString("  ")
		}
		line.WriteString(padRight(h, widths[i]))
	}
	if _, err := fmt.Fprintln(out, line.String()); err != nil {
		return err
	}
	for _, row := range rows {
		line.Reset()
		for i := range headers {
			if i > 0 {
				line.WriteString("  ")
			}
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			line.WriteString(padRight(cell, widths[i]))
		}
		if _, err := fmt.Fprintln(out, line.String()); err != nil {
			return err
		}
	}
	return nil
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func completeCommands(input string, limit int) []string {
	input = strings.ToLower(input)
	trailingSpace := strings.HasSuffix(input, " ")
	inputTokens := strings.Fields(input)
	if limit <= 0 {
		limit = 1
	}
	matches := make([]string, 0, len(replCommandSpecs()))
	for _, spec := range replCommandSpecs() {
		if commandMatchesPrefix(spec.Name, inputTokens, trailingSpace) {
			matches = append(matches, spec.Name)
		}
	}
	sort.Strings(matches)
	if len(matches) > limit {
		return matches[:limit]
	}
	return matches
}

func commandMatchesPrefix(commandName string, inputTokens []string, trailingSpace bool) bool {
	if len(inputTokens) == 0 {
		return true
	}
	commandTokens := strings.Fields(strings.ToLower(strings.TrimSpace(commandName)))
	if len(commandTokens) == 0 {
		return false
	}
	if trailingSpace {
		if len(inputTokens) >= len(commandTokens) {
			return false
		}
		for i, token := range inputTokens {
			if commandTokens[i] != token {
				return false
			}
		}
		return true
	}
	if len(inputTokens) > len(commandTokens) {
		return false
	}
	for i := 0; i < len(inputTokens)-1; i++ {
		if commandTokens[i] != inputTokens[i] {
			return false
		}
	}
	last := len(inputTokens) - 1
	return strings.HasPrefix(commandTokens[last], inputTokens[last])
}

func suggestApproxCommands(input string, limit int) []string {
	input = strings.TrimSpace(strings.ToLower(input))
	if limit <= 0 {
		limit = 1
	}
	type candidate struct {
		name  string
		score int
	}
	candidates := make([]candidate, 0, len(replCommandSpecs()))
	for _, spec := range replCommandSpecs() {
		name := strings.ToLower(spec.Name)
		score := 1000 + editDistance(input, name)
		switch {
		case input == "":
			score = 0
		case strings.HasPrefix(name, input):
			score = 0
		case strings.Contains(name, input):
			score = 1
		}
		candidates = append(candidates, candidate{name: spec.Name, score: score})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return candidates[i].name < candidates[j].name
		}
		return candidates[i].score < candidates[j].score
	})
	out := make([]string, 0, limit)
	for _, cand := range candidates {
		if len(out) >= limit {
			break
		}
		out = append(out, cand.name)
	}
	return out
}

var (
	docsRootOnce sync.Once
	docsRootPath string
)

func resolveDocsRoot() string {
	docsRootOnce.Do(func() {
		docsRootPath = discoverDocsRoot()
	})
	return docsRootPath
}

func discoverDocsRoot() string {
	envRoot := strings.TrimSpace(os.Getenv("TERMINALS_REPL_DOCS_ROOT"))
	if envRoot != "" {
		if dirExists(filepath.Join(envRoot, "docs", "repl")) {
			return filepath.Join(envRoot, "docs", "repl")
		}
		if dirExists(envRoot) && strings.HasSuffix(filepath.ToSlash(envRoot), "/docs/repl") {
			return envRoot
		}
	}

	if cwd, err := os.Getwd(); err == nil {
		if found := findDocsRootFrom(cwd); found != "" {
			return found
		}
	}
	if _, sourceFile, _, ok := runtime.Caller(0); ok {
		if found := findDocsRootFrom(filepath.Dir(sourceFile)); found != "" {
			return found
		}
	}
	return filepath.Join("docs", "repl")
}

func findDocsRootFrom(start string) string {
	dir := filepath.Clean(strings.TrimSpace(start))
	if dir == "" {
		return ""
	}
	for {
		candidate := filepath.Join(dir, "docs", "repl")
		if dirExists(candidate) {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func editDistance(a, b string) int {
	ar := []rune(a)
	br := []rune(b)
	if len(ar) == 0 {
		return len(br)
	}
	if len(br) == 0 {
		return len(ar)
	}
	dp := make([][]int, len(ar)+1)
	for i := range dp {
		dp[i] = make([]int, len(br)+1)
		dp[i][0] = i
	}
	for j := 0; j <= len(br); j++ {
		dp[0][j] = j
	}
	for i := 1; i <= len(ar); i++ {
		for j := 1; j <= len(br); j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			del := dp[i-1][j] + 1
			ins := dp[i][j-1] + 1
			sub := dp[i-1][j-1] + cost
			dp[i][j] = minInt(del, ins, sub)
		}
	}
	return dp[len(ar)][len(br)]
}

func minInt(vals ...int) int {
	out := vals[0]
	for _, v := range vals[1:] {
		if v < out {
			out = v
		}
	}
	return out
}
