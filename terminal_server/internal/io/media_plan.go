// Package io manages logical stream-routing state.
package io //nolint:revive

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/eventlog"
)

// FlowNodeKind identifies node behavior in a flow plan.
type FlowNodeKind string

// Flow node kind constants.
const (
	NodeSourceMic       FlowNodeKind = "source.mic"
	NodeSourceCamera    FlowNodeKind = "source.camera"
	NodeSourceTTS       FlowNodeKind = "source.tts"
	NodeSourceSensor    FlowNodeKind = "source.sensor"
	NodeSourceBluetooth FlowNodeKind = "source.bluetooth"
	NodeSourceWiFi      FlowNodeKind = "source.wifi"
	NodeBufferRecent    FlowNodeKind = "buffer.recent"
	NodeFeature         FlowNodeKind = "feature"
	NodeAnalyzer        FlowNodeKind = "analyzer"
	NodeTracker         FlowNodeKind = "tracker"
	NodeLocalizer       FlowNodeKind = "localizer"
	NodeFusion          FlowNodeKind = "fusion"
	NodeMixer           FlowNodeKind = "mixer"
	NodeCompositor      FlowNodeKind = "compositor"
	NodeRecorder        FlowNodeKind = "recorder"
	NodeArtifact        FlowNodeKind = "artifact"
	NodeSinkSpeaker     FlowNodeKind = "sink.speaker"
	NodeSinkDisplay     FlowNodeKind = "sink.display"
	NodeSinkSTT         FlowNodeKind = "sink.stt"
	NodeSinkStore       FlowNodeKind = "sink.store"
	NodeSinkEventBus    FlowNodeKind = "sink.event_bus"
	NodeFork            FlowNodeKind = "fork"
)

// MediaNodeKind remains as a backward-compatible alias.
type MediaNodeKind = FlowNodeKind

// ExecPolicy controls where a flow node should run.
type ExecPolicy string

const (
	// ExecAuto lets the scheduler choose where to run the node.
	ExecAuto ExecPolicy = "auto"
	// ExecPreferClient prefers client/edge execution when supported.
	ExecPreferClient ExecPolicy = "prefer_client"
	// ExecRequireClient requires client/edge execution.
	ExecRequireClient ExecPolicy = "require_client"
	// ExecServerOnly requires server-side execution.
	ExecServerOnly ExecPolicy = "server_only"
)

// FlowNode is one graph node.
type FlowNode struct {
	ID   string
	Kind FlowNodeKind
	Args map[string]string
	Exec ExecPolicy
}

// MediaNode remains as a backward-compatible alias.
type MediaNode = FlowNode

// FlowEdge links two nodes by ID.
type FlowEdge struct {
	From string
	To   string
}

// MediaEdge remains as a backward-compatible alias.
type MediaEdge = FlowEdge

// FlowPlan is a declarative topology graph for media/sensor/radio flows.
type FlowPlan struct {
	Nodes []FlowNode
	Edges []FlowEdge
}

// MediaPlan remains as a backward-compatible alias.
type MediaPlan = FlowPlan

// PlanHandle identifies one installed flow plan.
type PlanHandle string

// DeviceRef points at a concrete terminal.
type DeviceRef struct {
	DeviceID string
}

// Pose represents a world-space pose estimate.
type Pose struct {
	X          float64
	Y          float64
	Z          float64
	Yaw        float64
	Pitch      float64
	Roll       float64
	Confidence float64
}

// LocationEstimate represents a zone or world-space estimate.
type LocationEstimate struct {
	Zone       string
	Pose       *Pose
	RadiusM    float64
	Confidence float64
	Sources    []string
}

// ObservationProvenance records where an observation came from.
type ObservationProvenance struct {
	FlowID             string
	NodeID             string
	ExecSite           string
	ModelID            string
	CalibrationVersion string
}

// ArtifactRef points to optional evidence produced by a flow.
type ArtifactRef struct {
	ID        string
	Kind      string
	Source    DeviceRef
	StartTime time.Time
	EndTime   time.Time
	URI       string
}

