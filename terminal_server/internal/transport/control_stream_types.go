package transport

import (
	"context"
	"errors"
	"strings"
	"time"

	diagnosticsv1 "github.com/curtcox/terminals/terminal_server/gen/go/diagnostics/v1"
	iov1 "github.com/curtcox/terminals/terminal_server/gen/go/io/v1"
	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

var (
	// ErrInvalidClientMessage indicates an unsupported or empty client payload.
	ErrInvalidClientMessage = errors.New("invalid client message")
	// ErrInvalidCommandAction indicates an unsupported command action.
	ErrInvalidCommandAction = errors.New("invalid command action")
	// ErrInvalidCommandKind indicates an unsupported command kind.
	ErrInvalidCommandKind = errors.New("invalid command kind")
	// ErrMissingCommandIntent indicates required command intent is missing.
	ErrMissingCommandIntent = errors.New("missing command intent")
	// ErrMissingCommandText indicates required voice command text is missing.
	ErrMissingCommandText = errors.New("missing command text")
	// ErrMissingCommandDeviceID indicates required command device id is missing.
	ErrMissingCommandDeviceID = errors.New("missing command device id")
	// ErrBugReportIntakeUnavailable indicates bug-report handling is not configured.
	ErrBugReportIntakeUnavailable = errors.New("bug report intake unavailable")
)

// CapabilityUpdateRequest is a transport-neutral capability update payload.
type CapabilityUpdateRequest struct {
	DeviceID     string
	Capabilities map[string]string
}

// CapabilitySnapshotRequest is a full capability baseline with generation.
type CapabilitySnapshotRequest struct {
	DeviceID     string
	Generation   uint64
	Capabilities map[string]string
}

// CapabilityDeltaRequest is an incremental capability update with generation.
type CapabilityDeltaRequest struct {
	DeviceID     string
	Generation   uint64
	Reason       string
	Capabilities map[string]string
}

// HeartbeatRequest is a transport-neutral heartbeat payload.
type HeartbeatRequest struct {
	DeviceID string
}

// SensorDataRequest carries sensor telemetry from device clients.
type SensorDataRequest struct {
	DeviceID string
	UnixMS   int64
	Values   map[string]float64
}

// StreamReadyRequest indicates a media stream is ready on the client side.
type StreamReadyRequest struct {
	StreamID string
}

// WebRTCSignalRequest carries client-originated WebRTC signaling payloads.
type WebRTCSignalRequest struct {
	StreamID   string
	SignalType string
	Payload    string
}

// WebRTCSignalResponse carries server-originated WebRTC signaling payloads.
type WebRTCSignalResponse struct {
	StreamID   string
	SignalType string
	Payload    string
}

// RouteStreamResponse instructs clients to establish or acknowledge media routing.
type RouteStreamResponse struct {
	StreamID       string
	SourceDeviceID string
	TargetDeviceID string
	Kind           string
	Routing        *iov1.StreamRouting
}

// StartStreamResponse instructs clients to start an underlying media stream.
type StartStreamResponse struct {
	StreamID       string
	Kind           string
	SourceDeviceID string
	TargetDeviceID string
	Metadata       map[string]string
	Routing        *iov1.StreamRouting
	AudioMetadata  *iov1.StreamAudioMetadata
}

// StopStreamResponse instructs clients to stop an underlying media stream.
type StopStreamResponse struct {
	StreamID string
}

// InputRequest carries client input events relevant to active scenarios.
type InputRequest struct {
	DeviceID    string
	ComponentID string
	Action      string
	Value       string
	KeyText     string
}

// CommandRequest carries a client-issued scenario command.
type CommandRequest struct {
	RequestID string
	DeviceID  string
	Action    string // "start" (default) or "stop"
	Kind      string // "voice" or "manual"
	Text      string // voice transcript
	Intent    string // explicit scenario intent
	Arguments map[string]string
}

// VoiceAudioRequest carries a chunk of raw microphone audio from a device.
// Chunks are accumulated per device; on IsFinal the server runs STT on the
// assembled buffer and drives the voice command pipeline.
type VoiceAudioRequest struct {
	DeviceID   string
	Audio      []byte
	SampleRate int32
	IsFinal    bool
}

// ObservationRequest carries a typed observation from an edge flow.
type ObservationRequest struct {
	Observation iorouter.Observation
}

// ArtifactAvailableRequest reports an artifact that can be pulled by id.
type ArtifactAvailableRequest struct {
	Artifact iorouter.ArtifactRef
}

// FlowStatsRequest carries edge flow health and resource stats.
type FlowStatsRequest struct {
	FlowID        string
	CPUPct        float64
	MemMB         float64
	DroppedFrames uint64
	State         string
	StateEnum     iov1.FlowState
	Error         string
}

// ClockSampleRequest carries a timing sample from a device clock discipline loop.
type ClockSampleRequest struct {
	DeviceID     string
	ClientUnixMS int64
	ServerUnixMS int64
	ErrorMS      float64
}

// StartFlowResponse instructs a client to start a generalized flow.
type StartFlowResponse struct {
	FlowID string
	Plan   iorouter.FlowPlan
}

// PatchFlowResponse instructs a client to patch an existing flow.
type PatchFlowResponse struct {
	FlowID string
	Plan   iorouter.FlowPlan
}

// StopFlowResponse instructs a client to stop one flow.
type StopFlowResponse struct {
	FlowID string
}

// RequestArtifactResponse asks a client to materialize one artifact.
type RequestArtifactResponse struct {
	ArtifactID string
}

// PlayAudioResponse instructs a specific device to play synthesized audio.
type PlayAudioResponse struct {
	RequestID string
	DeviceID  string
	Audio     []byte
	Format    string
}

// ClientMessage is a one-of control stream message from client to server.
type ClientMessage struct {
	Hello           *HelloRequest
	CapabilitySnap  *CapabilitySnapshotRequest
	CapabilityDelta *CapabilityDeltaRequest
	Register        *RegisterRequest
	Capability      *CapabilityUpdateRequest
	Heartbeat       *HeartbeatRequest
	Sensor          *SensorDataRequest
	StreamReady     *StreamReadyRequest
	WebRTCSignal    *WebRTCSignalRequest
	Input           *InputRequest
	Command         *CommandRequest
	VoiceAudio      *VoiceAudioRequest
	Observation     *ObservationRequest
	ArtifactReady   *ArtifactAvailableRequest
	FlowStats       *FlowStatsRequest
	ClockSample     *ClockSampleRequest
	BugReport       *diagnosticsv1.BugReport
	SessionDeviceID string
}

// ServerMessage is a one-of control stream message from server to client.
type ServerMessage struct {
	HelloAck        *HelloResponse
	CapabilityAck   *CapabilityLifecycleAck
	RegisterAck     *RegisterResponse
	CommandAck      string
	SetUI           *ui.Descriptor
	UpdateUI        *UIUpdate
	StartStream     *StartStreamResponse
	StopStream      *StopStreamResponse
	RouteStream     *RouteStreamResponse
	WebRTCSignal    *WebRTCSignalResponse
	TransitionUI    *UITransition
	PlayAudio       *PlayAudioResponse
	BugReportAck    *diagnosticsv1.BugReportAck
	StartFlow       *StartFlowResponse
	PatchFlow       *PatchFlowResponse
	StopFlow        *StopFlowResponse
	RequestArtifact *RequestArtifactResponse
	Notification    string
	ScenarioStart   string
	ScenarioStop    string
	Data            map[string]string
	ErrorCode       string
	Error           string
	RelayToDeviceID string
}

// UIUpdate carries a server-driven patch to a specific UI component.
type UIUpdate struct {
	ComponentID string
	Node        ui.Descriptor
}

// UITransition carries a UI transition hint for the active device UI.
type UITransition struct {
	Transition string
	DurationMS int32
}

// DeviceAudioPublisher receives live mic-audio chunks keyed by device id so
// scenarios subscribed via scenario.Environment.DeviceAudio can analyze the
// live stream alongside any voice-command pipeline already consuming the
// buffered audio.
type DeviceAudioPublisher interface {
	Publish(deviceID string, chunk []byte)
}

// WebRTCSignalEngine handles server-managed WebRTC signaling per stream/device.
type WebRTCSignalEngine interface {
	HandleSignal(ctx context.Context, req WebRTCSignalEngineRequest) ([]WebRTCSignalEngineResponse, error)
	RemoveStream(streamID string)
}

// WebRTCSignalEngineRequest is input to the server-side signaling engine.
type WebRTCSignalEngineRequest struct {
	StreamID string
	DeviceID string
	Signal   WebRTCSignalRequest
}

// WebRTCSignalEngineResponse is a server-generated outbound signal.
type WebRTCSignalEngineResponse struct {
	TargetDeviceID string
	Signal         WebRTCSignalResponse
}

// BugReportIntake handles persisted bug-report intake for control messages.
type BugReportIntake interface {
	File(context.Context, *diagnosticsv1.BugReport) (*diagnosticsv1.BugReportAck, error)
}

// Actor is a resolved identity principal used by menu composition policy.
type Actor struct {
	Kind string
	ID   string
}

// IdentityService resolves the actor for a given device id.
type IdentityService interface {
	ResolveActor(deviceID string) Actor
}

// MenuAppPolicy filters or rewrites menu app visibility by actor.
type MenuAppPolicy interface {
	VisibleApps(actor Actor, apps []string) []string
}

type allowAllMenuAppPolicy struct{}

func (allowAllMenuAppPolicy) VisibleApps(_ Actor, apps []string) []string {
	return append([]string(nil), apps...)
}

type mediaStreamState struct {
	StreamID          string
	Kind              string
	SourceDeviceID    string
	TargetDeviceID    string
	Metadata          map[string]string
	RoutingWebRTCMode iov1.WebRTCMode
	AudioMetadata     *iov1.StreamAudioMetadata
	Ready             bool
}

type multiWindowResumeState struct {
	PriorScenario string
	PriorUI       ui.Descriptor
	HasPriorUI    bool
}

type menuOverlayState struct {
	ActivationID string
	Policy       overlayInputPolicyConfig
	Suspended    []iorouter.Route
}

type sensorSnapshot struct {
	UnixMS int64
	Values map[string]float64
}

type overlayInputPolicy string

const (
	overlayInputPolicyLive   overlayInputPolicy = "LIVE"
	overlayInputPolicyPaused overlayInputPolicy = "PAUSED"
	overlayInputPolicyMixed  overlayInputPolicy = "MIXED"
)

type overlayInputStream string

const (
	overlayStreamPointer  overlayInputStream = "pointer"
	overlayStreamTouch    overlayInputStream = "touch"
	overlayStreamKeyboard overlayInputStream = "keyboard"
	overlayStreamAudio    overlayInputStream = "audio"
	overlayStreamCamera   overlayInputStream = "camera"
)

type overlayInputPolicyConfig struct {
	Mode      overlayInputPolicy
	Overrides map[overlayInputStream]bool // true keeps main activation live for stream.
}

const (
	defaultTerminalReadDeadline = 180 * time.Millisecond
	defaultTerminalReadInterval = 10 * time.Millisecond
	defaultTerminalUIInterval   = 800 * time.Millisecond
	defaultTerminalReplAdminURL = "http://127.0.0.1:50053"
	defaultPhotoFrameInterval   = 12 * time.Second
	bugReportButtonID           = "global_bug_report_button"
	bugReportActionPrefix       = "bug_report"
	defaultCornerPlacement      = "bottom-right"
	cornerAffordanceLogicalID   = "__affordance.corner__"
	menuOverlayActivationPrefix = "menu-overlay:"
)

// CommandEvent is a bounded audit record of command handling.
type CommandEvent struct {
	RequestID string
	DeviceID  string
	Kind      string
	Action    string
	Intent    string
	Outcome   string
	WhenUnix  int64
}

func enterTransitionForScenario(name string) (UITransition, bool) {
	switch strings.TrimSpace(name) {
	case "terminal":
		return UITransition{Transition: "terminal_enter", DurationMS: 220}, true
	case "photo_frame":
		return UITransition{Transition: "photo_frame_enter", DurationMS: 220}, true
	default:
		return UITransition{}, false
	}
}

func defaultPhotoFrameSlides() []string {
	return []string{
		"https://picsum.photos/id/1015/1920/1080",
		"https://picsum.photos/id/1016/1920/1080",
		"https://picsum.photos/id/1025/1920/1080",
		"https://picsum.photos/id/1035/1920/1080",
		"https://picsum.photos/id/1043/1920/1080",
	}
}

