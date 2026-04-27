// Package mcpadapter bridges the MCP tool-call protocol to the REPL capability layer.
package mcpadapter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/repl"
	"github.com/curtcox/terminals/terminal_server/internal/replsession"
)

// Tool name constants for special REPL meta-tools.
const (
	ToolReplComplete = "repl_complete"
	ToolReplDescribe = "repl_describe"
)

// Sentinel errors returned by the adapter.
var (
	ErrUnknownTool    = errors.New("unknown tool")
	ErrUnknownSession = errors.New("unknown mcp session")
)

// MutatingCapability describes how a session authorizes mutating tool calls.
type MutatingCapability string

// MutatingCapability values.
const (
	MutatingViaElicitation MutatingCapability = "mutating_via_elicitation"
	MutatingViaFallback    MutatingCapability = "mutating_via_fallback"
	MutatingUnavailable    MutatingCapability = "mutating_unavailable"
)

// ClientCapabilities describes optional protocol features supported by an MCP client.
type ClientCapabilities struct {
	SupportsElicitation bool
	SupportsFallbackID  bool
}

// SessionInfo holds the negotiated state for an open MCP session.
type SessionInfo struct {
	SessionID      string
	Capability     MutatingCapability
	ClientIdentity string
}

// ElicitRequest carries the parameters for a human-approval elicitation call.
type ElicitRequest struct {
	SessionID       string
	ToolName        string
	RenderedCommand string
	Classification  repl.CommandClassification
}

// ElicitResponse carries the human approval decision from an elicitation call.
type ElicitResponse struct {
	Approved bool
}

// UnsafeConfirmationEvent is emitted when a mutation is confirmed suspiciously fast.
type UnsafeConfirmationEvent struct {
	SessionID   string
	ClientID    string
	ToolName    string
	CommandHash string
	Latency     time.Duration
	Path        string
}

// Config holds construction-time settings for the Adapter.
type Config struct {
	AdminBaseURL       string
	Now                func() time.Time
	ConfirmationTTL    time.Duration
	MinHumanLatency    time.Duration
	OperationalMax     int
	OperationalTTL     time.Duration
	Elicit             func(context.Context, ElicitRequest) (ElicitResponse, error)
	UnsafeConfirmation func(context.Context, UnsafeConfirmationEvent)
}

// Tool describes a single callable tool exposed via the MCP adapter.
type Tool struct {
	Name                 string
	CommandName          string
	Classification       repl.CommandClassification
	Description          string
	DiscouragedForAgents bool
	ArgumentsSchema      map[string]any
}

// CallToolRequest is the input to a single MCP tool call.
type CallToolRequest struct {
	SessionID          string
	ToolName           string
	Arguments          map[string]any
	MetaConfirmationID string
}

// CallToolResponse is the output from a single MCP tool call.
type CallToolResponse struct {
	Status          string
	Output          string
	ErrorCode       string
	ErrorMessage    string
	ConfirmationID  string
	ExpiresAt       time.Time
	RenderedCommand string
	Classification  repl.CommandClassification
	Metadata        map[string]any
}

// Adapter bridges MCP tool-call requests to the REPL capability layer.
type Adapter struct {
	cfg             Config
	toolsByName     map[string]Tool
	tools           []Tool
	registryVersion string

	mu            sync.Mutex
	sessions      map[string]SessionInfo
	confirmations map[string]pendingConfirmation
	operational   map[string]int
}

type pendingConfirmation struct {
	SessionID       string
	ToolName        string
	CanonicalArgs   string
	RenderedCommand string
	CreatedAt       time.Time
	ExpiresAt       time.Time
	Consumed        bool
}

// New creates an Adapter with the given configuration.
func New(cfg Config) *Adapter {
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	if cfg.ConfirmationTTL <= 0 {
		cfg.ConfirmationTTL = 2 * time.Minute
	}
	if cfg.MinHumanLatency <= 0 {
		cfg.MinHumanLatency = 500 * time.Millisecond
	}
	if cfg.OperationalMax <= 0 {
		cfg.OperationalMax = 3
	}
	if cfg.OperationalTTL <= 0 {
		cfg.OperationalTTL = 2 * time.Minute
	}
	cfg.Now = now

	tools := generateTools()
	toolsByName := make(map[string]Tool, len(tools))
	for _, tool := range tools {
		toolsByName[tool.Name] = tool
	}
	return &Adapter{
		cfg:             cfg,
		toolsByName:     toolsByName,
		tools:           tools,
		registryVersion: computeRegistryVersion(tools),
		sessions:        map[string]SessionInfo{},
		confirmations:   map[string]pendingConfirmation{},
		operational:     map[string]int{},
	}
}

