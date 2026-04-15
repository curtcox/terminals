// Package io manages logical stream-routing state.
package io //nolint:revive

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// MediaNodeKind identifies node behavior in a media plan.
type MediaNodeKind string

// Media node kind constants.
const (
	NodeSourceMic    MediaNodeKind = "source.mic"
	NodeSourceCamera MediaNodeKind = "source.camera"
	NodeSourceTTS    MediaNodeKind = "source.tts"
	NodeSinkSpeaker  MediaNodeKind = "sink.speaker"
	NodeSinkDisplay  MediaNodeKind = "sink.display"
	NodeSinkSTT      MediaNodeKind = "sink.stt"
	NodeAnalyzer     MediaNodeKind = "analyzer"
	NodeRecorder     MediaNodeKind = "recorder"
	NodeFork         MediaNodeKind = "fork"
)

// MediaNode is one graph node.
type MediaNode struct {
	ID   string
	Kind MediaNodeKind
	Args map[string]string
}

// MediaEdge links two nodes by ID.
type MediaEdge struct {
	From string
	To   string
}

// MediaPlan is a declarative topology graph.
type MediaPlan struct {
	Nodes []MediaNode
	Edges []MediaEdge
}

// PlanHandle identifies one installed media plan.
type PlanHandle string

// AnalyzerEvent is emitted by analyzer nodes.
type AnalyzerEvent struct {
	Kind       string
	Subject    string
	Attributes map[string]string
	OccurredAt time.Time
}

// AnalyzerRunner executes analyzer nodes for a source device.
type AnalyzerRunner interface {
	StartAnalyzer(
		ctx context.Context,
		sourceDeviceID string,
		analyzer string,
		emit func(AnalyzerEvent),
	) (func(), error)
}

// MediaPlanner installs media plans and compiles them into logical routes.
type MediaPlanner struct {
	mu sync.Mutex

	router       *Router
	nextID       uint64
	active       map[PlanHandle]planRuntime
	analyzer     AnalyzerRunner
	analyzerSink func(AnalyzerEvent)
}

type planRuntime struct {
	routes []Route
	stops  []func()
}

var (
	// ErrInvalidMediaPlan indicates malformed nodes/edges.
	ErrInvalidMediaPlan = errors.New("invalid media plan")
)

// NewMediaPlanner returns an empty planner.
func NewMediaPlanner(router *Router) *MediaPlanner {
	return &MediaPlanner{
		router: router,
		active: make(map[PlanHandle]planRuntime),
	}
}

// SetAnalyzerRunner sets optional analyzer runtime support.
func (p *MediaPlanner) SetAnalyzerRunner(runner AnalyzerRunner) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.analyzer = runner
}

// AnalyzerEnabled reports whether analyzer nodes can execute.
func (p *MediaPlanner) AnalyzerEnabled() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.analyzer != nil
}

// SetAnalyzerSink sets optional analyzer event sink callback.
func (p *MediaPlanner) SetAnalyzerSink(sink func(AnalyzerEvent)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.analyzerSink = sink
}

// Apply compiles and installs a media plan.
func (p *MediaPlanner) Apply(ctx context.Context, plan MediaPlan) (PlanHandle, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.router == nil {
		return "", ErrInvalidMediaPlan
	}
	compiled, err := p.compileLocked(ctx, plan)
	if err != nil {
		return "", err
	}

	p.nextID++
	handle := PlanHandle(fmt.Sprintf("plan-%d", p.nextID))
	p.active[handle] = compiled
	return handle, nil
}

// Patch replaces an existing plan with a new one.
func (p *MediaPlanner) Patch(ctx context.Context, handle PlanHandle, plan MediaPlan) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	previous, ok := p.active[handle]
	if !ok {
		return ErrInvalidMediaPlan
	}
	for _, stop := range previous.stops {
		stop()
	}
	for _, route := range previous.routes {
		_ = p.router.Disconnect(route.SourceID, route.TargetID, route.StreamKind)
	}

	compiled, err := p.compileLocked(ctx, plan)
	if err != nil {
		return err
	}
	p.active[handle] = compiled
	return nil
}

// Tear uninstalls a plan.
func (p *MediaPlanner) Tear(_ context.Context, handle PlanHandle) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	runtime, ok := p.active[handle]
	if !ok {
		return nil
	}
	delete(p.active, handle)

	for _, stop := range runtime.stops {
		stop()
	}
	for _, route := range runtime.routes {
		_ = p.router.Disconnect(route.SourceID, route.TargetID, route.StreamKind)
	}
	return nil
}

