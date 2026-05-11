package repl

import (
	"sort"
	"strings"
)

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
		{Name: "app rollback", Usage: "app rollback <app> [--keep-data|--archive-data|--purge] [--json]", Summary: "Rollback an app package", Classification: commandMutating, RelatedDocs: []string{"repl/commands/app"}},
		{Name: "apps migrate status", Usage: "apps migrate status <app> [--json]", Summary: "Show migration status for one app", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/app"}},
		{Name: "apps migrate logs", Usage: "apps migrate logs <app> [--step <n>] [--json]", Summary: "Tail structured migration logs for one app", Classification: commandOperational, RelatedDocs: []string{"repl/commands/app"}},
		{Name: "apps migrate retry", Usage: "apps migrate retry <app> [--json]", Summary: "Retry app migration execution", Classification: commandCriticalMutating, RelatedDocs: []string{"repl/commands/app"}},
		{Name: "apps migrate abort", Usage: "apps migrate abort <app> [--to <checkpoint|baseline>] [--json]", Summary: "Abort in-flight app migration execution", Classification: commandCriticalMutating, RelatedDocs: []string{"repl/commands/app"}},
		{Name: "apps migrate drain-ready", Usage: "apps migrate drain-ready <app> <true|false> [--json]", Summary: "Mark whether drain prerequisites are satisfied", Classification: commandCriticalMutating, RelatedDocs: []string{"repl/commands/app"}},
		{Name: "apps migrate reconcile", Usage: "apps migrate reconcile <app> <record-id> <resolution> [--json]", Summary: "Resolve one migration reconciliation record", Classification: commandCriticalMutating, RelatedDocs: []string{"repl/commands/app"}},
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
		{Name: "ai ask", Usage: "ai ask <prompt> [--json]", Summary: "Ask the configured AI provider a question", Classification: commandOperational, RelatedDocs: []string{"repl/commands/ai"}, DiscouragedForAgents: true},
		{Name: "ai gen", Usage: "ai gen <description> [--json]", Summary: "Generate text/code from the configured AI provider", Classification: commandOperational, RelatedDocs: []string{"repl/commands/ai"}, DiscouragedForAgents: true},
		{Name: "ai run", Usage: "ai run [--json]", Summary: "Execute the pending AI-proposed command (alias of ai approve)", Classification: commandMutating, RelatedDocs: []string{"repl/commands/ai"}, DiscouragedForAgents: true},
		{Name: "ai approve", Usage: "ai approve [--json]", Summary: "Approve and execute the pending AI-proposed command", Classification: commandMutating, RelatedDocs: []string{"repl/commands/ai"}, DiscouragedForAgents: true},
		{Name: "ai reject", Usage: "ai reject [--json]", Summary: "Reject and clear the pending AI-proposed command", Classification: commandMutating, RelatedDocs: []string{"repl/commands/ai"}, DiscouragedForAgents: true},
		{Name: "ai context", Usage: "ai context [--json]", Summary: "Show pinned AI context refs", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/ai"}, DiscouragedForAgents: true},
		{Name: "ai context add", Usage: "ai context add <ref> [--json]", Summary: "Add one-shot AI context for the next turn", Classification: commandMutating, RelatedDocs: []string{"repl/commands/ai"}, DiscouragedForAgents: true},
		{Name: "ai context pin", Usage: "ai context pin <ref> [--json]", Summary: "Pin AI context across turns", Classification: commandMutating, RelatedDocs: []string{"repl/commands/ai"}, DiscouragedForAgents: true},
		{Name: "ai context unpin", Usage: "ai context unpin <ref> [--json]", Summary: "Remove one pinned AI context ref", Classification: commandMutating, RelatedDocs: []string{"repl/commands/ai"}, DiscouragedForAgents: true},
		{Name: "ai context clear", Usage: "ai context clear [--json]", Summary: "Clear pinned AI context refs", Classification: commandMutating, RelatedDocs: []string{"repl/commands/ai"}, DiscouragedForAgents: true},
		{Name: "ai policy show", Usage: "ai policy show [--json]", Summary: "Show AI approval policy", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/ai"}, DiscouragedForAgents: true},
		{Name: "ai policy set", Usage: "ai policy set <auto-readonly|prompt-all|prompt-mutating> [--json]", Summary: "Set AI approval policy for this session", Classification: commandMutating, RelatedDocs: []string{"repl/commands/ai"}, DiscouragedForAgents: true},
		{Name: "ai history", Usage: "ai history [--json]", Summary: "Show AI thread id and recent exchange history", Classification: commandReadOnly, RelatedDocs: []string{"repl/commands/ai"}, DiscouragedForAgents: true},
		{Name: "ai reset", Usage: "ai reset [--json]", Summary: "Clear AI thread id and exchange history", Classification: commandMutating, RelatedDocs: []string{"repl/commands/ai"}, DiscouragedForAgents: true},
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