// Observation is the typed record emitted by analyzers, trackers, localizers,
// and fusers. It is compact enough to transport over the control plane.
type Observation struct {
	Kind         string
	Subject      string
	SourceDevice DeviceRef
	OccurredAt   time.Time
	Confidence   float64
	Zone         string
	Location     *LocationEstimate
	TrackID      string
	Attributes   map[string]string
	Evidence     []ArtifactRef
	Provenance   ObservationProvenance
}

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

// MediaPlanner installs media/flow plans and compiles them into logical routes.
type MediaPlanner struct {
	mu sync.Mutex

	router          *Router
	nextID          uint64
	active          map[PlanHandle]planRuntime
	analyzer        AnalyzerRunner
	analyzerSink    func(AnalyzerEvent)
	observationSink func(Observation)
}

type planRuntime struct {
	routes []Route
	stops  []func()
}

// ErrInvalidMediaPlan indicates malformed nodes/edges.
var ErrInvalidMediaPlan = errors.New("invalid media plan")

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

// SetObservationSink sets optional observation sink callback.
func (p *MediaPlanner) SetObservationSink(sink func(Observation)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.observationSink = sink
}

// Apply compiles and installs a media plan.
func (p *MediaPlanner) Apply(ctx context.Context, plan MediaPlan) (PlanHandle, error) {
	return p.ApplyFlow(ctx, plan)
}

// ApplyFlow compiles and installs a generalized flow plan.
func (p *MediaPlanner) ApplyFlow(ctx context.Context, plan FlowPlan) (PlanHandle, error) {
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
	eventlog.Emit(ctx, "io.flow.started", slog.LevelInfo, "flow started",
		slog.String("component", "io.flow"),
		slog.String("flow_id", string(handle)),
		slog.Int("route_count", len(compiled.routes)),
		slog.Int("node_count", len(plan.Nodes)),
		slog.Int("edge_count", len(plan.Edges)),
	)
	return handle, nil
}

// Patch replaces an existing plan with a new one.
func (p *MediaPlanner) Patch(ctx context.Context, handle PlanHandle, plan MediaPlan) error {
	return p.PatchFlow(ctx, handle, plan)
}

// PatchFlow replaces an existing flow plan with a new one.
func (p *MediaPlanner) PatchFlow(ctx context.Context, handle PlanHandle, plan FlowPlan) error {
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
		eventlog.Emit(ctx, "io.route.torn_down", slog.LevelInfo, "route torn down",
			slog.String("component", "io.router"),
			slog.String("flow_id", string(handle)),
			slog.String("source_device_id", route.SourceID),
			slog.String("target_device_id", route.TargetID),
			slog.String("stream_kind", route.StreamKind),
		)
	}

	compiled, err := p.compileLocked(ctx, plan)
	if err != nil {
		return err
	}
	p.active[handle] = compiled
	eventlog.Emit(ctx, "io.flow.patched", slog.LevelInfo, "flow patched",
		slog.String("component", "io.flow"),
		slog.String("flow_id", string(handle)),
		slog.Int("route_count", len(compiled.routes)),
		slog.Int("node_count", len(plan.Nodes)),
		slog.Int("edge_count", len(plan.Edges)),
	)
	return nil
}

// Tear uninstalls a plan.
func (p *MediaPlanner) Tear(ctx context.Context, handle PlanHandle) error {
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
		eventlog.Emit(ctx, "io.route.torn_down", slog.LevelInfo, "route torn down",
			slog.String("component", "io.router"),
			slog.String("flow_id", string(handle)),
			slog.String("source_device_id", route.SourceID),
			slog.String("target_device_id", route.TargetID),
			slog.String("stream_kind", route.StreamKind),
		)
	}
	eventlog.Emit(ctx, "io.flow.stopped", slog.LevelInfo, "flow stopped",
		slog.String("component", "io.flow"),
		slog.String("flow_id", string(handle)),
		slog.Int("route_count", len(runtime.routes)),
	)
	return nil
}