func (p *MediaPlanner) compileLocked(ctx context.Context, plan MediaPlan) (planRuntime, error) {
	nodeByID := make(map[string]MediaNode, len(plan.Nodes))
	incoming := make(map[string][]string)
	for _, node := range plan.Nodes {
		id := strings.TrimSpace(node.ID)
		if id == "" {
			return planRuntime{}, ErrInvalidMediaPlan
		}
		node.ID = id
		if node.Args == nil {
			node.Args = map[string]string{}
		}
		nodeByID[id] = node
	}
	for _, edge := range plan.Edges {
		from := strings.TrimSpace(edge.From)
		to := strings.TrimSpace(edge.To)
		if from == "" || to == "" {
			return planRuntime{}, ErrInvalidMediaPlan
		}
		if _, ok := nodeByID[from]; !ok {
			return planRuntime{}, ErrInvalidMediaPlan
		}
		if _, ok := nodeByID[to]; !ok {
			return planRuntime{}, ErrInvalidMediaPlan
		}
		incoming[to] = append(incoming[to], from)
	}

	runtime := planRuntime{}
	for _, edge := range plan.Edges {
		from := nodeByID[strings.TrimSpace(edge.From)]
		to := nodeByID[strings.TrimSpace(edge.To)]

		// Route concrete source->sink links onto the imperative router.
		if isSinkNode(to.Kind) {
			sourceNode, ok := resolveSourceNode(from.ID, nodeByID, incoming, map[string]struct{}{})
			if !ok {
				continue
			}
			sourceDeviceID := strings.TrimSpace(sourceNode.Args["device_id"])
			targetDeviceID := strings.TrimSpace(to.Args["device_id"])
			if sourceDeviceID == "" || targetDeviceID == "" {
				continue
			}
			streamKind := streamKindFor(sourceNode.Kind, to.Kind)
			if override := strings.TrimSpace(to.Args["stream_kind"]); override != "" {
				streamKind = override
			}
			if streamKind == "" {
				continue
			}
			if err := p.router.Connect(sourceDeviceID, targetDeviceID, streamKind); err != nil && !errors.Is(err, ErrRouteExists) {
				return planRuntime{}, err
			}
			runtime.routes = append(runtime.routes, Route{
				SourceID:   sourceDeviceID,
				TargetID:   targetDeviceID,
				StreamKind: streamKind,
			})
		}

		// Analyzer nodes publish typed events through the planner sink.
		if to.Kind == NodeAnalyzer && p.analyzer != nil {
			sourceNode, ok := resolveSourceNode(from.ID, nodeByID, incoming, map[string]struct{}{})
			if !ok {
				continue
			}
			sourceDeviceID := strings.TrimSpace(sourceNode.Args["device_id"])
			if sourceDeviceID == "" {
				continue
			}
			analyzerName := strings.TrimSpace(to.Args["name"])
			if analyzerName == "" {
				analyzerName = "sound"
			}
			stop, err := p.analyzer.StartAnalyzer(ctx, sourceDeviceID, analyzerName, p.emitAnalyzerEvent)
			if err != nil {
				return planRuntime{}, err
			}
			if stop != nil {
				runtime.stops = append(runtime.stops, stop)
			}
		}
	}
	return runtime, nil
}

func (p *MediaPlanner) emitAnalyzerEvent(event AnalyzerEvent) {
	if p == nil || p.analyzerSink == nil {
		return
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	if event.Attributes == nil {
		event.Attributes = map[string]string{}
	}
	p.analyzerSink(event)
}

func resolveSourceNode(
	nodeID string,
	nodeByID map[string]MediaNode,
	incoming map[string][]string,
	visited map[string]struct{},
) (MediaNode, bool) {
	if _, seen := visited[nodeID]; seen {
		return MediaNode{}, false
	}
	visited[nodeID] = struct{}{}

	node, ok := nodeByID[nodeID]
	if !ok {
		return MediaNode{}, false
	}
	switch node.Kind {
	case NodeSourceMic, NodeSourceCamera, NodeSourceTTS:
		return node, true
	case NodeFork, NodeAnalyzer:
		// Traverse upstream through processing/topology nodes.
	case NodeSinkSpeaker, NodeSinkDisplay, NodeSinkSTT, NodeRecorder:
		return MediaNode{}, false
	}
	for _, parent := range incoming[nodeID] {
		if out, ok := resolveSourceNode(parent, nodeByID, incoming, visited); ok {
			return out, true
		}
	}
	return MediaNode{}, false
}

func isSinkNode(kind MediaNodeKind) bool {
	switch kind {
	case NodeSinkSpeaker, NodeSinkDisplay, NodeSinkSTT, NodeRecorder:
		return true
	case NodeSourceMic, NodeSourceCamera, NodeSourceTTS, NodeAnalyzer, NodeFork:
		return false
	}
	return false
}

func streamKindFor(source, sink MediaNodeKind) string {
	switch {
	case source == NodeSourceCamera && sink == NodeSinkDisplay:
		return "video"
	case source == NodeSourceMic && sink == NodeSinkSpeaker:
		return "audio"
	case source == NodeSourceMic && sink == NodeSinkSTT:
		return "audio_stt"
	case source == NodeSourceMic && sink == NodeRecorder:
		return "audio_record"
	case source == NodeSourceTTS && sink == NodeSinkSpeaker:
		return "tts_audio"
	default:
		return ""
	}
}
