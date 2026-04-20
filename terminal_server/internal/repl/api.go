package repl

import (
	"bytes"
	"context"
	"strings"
)

// CommandClassification describes the mutation level of a REPL command.
type CommandClassification string

const (
	CommandClassificationReadOnly    CommandClassification = "read_only"
	CommandClassificationOperational CommandClassification = "operational"
	CommandClassificationMutating    CommandClassification = "mutating"
)

// DocsRenderMode controls how docs commands are rendered.
type DocsRenderMode string

const (
	DocsRenderModeTerminal DocsRenderMode = "terminal"
	DocsRenderModeMarkdown DocsRenderMode = "markdown"
)

// CommandSpec describes one registered REPL command.
type CommandSpec struct {
	Name                 string
	Usage                string
	Summary              string
	Classification       CommandClassification
	Examples             []string
	RelatedDocs          []string
	DiscouragedForAgents bool
}

// ExecuteOptions controls one-shot command execution.
type ExecuteOptions struct {
	AdminBaseURL string
	SessionID    string
	DocsMode     DocsRenderMode
}

// ExecuteResult returns captured output from one-shot execution.
type ExecuteResult struct {
	Output string
}

// ExecuteCommand runs a REPL command line without interactive prompting.
func ExecuteCommand(ctx context.Context, line string, opts ExecuteOptions) (ExecuteResult, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return ExecuteResult{}, nil
	}
	var out bytes.Buffer
	st := newStateWithDocsMode(&out, opts.AdminBaseURL, opts.SessionID, opts.DocsMode)
	_, err := st.eval(ctx, line)
	return ExecuteResult{Output: out.String()}, err
}

// CommandSpecs returns a stable snapshot of REPL command metadata.
func CommandSpecs() []CommandSpec {
	raw := replCommandSpecs()
	out := make([]CommandSpec, 0, len(raw))
	for _, spec := range raw {
		out = append(out, toPublicCommandSpec(spec))
	}
	return out
}

// DescribeCommand returns metadata for one command.
func DescribeCommand(command string) (CommandSpec, bool) {
	spec, ok := replCommandSpec(command)
	if !ok {
		return CommandSpec{}, false
	}
	return toPublicCommandSpec(spec), true
}

// Complete returns command completions for a prefix.
func Complete(prefix string, limit int) []string {
	return suggestCommands(prefix, limit)
}

func toPublicCommandSpec(spec commandSpec) CommandSpec {
	return CommandSpec{
		Name:                 spec.Name,
		Usage:                spec.Usage,
		Summary:              spec.Summary,
		Classification:       CommandClassification(spec.Classification),
		Examples:             append([]string(nil), spec.Examples...),
		RelatedDocs:          append([]string(nil), spec.RelatedDocs...),
		DiscouragedForAgents: spec.DiscouragedForAgents,
	}
}

func normalizeDocsRenderMode(mode DocsRenderMode) DocsRenderMode {
	if strings.EqualFold(strings.TrimSpace(string(mode)), string(DocsRenderModeMarkdown)) {
		return DocsRenderModeMarkdown
	}
	return DocsRenderModeTerminal
}
