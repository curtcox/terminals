package io

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/curtcox/terminals/terminal_server/internal/eventlog"
)

func indexMediaPlanNodes(plan FlowPlan) (map[string]FlowNode, map[string][]string, error) {
	nodeByID, err := indexMediaPlanNodeIDs(plan.Nodes)
	if err != nil {
		return nil, nil, err
	}
	incoming, err := indexMediaPlanEdges(plan.Edges, nodeByID)
	if err != nil {
		return nil, nil, err
	}
	return nodeByID, incoming, nil
}

func indexMediaPlanNodeIDs(nodes []FlowNode) (map[string]FlowNode, error) {
	nodeByID := make(map[string]FlowNode, len(nodes))
	for _, node := range nodes {
		id := strings.TrimSpace(node.ID)
		if id == "" {
			return nil, ErrInvalidMediaPlan
		}
		node.ID = id
		if node.Args == nil {
			node.Args = map[string]string{}
		}
		if node.Exec == "" {
			node.Exec = ExecAuto
		}
		nodeByID[id] = node
	}
	return nodeByID, nil
}

func indexMediaPlanEdges(edges []FlowEdge, nodeByID map[string]FlowNode) (map[string][]string, error) {
	incoming := make(map[string][]string)
	for _, edge := range edges {
		from := strings.TrimSpace(edge.From)
		to := strings.TrimSpace(edge.To)
		if from == "" || to == "" {
			return nil, ErrInvalidMediaPlan
		}
		if _, ok := nodeByID[from]; !ok {
			return nil, ErrInvalidMediaPlan
		}
		if _, ok := nodeByID[to]; !ok {
			return nil, ErrInvalidMediaPlan
		}
		incoming[to] = append(incoming[to], from)
	}
	return incoming, nil
}

func (p *MediaPlanner) applySinkEdge(
	ctx context.Context,
	runtime *planRuntime,
	from, to FlowNode,
	nodeByID map[string]FlowNode,
	incoming map[string][]string,
) error {
	if !isSinkNode(to.Kind) {
		return nil
	}
	sourceNode, ok := resolveSourceNode(from.ID, nodeByID, incoming, map[string]struct{}{})
	if !ok {
		return nil
	}
	sourceDeviceID := strings.TrimSpace(sourceNode.Args["device_id"])
	targetDeviceID := strings.TrimSpace(to.Args["device_id"])
	if sourceDeviceID == "" || targetDeviceID == "" {
		return nil
	}
	sourceResource := strings.TrimSpace(sourceNode.Args["resource"])
	targetResource := strings.TrimSpace(to.Args["resource"])
	streamKind := streamKindFor(sourceNode.Kind, to.Kind)
	if inferred := streamKindForResources(sourceResource, targetResource); inferred != "" {
		streamKind = inferred
	}
	if override := strings.TrimSpace(to.Args["stream_kind"]); override != "" {
		streamKind = override
	}
	if streamKind == "" {
		return nil
	}
	if err := p.router.Connect(sourceDeviceID, targetDeviceID, streamKind); err != nil && !errors.Is(err, ErrRouteExists) {
		return err
	}
	runtime.routes = append(runtime.routes, Route{
		SourceID:   sourceDeviceID,
		TargetID:   targetDeviceID,
		StreamKind: streamKind,
	})
	eventlog.Emit(ctx, "io.route.applied", slog.LevelInfo, "route applied",
		slog.String("component", "io.router"),
		slog.String("source_device_id", sourceDeviceID),
		slog.String("target_device_id", targetDeviceID),
		slog.String("stream_kind", streamKind),
	)
	return nil
}

func (p *MediaPlanner) applyAnalyzerEdge(
	ctx context.Context,
	runtime *planRuntime,
	from, to FlowNode,
	nodeByID map[string]FlowNode,
	incoming map[string][]string,
) error {
	if to.Kind != NodeAnalyzer || p.analyzer == nil {
		return nil
	}
	sourceNode, ok := resolveSourceNode(from.ID, nodeByID, incoming, map[string]struct{}{})
	if !ok {
		return nil
	}
	sourceDeviceID := strings.TrimSpace(sourceNode.Args["device_id"])
	if sourceDeviceID == "" {
		return nil
	}
	analyzerName := strings.TrimSpace(to.Args["name"])
	if analyzerName == "" {
		analyzerName = "sound"
	}
	stop, err := p.analyzer.StartAnalyzer(ctx, sourceDeviceID, analyzerName, p.emitAnalyzerEvent)
	if err != nil {
		return err
	}
	if stop != nil {
		runtime.stops = append(runtime.stops, stop)
	}
	return nil
}