// RegistryVersion returns a hash that changes whenever the tool registry changes.
func (a *Adapter) RegistryVersion() string {
	return a.registryVersion
}

// Tools returns a snapshot of the registered tools.
func (a *Adapter) Tools() []Tool {
	out := make([]Tool, len(a.tools))
	copy(out, a.tools)
	return out
}

// OpenSession registers a new MCP session and returns its negotiated info.
func (a *Adapter) OpenSession(sessionID, clientIdentity string, caps ClientCapabilities) SessionInfo {
	sessionID = strings.TrimSpace(sessionID)
	clientIdentity = strings.TrimSpace(clientIdentity)
	capability := MutatingUnavailable
	switch {
	case caps.SupportsElicitation:
		capability = MutatingViaElicitation
	case caps.SupportsFallbackID:
		capability = MutatingViaFallback
	}
	info := SessionInfo{SessionID: sessionID, Capability: capability, ClientIdentity: clientIdentity}
	a.mu.Lock()
	a.sessions[sessionID] = info
	a.mu.Unlock()
	return info
}

// CloseSession removes the session with the given ID.
func (a *Adapter) CloseSession(sessionID string) {
	a.mu.Lock()
	delete(a.sessions, strings.TrimSpace(sessionID))
	a.mu.Unlock()
}

// SessionInfo returns the negotiated info for the given session ID.
func (a *Adapter) SessionInfo(sessionID string) (SessionInfo, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	info, ok := a.sessions[strings.TrimSpace(sessionID)]
	return info, ok
}

// SetSessionCapability updates the mutating capability for the given session.
func (a *Adapter) SetSessionCapability(sessionID string, capability MutatingCapability) bool {
	sessionID = strings.TrimSpace(sessionID)
	a.mu.Lock()
	defer a.mu.Unlock()
	info, ok := a.sessions[sessionID]
	if !ok {
		return false
	}
	info.Capability = capability
	a.sessions[sessionID] = info
	return true
}

// SetElicitHook installs a callback invoked when a mutating tool requires human approval.
func (a *Adapter) SetElicitHook(hook func(context.Context, ElicitRequest) (ElicitResponse, error)) {
	a.mu.Lock()
	a.cfg.Elicit = hook
	a.mu.Unlock()
}

// CallTool executes a single MCP tool call and returns the response.
func (a *Adapter) CallTool(ctx context.Context, req CallToolRequest) (CallToolResponse, error) {
	return a.CallToolStream(ctx, req, nil)
}

