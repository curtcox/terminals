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
	SessionID    string
}

type commandClassification string

const (
	commandReadOnly commandClassification = "read_only"
	commandMutating commandClassification = "mutating"
)

type commandSpec struct {
	Name           string
	Usage          string
	Summary        string
	Classification commandClassification
	Examples       []string
	RelatedDocs    []string
}

func replCommandSpecs() []commandSpec {
	return []commandSpec{
		{Name: "help", Usage: "help [command]", Summary: "Show REPL help or help for one command", Classification: commandReadOnly, Examples: []string{"help", "help app reload"}},
		{Name: "describe", Usage: "describe <command>", Summary: "Show a detailed command description", Classification: commandReadOnly, Examples: []string{"describe sessions terminate"}},
		{Name: "complete", Usage: "complete <prefix>", Summary: "List command completions for a prefix", Classification: commandReadOnly, Examples: []string{"complete app r"}},
		{Name: "echo", Usage: "echo <text>", Summary: "Print text", Classification: commandReadOnly},
		{Name: "sleep", Usage: "sleep <seconds>", Summary: "Sleep for N seconds", Classification: commandReadOnly},
		{Name: "printf", Usage: "printf <text>", Summary: "Print text without newline (supports \\xNN escapes)", Classification: commandReadOnly},
		{Name: "clear", Usage: "clear", Summary: "Clear terminal display", Classification: commandReadOnly},
		{Name: "exit", Usage: "exit", Summary: "Exit REPL", Classification: commandReadOnly},
		{Name: "devices ls", Usage: "devices ls [--json]", Summary: "List devices", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/devices"}},
		{Name: "sessions ls", Usage: "sessions ls [--json]", Summary: "List REPL sessions", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/sessions"}},
		{Name: "sessions show", Usage: "sessions show <session>", Summary: "Show one REPL session", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/sessions"}},
		{Name: "sessions terminate", Usage: "sessions terminate <session>", Summary: "Terminate one REPL session", Classification: commandMutating, RelatedDocs: []string{"repl/commands/sessions"}},
		{Name: "activations ls", Usage: "activations ls [--json]", Summary: "List active scenario by device", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/activations"}},
		{Name: "claims tree", Usage: "claims tree [--json]", Summary: "Show claims grouped by device", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/claims"}},
		{Name: "app ls", Usage: "app ls [--json]", Summary: "List loaded apps", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/app"}},
		{Name: "app reload", Usage: "app reload <app> [--json]", Summary: "Reload an app package", Classification: commandMutating, RelatedDocs: []string{"repl/commands/app"}},
		{Name: "app rollback", Usage: "app rollback <app> [--json]", Summary: "Rollback an app package", Classification: commandMutating, RelatedDocs: []string{"repl/commands/app"}},
		{Name: "config show", Usage: "config show [--json]", Summary: "Show effective config", Classification: commandReadOnly},
		{Name: "docs ls", Usage: "docs ls", Summary: "List documentation topics", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/docs"}},
		{Name: "docs search", Usage: "docs search <query>", Summary: "Search documentation topics", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/docs"}},
		{Name: "docs open", Usage: "docs open <topic>", Summary: "Open one documentation topic", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/docs"}},
		{Name: "docs examples", Usage: "docs examples [filter]", Summary: "List example topics", Classification: commandReadOnly, RelatedDocs: []string{"repl/examples/app-dev-loop"}},
		{Name: "ai providers", Usage: "ai providers [--json]", Summary: "List configured AI providers", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/ai"}},
		{Name: "ai models", Usage: "ai models [provider] [--json]", Summary: "List models for a provider", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/ai"}},
		{Name: "ai use", Usage: "ai use <provider> <model> [--json]", Summary: "Set sticky provider/model selection for this session", Classification: commandMutating, RelatedDocs: []string{"repl/commands/ai"}},
		{Name: "ai status", Usage: "ai status [--json]", Summary: "Show current provider/model selection for this session", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/ai"}},
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

	state := newState(out, opts.AdminBaseURL, opts.SessionID)
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
	session  string
	client   *http.Client
}

func newState(out io.Writer, adminBaseURL, sessionID string) *state {
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
		s.printHelp(tokens[1:])
		return false, nil
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
	case "devices", "sessions", "activations", "claims", "app", "config", "docs", "ai":
		return false, s.evalControlPlane(ctx, cmd, tokens[1:])
	default:
		input := strings.ToLower(strings.TrimSpace(strings.Join(tokens, " ")))
		suggestions := suggestCommands(input, 3)
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
			fmt.Fprintf(s.out, "OK  terminated session %s\n", sessionID)
			return nil
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
			fmt.Fprintf(s.out, "OK  app=%s action=%s version=%s\n", appName, sub, toString(body["version"]))
			return nil
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
				fmt.Fprintln(s.out, toString(model))
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
			fmt.Fprintf(s.out, "provider: %s  model: %s (sticky for %s)\n", toString(body["provider"]), toString(body["model"]), s.session)
			return nil
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
			fmt.Fprintf(s.out, "session: %s\nprovider: %s\nmodel: %s\n", toString(body["session_id"]), toString(body["provider"]), toString(body["model"]))
			return nil
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

func (s *state) printHelp(args []string) {
	query := strings.TrimSpace(strings.ToLower(strings.Join(args, " ")))
	if query != "" {
		spec, ok := replCommandSpec(query)
		if !ok {
			fmt.Fprintf(s.out, "unknown command %q\n", query)
			return
		}
		s.renderCommandSpec(spec)
		return
	}

	rows := make([][]string, 0, len(replCommandSpecs()))
	specs := replCommandSpecs()
	sort.Slice(specs, func(i, j int) bool { return specs[i].Name < specs[j].Name })
	for _, spec := range specs {
		rows = append(rows, []string{spec.Usage, string(spec.Classification), spec.Summary})
	}
	_ = printTable(s.out, []string{"COMMAND", "CLASS", "SUMMARY"}, rows)
	fmt.Fprintln(s.out, "Run `help <command>` or `describe <command>` for details.")
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
	s.renderCommandSpec(spec)
	return nil
}

func (s *state) completeCommand(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: complete <prefix>")
	}
	prefix := strings.TrimSpace(strings.ToLower(strings.Join(args, " ")))
	matches := suggestCommands(prefix, 32)
	if len(matches) == 0 {
		fmt.Fprintln(s.out, "(no completions)")
		return nil
	}
	for _, match := range matches {
		fmt.Fprintln(s.out, match)
	}
	return nil
}

func (s *state) renderCommandSpec(spec commandSpec) {
	fmt.Fprintf(s.out, "%s\n", spec.Usage)
	fmt.Fprintf(s.out, "classification: %s\n", spec.Classification)
	fmt.Fprintln(s.out, spec.Summary)
	for _, ex := range spec.Examples {
		fmt.Fprintf(s.out, "example: %s\n", ex)
	}
	for _, ref := range spec.RelatedDocs {
		fmt.Fprintf(s.out, "docs: %s\n", ref)
	}
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

func suggestCommands(input string, limit int) []string {
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
		if input == "" {
			score = 0
		} else if strings.HasPrefix(name, input) {
			score = 0
		} else if strings.Contains(name, input) {
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
