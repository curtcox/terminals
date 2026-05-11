package transport

import (
	"context"
	"strings"
	"sync"
	"time"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/recording"
	"github.com/curtcox/terminals/terminal_server/internal/replsession"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/terminal"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

type observationSink interface {
	AddObservation(context.Context, iorouter.Observation)
}

// StreamHandler processes control stream messages.
type StreamHandler struct {
	control   *ControlService
	runtime   *scenario.Runtime
	metrics   *Metrics
	mu        sync.Mutex
	seen      map[string]ServerMessage
	seenOrder []string
	seenLimit int

	terminals            *terminal.Manager
	replSessions         *replsession.Service
	terminalReadDeadline time.Duration
	terminalReadInterval time.Duration
	terminalUIInterval   time.Duration
	terminalReplAdminURL string
	uiSession            *UISessionState
	menuOverlayByDevice  map[string]menuOverlayState
	photoFrameSlides     []string
	photoFrameIndexByDev map[string]int
	photoFrameLastByDev  map[string]time.Time
	photoFrameInterval   time.Duration

	mediaControl            *MediaControlState
	sensorsByDevice         map[string]sensorSnapshot
	suspendedClaimsByDevice map[string][]iorouter.Claim
	// routeReplay captures route sets at disconnect so subsequent reconnects
	// can replay StartStream/RouteStream messages even though the live
	// router state was torn down on disconnect.
	routeReplay *RouteReplayStore

	diagnostics    *DiagnosticsIntake
	uiOwners       *uiActionOwnershipTracker
	voicePipeline  *VoicePipeline
	wakeWordDedupe *wakeWordDedupeStage

	identityService   IdentityService
	menuAppPolicy     MenuAppPolicy
	menuOverlayPolicy overlayInputPolicyConfig

	// capability lifecycle collaborator (hello/register/snapshot/delta).
	capabilityLifecycle *CapabilityLifecycle

	// command dispatch collaborator (validation orchestration / audit
	// buffer / post-command fan-out / RememberSetUI).
	commandDispatcher *CommandDispatcher
}

func (h *StreamHandler) activeScenarioName(deviceID string) string {
	if h.runtime == nil || h.runtime.Engine == nil {
		return ""
	}
	name, ok := h.runtime.Engine.Active(strings.TrimSpace(deviceID))
	if !ok {
		return ""
	}
	return name
}

func (h *StreamHandler) captureMultiWindowResume(deviceID, priorScenario string) {
	h.uiSession.CaptureMultiWindowResume(deviceID, priorScenario)
}

func (h *StreamHandler) restoreMultiWindowResume(deviceID string) (*ui.Descriptor, *UITransition, bool) {
	priorScenario, priorUI, hasPriorUI, taken := h.uiSession.TakeMultiWindowResume(deviceID)
	if !taken {
		return nil, nil, false
	}

	var restoredUI *ui.Descriptor
	if hasPriorUI {
		copyUI := priorUI
		restoredUI = &copyUI
	}

	var restoredTransition *UITransition
	if transition, ok := enterTransitionForScenario(priorScenario); ok {
		copyTransition := transition
		restoredTransition = &copyTransition
	}
	return restoredUI, restoredTransition, true
}

// NewStreamHandler creates a handler for control stream messages.
func NewStreamHandler(control *ControlService) *StreamHandler {
	return newStreamHandler(control, nil)
}

// newStreamHandler centralizes StreamHandler initialization. Both public
// constructors delegate here so field defaults stay in one place.
func newStreamHandler(control *ControlService, runtime *scenario.Runtime) *StreamHandler {
	handler := &StreamHandler{
		// transport dispatch / metrics
		control:   control,
		runtime:   runtime,
		metrics:   &Metrics{},
		seen:      map[string]ServerMessage{},
		seenLimit: 1024,

		// terminal / repl session support
		terminals:            terminal.NewManager(),
		terminalReadDeadline: defaultTerminalReadDeadline,
		terminalReadInterval: defaultTerminalReadInterval,
		terminalUIInterval:   defaultTerminalUIInterval,
		terminalReplAdminURL: defaultTerminalReplAdminURL,

		// UI session state
		uiSession:           NewUISessionState(),
		menuOverlayByDevice: map[string]menuOverlayState{},

		// photo-frame scenario defaults
		photoFrameSlides:     defaultPhotoFrameSlides(),
		photoFrameIndexByDev: map[string]int{},
		photoFrameLastByDev:  map[string]time.Time{},
		photoFrameInterval:   defaultPhotoFrameInterval,

		// media control / route replay / voice
		mediaControl:            NewMediaControlState(),
		sensorsByDevice:         map[string]sensorSnapshot{},
		suspendedClaimsByDevice: map[string][]iorouter.Claim{},
		routeReplay:             NewRouteReplayStore(),

		// diagnostics / collaborators with default impls
		uiOwners:          newUIActionOwnershipTracker(),
		wakeWordDedupe:    newWakeWordDedupeStage(0, ""),
		menuAppPolicy:     allowAllMenuAppPolicy{},
		menuOverlayPolicy: defaultOverlayInputPolicy(),
	}
	handler.replSessions = replsession.NewService(handler.terminals)
	handler.capabilityLifecycle = NewCapabilityLifecycle(control)
	handler.commandDispatcher = NewCommandDispatcher(handler, handler.handleCommand, 200)
	handler.diagnostics = NewDiagnosticsIntake(nil)
	handler.voicePipeline = NewVoicePipeline(handler)
	return handler
}

// SetDeviceAudioPublisher wires a live audio publisher so incoming VoiceAudio
// chunks are fanned out to scenarios that need to analyze the device's
// mic stream in real time. Safe to call once before any control streams are
// handled; subsequent calls replace the publisher.
func (h *StreamHandler) SetDeviceAudioPublisher(pub DeviceAudioPublisher) {
	h.voicePipeline.SetDeviceAudioPublisher(pub)
}

// NewStreamHandlerWithRuntime creates a handler with scenario runtime support.
func NewStreamHandlerWithRuntime(control *ControlService, runtime *scenario.Runtime) *StreamHandler {
	return newStreamHandler(control, runtime)
}

// SetRecordingManager wires stream recording lifecycle hooks used when routes
// start and stop. Passing nil restores the no-op manager.
func (h *StreamHandler) SetRecordingManager(mgr recording.Manager) {
	h.mediaControl.SetRecordingManager(mgr)
}

// SetWebRTCSignalEngine wires a server-side signaling engine used for streams
// marked as server-managed. Passing nil disables server-managed signaling.
func (h *StreamHandler) SetWebRTCSignalEngine(engine WebRTCSignalEngine) {
	h.mediaControl.SetWebRTCSignalEngine(engine)
}

// SetBugReportIntake wires a persisted bug-report intake for control streams.
// Delegates to the DiagnosticsIntake collaborator; kept as a passthrough so
// existing callers continue to compile.
func (h *StreamHandler) SetBugReportIntake(intake BugReportIntake) {
	h.diagnostics.SetIntake(intake)
}

// SetIdentityService wires actor resolution used by menu composition.
func (h *StreamHandler) SetIdentityService(identity IdentityService) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.identityService = identity
}

// SetMenuAppPolicy configures actor-aware menu app filtering.
func (h *StreamHandler) SetMenuAppPolicy(policy MenuAppPolicy) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if policy == nil {
		h.menuAppPolicy = allowAllMenuAppPolicy{}
		return
	}
	h.menuAppPolicy = policy
}