// CallToolStream executes a tool call and delivers output chunks via onChunk as they are produced.
func (a *Adapter) CallToolStream(ctx context.Context, req CallToolRequest, onChunk func(string) error) (CallToolResponse, error) {
	tool, ok := a.toolsByName[strings.TrimSpace(req.ToolName)]
	if !ok {
		return CallToolResponse{}, ErrUnknownTool
	}
	sess, ok := a.SessionInfo(req.SessionID)
	if !ok {
		return CallToolResponse{}, ErrUnknownSession
	}
	if req.Arguments == nil {
		req.Arguments = map[string]any{}
	}

	if tool.Name == ToolReplComplete {
		prefix := anyString(req.Arguments["prefix"])
		limit := anyInt(req.Arguments["limit"])
		return CallToolResponse{
			Status: "ok",
			Metadata: map[string]any{
				"matches": repl.Complete(prefix, limit),
			},
		}, nil
	}
	if tool.Name == ToolReplDescribe {
		name := strings.TrimSpace(anyString(req.Arguments["command"]))
		if name == "" {
			return CallToolResponse{
				Status: "ok",
				Metadata: map[string]any{
					"commands": repl.CommandSpecs(),
				},
			}, nil
		}
		spec, found := repl.DescribeCommand(name)
		if !found {
			return CallToolResponse{Status: "error", ErrorCode: "unknown_command", ErrorMessage: "unknown command"}, nil
		}
		return CallToolResponse{Status: "ok", Metadata: map[string]any{"command": spec}}, nil
	}

	rendered, canonicalArgs, err := renderCommand(tool, req.Arguments)
	if err != nil {
		return CallToolResponse{Status: "error", ErrorCode: "invalid_arguments", ErrorMessage: err.Error()}, nil
	}
	if tool.Classification == repl.CommandClassificationMutating {
		gate, err := a.authorizeMutation(ctx, sess, tool, rendered, canonicalArgs, strings.TrimSpace(req.MetaConfirmationID))
		if err != nil {
			return CallToolResponse{}, err
		}
		if gate.Status != "approved" {
			return gate, nil
		}
	}

	if tool.Classification == repl.CommandClassificationOperational {
		release, budgetDenied := a.acquireOperationalSlot(req.SessionID, rendered, tool.Classification)
		if budgetDenied != nil {
			return *budgetDenied, nil
		}
		defer release()
		ctxWithTTL, cancel := context.WithTimeout(ctx, a.cfg.OperationalTTL)
		defer cancel()
		result, err := repl.ExecuteCommandStream(ctxWithTTL, rendered, repl.ExecuteOptions{
			AdminBaseURL: a.cfg.AdminBaseURL,
			SessionID:    req.SessionID,
			DocsMode:     repl.DocsRenderModeMarkdown,
		}, onChunk)
		if errors.Is(err, context.DeadlineExceeded) {
			return CallToolResponse{
				Status:          "error",
				ErrorCode:       "operational_ttl_exceeded",
				ErrorMessage:    "operational command exceeded session stream_ttl budget",
				RenderedCommand: rendered,
				Classification:  tool.Classification,
			}, nil
		}
		if err != nil {
			return CallToolResponse{Status: "error", ErrorCode: "command_failed", ErrorMessage: err.Error(), RenderedCommand: rendered, Classification: tool.Classification}, nil
		}
		return CallToolResponse{Status: "ok", Output: result.Output, RenderedCommand: rendered, Classification: tool.Classification}, nil
	}
	result, err := repl.ExecuteCommandStream(ctx, rendered, repl.ExecuteOptions{
		AdminBaseURL: a.cfg.AdminBaseURL,
		SessionID:    req.SessionID,
		DocsMode:     repl.DocsRenderModeMarkdown,
	}, onChunk)
	if err != nil {
		return CallToolResponse{Status: "error", ErrorCode: "command_failed", ErrorMessage: err.Error(), RenderedCommand: rendered, Classification: tool.Classification}, nil
	}
	return CallToolResponse{Status: "ok", Output: result.Output, RenderedCommand: rendered, Classification: tool.Classification}, nil
}

func (a *Adapter) acquireOperationalSlot(sessionID, rendered string, classification repl.CommandClassification) (func(), *CallToolResponse) {
	a.mu.Lock()
	if a.operational[sessionID] >= a.cfg.OperationalMax {
		a.mu.Unlock()
		return nil, &CallToolResponse{
			Status:          "error",
			ErrorCode:       "rate_limited",
			ErrorMessage:    "operational command budget exceeded for session",
			RenderedCommand: rendered,
			Classification:  classification,
		}
	}
	a.operational[sessionID]++
	a.mu.Unlock()
	return func() {
		a.mu.Lock()
		if a.operational[sessionID] <= 1 {
			delete(a.operational, sessionID)
		} else {
			a.operational[sessionID]--
		}
		a.mu.Unlock()
	}, nil
}

func (a *Adapter) authorizeMutation(ctx context.Context, sess SessionInfo, tool Tool, rendered, canonicalArgs, metaConfirmationID string) (CallToolResponse, error) {
	switch sess.Capability {
	case MutatingUnavailable:
		return CallToolResponse{Status: "error", ErrorCode: "unsupported_client", ErrorMessage: "mutating tools require elicitation or confirmation_id fallback support", RenderedCommand: rendered, Classification: tool.Classification}, nil
	case MutatingViaElicitation:
		a.mu.Lock()
		elicit := a.cfg.Elicit
		a.mu.Unlock()
		if elicit == nil {
			return CallToolResponse{Status: "error", ErrorCode: "elicit_unavailable", ErrorMessage: "elicitation hook is not configured", RenderedCommand: rendered, Classification: tool.Classification}, nil
		}
		start := a.cfg.Now()
		resp, err := elicit(ctx, ElicitRequest{
			SessionID:       sess.SessionID,
			ToolName:        tool.Name,
			RenderedCommand: rendered,
			Classification:  tool.Classification,
		})
		if err != nil {
			return CallToolResponse{}, err
		}
		latency := a.cfg.Now().Sub(start)
		if latency < a.cfg.MinHumanLatency {
			a.emitUnsafe(ctx, sess, tool.Name, canonicalArgs, latency, "elicitation")
		}
		if !resp.Approved {
			return CallToolResponse{Status: "rejected", ErrorCode: "approval_rejected", ErrorMessage: "mutation not approved", RenderedCommand: rendered, Classification: tool.Classification}, nil
		}
		return CallToolResponse{Status: "approved"}, nil
	case MutatingViaFallback:
		return a.authorizeViaFallback(ctx, sess, tool, rendered, canonicalArgs, metaConfirmationID), nil
	default:
		return CallToolResponse{Status: "error", ErrorCode: "unsupported_client", ErrorMessage: "unknown capability state"}, nil
	}
}

