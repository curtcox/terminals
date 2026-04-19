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
	"sort"
	"strconv"
	"strings"
	"time"
)

// Options configures a REPL run.
type Options struct {
	Prompt       string
	AdminBaseURL string
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

	state := newState(out, opts.AdminBaseURL)
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 1024), 1024*1024)

	fmt.Fprintf(out, "Terminals REPL (control-plane only). Type 'help' for commands.\n")
	fmt.Fprint(out, prompt)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			fmt.Fprint(out, prompt)
			continue
		}
		exit, err := state.eval(ctx, line)
		if err != nil {
			fmt.Fprintf(out, "error: %v\n", err)
		}
		if exit {
			return nil
		}
		fmt.Fprint(out, prompt)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

type state struct {
	out      io.Writer
	adminURL string
	client   *http.Client
}

func newState(out io.Writer, adminBaseURL string) *state {
	adminBaseURL = strings.TrimSpace(adminBaseURL)
	if adminBaseURL == "" {
		adminBaseURL = strings.TrimSpace(os.Getenv("TERMINALS_REPL_ADMIN_URL"))
	}
	if adminBaseURL == "" {
		adminBaseURL = "http://127.0.0.1:50053"
	}
	adminBaseURL = strings.TrimSuffix(adminBaseURL, "/")
	return &state{
		out:      out,
		adminURL: adminBaseURL,
		client:   &http.Client{Timeout: 3 * time.Second},
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
		s.printHelp()
		return false, nil
	case "echo":
		fmt.Fprintln(s.out, strings.Join(tokens[1:], " "))
		return false, nil
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
		fmt.Fprint(s.out, text)
		return false, nil
	case "clear":
		fmt.Fprint(s.out, "\033[2J\033[H")
		return false, nil
	case "exit", "quit":
		fmt.Fprintln(s.out, "bye")
		return true, nil
	case "devices", "sessions", "activations", "claims", "app", "config", "docs":
		return false, s.evalControlPlane(ctx, cmd, tokens[1:])
	default:
		return false, fmt.Errorf("unknown command: %s", tokens[0])
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
				attached, _ := row["attached_devices"].([]any)
				rows = append(rows, []string{
					toString(row["id"]),
					toString(row["owner_activation_id"]),
					strconv.Itoa(len(attached)),
					toString(row["idle"]),
					formatUnixMillis(row["created_at"]),
				})
			}
			return printTable(s.out, []string{"ID", "OWNER", "ATTACHED", "IDLE", "CREATED"}, rows)
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
			session, _ := body["session"]
			return writeJSON(s.out, session)
		default:
			return fmt.Errorf("unknown command: sessions %s", sub)
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
			fmt.Fprintln(s.out, "(no claims)")
			return nil
		}
		deviceIDs := make([]string, 0, len(claimsByDevice))
		for deviceID := range claimsByDevice {
			deviceIDs = append(deviceIDs, deviceID)
		}
		sort.Strings(deviceIDs)
		for _, deviceID := range deviceIDs {
			fmt.Fprintf(s.out, "%s\n", deviceID)
			claims, _ := claimsByDevice[deviceID].([]any)
			if len(claims) == 0 {
				fmt.Fprintln(s.out, "  (none)")
				continue
			}
			for _, claimAny := range claims {
				claim, _ := claimAny.(map[string]any)
				if claim == nil {
					continue
				}
				fmt.Fprintf(s.out, "  - %s by %s\n", toString(claim["resource"]), toString(claim["activation_id"]))
			}
		}
		return nil
	case "app":
		if sub != "ls" {
			return fmt.Errorf("unknown command: app %s", sub)
		}
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
	case "config":
		if sub != "show" {
			return fmt.Errorf("unknown command: config %s", sub)
		}
		body, err := s.fetchJSON(ctx, "/admin/api/status")
		if err != nil {
			return err
		}
		cfg, _ := body["config"]
		if jsonOut {
			return writeJSON(s.out, cfg)
		}
		return writeJSON(s.out, cfg)
	case "docs":
		switch sub {
		case "ls":
			topics, err := listDocTopics("docs/repl")
			if err != nil {
				return err
			}
			for _, topic := range topics {
				fmt.Fprintln(s.out, topic)
			}
			return nil
		case "search":
			if len(args) < 2 {
				return errors.New("usage: docs search <query>")
			}
			query := strings.ToLower(strings.TrimSpace(strings.Join(args[1:], " ")))
			matches, err := searchDocTopics("docs/repl", query)
			if err != nil {
				return err
			}
			fmt.Fprintf(s.out, "search results for %q\n", strings.Join(args[1:], " "))
			if len(matches) == 0 {
				fmt.Fprintln(s.out, "(no matches)")
				return nil
			}
			for _, topic := range matches {
				fmt.Fprintf(s.out, "- %s\n", topic)
			}
			return nil
		case "open":
			if len(args) < 2 {
				return errors.New("usage: docs open <topic>")
			}
			topic := strings.TrimSpace(strings.Join(args[1:], " "))
			path := resolveDocTopicPath("docs/repl", topic)
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			fmt.Fprintln(s.out, string(content))
			return nil
		case "examples":
			filter := ""
			if len(args) > 1 {
				filter = strings.ToLower(strings.Join(args[1:], " "))
			}
			topics, err := listDocTopics(filepath.Join("docs/repl", "examples"))
			if err != nil {
				return err
			}
			for _, topic := range topics {
				if filter == "" || strings.Contains(strings.ToLower(topic), filter) {
					fmt.Fprintln(s.out, topic)
				}
			}
			return nil
		default:
			return fmt.Errorf("unknown command: docs %s", sub)
		}
	default:
		return fmt.Errorf("unsupported command group: %s", group)
	}
}

func (s *state) fetchJSON(ctx context.Context, route string) (map[string]any, error) {
	u, err := url.JoinPath(s.adminURL, route)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
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

func (s *state) printHelp() {
	fmt.Fprintln(s.out, "help                         Show this help")
	fmt.Fprintln(s.out, "echo <text>                  Print text")
	fmt.Fprintln(s.out, "sleep <seconds>              Sleep for N seconds")
	fmt.Fprintln(s.out, "printf <text>                Print text without newline (supports \\xNN escapes)")
	fmt.Fprintln(s.out, "devices ls [--json]          List devices")
	fmt.Fprintln(s.out, "sessions ls [--json]         List REPL sessions")
	fmt.Fprintln(s.out, "sessions show <session>      Show one REPL session")
	fmt.Fprintln(s.out, "activations ls [--json]      List active scenario by device")
	fmt.Fprintln(s.out, "claims tree [--json]         Show claims grouped by device")
	fmt.Fprintln(s.out, "app ls [--json]              List loaded apps")
	fmt.Fprintln(s.out, "config show [--json]         Show effective config")
	fmt.Fprintln(s.out, "docs <ls|search|open|examples>")
	fmt.Fprintln(s.out, "clear                        Clear terminal")
	fmt.Fprintln(s.out, "exit                         Exit REPL")
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
	fmt.Fprintln(out, line.String())
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
		fmt.Fprintln(out, line.String())
	}
	return nil
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
