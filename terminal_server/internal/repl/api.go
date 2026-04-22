package repl

import (
	"bytes"
	"context"
	"io"
	"strings"
)

// CommandClassification describes the mutation level of a REPL command.
type CommandClassification string

// CommandClassification values.
const (
	CommandClassificationReadOnly    CommandClassification = "read_only"
	CommandClassificationOperational CommandClassification = "operational"
	CommandClassificationMutating    CommandClassification = "mutating"
)

// DocsRenderMode controls how docs commands are rendered.
type DocsRenderMode string

// DocsRenderMode values.
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
	return ExecuteCommandStream(ctx, line, opts, nil)
}

// ExecuteCommandStream runs a REPL command and emits output chunks as they are
// written when onChunk is provided.
func ExecuteCommandStream(ctx context.Context, line string, opts ExecuteOptions, onChunk func(string) error) (ExecuteResult, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return ExecuteResult{}, nil
	}
	var out bytes.Buffer
	writer := io.Writer(&out)
	if onChunk != nil {
		writer = &chunkWriter{
			base: &out,
			emit: onChunk,
		}
	}
	st := newStateWithDocsMode(writer, opts.AdminBaseURL, opts.SessionID, opts.DocsMode)
	_, err := st.eval(ctx, line)
	return ExecuteResult{Output: out.String()}, err
}

type chunkWriter struct {
	base io.Writer
	emit func(string) error
}

func (w *chunkWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if _, err := w.base.Write(p); err != nil {
		return 0, err
	}
	if w.emit != nil {
		if err := w.emit(string(p)); err != nil {
			return 0, err
		}
	}
	return len(p), nil
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
	return completeCommands(prefix, limit)
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