func (p *MediaPlanner) compileLocked(ctx context.Context, plan FlowPlan) (planRuntime, error) {
	nodeByID := make(map[string]FlowNode, len(plan.Nodes))
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
		if node.Exec == "" {
			node.Exec = ExecAuto
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
			eventlog.Emit(ctx, "io.route.applied", slog.LevelInfo, "route applied",
				slog.String("component", "io.router"),
				slog.String("source_device_id", sourceDeviceID),
				slog.String("target_device_id", targetDeviceID),
				slog.String("stream_kind", streamKind),
			)
		}

		// Analyzer nodes publish typed events through the planner sinks.
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
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	if event.Attributes == nil {
		event.Attributes = map[string]string{}
	}

	if p != nil && p.analyzerSink != nil {
		p.analyzerSink(event)
	}
	eventlog.Emit(context.Background(), "io.analyzer.event", slog.LevelInfo, "analyzer event",
		slog.String("component", "io.flow"),
		slog.String("kind", strings.TrimSpace(event.Kind)),
		slog.String("subject", strings.TrimSpace(event.Subject)),
		slog.Int("attribute_count", len(event.Attributes)),
	)
	if p != nil && p.observationSink != nil {
		observation := Observation{
			Kind:       strings.TrimSpace(event.Kind),
			Subject:    strings.TrimSpace(event.Subject),
			OccurredAt: event.OccurredAt,
			Attributes: copyStringMap(event.Attributes),
			Provenance: ObservationProvenance{
				ExecSite: "server",
			},
		}
		if observation.Kind == "" {
			observation.Kind = "analyzer.event"
		}
		p.observationSink(observation)
	}
}

func resolveSourceNode(
	nodeID string,
	nodeByID map[string]FlowNode,
	incoming map[string][]string,
	visited map[string]struct{},
) (FlowNode, bool) {
	if _, seen := visited[nodeID]; seen {
		return FlowNode{}, false
	}
	visited[nodeID] = struct{}{}

	node, ok := nodeByID[nodeID]
	if !ok {
		return FlowNode{}, false
	}
	switch node.Kind {
	case NodeSourceMic, NodeSourceCamera, NodeSourceTTS, NodeSourceSensor, NodeSourceBluetooth, NodeSourceWiFi:
		return node, true
	case NodeFork, NodeAnalyzer, NodeFeature, NodeTracker, NodeLocalizer, NodeFusion, NodeMixer, NodeCompositor, NodeBufferRecent, NodeArtifact:
		// Traverse upstream through processing/topology nodes.
	case NodeSinkSpeaker, NodeSinkDisplay, NodeSinkSTT, NodeRecorder, NodeSinkStore, NodeSinkEventBus:
		return FlowNode{}, false
	}
	for _, parent := range incoming[nodeID] {
		if out, ok := resolveSourceNode(parent, nodeByID, incoming, visited); ok {
			return out, true
		}
	}
	return FlowNode{}, false
}

func isSinkNode(kind FlowNodeKind) bool {
	switch kind {
	case NodeSinkSpeaker, NodeSinkDisplay, NodeSinkSTT, NodeRecorder, NodeSinkStore, NodeSinkEventBus:
		return true
	case NodeSourceMic, NodeSourceCamera, NodeSourceTTS, NodeSourceSensor, NodeSourceBluetooth, NodeSourceWiFi, NodeAnalyzer, NodeFork, NodeBufferRecent, NodeFeature, NodeTracker, NodeLocalizer, NodeFusion, NodeMixer, NodeCompositor, NodeArtifact:
		return false
	}
	return false
}

func streamKindFor(source, sink FlowNodeKind) string {
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
	case source == NodeSourceSensor && sink == NodeSinkStore:
		return "sensor"
	case source == NodeSourceBluetooth && sink == NodeSinkStore:
		return "radio_ble"
	case source == NodeSourceWiFi && sink == NodeSinkStore:
		return "radio_wifi"
	default:
		return ""
	}
}

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
