// Package repl implements the control-plane REPL used by terminal sessions.
package repl

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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
	DocsMode     DocsRenderMode
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
	pending  *pendingAIProposal
}

type pendingAIProposal struct {
	Command string
	Summary string
	Source  string
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
	case "devices", "sessions", "identity", "session", "message", "board", "artifact", "canvas", "search", "memory", "bug", "placement", "cohort", "ui", "recent", "store", "bus", "handlers", "scenarios", "sim", "scripts", "activations", "claims", "app", "apps", "config", "docs", "logs", "observe", "ai":
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

func (s *state) capturePendingAIProposal(body map[string]any) {
	command := strings.TrimSpace(toString(lookupMapAny(body,
		"proposed_command",
		"pending_command",
		"proposal_command",
		"approval_command",
	)))
	if command == "" {
		s.pending = nil
		return
	}
	summary := strings.TrimSpace(toString(lookupMapAny(body,
		"proposal_summary",
		"pending_summary",
		"approval_summary",
	)))
	source := strings.TrimSpace(toString(lookupMapAny(body,
		"proposal_source",
		"pending_source",
	)))
	if source == "" || source == "<nil>" {
		source = "ai"
	}
	s.pending = &pendingAIProposal{Command: command, Summary: summary, Source: source}
	_, _ = fmt.Fprintf(s.out, "pending proposal (%s): %s\n", source, command)
	if summary != "" {
		_, _ = fmt.Fprintf(s.out, "summary: %s\n", summary)
	}
	_, _ = fmt.Fprintln(s.out, "approve? (ai approve / ai run / ai reject)")
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

