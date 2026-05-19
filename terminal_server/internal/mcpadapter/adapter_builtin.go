package mcpadapter

import (
	"context"
	"errors"
	"strings"

	"github.com/curtcox/terminals/terminal_server/internal/repl"
)

func (a *Adapter) callBuiltinTool(tool Tool, req CallToolRequest) (CallToolResponse, bool, error) {
	switch tool.Name {
	case ToolReplComplete:
		prefix := anyString(req.Arguments["prefix"])
		limit := anyInt(req.Arguments["limit"])
		return CallToolResponse{
			Status: "ok",
			Metadata: map[string]any{
				"matches": repl.Complete(prefix, limit),
			},
		}, true, nil
	case ToolReplDescribe:
		return a.callReplDescribeTool(req.Arguments), true, nil
	default:
		return CallToolResponse{}, false, nil
	}
}

func (a *Adapter) callReplDescribeTool(args map[string]any) CallToolResponse {
	name := strings.TrimSpace(anyString(args["command"]))
	if name == "" {
		return CallToolResponse{
			Status: "ok",
			Metadata: map[string]any{
				"commands": repl.CommandSpecs(),
			},
		}
	}
	spec, found := repl.DescribeCommand(name)
	if !found {
		return CallToolResponse{Status: "error", ErrorCode: "unknown_command", ErrorMessage: "unknown command"}
	}
	return CallToolResponse{Status: "ok", Metadata: map[string]any{"command": spec}}
}

func (a *Adapter) executeRenderedTool(
	ctx context.Context,
	req CallToolRequest,
	tool Tool,
	sess SessionInfo,
	rendered string,
	canonicalArgs string,
	onChunk func(string) error,
) (CallToolResponse, error) {
	if tool.Classification.RequiresApproval() {
		gate, err := a.authorizeMutation(ctx, sess, tool, rendered, canonicalArgs, strings.TrimSpace(req.MetaConfirmationID))
		if err != nil {
			return CallToolResponse{}, err
		}
		if gate.Status != "approved" {
			return gate, nil
		}
	}
	if tool.Classification == repl.CommandClassificationOperational {
		return a.executeOperationalTool(ctx, req, tool, rendered, onChunk)
	}
	return a.executeStandardTool(ctx, req, tool, rendered, onChunk)
}

func (a *Adapter) executeOperationalTool(
	ctx context.Context,
	req CallToolRequest,
	tool Tool,
	rendered string,
	onChunk func(string) error,
) (CallToolResponse, error) {
	release, budgetDenied := a.acquireOperationalSlot(req.SessionID, rendered, tool.Classification)
	if budgetDenied != nil {
		return *budgetDenied, nil
	}
	defer release()
	ctxWithTTL, cancel := context.WithTimeout(ctx, a.cfg.OperationalTTL)
	defer cancel()
	return a.runRenderedCommand(ctxWithTTL, req, tool, rendered, onChunk, "operational_ttl_exceeded", "operational command exceeded session stream_ttl budget")
}

func (a *Adapter) executeStandardTool(
	ctx context.Context,
	req CallToolRequest,
	tool Tool,
	rendered string,
	onChunk func(string) error,
) (CallToolResponse, error) {
	return a.runRenderedCommand(ctx, req, tool, rendered, onChunk, "", "")
}

func (a *Adapter) runRenderedCommand(
	ctx context.Context,
	req CallToolRequest,
	tool Tool,
	rendered string,
	onChunk func(string) error,
	ttlErrorCode, ttlErrorMessage string,
) (CallToolResponse, error) {
	result, err := repl.ExecuteCommandStream(ctx, rendered, repl.ExecuteOptions{
		AdminBaseURL: a.cfg.AdminBaseURL,
		SessionID:    req.SessionID,
		DocsMode:     repl.DocsRenderModeMarkdown,
	}, onChunk)
	if ttlErrorCode != "" && errors.Is(err, context.DeadlineExceeded) {
		return CallToolResponse{
			Status:          "error",
			ErrorCode:       ttlErrorCode,
			ErrorMessage:    ttlErrorMessage,
			RenderedCommand: rendered,
			Classification:  tool.Classification,
		}, nil
	}
	if err != nil {
		return CallToolResponse{
			Status:          "error",
			ErrorCode:       "command_failed",
			ErrorMessage:    err.Error(),
			RenderedCommand: rendered,
			Classification:  tool.Classification,
		}, nil
	}
	return CallToolResponse{
		Status:          "ok",
		Output:          result.Output,
		RenderedCommand: rendered,
		Classification:  tool.Classification,
	}, nil
}