func (a *Adapter) authorizeViaFallback(ctx context.Context, sess SessionInfo, tool Tool, rendered, canonicalArgs, metaConfirmationID string) CallToolResponse {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.cfg.Now()
	if metaConfirmationID == "" {
		return a.issueConfirmationLocked(now, sess, tool, rendered, canonicalArgs)
	}
	pending, ok := a.confirmations[metaConfirmationID]
	if !ok || pending.Consumed || now.After(pending.ExpiresAt) || pending.SessionID != sess.SessionID || pending.ToolName != tool.Name || pending.CanonicalArgs != canonicalArgs {
		return a.issueConfirmationLocked(now, sess, tool, rendered, canonicalArgs)
	}
	pending.Consumed = true
	a.confirmations[metaConfirmationID] = pending
	latency := now.Sub(pending.CreatedAt)
	if latency < a.cfg.MinHumanLatency {
		a.emitUnsafe(ctx, sess, tool.Name, canonicalArgs, latency, "confirmation_id")
	}
	return CallToolResponse{Status: "approved"}
}

func (a *Adapter) issueConfirmationLocked(now time.Time, sess SessionInfo, tool Tool, rendered, canonicalArgs string) CallToolResponse {
	expires := now.Add(a.cfg.ConfirmationTTL)
	token := newConfirmationToken(sess.SessionID, tool.Name, canonicalArgs, now)
	a.confirmations[token] = pendingConfirmation{
		SessionID:       sess.SessionID,
		ToolName:        tool.Name,
		CanonicalArgs:   canonicalArgs,
		RenderedCommand: rendered,
		CreatedAt:       now,
		ExpiresAt:       expires,
	}
	return CallToolResponse{
		Status:          "confirmation_required",
		ConfirmationID:  token,
		ExpiresAt:       expires,
		RenderedCommand: rendered,
		Classification:  tool.Classification,
	}
}

func (a *Adapter) emitUnsafe(ctx context.Context, sess SessionInfo, toolName, canonicalArgs string, latency time.Duration, path string) {
	if a.cfg.UnsafeConfirmation == nil {
		return
	}
	sum := sha256.Sum256([]byte(canonicalArgs))
	a.cfg.UnsafeConfirmation(ctx, UnsafeConfirmationEvent{
		SessionID:   sess.SessionID,
		ClientID:    sess.ClientIdentity,
		ToolName:    toolName,
		CommandHash: hex.EncodeToString(sum[:]),
		Latency:     latency,
		Path:        path,
	})
}

