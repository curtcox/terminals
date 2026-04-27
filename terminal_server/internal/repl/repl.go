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
	commandReadOnly         commandClassification = "read_only"
	commandOperational      commandClassification = "operational"
	commandMutating         commandClassification = "mutating"
	commandCriticalMutating commandClassification = "critical_mutating"
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
		{Name: "identity show", Usage: "identity show <identity> [--json]", Summary: "Show one identity", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/identity"}},
		{Name: "identity groups", Usage: "identity groups [--json]", Summary: "List identity groups", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/identity"}},
		{Name: "identity resolve", Usage: "identity resolve <audience> [--json]", Summary: "Resolve an audience to identities", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/identity"}},
		{Name: "identity prefs", Usage: "identity prefs <identity> [--json]", Summary: "Show identity preferences", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/identity"}},
		{Name: "identity ack ls", Usage: "identity ack ls [subject-ref] [--json]", Summary: "List acknowledgements", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/identity"}},
		{Name: "identity ack show", Usage: "identity ack show <subject-ref> [--json]", Summary: "Show acknowledgements for one subject", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/identity"}},
		{Name: "identity ack record", Usage: "identity ack record <subject-ref> --actor <actor-ref> [--mode <mode>] [--json]", Summary: "Record an acknowledgement", Classification: commandMutating, RelatedDocs: []string{"repl/commands/identity"}},
		{Name: "session ls", Usage: "session ls [--json]", Summary: "List interactive sessions", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/session"}},
		{Name: "session create", Usage: "session create <kind> <target> [--json]", Summary: "Create a generalized interactive session", Classification: commandMutating, RelatedDocs: []string{"repl/commands/session"}},
		{Name: "session show", Usage: "session show <session> [--json]", Summary: "Show one interactive session", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/session"}},
		{Name: "session members", Usage: "session members <session> [--json]", Summary: "List interactive session participants", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/session"}},
		{Name: "session join", Usage: "session join <session> <participant> [--json]", Summary: "Join a participant to an interactive session", Classification: commandMutating, RelatedDocs: []string{"repl/commands/session"}},
		{Name: "session leave", Usage: "session leave <session> <participant> [--json]", Summary: "Remove a participant from an interactive session", Classification: commandMutating, RelatedDocs: []string{"repl/commands/session"}},
		{Name: "session attach", Usage: "session attach <session> <device-ref> [--json]", Summary: "Attach a device to an interactive session", Classification: commandMutating, RelatedDocs: []string{"repl/commands/session"}},
		{Name: "session detach", Usage: "session detach <session> <device-ref> [--json]", Summary: "Detach a device from an interactive session", Classification: commandMutating, RelatedDocs: []string{"repl/commands/session"}},
		{Name: "session control request", Usage: "session control request <session> <participant> [control-type] [--json]", Summary: "Request control for one participant", Classification: commandMutating, RelatedDocs: []string{"repl/commands/session"}},
		{Name: "session control grant", Usage: "session control grant <session> <participant> [granted-by] [control-type] [--json]", Summary: "Grant control to one participant", Classification: commandMutating, RelatedDocs: []string{"repl/commands/session"}},
		{Name: "session control revoke", Usage: "session control revoke <session> <participant> [revoked-by] [--json]", Summary: "Revoke control from one participant", Classification: commandMutating, RelatedDocs: []string{"repl/commands/session"}},
		{Name: "message rooms", Usage: "message rooms [--json]", Summary: "List durable message rooms", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/message"}},
		{Name: "message room new", Usage: "message room new <name> [--json]", Summary: "Create a durable message room", Classification: commandMutating, RelatedDocs: []string{"repl/commands/message"}},
		{Name: "message room show", Usage: "message room show <room> [--json]", Summary: "Show one durable message room", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/message"}},
		{Name: "message ls", Usage: "message ls [room] [--json]", Summary: "List messages", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/message"}},
		{Name: "message get", Usage: "message get <message> [--json]", Summary: "Show one message", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/message"}},
		{Name: "message unread", Usage: "message unread <identity> [room] [--json]", Summary: "List unread messages for an identity", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/message"}},
		{Name: "message post", Usage: "message post <room> <text> [--json]", Summary: "Post a room/direct message", Classification: commandMutating, RelatedDocs: []string{"repl/commands/message"}},
		{Name: "message dm", Usage: "message dm <target> <text> [--json]", Summary: "Send a direct message", Classification: commandMutating, RelatedDocs: []string{"repl/commands/message"}},
		{Name: "message thread", Usage: "message thread <root-message> <text> [--json]", Summary: "Reply in a message thread", Classification: commandMutating, RelatedDocs: []string{"repl/commands/message"}},
		{Name: "message ack", Usage: "message ack <identity> <message> [--json]", Summary: "Acknowledge a message for an identity", Classification: commandMutating, RelatedDocs: []string{"repl/commands/message"}},
		{Name: "board ls", Usage: "board ls [board] [--json]", Summary: "List board or bulletin entries", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/board"}},
		{Name: "board post", Usage: "board post <board> <text> [--json]", Summary: "Post a board entry", Classification: commandMutating, RelatedDocs: []string{"repl/commands/board"}},
		{Name: "board pin", Usage: "board pin <board> <text> [--json]", Summary: "Pin a bulletin entry", Classification: commandMutating, RelatedDocs: []string{"repl/commands/board"}},
		{Name: "artifact ls", Usage: "artifact ls [--json]", Summary: "List shared artifacts", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/artifact"}},
		{Name: "artifact show", Usage: "artifact show <artifact> [--json]", Summary: "Show one shared artifact", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/artifact"}},
		{Name: "artifact history", Usage: "artifact history <artifact> [--json]", Summary: "Show version history for one artifact", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/artifact"}},
		{Name: "artifact create", Usage: "artifact create <kind> <title> [--json]", Summary: "Create a shared artifact", Classification: commandMutating, RelatedDocs: []string{"repl/commands/artifact"}},
		{Name: "artifact patch", Usage: "artifact patch <artifact> <title> [--json]", Summary: "Patch shared artifact metadata", Classification: commandMutating, RelatedDocs: []string{"repl/commands/artifact"}},
		{Name: "artifact replace", Usage: "artifact replace <artifact> <title> [--json]", Summary: "Replace shared artifact content metadata", Classification: commandMutating, RelatedDocs: []string{"repl/commands/artifact"}},
		{Name: "artifact template save", Usage: "artifact template save <name> <source-artifact> [--json]", Summary: "Save a reusable artifact template", Classification: commandMutating, RelatedDocs: []string{"repl/commands/artifact"}},
		{Name: "artifact template apply", Usage: "artifact template apply <name> <target-artifact> [--json]", Summary: "Apply a saved artifact template to a target", Classification: commandMutating, RelatedDocs: []string{"repl/commands/artifact"}},
		{Name: "canvas ls", Usage: "canvas ls [canvas] [--json]", Summary: "List canvas annotations", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/canvas"}},
		{Name: "canvas annotate", Usage: "canvas annotate <canvas> <text> [--json]", Summary: "Annotate a shared canvas", Classification: commandMutating, RelatedDocs: []string{"repl/commands/canvas"}},
		{Name: "search query", Usage: "search query <text> [--json]", Summary: "Run unified search", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/search"}},
		{Name: "search timeline", Usage: "search timeline [scope] [--json]", Summary: "View timeline-oriented search results", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/search"}},
		{Name: "search related", Usage: "search related <subject-ref> [--json]", Summary: "Find related indexed content", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/search"}},
		{Name: "search recent", Usage: "search recent [scope] [--json]", Summary: "List recent search-visible activity", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/search"}},
		{Name: "memory remember", Usage: "memory remember <scope> <text> [--json]", Summary: "Store a memory entry", Classification: commandMutating, RelatedDocs: []string{"repl/commands/memory"}},
		{Name: "memory recall", Usage: "memory recall <text> [--json]", Summary: "Recall memory entries", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/memory"}},
		{Name: "memory stream", Usage: "memory stream [scope] [--json]", Summary: "Show memory stream entries", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/memory"}},
		{Name: "bug ls", Usage: "bug ls [--json]", Summary: "List bug reports", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/bug"}},
		{Name: "bug show", Usage: "bug show <report-id> [--json]", Summary: "Show one bug report", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/bug"}},
		{Name: "bug file", Usage: "bug file <reporter-device-id> <subject-device-id> <description> [--source <source>] [--tags <tag[,tag...]>] [--json]", Summary: "File a bug report", Classification: commandMutating, RelatedDocs: []string{"repl/commands/bug"}},
		{Name: "bug confirm", Usage: "bug confirm <report-id> [--json]", Summary: "Confirm one bug report", Classification: commandMutating, RelatedDocs: []string{"repl/commands/bug"}},
		{Name: "bug tail", Usage: "bug tail [<query>]", Summary: "Tail bug-reporting control-plane logs", Classification: commandOperational, RelatedDocs: []string{"repl/commands/bug"}},
		{Name: "placement ls", Usage: "placement ls [--json]", Summary: "List placement metadata", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/placement"}},
		{Name: "cohort ls", Usage: "cohort ls [--json]", Summary: "List named device cohorts", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/cohort"}},
		{Name: "cohort show", Usage: "cohort show <name> [--json]", Summary: "Show one cohort with resolved members", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/cohort"}},
		{Name: "cohort put", Usage: "cohort put <name> --selectors <selector[,selector...]> [--json]", Summary: "Create or update a named device cohort", Classification: commandMutating, RelatedDocs: []string{"repl/commands/cohort"}},
		{Name: "cohort del", Usage: "cohort del <name> [--json]", Summary: "Delete a named device cohort", Classification: commandMutating, RelatedDocs: []string{"repl/commands/cohort"}},
		{Name: "ui push", Usage: "ui push <device> <descriptor-expr> [--root <id>] [--json]", Summary: "Push an authored UI descriptor to a device", Classification: commandMutating, RelatedDocs: []string{"repl/commands/ui"}},
		{Name: "ui patch", Usage: "ui patch <device> <component-id> <descriptor-expr> [--json]", Summary: "Patch one UI component on a device", Classification: commandMutating, RelatedDocs: []string{"repl/commands/ui"}},
		{Name: "ui transition", Usage: "ui transition <device> <component-id> <transition> [--duration-ms <n>] [--json]", Summary: "Apply a UI transition hint on a device", Classification: commandMutating, RelatedDocs: []string{"repl/commands/ui"}},
		{Name: "ui broadcast", Usage: "ui broadcast <cohort> <descriptor-expr> [--patch <component-id>] [--json]", Summary: "Broadcast a UI descriptor to all cohort members", Classification: commandMutating, RelatedDocs: []string{"repl/commands/ui"}},
		{Name: "ui subscribe", Usage: "ui subscribe <device> --to <activation|cohort> [--json]", Summary: "Subscribe a device to a cohort or activation UI stream", Classification: commandMutating, RelatedDocs: []string{"repl/commands/ui"}},
		{Name: "ui snapshot", Usage: "ui snapshot <device> [--json]", Summary: "Show current authored UI snapshot for a device", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/ui"}},
		{Name: "ui views ls", Usage: "ui views ls [--json]", Summary: "List authored UI views", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/ui"}},
		{Name: "ui views show", Usage: "ui views show <view-id> [--json]", Summary: "Show one authored UI view", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/ui"}},
		{Name: "ui views rm", Usage: "ui views rm <view-id> [--json]", Summary: "Remove one authored UI view", Classification: commandMutating, RelatedDocs: []string{"repl/commands/ui"}},
		{Name: "recent ls", Usage: "recent ls [--json]", Summary: "List recent activity", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/recent"}},
		{Name: "store ns ls", Usage: "store ns ls [--json]", Summary: "List store namespaces", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/store"}},
		{Name: "store put", Usage: "store put <namespace> <key> <value> [--ttl <duration>] [--json]", Summary: "Write typed key-value state", Classification: commandMutating, RelatedDocs: []string{"repl/commands/store"}},
		{Name: "store get", Usage: "store get <namespace> <key> [--json]", Summary: "Read typed key-value state", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/store"}},
		{Name: "store ls", Usage: "store ls <namespace> [--json]", Summary: "List typed key-value state", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/store"}},
		{Name: "store del", Usage: "store del <namespace> <key> [--json]", Summary: "Delete typed key-value state", Classification: commandMutating, RelatedDocs: []string{"repl/commands/store"}},
		{Name: "store watch", Usage: "store watch <namespace> [--prefix <p>] [--json]", Summary: "Watch typed key-value state by prefix", Classification: commandOperational, RelatedDocs: []string{"repl/commands/store"}},
		{Name: "store bind", Usage: "store bind <namespace> <key> --to <device>:<scenario> [--json]", Summary: "Bind state keys to a device/scenario pair", Classification: commandMutating, RelatedDocs: []string{"repl/commands/store"}},
		{Name: "bus emit", Usage: "bus emit <kind> <name> [payload] [--json]", Summary: "Emit typed bus events or intents", Classification: commandMutating, RelatedDocs: []string{"repl/commands/bus"}},
		{Name: "bus tail", Usage: "bus tail [--kind <kind>] [--name <name>] [--limit <n>] [--json]", Summary: "Tail recent bus events", Classification: commandOperational, RelatedDocs: []string{"repl/commands/bus"}},
		{Name: "bus replay", Usage: "bus replay <from-id> <to-id> [--kind <kind>] [--name <name>] [--limit <n>] [--json]", Summary: "Replay a filtered bus event window", Classification: commandOperational, RelatedDocs: []string{"repl/commands/bus"}},
		{Name: "handlers ls", Usage: "handlers ls [--json]", Summary: "List registered input/event handlers", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/handlers"}},
		{Name: "handlers on", Usage: "handlers on <selector> <action> (--run <command> | --emit <kind> <name> [payload]) [--json]", Summary: "Register an input/event handler", Classification: commandMutating, RelatedDocs: []string{"repl/commands/handlers"}},
		{Name: "handlers off", Usage: "handlers off <handler-id> [--json]", Summary: "Remove one registered handler", Classification: commandMutating, RelatedDocs: []string{"repl/commands/handlers"}},
		{Name: "scenarios ls", Usage: "scenarios ls [--json]", Summary: "List inline REPL-authored scenarios", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/scenarios"}},
		{Name: "scenarios show", Usage: "scenarios show <name> [--json]", Summary: "Show one inline REPL-authored scenario", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/scenarios"}},
		{Name: "scenarios define", Usage: "scenarios define <name> [--match <intent|intent=x|event=y>]... [--priority <p>] [--on-start <command>] [--on-input <command>] [--on-event <kind> <command>]... [--on-suspend <command>] [--on-resume <command>] [--on-stop <command>] [--json]", Summary: "Define or update one inline scenario", Classification: commandMutating, RelatedDocs: []string{"repl/commands/scenarios"}},
		{Name: "scenarios undefine", Usage: "scenarios undefine <name> [--json]", Summary: "Remove one inline REPL-authored scenario", Classification: commandMutating, RelatedDocs: []string{"repl/commands/scenarios"}},
		{Name: "sim device new", Usage: "sim device new <id> [--caps <cap[,cap...]>] [--json]", Summary: "Register a virtual simulation device", Classification: commandMutating, RelatedDocs: []string{"repl/commands/sim"}},
		{Name: "sim device rm", Usage: "sim device rm <id> [--json]", Summary: "Remove a virtual simulation device", Classification: commandMutating, RelatedDocs: []string{"repl/commands/sim"}},
		{Name: "sim input", Usage: "sim input <id> <component-id> <action> [<value>] [--json]", Summary: "Inject synthetic input into a simulation device", Classification: commandMutating, RelatedDocs: []string{"repl/commands/sim"}},
		{Name: "sim ui", Usage: "sim ui <id> [--json]", Summary: "Inspect captured simulated UI state and inputs", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/sim"}},
		{Name: "sim expect", Usage: "sim expect <id> <ui|message> <selector> [--within <duration>] [--json]", Summary: "Assert simulated UI/message output", Classification: commandOperational, RelatedDocs: []string{"repl/commands/sim"}},
		{Name: "sim record", Usage: "sim record <id> [--duration <duration>] [--json]", Summary: "Capture simulated state over a recording window", Classification: commandOperational, RelatedDocs: []string{"repl/commands/sim"}},
		{Name: "scripts dry-run", Usage: "scripts dry-run <path> [--json]", Summary: "Parse and summarize a script without executing it", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/scripts"}},
		{Name: "scripts run", Usage: "scripts run <path> [--json]", Summary: "Execute a REPL script non-interactively", Classification: commandOperational, RelatedDocs: []string{"repl/commands/scripts"}},
		{Name: "activations ls", Usage: "activations ls [--json]", Summary: "List active scenario by device", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/activations"}},
		{Name: "claims tree", Usage: "claims tree [--json]", Summary: "Show claims grouped by device", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/claims"}},
		{Name: "app ls", Usage: "app ls [--json]", Summary: "List loaded apps", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/app"}},
		{Name: "app logs", Usage: "app logs <app> [<query>]", Summary: "Query app-related logs", Classification: commandOperational, RelatedDocs: []string{"repl/commands/app"}},
		{Name: "app reload", Usage: "app reload <app> [--json]", Summary: "Reload an app package", Classification: commandMutating, RelatedDocs: []string{"repl/commands/app"}},
		{Name: "app rollback", Usage: "app rollback <app> [--json]", Summary: "Rollback an app package", Classification: commandMutating, RelatedDocs: []string{"repl/commands/app"}},
		{Name: "apps keys ls", Usage: "apps keys ls [--json]", Summary: "List trust-store keys", Classification: commandReadOnly},
		{Name: "apps keys show", Usage: "apps keys show <key-id> [--json]", Summary: "Show one trust key", Classification: commandReadOnly},
		{Name: "apps keys add", Usage: "apps keys add <key-id> <role[,role]> [note]", Summary: "Add a key to the trust store", Classification: commandMutating},
		{Name: "apps keys confirm", Usage: "apps keys confirm <key-id> [--json]", Summary: "Confirm a candidate key", Classification: commandCriticalMutating},
		{Name: "apps keys revoke", Usage: "apps keys revoke <key-id> [reason]", Summary: "Revoke a key", Classification: commandCriticalMutating},
		{Name: "apps keys archive", Usage: "apps keys archive <key-id> [--json]", Summary: "Archive a non-active key", Classification: commandMutating},
		{Name: "apps keys rotate --accept", Usage: "apps keys rotate --accept <rotation-json>", Summary: "Accept a key rotation (both statements required)", Classification: commandCriticalMutating},
		{Name: "apps keys rotate --rollback", Usage: "apps keys rotate --rollback <accepted-seq>", Summary: "Roll back a key rotation by accepted-seq", Classification: commandCriticalMutating},
		{Name: "apps keys rotate --emit", Usage: "apps keys rotate --emit <old-key-id> <new-key-id> [name ...]", Summary: "Print unsigned rotation statement template", Classification: commandReadOnly},
		{Name: "apps keys rotate-installer", Usage: "apps keys rotate-installer", Summary: "Generate a new installer key pair and rotate", Classification: commandCriticalMutating},
		{Name: "apps keys rotations", Usage: "apps keys rotations [--json]", Summary: "List all rotation records", Classification: commandReadOnly},
		{Name: "apps keys verify", Usage: "apps keys verify [--json]", Summary: "Verify trust log chain integrity", Classification: commandReadOnly},
		{Name: "apps keys log", Usage: "apps keys log [--json]", Summary: "Show trust log entries", Classification: commandReadOnly},
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
	case "apps":
		switch sub {
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

func (s *state) postJSON(ctx context.Context, route string, payload map[string]any) (map[string]any, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return s.doJSON(ctx, http.MethodPost, route, "application/json", bytes.NewReader(b))
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

func flagValue(args []string, name string) string {
	for i := range args {
		if !strings.EqualFold(strings.TrimSpace(args[i]), name) {
			continue
		}
		if i+1 >= len(args) {
			return ""
		}
		next := strings.TrimSpace(args[i+1])
		if strings.HasPrefix(next, "--") {
			return ""
		}
		return next
	}
	return ""
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

func normalizeBugSource(raw string) string {
	source := strings.TrimSpace(strings.ToUpper(raw))
	if source == "" {
		return "BUG_REPORT_SOURCE_ADMIN"
	}
	if !strings.HasPrefix(source, "BUG_REPORT_SOURCE_") {
		source = "BUG_REPORT_SOURCE_" + source
	}
	return source
}

func nonFlagArgsSkippingFlagValues(args []string, valueFlags ...string) []string {
	skipValueFlags := make(map[string]struct{}, len(valueFlags))
	for _, flag := range valueFlags {
		skipValueFlags[strings.ToLower(strings.TrimSpace(flag))] = struct{}{}
	}
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		trimmed := strings.TrimSpace(args[i])
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "--") {
			if _, ok := skipValueFlags[strings.ToLower(trimmed)]; ok {
				if i+1 < len(args) {
					next := strings.TrimSpace(args[i+1])
					if !strings.HasPrefix(next, "--") {
						i++
					}
				}
			}
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func parseHandlersEmitValue(args []string) (kind string, name string, payload string) {
	for i := 0; i < len(args); i++ {
		if !strings.EqualFold(strings.TrimSpace(args[i]), "--emit") {
			continue
		}
		if i+1 >= len(args) {
			return "", "", ""
		}
		kind = strings.TrimSpace(args[i+1])
		if i+2 >= len(args) {
			return kind, "", ""
		}
		name = strings.TrimSpace(args[i+2])
		if i+3 >= len(args) {
			return kind, name, ""
		}
		payloadParts := make([]string, 0, len(args)-(i+3))
		for j := i + 3; j < len(args); j++ {
			part := strings.TrimSpace(args[j])
			if strings.HasPrefix(part, "--") {
				break
			}
			payloadParts = append(payloadParts, part)
		}
		return kind, name, strings.Join(payloadParts, " ")
	}
	return "", "", ""
}

func parseCSVValues(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

type scenarioDefineHook struct {
	kind    string
	command string
}

type scenarioDefineArgs struct {
	name         string
	matchIntents []string
	matchEvents  []string
	priority     string
	onStart      string
	onInput      string
	onEvents     []scenarioDefineHook
	onSuspend    string
	onResume     string
	onStop       string
}

func parseScenariosDefineArgs(args []string) (scenarioDefineArgs, error) {
	out := scenarioDefineArgs{}
	if len(args) == 0 {
		return out, errors.New("usage: scenarios define <name> [--match <intent|intent=x|event=y>]... [--priority <p>] [--on-start <command>] [--on-input <command>] [--on-event <kind> <command>]... [--on-suspend <command>] [--on-resume <command>] [--on-stop <command>]")
	}
	out.name = strings.TrimSpace(args[0])
	if out.name == "" || strings.HasPrefix(out.name, "--") {
		return out, errors.New("usage: scenarios define <name> [--match <intent|intent=x|event=y>]... [--priority <p>] [--on-start <command>] [--on-input <command>] [--on-event <kind> <command>]... [--on-suspend <command>] [--on-resume <command>] [--on-stop <command>]")
	}

	for i := 1; i < len(args); i++ {
		flag := strings.TrimSpace(args[i])
		if flag == "" || !strings.HasPrefix(flag, "--") {
			return out, fmt.Errorf("unexpected token in scenarios define: %s", args[i])
		}
		switch flag {
		case "--json":
			continue
		case "--match":
			if i+1 >= len(args) || strings.HasPrefix(strings.TrimSpace(args[i+1]), "--") {
				return out, errors.New("usage: scenarios define <name> ... --match <intent|intent=x|event=y>")
			}
			for _, token := range strings.Split(strings.TrimSpace(args[i+1]), ",") {
				token = strings.TrimSpace(token)
				if token == "" {
					continue
				}
				lower := strings.ToLower(token)
				switch {
				case strings.HasPrefix(lower, "event="):
					out.matchEvents = append(out.matchEvents, strings.TrimSpace(token[len("event="):]))
				case strings.HasPrefix(lower, "intent="):
					out.matchIntents = append(out.matchIntents, strings.TrimSpace(token[len("intent="):]))
				default:
					out.matchIntents = append(out.matchIntents, token)
				}
			}
			i++
		case "--priority":
			if i+1 >= len(args) || strings.HasPrefix(strings.TrimSpace(args[i+1]), "--") {
				return out, errors.New("usage: scenarios define <name> ... --priority <p>")
			}
			out.priority = strings.TrimSpace(args[i+1])
			i++
		case "--on-start":
			if i+1 >= len(args) || strings.HasPrefix(strings.TrimSpace(args[i+1]), "--") {
				return out, errors.New("usage: scenarios define <name> ... --on-start <command>")
			}
			out.onStart = strings.TrimSpace(args[i+1])
			i++
		case "--on-input":
			if i+1 >= len(args) || strings.HasPrefix(strings.TrimSpace(args[i+1]), "--") {
				return out, errors.New("usage: scenarios define <name> ... --on-input <command>")
			}
			out.onInput = strings.TrimSpace(args[i+1])
			i++
		case "--on-event":
			if i+2 >= len(args) {
				return out, errors.New("usage: scenarios define <name> ... --on-event <kind> <command>")
			}
			kind := strings.TrimSpace(args[i+1])
			if kind == "" || strings.HasPrefix(kind, "--") {
				return out, errors.New("usage: scenarios define <name> ... --on-event <kind> <command>")
			}
			j := i + 2
			parts := make([]string, 0, 2)
			for ; j < len(args); j++ {
				part := strings.TrimSpace(args[j])
				if strings.HasPrefix(part, "--") {
					break
				}
				parts = append(parts, part)
			}
			if len(parts) == 0 {
				return out, errors.New("usage: scenarios define <name> ... --on-event <kind> <command>")
			}
			out.onEvents = append(out.onEvents, scenarioDefineHook{kind: kind, command: strings.Join(parts, " ")})
			i = j - 1
		case "--on-suspend":
			if i+1 >= len(args) || strings.HasPrefix(strings.TrimSpace(args[i+1]), "--") {
				return out, errors.New("usage: scenarios define <name> ... --on-suspend <command>")
			}
			out.onSuspend = strings.TrimSpace(args[i+1])
			i++
		case "--on-resume":
			if i+1 >= len(args) || strings.HasPrefix(strings.TrimSpace(args[i+1]), "--") {
				return out, errors.New("usage: scenarios define <name> ... --on-resume <command>")
			}
			out.onResume = strings.TrimSpace(args[i+1])
			i++
		case "--on-stop":
			if i+1 >= len(args) || strings.HasPrefix(strings.TrimSpace(args[i+1]), "--") {
				return out, errors.New("usage: scenarios define <name> ... --on-stop <command>")
			}
			out.onStop = strings.TrimSpace(args[i+1])
			i++
		default:
			return out, fmt.Errorf("unknown flag for scenarios define: %s", flag)
		}
	}
	out.matchIntents = uniqueStrings(out.matchIntents)
	out.matchEvents = uniqueStrings(out.matchEvents)
	return out, nil
}

func uniqueStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		exists := false
		for _, existing := range out {
			if strings.EqualFold(existing, trimmed) {
				exists = true
				break
			}
		}
		if !exists {
			out = append(out, trimmed)
		}
	}
	return out
}

func joinAnyStrings(value any, sep string) string {
	items, ok := value.([]any)
	if !ok {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		text := strings.TrimSpace(toString(item))
		if text == "" {
			continue
		}
		parts = append(parts, text)
	}
	return strings.Join(parts, sep)
}

func defaultIfBlank(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
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