func generateTools() []Tool {
	base := make([]Tool, 0, len(repl.CommandSpecs())+2)
	for _, spec := range repl.CommandSpecs() {
		name := strings.ReplaceAll(spec.Name, " ", "_")
		if strings.Contains(name, "_confirm") || strings.Contains(name, "_force") {
			continue
		}
		base = append(base, Tool{
			Name:                 name,
			CommandName:          spec.Name,
			Classification:       spec.Classification,
			Description:          buildDescription(spec),
			DiscouragedForAgents: spec.DiscouragedForAgents,
			ArgumentsSchema:      buildArgumentsSchema(spec),
		})
	}
	base = append(base,
		Tool{
			Name:            ToolReplComplete,
			Description:     "Mirror REPL completion for agent discovery.",
			Classification:  repl.CommandClassificationReadOnly,
			ArgumentsSchema: map[string]any{"type": "object", "properties": map[string]any{"prefix": map[string]any{"type": "string"}, "limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 128}}, "required": []string{"prefix"}},
		},
		Tool{
			Name:            ToolReplDescribe,
			Description:     "Mirror REPL describe metadata for one command, or return registry summary when command is omitted.",
			Classification:  repl.CommandClassificationReadOnly,
			ArgumentsSchema: map[string]any{"type": "object", "properties": map[string]any{"command": map[string]any{"type": "string"}}},
		},
	)
	sort.Slice(base, func(i, j int) bool { return base[i].Name < base[j].Name })
	return base
}

func buildDescription(spec repl.CommandSpec) string {
	parts := []string{spec.Summary, "classification: " + string(spec.Classification), "usage: " + spec.Usage}
	if spec.DiscouragedForAgents {
		parts = append(parts, "discouraged_for_agents: true")
	}
	if len(spec.Examples) > 0 {
		parts = append(parts, "examples: "+strings.Join(spec.Examples, " | "))
	}
	return strings.Join(parts, "\n")
}

func buildArgumentsSchema(spec repl.CommandSpec) map[string]any {
	params := usageParams(spec.Usage)
	properties := map[string]any{}
	required := make([]string, 0, len(params))
	for _, param := range params {
		if param.Name == "json" {
			properties[param.Name] = map[string]any{"type": "boolean", "description": "Return JSON output when command supports --json."}
			continue
		}
		properties[param.Name] = map[string]any{"type": "string"}
		if !param.Optional {
			required = append(required, param.Name)
		}
	}
	schema := map[string]any{"type": "object", "properties": properties}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

type usageParam struct {
	Name     string
	Optional bool
}

func usageParams(usage string) []usageParam {
	tokens := strings.Fields(strings.TrimSpace(usage))
	if len(tokens) <= 1 {
		return nil
	}
	out := make([]usageParam, 0)
	for _, token := range tokens[1:] {
		switch {
		case token == "[--json]" || token == "--json":
			out = append(out, usageParam{Name: "json", Optional: true})
		case strings.HasPrefix(token, "<") && strings.HasSuffix(token, ">"):
			name := strings.TrimSuffix(strings.TrimPrefix(token, "<"), ">")
			name = strings.ReplaceAll(name, "-", "_")
			out = append(out, usageParam{Name: name})
		case strings.HasPrefix(token, "[") && strings.HasSuffix(token, "]"):
			inner := strings.TrimSuffix(strings.TrimPrefix(token, "["), "]")
			if strings.HasPrefix(inner, "<") && strings.HasSuffix(inner, ">") {
				name := strings.TrimSuffix(strings.TrimPrefix(inner, "<"), ">")
				name = strings.ReplaceAll(name, "-", "_")
				out = append(out, usageParam{Name: name, Optional: true})
				continue
			}
			inner = strings.ReplaceAll(inner, "-", "_")
			out = append(out, usageParam{Name: inner, Optional: true})
		}
	}
	return out
}

func renderCommand(tool Tool, args map[string]any) (rendered string, canonicalArgs string, err error) {
	spec, ok := repl.DescribeCommand(tool.CommandName)
	if !ok {
		return "", "", fmt.Errorf("unknown command %s", tool.CommandName)
	}
	params := usageParams(spec.Usage)
	parts := []string{spec.Name}
	canonical := make([]string, 0, len(params))
	for _, param := range params {
		if param.Name == "json" {
			if anyBool(args[param.Name]) {
				parts = append(parts, "--json")
				canonical = append(canonical, "json=true")
			}
			continue
		}
		value := strings.TrimSpace(anyString(args[param.Name]))
		if value == "" {
			if param.Optional {
				continue
			}
			return "", "", fmt.Errorf("missing required argument: %s", param.Name)
		}
		parts = append(parts, value)
		canonical = append(canonical, param.Name+"="+value)
	}
	return strings.Join(parts, " "), strings.Join(canonical, "|"), nil
}

func anyString(v any) string {
	if v == nil {
		return ""
	}
	switch typed := v.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func anyInt(v any) int {
	switch typed := v.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func anyBool(v any) bool {
	switch typed := v.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return false
	}
}

func computeRegistryVersion(tools []Tool) string {
	h := sha256.New()
	for _, tool := range tools {
		_, _ = h.Write([]byte(tool.Name))
		_, _ = h.Write([]byte("|"))
		_, _ = h.Write([]byte(tool.CommandName))
		_, _ = h.Write([]byte("|"))
		_, _ = h.Write([]byte(tool.Description))
		_, _ = h.Write([]byte("\n"))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func newConfirmationToken(sessionID, toolName, canonicalArgs string, now time.Time) string {
	raw := fmt.Sprintf("%s|%s|%s|%d", sessionID, toolName, canonicalArgs, now.UnixNano())
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:16])
}

// ReplSessionCreateRequest builds a replsession.CreateSessionRequest from the given MCP session info.
func ReplSessionCreateRequest(deviceID, ownerActivationID string, info SessionInfo) replsession.CreateSessionRequest {
	return replsession.CreateSessionRequest{
		DeviceID:          strings.TrimSpace(deviceID),
		OwnerActivationID: strings.TrimSpace(ownerActivationID),
		Origin:            replsession.SessionOriginMCP,
		AgentIdentity:     strings.TrimSpace(info.ClientIdentity),
		AgentCapability:   string(info.Capability),
	}
}
