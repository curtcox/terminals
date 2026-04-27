// Package admin serves a lightweight web dashboard and JSON admin APIs.
package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/appruntime"
	"github.com/curtcox/terminals/terminal_server/internal/capability"
	"github.com/curtcox/terminals/terminal_server/internal/config"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/diagnostics/bugreport"
	"github.com/curtcox/terminals/terminal_server/internal/eventlog"
	"github.com/curtcox/terminals/terminal_server/internal/eventlog/query"
	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/replai"
	"github.com/curtcox/terminals/terminal_server/internal/replsession"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/transport"
	"github.com/curtcox/terminals/terminal_server/internal/world"
)

type replSessionService interface {
	ListSessions(ctx context.Context, req replsession.ListSessionsRequest) (*replsession.ListSessionsResponse, error)
	GetSession(ctx context.Context, req replsession.GetSessionRequest) (*replsession.GetSessionResponse, error)
	TerminateSession(ctx context.Context, req replsession.TerminateSessionRequest) (*replsession.TerminateSessionResponse, error)
}

type replAIService interface {
	ListProviders(ctx context.Context, req replai.ListProvidersRequest) (*replai.ListProvidersResponse, error)
	ListModels(ctx context.Context, req replai.ListModelsRequest) (*replai.ListModelsResponse, error)
	GetSelection(ctx context.Context, req replai.GetSelectionRequest) (*replai.GetSelectionResponse, error)
	SetSelection(ctx context.Context, req replai.SetSelectionRequest) (*replai.SetSelectionResponse, error)
	Ask(ctx context.Context, req replai.AskRequest) (*replai.AskResponse, error)
	Generate(ctx context.Context, req replai.GenerateRequest) (*replai.GenerateResponse, error)
	GetContext(ctx context.Context, req replai.GetContextRequest) (*replai.GetContextResponse, error)
	AddContext(ctx context.Context, req replai.AddContextRequest) (*replai.AddContextResponse, error)
	PinContext(ctx context.Context, req replai.PinContextRequest) (*replai.PinContextResponse, error)
	UnpinContext(ctx context.Context, req replai.UnpinContextRequest) (*replai.UnpinContextResponse, error)
	ClearContext(ctx context.Context, req replai.ClearContextRequest) (*replai.ClearContextResponse, error)
	GetPolicy(ctx context.Context, req replai.GetPolicyRequest) (*replai.GetPolicyResponse, error)
	SetPolicy(ctx context.Context, req replai.SetPolicyRequest) (*replai.SetPolicyResponse, error)
	GetThread(ctx context.Context, req replai.GetThreadRequest) (*replai.GetThreadResponse, error)
	ResetThread(ctx context.Context, req replai.ResetThreadRequest) (*replai.ResetThreadResponse, error)
}

type worldAdminModel interface {
	ListGeometries(ctx context.Context) []world.DeviceGeometry
	CalibrationHistory(ctx context.Context, deviceID string, limit int) ([]world.CalibrationEvent, error)
	VerifyDevice(ctx context.Context, deviceID string, method string) error
}

// Handler serves a lightweight admin dashboard and JSON control APIs.
type Handler struct {
	control     *transport.ControlService
	runtime     *scenario.Runtime
	repl        replSessionService
	ai          replAIService
	appRuntime  *appruntime.Runtime
	syncAppDefs func()
	devices     *device.Manager
	bugReports  *bugreport.Service
	capability  *capability.Service
	world       worldAdminModel
	trust       trustService
	cfg         config.Config
	now         func() time.Time
}

// NewHandler builds an admin handler with dashboard and API routes.
func NewHandler(
	control *transport.ControlService,
	runtime *scenario.Runtime,
	repl replSessionService,
	ai replAIService,
	appRuntime *appruntime.Runtime,
	syncAppDefs func(),
	devices *device.Manager,
	cfg config.Config,
	worldModel worldAdminModel,
	trustSvc trustService,
) http.Handler {
	h := &Handler{
		control:     control,
		runtime:     runtime,
		repl:        repl,
		ai:          ai,
		appRuntime:  appRuntime,
		syncAppDefs: syncAppDefs,
		devices:     devices,
		trust:       trustSvc,
		bugReports:  bugreport.NewService(cfg.LogDir, devices, runtime),
		capability:  capability.NewService(),
		world:       worldModel,
		cfg:         cfg,
		now:         time.Now,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/admin", h.handleDashboard)
	mux.HandleFunc("/admin/api/status", h.handleStatus)
	mux.HandleFunc("/admin/api/devices", h.handleDevices)
	mux.HandleFunc("/admin/api/devices/placement", h.handleDevicePlacementUpdate)
	mux.HandleFunc("/admin/api/scenarios", h.handleScenarios)
	mux.HandleFunc("/admin/api/scenarios/start", h.handleStartScenario)
	mux.HandleFunc("/admin/api/scenarios/stop", h.handleStopScenario)
	mux.HandleFunc("/admin/api/activations", h.handleActivations)
	mux.HandleFunc("/admin/api/repl/sessions", h.handleReplSessions)
	mux.HandleFunc("/admin/api/repl/sessions/", h.handleReplSession)
	mux.HandleFunc("/admin/api/repl/ai/providers", h.handleReplAIProviders)
	mux.HandleFunc("/admin/api/repl/ai/models", h.handleReplAIModels)
	mux.HandleFunc("/admin/api/repl/ai/selection", h.handleReplAISelection)
	mux.HandleFunc("/admin/api/repl/ai/ask", h.handleReplAIAsk)
	mux.HandleFunc("/admin/api/repl/ai/gen", h.handleReplAIGenerate)
	mux.HandleFunc("/admin/api/repl/ai/context", h.handleReplAIContext)
	mux.HandleFunc("/admin/api/repl/ai/context/pin", h.handleReplAIPinContext)
	mux.HandleFunc("/admin/api/repl/ai/context/unpin", h.handleReplAIUnpinContext)
	mux.HandleFunc("/admin/api/repl/ai/context/clear", h.handleReplAIClearContext)
	mux.HandleFunc("/admin/api/repl/ai/policy", h.handleReplAIPolicy)
	mux.HandleFunc("/admin/api/repl/ai/history", h.handleReplAIHistory)
	mux.HandleFunc("/admin/api/repl/ai/reset", h.handleReplAIReset)
	mux.HandleFunc("/admin/api/identity", h.handleIdentity)
	mux.HandleFunc("/admin/api/identity/show", h.handleIdentityShow)
	mux.HandleFunc("/admin/api/identity/groups", h.handleIdentityGroups)
	mux.HandleFunc("/admin/api/identity/prefs", h.handleIdentityPreferences)
	mux.HandleFunc("/admin/api/identity/resolve", h.handleIdentityResolve)
	mux.HandleFunc("/admin/api/identity/ack", h.handleIdentityAcknowledgements)
	mux.HandleFunc("/admin/api/session", h.handleInteractiveSessions)
	mux.HandleFunc("/admin/api/session/show", h.handleInteractiveSessionShow)
	mux.HandleFunc("/admin/api/session/members", h.handleInteractiveSessionMembers)
	mux.HandleFunc("/admin/api/session/create", h.handleInteractiveSessionCreate)
	mux.HandleFunc("/admin/api/session/join", h.handleInteractiveSessionJoin)
	mux.HandleFunc("/admin/api/session/leave", h.handleInteractiveSessionLeave)
	mux.HandleFunc("/admin/api/session/attach", h.handleInteractiveSessionAttachDevice)
	mux.HandleFunc("/admin/api/session/detach", h.handleInteractiveSessionDetachDevice)
	mux.HandleFunc("/admin/api/session/control/request", h.handleInteractiveSessionRequestControl)
	mux.HandleFunc("/admin/api/session/control/grant", h.handleInteractiveSessionGrantControl)
	mux.HandleFunc("/admin/api/session/control/revoke", h.handleInteractiveSessionRevokeControl)
	mux.HandleFunc("/admin/api/message/rooms", h.handleMessageRooms)
	mux.HandleFunc("/admin/api/message/room", h.handleMessageRoom)
	mux.HandleFunc("/admin/api/message", h.handleMessages)
	mux.HandleFunc("/admin/api/message/get", h.handleMessageGet)
	mux.HandleFunc("/admin/api/message/unread", h.handleMessageUnread)
	mux.HandleFunc("/admin/api/message/post", h.handleMessagePost)
	mux.HandleFunc("/admin/api/message/dm", h.handleMessageDirect)
	mux.HandleFunc("/admin/api/message/thread", h.handleMessageThread)
	mux.HandleFunc("/admin/api/message/ack", h.handleMessageAck)
	mux.HandleFunc("/admin/api/board", h.handleBoard)
	mux.HandleFunc("/admin/api/board/post", h.handleBoardPost)
	mux.HandleFunc("/admin/api/board/pin", h.handleBoardPin)
	mux.HandleFunc("/admin/api/artifact", h.handleArtifacts)
	mux.HandleFunc("/admin/api/artifact/get", h.handleArtifactGet)
	mux.HandleFunc("/admin/api/artifact/history", h.handleArtifactHistory)
	mux.HandleFunc("/admin/api/artifact/create", h.handleArtifactCreate)
	mux.HandleFunc("/admin/api/artifact/patch", h.handleArtifactPatch)
	mux.HandleFunc("/admin/api/artifact/replace", h.handleArtifactReplace)
	mux.HandleFunc("/admin/api/artifact/template/save", h.handleArtifactTemplateSave)
	mux.HandleFunc("/admin/api/artifact/template/apply", h.handleArtifactTemplateApply)
	mux.HandleFunc("/admin/api/canvas", h.handleCanvas)
	mux.HandleFunc("/admin/api/canvas/annotate", h.handleCanvasAnnotate)
	mux.HandleFunc("/admin/api/search", h.handleSearch)
	mux.HandleFunc("/admin/api/search/timeline", h.handleSearchTimeline)
	mux.HandleFunc("/admin/api/search/related", h.handleSearchRelated)
	mux.HandleFunc("/admin/api/search/recent", h.handleSearchRecent)
	mux.HandleFunc("/admin/api/memory", h.handleMemoryRecall)
	mux.HandleFunc("/admin/api/memory/stream", h.handleMemoryStream)
	mux.HandleFunc("/admin/api/memory/remember", h.handleMemoryRemember)
	mux.HandleFunc("/admin/api/placement", h.handlePlacement)
	mux.HandleFunc("/admin/api/cohort", h.handleCohorts)
	mux.HandleFunc("/admin/api/cohort/upsert", h.handleCohortUpsert)
	mux.HandleFunc("/admin/api/cohort/del", h.handleCohortDelete)
	mux.HandleFunc("/admin/api/ui/views", h.handleUIViews)
	mux.HandleFunc("/admin/api/ui/views/upsert", h.handleUIViewUpsert)
	mux.HandleFunc("/admin/api/ui/views/del", h.handleUIViewDelete)
	mux.HandleFunc("/admin/api/ui/push", h.handleUIPush)
	mux.HandleFunc("/admin/api/ui/patch", h.handleUIPatch)
	mux.HandleFunc("/admin/api/ui/transition", h.handleUITransition)
	mux.HandleFunc("/admin/api/ui/broadcast", h.handleUIBroadcast)
	mux.HandleFunc("/admin/api/ui/subscribe", h.handleUISubscribe)
	mux.HandleFunc("/admin/api/ui/snapshot", h.handleUISnapshot)
	mux.HandleFunc("/admin/api/recent", h.handleRecent)
	mux.HandleFunc("/admin/api/sim/devices", h.handleSimDevices)
	mux.HandleFunc("/admin/api/sim/devices/new", h.handleSimDeviceNew)
	mux.HandleFunc("/admin/api/sim/devices/rm", h.handleSimDeviceRemove)
	mux.HandleFunc("/admin/api/sim/input", h.handleSimInput)
	mux.HandleFunc("/admin/api/sim/ui", h.handleSimUI)
	mux.HandleFunc("/admin/api/sim/expect", h.handleSimExpect)
	mux.HandleFunc("/admin/api/sim/record", h.handleSimRecord)
	mux.HandleFunc("/admin/api/scripts/dry-run", h.handleScriptsDryRun)
	mux.HandleFunc("/admin/api/scripts/run", h.handleScriptsRun)
	mux.HandleFunc("/admin/api/world/calibration", h.handleWorldCalibration)
	mux.HandleFunc("/admin/api/world/verify", h.handleWorldVerify)
	mux.HandleFunc("/admin/api/store/get", h.handleStoreGet)
	mux.HandleFunc("/admin/api/store/ns", h.handleStoreNamespaces)
	mux.HandleFunc("/admin/api/store/ls", h.handleStoreList)
	mux.HandleFunc("/admin/api/store/put", h.handleStorePut)
	mux.HandleFunc("/admin/api/store/del", h.handleStoreDelete)
	mux.HandleFunc("/admin/api/store/watch", h.handleStoreWatch)
	mux.HandleFunc("/admin/api/store/bind", h.handleStoreBind)
	mux.HandleFunc("/admin/api/bus", h.handleBusTail)
	mux.HandleFunc("/admin/api/bus/emit", h.handleBusEmit)
	mux.HandleFunc("/admin/api/bus/replay", h.handleBusReplay)
	mux.HandleFunc("/admin/api/handlers", h.handleHandlers)
	mux.HandleFunc("/admin/api/handlers/on", h.handleHandlersOn)
	mux.HandleFunc("/admin/api/handlers/off", h.handleHandlersOff)
	mux.HandleFunc("/admin/api/scenarios/inline", h.handleInlineScenarios)
	mux.HandleFunc("/admin/api/scenarios/inline/define", h.handleInlineScenarioDefine)
	mux.HandleFunc("/admin/api/scenarios/inline/undefine", h.handleInlineScenarioUndefine)
	mux.HandleFunc("/admin/api/apps", h.handleApps)
	mux.HandleFunc("/admin/api/apps/reload", h.handleReloadApp)
	mux.HandleFunc("/admin/api/apps/rollback", h.handleRollbackApp)
	mux.HandleFunc("/admin/api/apps/migrate/status", h.handleAppMigrationStatus)
	mux.HandleFunc("/admin/api/apps/migrate/retry", h.handleAppMigrationRetry)
	mux.HandleFunc("/admin/api/apps/migrate/abort", h.handleAppMigrationAbort)
	mux.HandleFunc("/admin/api/apps/migrate/reconcile", h.handleAppMigrationReconcile)
	mux.HandleFunc("/admin/api/trust/keys", h.handleTrustKeys)
	mux.HandleFunc("/admin/api/trust/keys/confirm", h.handleTrustKeyConfirm)
	mux.HandleFunc("/admin/api/trust/keys/revoke", h.handleTrustKeyRevoke)
	mux.HandleFunc("/admin/api/trust/keys/archive", h.handleTrustKeyArchive)
	mux.HandleFunc("/admin/api/trust/keys/rotate", h.handleTrustKeyRotateAccept)
	mux.HandleFunc("/admin/api/trust/keys/rotate/rollback", h.handleTrustKeyRotateRollback)
	mux.HandleFunc("/admin/api/trust/keys/rotate-installer", h.handleTrustRotateInstaller)
	mux.HandleFunc("/admin/api/trust/rotations", h.handleTrustRotations)
	mux.HandleFunc("/admin/api/trust/verify", h.handleTrustVerify)
	mux.HandleFunc("/admin/api/trust/log", h.handleTrustLog)
	mux.HandleFunc("/admin/logs", h.handleLogs)
	mux.HandleFunc("/admin/logs.jsonl", h.handleLogsJSONL)
	mux.HandleFunc("/admin/logs/trace/", h.handleLogsTrace)
	mux.HandleFunc("/admin/logs/activation/", h.handleLogsActivation)
	mux.HandleFunc("/admin/bugs", h.handleBugsListPage)
	mux.HandleFunc("/admin/bugs/", h.handleBugDetailPage)
	mux.HandleFunc("/admin/bugs/new", h.handleBugNewPage)
	mux.HandleFunc("/admin/api/bugs", h.handleBugsListAPI)
	mux.HandleFunc("/admin/api/bugs/", h.handleBugReportAPI)
	mux.HandleFunc("/bug", h.handleBugNewPage)
	mux.HandleFunc("/bug/intake", h.handleBugIntake)
	mux.HandleFunc("/admin/api/chat/messages", h.handleChatMessages)
	mux.HandleFunc("/admin/api/chat/send", h.handleChatSend)
	return h.withRequestLogging(mux)
}

func (h *Handler) handleChatMessages(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	msgs := scenario.SharedRoom().Messages()
	out := make([]map[string]any, 0, len(msgs))
	for _, msg := range msgs {
		out = append(out, map[string]any{
			"id":        msg.ID,
			"device_id": msg.DeviceID,
			"name":      msg.Name,
			"text":      msg.Text,
			"at":        msg.At.UTC().Format(time.RFC3339),
		})
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"messages":     out,
		"participants": scenario.SharedRoom().Participants(),
	})
}

func (h *Handler) handleChatSend(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		DeviceID string `json:"device_id"`
		Name     string `json:"name"`
		Text     string `json:"text"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	body.DeviceID = strings.TrimSpace(body.DeviceID)
	body.Name = strings.TrimSpace(body.Name)
	body.Text = strings.TrimSpace(body.Text)
	if body.DeviceID == "" || body.Text == "" {
		h.writeJSONError(w, http.StatusBadRequest, "device_id and text are required")
		return
	}
	room := scenario.SharedRoom()
	if body.Name != "" {
		room.SetName(body.DeviceID, body.Name)
	}
	msg, ok := room.Post(body.DeviceID, body.Name, body.Text)
	if !ok {
		h.writeJSONError(w, http.StatusBadRequest, "message rejected")
		return
	}
	// Broadcast UI updates to any connected chat participants.
	transport.BroadcastChatMessagesUpdate()
	h.writeJSON(w, http.StatusOK, map[string]any{
		"message": map[string]any{
			"id":        msg.ID,
			"device_id": msg.DeviceID,
			"name":      msg.Name,
			"text":      msg.Text,
			"at":        msg.At.UTC().Format(time.RFC3339),
		},
	})
}

func (h *Handler) handleIdentity(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"identities": h.capability.ListIdentities()})
}

func (h *Handler) handleIdentityShow(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	identityRef := strings.TrimSpace(req.URL.Query().Get("identity"))
	if identityRef == "" {
		h.writeJSONError(w, http.StatusBadRequest, "identity is required")
		return
	}
	identity, ok := h.capability.GetIdentity(identityRef)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "identity not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"identity": identity})
}

func (h *Handler) handleIdentityGroups(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"groups": h.capability.ListGroups()})
}

func (h *Handler) handleIdentityPreferences(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	identityRef := strings.TrimSpace(req.URL.Query().Get("identity"))
	if identityRef == "" {
		h.writeJSONError(w, http.StatusBadRequest, "identity is required")
		return
	}
	prefs, ok := h.capability.GetPreferences(identityRef)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "identity not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"identity": identityRef, "preferences": prefs})
}

func (h *Handler) handleIdentityResolve(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	audience := strings.TrimSpace(req.URL.Query().Get("audience"))
	h.writeJSON(w, http.StatusOK, map[string]any{
		"audience":   audience,
		"identities": h.capability.ResolveAudience(audience),
	})
}

func (h *Handler) handleIdentityAcknowledgements(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		subjectRef := strings.TrimSpace(req.URL.Query().Get("subject_ref"))
		h.writeJSON(w, http.StatusOK, map[string]any{
			"subject_ref":      subjectRef,
			"acknowledgements": h.capability.GetAcknowledgements(subjectRef),
		})
	case http.MethodPost:
		if err := req.ParseForm(); err != nil {
			h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
			return
		}
		ack, ok := h.capability.RecordAcknowledgement(req.Form.Get("subject_ref"), req.Form.Get("actor"), req.Form.Get("mode"))
		if !ok {
			h.writeJSONError(w, http.StatusBadRequest, "invalid acknowledgement")
			return
		}
		h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "ack": ack})
	default:
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleInteractiveSessions(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"sessions": h.capability.ListSessions()})
}

func (h *Handler) handleInteractiveSessionShow(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	sessionID := strings.TrimSpace(req.URL.Query().Get("session_id"))
	session, ok := h.capability.GetSession(sessionID)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"session": session})
}

func (h *Handler) handleInteractiveSessionMembers(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	sessionID := strings.TrimSpace(req.URL.Query().Get("session_id"))
	participants, ok := h.capability.ListSessionParticipants(sessionID)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"session_id":   sessionID,
		"participants": participants,
	})
}

func (h *Handler) handleInteractiveSessionCreate(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	session := h.capability.CreateSession(req.Form.Get("kind"), req.Form.Get("target"))
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "session": session})
}

func (h *Handler) handleInteractiveSessionJoin(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	session, ok := h.capability.JoinSession(req.Form.Get("session_id"), req.Form.Get("participant"))
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "session": session})
}

func (h *Handler) handleInteractiveSessionLeave(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	session, ok := h.capability.LeaveSession(req.Form.Get("session_id"), req.Form.Get("participant"))
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "session": session})
}

func (h *Handler) handleInteractiveSessionAttachDevice(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	session, ok := h.capability.AttachDevice(req.Form.Get("session_id"), req.Form.Get("device_ref"))
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "session": session})
}

func (h *Handler) handleInteractiveSessionDetachDevice(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	session, ok := h.capability.DetachDevice(req.Form.Get("session_id"), req.Form.Get("device_ref"))
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "session": session})
}

func (h *Handler) handleInteractiveSessionRequestControl(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	session, ok := h.capability.RequestControl(req.Form.Get("session_id"), req.Form.Get("participant"), req.Form.Get("control_type"))
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "session": session})
}

func (h *Handler) handleInteractiveSessionGrantControl(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	session, ok := h.capability.GrantControl(req.Form.Get("session_id"), req.Form.Get("participant"), req.Form.Get("granted_by"), req.Form.Get("control_type"))
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "session": session})
}

func (h *Handler) handleInteractiveSessionRevokeControl(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	session, ok := h.capability.RevokeControl(req.Form.Get("session_id"), req.Form.Get("participant"), req.Form.Get("revoked_by"))
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "session": session})
}

func (h *Handler) handleMessages(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"messages": h.capability.ListMessages(req.URL.Query().Get("room"))})
}

func (h *Handler) handleMessageRooms(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"rooms": h.capability.ListMessageRooms()})
}

func (h *Handler) handleMessageRoom(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		roomRef := strings.TrimSpace(req.URL.Query().Get("room"))
		if roomRef == "" {
			h.writeJSONError(w, http.StatusBadRequest, "room is required")
			return
		}
		room, ok := h.capability.GetMessageRoom(roomRef)
		if !ok {
			h.writeJSONError(w, http.StatusNotFound, "room not found")
			return
		}
		h.writeJSON(w, http.StatusOK, map[string]any{"room": room})
	case http.MethodPost:
		if err := req.ParseForm(); err != nil {
			h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
			return
		}
		name := strings.TrimSpace(req.Form.Get("name"))
		if name == "" {
			h.writeJSONError(w, http.StatusBadRequest, "name is required")
			return
		}
		room := h.capability.CreateMessageRoom(name)
		h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "room": room})
	default:
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleMessageGet(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	messageID := strings.TrimSpace(req.URL.Query().Get("message_id"))
	if messageID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "message_id is required")
		return
	}
	message, ok := h.capability.GetMessage(messageID)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "message not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"message": message})
}

func (h *Handler) handleMessageUnread(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	identityID := strings.TrimSpace(req.URL.Query().Get("identity_id"))
	h.writeJSON(w, http.StatusOK, map[string]any{
		"identity_id": identityID,
		"messages":    h.capability.ListUnreadMessages(identityID, req.URL.Query().Get("room")),
	})
}

func (h *Handler) handleMessagePost(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	message := h.capability.PostMessage(req.Form.Get("room"), req.Form.Get("text"))
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "message": message})
}

func (h *Handler) handleMessageDirect(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	message := h.capability.SendDirectMessage(req.Form.Get("target_ref"), req.Form.Get("text"))
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "message": message})
}

func (h *Handler) handleMessageThread(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	message, ok := h.capability.ReplyMessageThread(req.Form.Get("root_ref"), req.Form.Get("text"))
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "root message not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "message": message})
}

func (h *Handler) handleMessageAck(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	ack, ok := h.capability.AcknowledgeMessage(req.Form.Get("identity_id"), req.Form.Get("message_id"))
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "message not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "ack": ack})
}

func (h *Handler) handleBoard(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"items": h.capability.ListBoard(req.URL.Query().Get("board"))})
}

func (h *Handler) handleBoardPost(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	item := h.capability.PostBoard(req.Form.Get("board"), req.Form.Get("text"))
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "item": item})
}

func (h *Handler) handleBoardPin(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	item := h.capability.PinBoard(req.Form.Get("board"), req.Form.Get("text"))
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "item": item})
}

func (h *Handler) handleArtifacts(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"artifacts": h.capability.ListArtifacts()})
}

func (h *Handler) handleArtifactGet(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	artifactID := strings.TrimSpace(req.URL.Query().Get("artifact_id"))
	if artifactID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "artifact_id is required")
		return
	}
	artifact, ok := h.capability.GetArtifact(artifactID)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "artifact not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"artifact": artifact})
}

func (h *Handler) handleArtifactHistory(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	artifactID := strings.TrimSpace(req.URL.Query().Get("artifact_id"))
	if artifactID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "artifact_id is required")
		return
	}
	history, ok := h.capability.ArtifactHistory(artifactID)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "artifact not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"artifact_id": artifactID, "versions": history})
}

func (h *Handler) handleArtifactCreate(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	artifact := h.capability.CreateArtifact(req.Form.Get("kind"), req.Form.Get("title"))
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "artifact": artifact})
}

func (h *Handler) handleArtifactPatch(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	artifact, ok := h.capability.PatchArtifact(req.Form.Get("artifact_id"), req.Form.Get("title"))
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "artifact not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "artifact": artifact})
}

func (h *Handler) handleArtifactReplace(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	artifact, ok := h.capability.ReplaceArtifact(req.Form.Get("artifact_id"), req.Form.Get("title"))
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "artifact not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "artifact": artifact})
}

func (h *Handler) handleArtifactTemplateSave(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	template, ok := h.capability.SaveArtifactTemplate(req.Form.Get("name"), req.Form.Get("source_artifact_id"))
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "artifact template source not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "template": template})
}

func (h *Handler) handleArtifactTemplateApply(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	artifact, ok := h.capability.ApplyArtifactTemplate(req.Form.Get("name"), req.Form.Get("target_artifact_id"))
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "artifact template or target not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "artifact": artifact})
}

func (h *Handler) handleCanvas(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"annotations": h.capability.ListCanvas(req.URL.Query().Get("canvas"))})
}

func (h *Handler) handleCanvasAnnotate(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	annotation := h.capability.AnnotateCanvas(req.Form.Get("canvas"), req.Form.Get("text"))
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "annotation": annotation})
}

func (h *Handler) handleSearch(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"results": h.capability.Search(req.URL.Query().Get("q"))})
}

func (h *Handler) handleSearchTimeline(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"items": h.capability.SearchTimeline(req.URL.Query().Get("scope"))})
}

func (h *Handler) handleSearchRelated(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"results": h.capability.SearchRelated(req.URL.Query().Get("subject"))})
}

func (h *Handler) handleSearchRecent(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"items": h.capability.SearchRecent(req.URL.Query().Get("scope"), 20)})
}

func (h *Handler) handleMemoryRecall(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"memories": h.capability.Recall(req.URL.Query().Get("q"))})
}

func (h *Handler) handleMemoryStream(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"memories": h.capability.MemoryStream(req.URL.Query().Get("scope"))})
}

func (h *Handler) handleMemoryRemember(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	memory := h.capability.Remember(req.Form.Get("scope"), req.Form.Get("text"))
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "memory": memory})
}

func (h *Handler) handlePlacement(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	placements := make([]map[string]any, 0)
	for _, d := range h.devices.List() {
		placements = append(placements, map[string]any{
			"device_id": d.DeviceID,
			"zone":      d.Placement.Zone,
			"roles":     append([]string(nil), d.Placement.Roles...),
			"mobility":  d.Placement.Mobility,
			"affinity":  d.Placement.Affinity,
		})
	}
	sort.Slice(placements, func(i, j int) bool {
		return fmt.Sprintf("%v", placements[i]["device_id"]) < fmt.Sprintf("%v", placements[j]["device_id"])
	})
	h.writeJSON(w, http.StatusOK, map[string]any{"placements": placements})
}

func (h *Handler) handleCohorts(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	name := strings.TrimSpace(req.URL.Query().Get("name"))
	if name == "" {
		h.writeJSON(w, http.StatusOK, map[string]any{"cohorts": h.capability.CohortList()})
		return
	}
	cohort, ok := h.capability.CohortGet(name)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "cohort not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"cohort":  cohort,
		"members": h.resolveCohortMembers(cohort.Selectors),
	})
}

func (h *Handler) handleCohortUpsert(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	name := strings.TrimSpace(req.Form.Get("name"))
	if name == "" {
		h.writeJSONError(w, http.StatusBadRequest, "name is required")
		return
	}
	selectors := parseSelectors(req.Form.Get("selectors"))
	cohort := h.capability.CohortUpsert(name, selectors)
	h.writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"cohort":  cohort,
		"members": h.resolveCohortMembers(cohort.Selectors),
	})
}

func (h *Handler) handleCohortDelete(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	name := strings.TrimSpace(req.Form.Get("name"))
	if name == "" {
		h.writeJSONError(w, http.StatusBadRequest, "name is required")
		return
	}
	deleted := h.capability.CohortDelete(name)
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "deleted": deleted})
}

func (h *Handler) handleUIViews(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	viewID := strings.TrimSpace(req.URL.Query().Get("view_id"))
	if viewID == "" {
		h.writeJSON(w, http.StatusOK, map[string]any{"views": h.capability.UIViewList()})
		return
	}
	view, ok := h.capability.UIViewGet(viewID)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "ui view not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"view": view})
}

func (h *Handler) handleUIViewUpsert(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	viewID := strings.TrimSpace(req.Form.Get("view_id"))
	if viewID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "view_id is required")
		return
	}
	view := h.capability.UIViewUpsert(
		viewID,
		strings.TrimSpace(req.Form.Get("root_id")),
		strings.TrimSpace(req.Form.Get("descriptor")),
	)
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "view": view})
}

func (h *Handler) handleUIViewDelete(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	viewID := strings.TrimSpace(req.Form.Get("view_id"))
	if viewID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "view_id is required")
		return
	}
	deleted := h.capability.UIViewDelete(viewID)
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "deleted": deleted})
}

func (h *Handler) handleUIPush(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	deviceID := strings.TrimSpace(req.Form.Get("device_id"))
	descriptor := strings.TrimSpace(req.Form.Get("descriptor"))
	if deviceID == "" || descriptor == "" {
		h.writeJSONError(w, http.StatusBadRequest, "device_id and descriptor are required")
		return
	}
	snapshot := h.capability.UIPush(deviceID, descriptor, strings.TrimSpace(req.Form.Get("root_id")))
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "snapshot": snapshot})
}

func (h *Handler) handleUIPatch(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	deviceID := strings.TrimSpace(req.Form.Get("device_id"))
	componentID := strings.TrimSpace(req.Form.Get("component_id"))
	descriptor := strings.TrimSpace(req.Form.Get("descriptor"))
	if deviceID == "" || componentID == "" || descriptor == "" {
		h.writeJSONError(w, http.StatusBadRequest, "device_id, component_id, and descriptor are required")
		return
	}
	snapshot := h.capability.UIPatch(deviceID, componentID, descriptor)
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "snapshot": snapshot})
}

func (h *Handler) handleUITransition(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	deviceID := strings.TrimSpace(req.Form.Get("device_id"))
	componentID := strings.TrimSpace(req.Form.Get("component_id"))
	transition := strings.TrimSpace(req.Form.Get("transition"))
	if deviceID == "" || componentID == "" || transition == "" {
		h.writeJSONError(w, http.StatusBadRequest, "device_id, component_id, and transition are required")
		return
	}
	durationMS := 0
	if raw := strings.TrimSpace(req.Form.Get("duration_ms")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			h.writeJSONError(w, http.StatusBadRequest, "duration_ms must be an integer")
			return
		}
		durationMS = parsed
	}
	snapshot := h.capability.UITransition(deviceID, componentID, transition, durationMS)
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "snapshot": snapshot})
}

func (h *Handler) handleUIBroadcast(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	cohortName := strings.TrimSpace(req.Form.Get("cohort"))
	descriptor := strings.TrimSpace(req.Form.Get("descriptor"))
	if cohortName == "" || descriptor == "" {
		h.writeJSONError(w, http.StatusBadRequest, "cohort and descriptor are required")
		return
	}
	cohort, ok := h.capability.CohortGet(cohortName)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "cohort not found")
		return
	}
	members := h.resolveCohortMembers(cohort.Selectors)
	broadcast := h.capability.UIBroadcast(cohort.Name, descriptor, strings.TrimSpace(req.Form.Get("patch_id")), members)
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "broadcast": broadcast, "members": members})
}

func (h *Handler) handleUISubscribe(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	deviceID := strings.TrimSpace(req.Form.Get("device_id"))
	target := strings.TrimSpace(req.Form.Get("to"))
	if deviceID == "" || target == "" {
		h.writeJSONError(w, http.StatusBadRequest, "device_id and to are required")
		return
	}
	snapshot := h.capability.UISubscribe(deviceID, target)
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "snapshot": snapshot})
}

func (h *Handler) handleUISnapshot(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	deviceID := strings.TrimSpace(req.URL.Query().Get("device_id"))
	if deviceID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "device_id is required")
		return
	}
	snapshot, ok := h.capability.UISnapshot(deviceID)
	if !ok {
		h.writeJSON(w, http.StatusOK, map[string]any{"snapshot": nil})
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"snapshot": snapshot})
}

func (h *Handler) handleRecent(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"items": h.capability.ListRecent()})
}

func (h *Handler) handleSimDevices(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"devices": h.capability.SimDeviceList()})
}

func (h *Handler) handleSimDeviceNew(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	deviceID := strings.TrimSpace(req.Form.Get("device_id"))
	if deviceID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "device_id is required")
		return
	}
	device := h.capability.SimDeviceUpsert(deviceID, parseCSVValues(req.Form["caps"]))
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "device": device})
}

func (h *Handler) handleSimDeviceRemove(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	deviceID := strings.TrimSpace(req.Form.Get("device_id"))
	if deviceID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "device_id is required")
		return
	}
	deleted := h.capability.SimDeviceDelete(deviceID)
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "deleted": deleted})
}

func (h *Handler) handleSimInput(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	deviceID := strings.TrimSpace(req.Form.Get("device_id"))
	componentID := strings.TrimSpace(req.Form.Get("component_id"))
	action := strings.TrimSpace(req.Form.Get("action"))
	if deviceID == "" || componentID == "" || action == "" {
		h.writeJSONError(w, http.StatusBadRequest, "device_id, component_id, and action are required")
		return
	}
	input, ok := h.capability.SimRecordInput(deviceID, componentID, action, req.Form.Get("value"))
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "sim device not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "input": input})
}

func (h *Handler) handleSimUI(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	deviceID := strings.TrimSpace(req.URL.Query().Get("device_id"))
	if deviceID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "device_id is required")
		return
	}
	device, ok := h.capability.SimDeviceGet(deviceID)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "sim device not found")
		return
	}
	snapshot, hasSnapshot := h.capability.UISnapshot(device.DeviceID)
	if !hasSnapshot {
		snapshot = capability.UISnapshot{DeviceID: device.DeviceID}
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"device":   device,
		"snapshot": snapshot,
		"inputs":   h.capability.SimInputs(device.DeviceID),
	})
}

func (h *Handler) handleSimExpect(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	deviceID := strings.TrimSpace(req.Form.Get("device_id"))
	kind := strings.TrimSpace(req.Form.Get("kind"))
	selector := strings.TrimSpace(req.Form.Get("selector"))
	if deviceID == "" || kind == "" {
		h.writeJSONError(w, http.StatusBadRequest, "device_id and kind are required")
		return
	}
	within := time.Duration(0)
	if rawWithin := strings.TrimSpace(req.Form.Get("within")); rawWithin != "" {
		parsedWithin, err := time.ParseDuration(rawWithin)
		if err != nil || parsedWithin <= 0 {
			h.writeJSONError(w, http.StatusBadRequest, "within must be a positive duration")
			return
		}
		within = parsedWithin
	}
	result, ok := h.capability.SimExpect(deviceID, kind, selector, within)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "sim device not found")
		return
	}
	if !result.Matched {
		h.writeJSON(w, http.StatusConflict, map[string]any{"status": "failed", "result": result})
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "result": result})
}

func (h *Handler) handleSimRecord(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	deviceID := strings.TrimSpace(req.Form.Get("device_id"))
	if deviceID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "device_id is required")
		return
	}
	duration := time.Duration(0)
	if rawDuration := strings.TrimSpace(req.Form.Get("duration")); rawDuration != "" {
		parsedDuration, err := time.ParseDuration(rawDuration)
		if err != nil || parsedDuration <= 0 {
			h.writeJSONError(w, http.StatusBadRequest, "duration must be a positive duration")
			return
		}
		duration = parsedDuration
	}
	record, ok := h.capability.SimRecord(deviceID, duration)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "sim device not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "result": record})
}

func (h *Handler) handleScriptsDryRun(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	path := strings.TrimSpace(req.Form.Get("path"))
	if path == "" {
		h.writeJSONError(w, http.StatusBadRequest, "path is required")
		return
	}
	content, err := os.ReadFile(path)
	if err != nil {
		h.writeJSONError(w, http.StatusNotFound, "script not found")
		return
	}
	result := h.capability.ScriptDryRun(path, string(content))
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "result": result})
}

func (h *Handler) handleScriptsRun(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	path := strings.TrimSpace(req.Form.Get("path"))
	if path == "" {
		h.writeJSONError(w, http.StatusBadRequest, "path is required")
		return
	}
	content, err := os.ReadFile(path)
	if err != nil {
		h.writeJSONError(w, http.StatusNotFound, "script not found")
		return
	}
	result := h.capability.ScriptRun(path, string(content))
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "result": result})
}

func (h *Handler) handleStoreGet(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	record, ok := h.capability.StoreGet(req.URL.Query().Get("namespace"), req.URL.Query().Get("key"))
	if !ok {
		h.writeJSON(w, http.StatusOK, map[string]any{"record": nil})
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"record": record})
}

func (h *Handler) handleStoreNamespaces(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"namespaces": h.capability.StoreNamespaces()})
}

func (h *Handler) handleStoreList(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"records": h.capability.StoreList(req.URL.Query().Get("namespace"))})
}

func (h *Handler) handleStorePut(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	ttl := time.Duration(0)
	ttlRaw := strings.TrimSpace(req.Form.Get("ttl"))
	if ttlRaw != "" {
		parsedTTL, err := time.ParseDuration(ttlRaw)
		if err != nil || parsedTTL <= 0 {
			h.writeJSONError(w, http.StatusBadRequest, "ttl must be a positive duration")
			return
		}
		ttl = parsedTTL
	}
	record := h.capability.StorePut(req.Form.Get("namespace"), req.Form.Get("key"), req.Form.Get("value"), ttl)
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "record": record})
}

func (h *Handler) handleStoreDelete(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	deleted := h.capability.StoreDelete(req.Form.Get("namespace"), req.Form.Get("key"))
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "deleted": deleted})
}

func (h *Handler) handleStoreWatch(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	namespace := req.URL.Query().Get("namespace")
	prefix := req.URL.Query().Get("prefix")
	h.writeJSON(w, http.StatusOK, map[string]any{
		"namespace": namespace,
		"prefix":    prefix,
		"records":   h.capability.StoreWatch(namespace, prefix),
	})
}

func (h *Handler) handleStoreBind(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	binding := strings.TrimSpace(req.Form.Get("to"))
	parts := strings.SplitN(binding, ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		h.writeJSONError(w, http.StatusBadRequest, "to must be formatted as <device>:<scenario>")
		return
	}
	record, ok := h.capability.StoreBind(req.Form.Get("namespace"), req.Form.Get("key"), binding)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "store record not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "record": record})
}

func (h *Handler) handleBusTail(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	limit := 0
	if rawLimit := strings.TrimSpace(req.URL.Query().Get("limit")); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit <= 0 {
			h.writeJSONError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		limit = parsedLimit
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"events": h.capability.BusTail(req.URL.Query().Get("kind"), req.URL.Query().Get("name"), limit)})
}

func (h *Handler) handleBusEmit(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	event := h.capability.BusEmit(req.Form.Get("kind"), req.Form.Get("name"), req.Form.Get("payload"))
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "event": event})
}

func (h *Handler) handleBusReplay(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	limit := 0
	if rawLimit := strings.TrimSpace(req.URL.Query().Get("limit")); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit <= 0 {
			h.writeJSONError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		limit = parsedLimit
	}
	events := h.capability.BusReplay(
		req.URL.Query().Get("from"),
		req.URL.Query().Get("to"),
		req.URL.Query().Get("kind"),
		req.URL.Query().Get("name"),
		limit,
	)
	h.writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

func (h *Handler) handleHandlers(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"handlers": h.capability.HandlerList()})
}

func (h *Handler) handleHandlersOn(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	selector := strings.TrimSpace(req.Form.Get("selector"))
	action := strings.TrimSpace(req.Form.Get("action"))
	if selector == "" || action == "" {
		h.writeJSONError(w, http.StatusBadRequest, "selector and action are required")
		return
	}
	runCommand := strings.TrimSpace(req.Form.Get("run"))
	emitKind := strings.TrimSpace(req.Form.Get("emit_kind"))
	emitName := strings.TrimSpace(req.Form.Get("emit_name"))
	emitPayload := strings.TrimSpace(req.Form.Get("emit_payload"))

	hasRun := runCommand != ""
	hasEmit := emitKind != "" || emitName != "" || emitPayload != ""
	if hasRun == hasEmit {
		h.writeJSONError(w, http.StatusBadRequest, "provide exactly one target: run or emit_kind/emit_name")
		return
	}

	var handler capability.HandlerRegistration
	if hasRun {
		handler = h.capability.HandlerOnRun(selector, action, runCommand)
	} else {
		if emitName == "" {
			h.writeJSONError(w, http.StatusBadRequest, "emit_name is required when using emit")
			return
		}
		handler = h.capability.HandlerOnEmit(selector, action, emitKind, emitName, emitPayload)
	}

	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "handler": handler})
}

func (h *Handler) handleHandlersOff(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	handlerID := strings.TrimSpace(req.Form.Get("handler_id"))
	if handlerID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "handler_id is required")
		return
	}
	deleted := h.capability.HandlerOff(handlerID)
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "deleted": deleted})
}

func (h *Handler) handleInlineScenarios(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	name := strings.TrimSpace(req.URL.Query().Get("name"))
	if name == "" {
		h.writeJSON(w, http.StatusOK, map[string]any{"scenarios": h.capability.ScenarioList()})
		return
	}
	def, ok := h.capability.ScenarioGet(name)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "inline scenario not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"scenario": def})
}

func (h *Handler) handleInlineScenarioDefine(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	name := strings.TrimSpace(req.Form.Get("name"))
	if name == "" {
		h.writeJSONError(w, http.StatusBadRequest, "name is required")
		return
	}
	onEventKinds := req.Form["on_event_kind"]
	onEventCommands := req.Form["on_event_command"]
	if len(onEventKinds) != len(onEventCommands) {
		h.writeJSONError(w, http.StatusBadRequest, "on_event_kind and on_event_command counts must match")
		return
	}
	onEvents := make([]capability.InlineScenarioEventHook, 0, len(onEventKinds))
	for i := range onEventKinds {
		kind := strings.TrimSpace(onEventKinds[i])
		command := strings.TrimSpace(onEventCommands[i])
		if kind == "" || command == "" {
			h.writeJSONError(w, http.StatusBadRequest, "on_event_kind and on_event_command values must be non-empty")
			return
		}
		onEvents = append(onEvents, capability.InlineScenarioEventHook{Kind: kind, Command: command})
	}
	def := h.capability.ScenarioDefine(capability.InlineScenarioDefinition{
		Name:         name,
		MatchIntents: parseCSVValues(req.Form["match_intent"]),
		MatchEvents:  parseCSVValues(req.Form["match_event"]),
		Priority:     strings.TrimSpace(req.Form.Get("priority")),
		OnStart:      strings.TrimSpace(req.Form.Get("on_start")),
		OnInput:      strings.TrimSpace(req.Form.Get("on_input")),
		OnEvents:     onEvents,
		OnSuspend:    strings.TrimSpace(req.Form.Get("on_suspend")),
		OnResume:     strings.TrimSpace(req.Form.Get("on_resume")),
		OnStop:       strings.TrimSpace(req.Form.Get("on_stop")),
	})
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "scenario": def})
}

func (h *Handler) handleInlineScenarioUndefine(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	name := strings.TrimSpace(req.Form.Get("name"))
	if name == "" {
		h.writeJSONError(w, http.StatusBadRequest, "name is required")
		return
	}
	deleted := h.capability.ScenarioUndefine(name)
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "deleted": deleted})
}

func (h *Handler) handleReplAIProviders(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.ai == nil {
		h.writeJSONError(w, http.StatusNotFound, "repl ai service not configured")
		return
	}
	resp, err := h.ai.ListProviders(req.Context(), replai.ListProvidersRequest{})
	if err != nil {
		h.writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"providers": resp.Providers})
}

func (h *Handler) handleReplAIModels(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.ai == nil {
		h.writeJSONError(w, http.StatusNotFound, "repl ai service not configured")
		return
	}
	provider := strings.TrimSpace(req.URL.Query().Get("provider"))
	resp, err := h.ai.ListModels(req.Context(), replai.ListModelsRequest{Provider: provider})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, replai.ErrMissingProvider) || errors.Is(err, replai.ErrProviderNotFound) {
			status = http.StatusBadRequest
		}
		h.writeJSONError(w, status, err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"provider": resp.Provider,
		"models":   resp.Models,
	})
}

func (h *Handler) handleReplAISelection(w http.ResponseWriter, req *http.Request) {
	if h.ai == nil {
		h.writeJSONError(w, http.StatusNotFound, "repl ai service not configured")
		return
	}
	switch req.Method {
	case http.MethodGet:
		sessionID := strings.TrimSpace(req.URL.Query().Get("session_id"))
		resp, err := h.ai.GetSelection(req.Context(), replai.GetSelectionRequest{SessionID: sessionID})
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, replai.ErrMissingSessionID) ||
				errors.Is(err, replai.ErrMissingProvider) ||
				errors.Is(err, replai.ErrMissingModel) ||
				errors.Is(err, replai.ErrProviderNotFound) {
				status = http.StatusBadRequest
			}
			if errors.Is(err, replsession.ErrSessionNotFound) {
				status = http.StatusNotFound
			}
			h.writeJSONError(w, status, err.Error())
			return
		}
		h.writeJSON(w, http.StatusOK, resp)
	case http.MethodPost:
		if err := req.ParseForm(); err != nil {
			h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
			return
		}
		resp, err := h.ai.SetSelection(req.Context(), replai.SetSelectionRequest{
			SessionID: strings.TrimSpace(req.Form.Get("session_id")),
			Provider:  strings.TrimSpace(req.Form.Get("provider")),
			Model:     strings.TrimSpace(req.Form.Get("model")),
		})
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, replai.ErrMissingSessionID) ||
				errors.Is(err, replai.ErrMissingProvider) ||
				errors.Is(err, replai.ErrMissingModel) ||
				errors.Is(err, replai.ErrProviderNotFound) {
				status = http.StatusBadRequest
			}
			if errors.Is(err, replsession.ErrSessionNotFound) {
				status = http.StatusNotFound
			}
			h.writeJSONError(w, status, err.Error())
			return
		}
		h.writeJSON(w, http.StatusOK, resp)
	default:
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleReplAIAsk(w http.ResponseWriter, req *http.Request) {
	if h.ai == nil {
		h.writeJSONError(w, http.StatusNotFound, "repl ai service not configured")
		return
	}
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	resp, err := h.ai.Ask(req.Context(), replai.AskRequest{
		SessionID: strings.TrimSpace(req.Form.Get("session_id")),
		Prompt:    strings.TrimSpace(req.Form.Get("prompt")),
	})
	if err != nil {
		h.writeReplAIError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleReplAIGenerate(w http.ResponseWriter, req *http.Request) {
	if h.ai == nil {
		h.writeJSONError(w, http.StatusNotFound, "repl ai service not configured")
		return
	}
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	resp, err := h.ai.Generate(req.Context(), replai.GenerateRequest{
		SessionID:   strings.TrimSpace(req.Form.Get("session_id")),
		Description: strings.TrimSpace(req.Form.Get("description")),
	})
	if err != nil {
		h.writeReplAIError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleReplAIContext(w http.ResponseWriter, req *http.Request) {
	if h.ai == nil {
		h.writeJSONError(w, http.StatusNotFound, "repl ai service not configured")
		return
	}
	switch req.Method {
	case http.MethodGet:
		sessionID := strings.TrimSpace(req.URL.Query().Get("session_id"))
		resp, err := h.ai.GetContext(req.Context(), replai.GetContextRequest{SessionID: sessionID})
		if err != nil {
			h.writeReplAIError(w, err)
			return
		}
		h.writeJSON(w, http.StatusOK, resp)
	case http.MethodPost:
		if err := req.ParseForm(); err != nil {
			h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
			return
		}
		resp, err := h.ai.AddContext(req.Context(), replai.AddContextRequest{
			SessionID: strings.TrimSpace(req.Form.Get("session_id")),
			Ref:       strings.TrimSpace(req.Form.Get("ref")),
		})
		if err != nil {
			h.writeReplAIError(w, err)
			return
		}
		h.writeJSON(w, http.StatusOK, resp)
	default:
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleReplAIPinContext(w http.ResponseWriter, req *http.Request) {
	if h.ai == nil {
		h.writeJSONError(w, http.StatusNotFound, "repl ai service not configured")
		return
	}
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	resp, err := h.ai.PinContext(req.Context(), replai.PinContextRequest{
		SessionID: strings.TrimSpace(req.Form.Get("session_id")),
		Ref:       strings.TrimSpace(req.Form.Get("ref")),
	})
	if err != nil {
		h.writeReplAIError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleReplAIUnpinContext(w http.ResponseWriter, req *http.Request) {
	if h.ai == nil {
		h.writeJSONError(w, http.StatusNotFound, "repl ai service not configured")
		return
	}
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	resp, err := h.ai.UnpinContext(req.Context(), replai.UnpinContextRequest{
		SessionID: strings.TrimSpace(req.Form.Get("session_id")),
		Ref:       strings.TrimSpace(req.Form.Get("ref")),
	})
	if err != nil {
		h.writeReplAIError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleReplAIClearContext(w http.ResponseWriter, req *http.Request) {
	if h.ai == nil {
		h.writeJSONError(w, http.StatusNotFound, "repl ai service not configured")
		return
	}
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	resp, err := h.ai.ClearContext(req.Context(), replai.ClearContextRequest{
		SessionID: strings.TrimSpace(req.Form.Get("session_id")),
	})
	if err != nil {
		h.writeReplAIError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleReplAIPolicy(w http.ResponseWriter, req *http.Request) {
	if h.ai == nil {
		h.writeJSONError(w, http.StatusNotFound, "repl ai service not configured")
		return
	}
	switch req.Method {
	case http.MethodGet:
		sessionID := strings.TrimSpace(req.URL.Query().Get("session_id"))
		resp, err := h.ai.GetPolicy(req.Context(), replai.GetPolicyRequest{SessionID: sessionID})
		if err != nil {
			h.writeReplAIError(w, err)
			return
		}
		h.writeJSON(w, http.StatusOK, resp)
	case http.MethodPost:
		if err := req.ParseForm(); err != nil {
			h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
			return
		}
		resp, err := h.ai.SetPolicy(req.Context(), replai.SetPolicyRequest{
			SessionID: strings.TrimSpace(req.Form.Get("session_id")),
			Policy:    strings.TrimSpace(req.Form.Get("policy")),
		})
		if err != nil {
			h.writeReplAIError(w, err)
			return
		}
		h.writeJSON(w, http.StatusOK, resp)
	default:
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleReplAIHistory(w http.ResponseWriter, req *http.Request) {
	if h.ai == nil {
		h.writeJSONError(w, http.StatusNotFound, "repl ai service not configured")
		return
	}
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	sessionID := strings.TrimSpace(req.URL.Query().Get("session_id"))
	resp, err := h.ai.GetThread(req.Context(), replai.GetThreadRequest{SessionID: sessionID})
	if err != nil {
		h.writeReplAIError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleReplAIReset(w http.ResponseWriter, req *http.Request) {
	if h.ai == nil {
		h.writeJSONError(w, http.StatusNotFound, "repl ai service not configured")
		return
	}
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	resp, err := h.ai.ResetThread(req.Context(), replai.ResetThreadRequest{
		SessionID: strings.TrimSpace(req.Form.Get("session_id")),
	})
	if err != nil {
		h.writeReplAIError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) writeReplAIError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	if errors.Is(err, replai.ErrMissingSessionID) ||
		errors.Is(err, replai.ErrMissingProvider) ||
		errors.Is(err, replai.ErrMissingModel) ||
		errors.Is(err, replai.ErrProviderNotFound) ||
		errors.Is(err, replai.ErrMissingContextRef) ||
		errors.Is(err, replai.ErrUnsupportedApprovalPolicy) ||
		errors.Is(err, replai.ErrMissingPrompt) {
		status = http.StatusBadRequest
	}
	if errors.Is(err, replsession.ErrSessionNotFound) {
		status = http.StatusNotFound
	}
	h.writeJSONError(w, status, err.Error())
}

func (h *Handler) handleReplSessions(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.repl == nil {
		h.writeJSON(w, http.StatusOK, map[string]any{"sessions": []replsession.ReplSession{}})
		return
	}
	list, err := h.repl.ListSessions(req.Context(), replsession.ListSessionsRequest{})
	if err != nil {
		h.writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"sessions": list.Sessions})
}

func (h *Handler) handleReplSession(w http.ResponseWriter, req *http.Request) {
	if h.repl == nil {
		h.writeJSONError(w, http.StatusNotFound, "repl session service not configured")
		return
	}
	sessionID := strings.TrimSpace(strings.TrimPrefix(req.URL.Path, "/admin/api/repl/sessions/"))
	if sessionID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "session id is required")
		return
	}
	switch req.Method {
	case http.MethodGet:
		session, err := h.repl.GetSession(req.Context(), replsession.GetSessionRequest{SessionID: sessionID})
		if err != nil {
			h.writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}
		h.writeJSON(w, http.StatusOK, map[string]any{"session": session.Session})
	case http.MethodDelete:
		if _, err := h.repl.TerminateSession(req.Context(), replsession.TerminateSessionRequest{SessionID: sessionID}); err != nil {
			h.writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}
		h.writeJSON(w, http.StatusOK, map[string]any{
			"status":     "ok",
			"session_id": sessionID,
		})
	default:
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleDashboard(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := dashboardTemplate.Execute(w, map[string]string{
		"ServerID": strings.TrimSpace(h.cfg.MDNSName),
	}); err != nil {
		h.writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("render dashboard: %v", err))
	}
}

func (h *Handler) handleStatus(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	status := map[string]any{
		"server":  h.control.StatusData(),
		"runtime": h.runtime.StatusData(),
		"config": map[string]any{
			"grpc_address":               h.cfg.GRPCAddress(),
			"mdns_service":               h.cfg.MDNSService,
			"mdns_name":                  h.cfg.MDNSName,
			"version":                    h.cfg.Version,
			"heartbeat_timeout_seconds":  h.cfg.HeartbeatTimeoutSeconds,
			"liveness_interval_seconds":  h.cfg.LivenessReconcileIntervalSecs,
			"due_timer_interval_seconds": h.cfg.DueTimerProcessIntervalSecs,
			"recording_dir":              h.cfg.RecordingDir,
			"log_dir":                    h.cfg.LogDir,
			"log_level":                  h.cfg.LogLevel,
			"log_max_bytes":              h.cfg.LogMaxBytes,
			"log_max_archives":           h.cfg.LogMaxArchives,
			"log_stderr":                 h.cfg.LogStderr,
			"photo_frame_dir":            h.cfg.PhotoFrameDir,
			"admin_http_host":            h.cfg.AdminHTTPHost,
			"admin_http_port":            h.cfg.AdminHTTPPort,
		},
		"timestamp_unix_ms": h.now().UTC().UnixMilli(),
	}
	h.writeJSON(w, http.StatusOK, status)
}

func (h *Handler) handleDevices(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	activeByDevice := h.runtime.Engine.ActiveSnapshot()
	type deviceView struct {
		DeviceID       string            `json:"device_id"`
		DeviceName     string            `json:"device_name"`
		DeviceType     string            `json:"device_type"`
		Platform       string            `json:"platform"`
		Zone           string            `json:"zone,omitempty"`
		Roles          []string          `json:"roles,omitempty"`
		Mobility       string            `json:"mobility,omitempty"`
		Affinity       string            `json:"affinity,omitempty"`
		State          string            `json:"state"`
		LastHeartbeat  int64             `json:"last_heartbeat_unix_ms"`
		RegisteredAt   int64             `json:"registered_at_unix_ms"`
		ActiveScenario string            `json:"active_scenario,omitempty"`
		Capabilities   map[string]string `json:"capabilities"`
	}

	devices := h.devices.List()
	views := make([]deviceView, 0, len(devices))
	for _, d := range devices {
		views = append(views, deviceView{
			DeviceID:       d.DeviceID,
			DeviceName:     d.DeviceName,
			DeviceType:     d.DeviceType,
			Platform:       d.Platform,
			Zone:           d.Placement.Zone,
			Roles:          d.Placement.Roles,
			Mobility:       d.Placement.Mobility,
			Affinity:       d.Placement.Affinity,
			State:          string(d.State),
			LastHeartbeat:  d.LastHeartbeat.UTC().UnixMilli(),
			RegisteredAt:   d.RegisteredAt.UTC().UnixMilli(),
			ActiveScenario: activeByDevice[d.DeviceID],
			Capabilities:   d.Capabilities,
		})
	}

	h.writeJSON(w, http.StatusOK, map[string]any{"devices": views})
}

func (h *Handler) handleScenarios(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	activeByDevice := h.runtime.Engine.ActiveSnapshot()
	registry := h.runtime.Engine.RegistrySnapshot()
	activeDevicesByScenario := make(map[string][]string)
	for deviceID, scenarioName := range activeByDevice {
		activeDevicesByScenario[scenarioName] = append(activeDevicesByScenario[scenarioName], deviceID)
	}

	type scenarioView struct {
		Name          string   `json:"name"`
		Priority      int      `json:"priority"`
		ActiveDevices []string `json:"active_devices"`
	}
	views := make([]scenarioView, 0, len(registry))
	for _, reg := range registry {
		activeDevices := activeDevicesByScenario[reg.Name]
		sort.Strings(activeDevices)
		views = append(views, scenarioView{
			Name:          reg.Name,
			Priority:      int(reg.Priority),
			ActiveDevices: activeDevices,
		})
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"scenarios": views})
}

func (h *Handler) handleDevicePlacementUpdate(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form")
		return
	}
	deviceID := strings.TrimSpace(req.FormValue("device_id"))
	if deviceID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "device_id is required")
		return
	}
	placement := device.PlacementMetadata{
		Zone:     strings.TrimSpace(req.FormValue("zone")),
		Roles:    parseDeviceIDs(req.FormValue("roles")),
		Mobility: strings.TrimSpace(req.FormValue("mobility")),
		Affinity: strings.TrimSpace(req.FormValue("affinity")),
	}
	if err := h.devices.UpdatePlacement(deviceID, placement); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	eventlog.Emit(req.Context(), "admin.action.applied", slog.LevelInfo, "admin placement update applied",
		slog.String("component", "admin.http"),
		slog.String("action", "device_placement.update"),
		slog.String("device_id", deviceID),
	)
	h.writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"device_id": deviceID,
		"placement": placement,
	})
}

func (h *Handler) handleStartScenario(w http.ResponseWriter, req *http.Request) {
	h.handleScenarioCommand(w, req, true)
}

func (h *Handler) handleStopScenario(w http.ResponseWriter, req *http.Request) {
	h.handleScenarioCommand(w, req, false)
}

func (h *Handler) handleScenarioCommand(w http.ResponseWriter, req *http.Request, start bool) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form")
		return
	}

	scenarioName := strings.TrimSpace(req.FormValue("scenario"))
	if scenarioName == "" {
		h.writeJSONError(w, http.StatusBadRequest, "scenario is required")
		return
	}

	deviceIDs := parseDeviceIDs(req.FormValue("device_ids"))
	if deviceID := strings.TrimSpace(req.FormValue("device_id")); deviceID != "" {
		deviceIDs = append(deviceIDs, deviceID)
	}
	deviceIDs = normalizeDeviceIDs(deviceIDs)

	ctx, cancel := context.WithTimeout(req.Context(), 3*time.Second)
	defer cancel()

	var (
		matched string
		err     error
		action  = "stop"
	)
	if start {
		action = "start"
		matched, err = h.runtime.StartScenario(ctx, scenarioName, deviceIDs)
	} else {
		matched, err = h.runtime.StopScenario(ctx, scenarioName, deviceIDs)
	}
	if err != nil {
		h.writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	eventlog.Emit(req.Context(), "admin.action.applied", slog.LevelInfo, "admin scenario command applied",
		slog.String("component", "admin.http"),
		slog.String("action", "scenario."+action),
		slog.String("scenario", matched),
		slog.Int("target_device_count", len(deviceIDs)),
	)

	h.writeJSON(w, http.StatusOK, map[string]any{
		"status":            "ok",
		"action":            action,
		"scenario":          matched,
		"requested":         scenarioName,
		"target_device_ids": deviceIDs,
	})
}

func (h *Handler) handleActivations(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	activeByDevice := h.runtime.Engine.ActiveSnapshot()
	suspendedByDevice := h.runtime.Engine.SuspendedSnapshot()

	claimsByDevice := map[string][]iorouter.Claim{}
	suspendedClaimsByDevice := map[string][]iorouter.Claim{}
	if routeIO, ok := h.runtime.Env.IO.(interface{ Claims() *iorouter.ClaimManager }); ok {
		claims := routeIO.Claims()
		if claims != nil {
			for _, d := range h.devices.List() {
				deviceID := d.DeviceID
				claimsByDevice[deviceID] = claims.Snapshot(deviceID)
				suspendedClaimsByDevice[deviceID] = claims.SuspendedSnapshot(deviceID)
			}
		}
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"active_by_device":           activeByDevice,
		"suspended_by_device":        suspendedByDevice,
		"claims_by_device":           claimsByDevice,
		"suspended_claims_by_device": suspendedClaimsByDevice,
		"event_tail":                 h.runtime.EventTail(50),
	})
}

func (h *Handler) handleApps(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.appRuntime == nil {
		h.writeJSON(w, http.StatusOK, map[string]any{"apps": []map[string]any{}})
		return
	}
	names := h.appRuntime.ListPackages()
	views := make([]map[string]any, 0, len(names))
	for _, name := range names {
		pkg, ok := h.appRuntime.GetPackage(name)
		if !ok {
			continue
		}
		history := h.appRuntime.ListPackageHistory(name)
		versions := make([]string, 0, len(history))
		for _, version := range history {
			versions = append(versions, strings.TrimSpace(version.Manifest.Version))
		}
		views = append(views, map[string]any{
			"name":             pkg.Manifest.Name,
			"version":          pkg.Manifest.Version,
			"revision":         pkg.Revision,
			"loaded_at_unixms": pkg.LoadedAt.UTC().UnixMilli(),
			"permissions":      pkg.Manifest.Permissions,
			"exports":          pkg.Manifest.Exports,
			"dev_mode":         pkg.Manifest.DevMode,
			"history_versions": versions,
		})
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"apps": views})
}

func (h *Handler) handleReloadApp(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.appRuntime == nil {
		h.writeJSONError(w, http.StatusBadRequest, "app runtime not configured")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form")
		return
	}
	name := strings.TrimSpace(req.FormValue("app"))
	if name == "" {
		name = strings.TrimSpace(req.FormValue("name"))
	}
	if name == "" {
		h.writeJSONError(w, http.StatusBadRequest, "app is required")
		return
	}
	ctx, cancel := context.WithTimeout(req.Context(), 3*time.Second)
	defer cancel()
	pkg, changed, err := h.appRuntime.ReloadPackage(ctx, name)
	if err != nil {
		h.writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if changed && h.syncAppDefs != nil {
		h.syncAppDefs()
	}
	eventlog.Emit(req.Context(), "admin.action.applied", slog.LevelInfo, "admin app reload applied",
		slog.String("component", "admin.http"),
		slog.String("action", "app.reload"),
		slog.String("app", pkg.Manifest.Name),
		slog.Bool("changed", changed),
		slog.String("version", pkg.Manifest.Version),
	)
	h.writeJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"action":   "reload",
		"app":      pkg.Manifest.Name,
		"changed":  changed,
		"version":  pkg.Manifest.Version,
		"revision": pkg.Revision,
	})
}

func (h *Handler) handleRollbackApp(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.appRuntime == nil {
		h.writeJSONError(w, http.StatusBadRequest, "app runtime not configured")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form")
		return
	}
	name := strings.TrimSpace(req.FormValue("app"))
	if name == "" {
		name = strings.TrimSpace(req.FormValue("name"))
	}
	if name == "" {
		h.writeJSONError(w, http.StatusBadRequest, "app is required")
		return
	}
	pkg, err := h.appRuntime.RollbackPackage(name)
	if err != nil {
		h.writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if h.syncAppDefs != nil {
		h.syncAppDefs()
	}
	eventlog.Emit(req.Context(), "admin.action.applied", slog.LevelInfo, "admin app rollback applied",
		slog.String("component", "admin.http"),
		slog.String("action", "app.rollback"),
		slog.String("app", pkg.Manifest.Name),
		slog.String("version", pkg.Manifest.Version),
	)
	h.writeJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"action":   "rollback",
		"app":      pkg.Manifest.Name,
		"version":  pkg.Manifest.Version,
		"revision": pkg.Revision,
	})
}

func (h *Handler) handleAppMigrationStatus(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.appRuntime == nil {
		h.writeJSONError(w, http.StatusBadRequest, "app runtime not configured")
		return
	}
	name := strings.TrimSpace(req.URL.Query().Get("app"))
	if name == "" {
		name = strings.TrimSpace(req.URL.Query().Get("name"))
	}
	if name == "" {
		h.writeJSONError(w, http.StatusBadRequest, "app is required")
		return
	}
	status, err := h.appRuntime.GetMigrationStatus(name)
	if err != nil {
		h.writeMigrationError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"migration": mapMigrationStatus(status),
	})
}

func (h *Handler) handleAppMigrationRetry(w http.ResponseWriter, req *http.Request) {
	h.handleAppMigrationAction(w, req, "retry")
}

func (h *Handler) handleAppMigrationAbort(w http.ResponseWriter, req *http.Request) {
	h.handleAppMigrationAction(w, req, "abort")
}

func (h *Handler) handleAppMigrationReconcile(w http.ResponseWriter, req *http.Request) {
	h.handleAppMigrationAction(w, req, "reconcile")
}

func (h *Handler) handleAppMigrationAction(w http.ResponseWriter, req *http.Request, action string) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.appRuntime == nil {
		h.writeJSONError(w, http.StatusBadRequest, "app runtime not configured")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form")
		return
	}
	name := strings.TrimSpace(req.FormValue("app"))
	if name == "" {
		name = strings.TrimSpace(req.FormValue("name"))
	}
	if name == "" {
		h.writeJSONError(w, http.StatusBadRequest, "app is required")
		return
	}

	var (
		status appruntime.MigrationStatus
		err    error
	)
	switch action {
	case "retry":
		status, err = h.appRuntime.RetryMigration(name)
	case "abort":
		status, err = h.appRuntime.AbortMigration(name)
	case "reconcile":
		recordID := strings.TrimSpace(req.FormValue("record_id"))
		resolution := strings.TrimSpace(req.FormValue("resolution"))
		if recordID == "" || resolution == "" {
			h.writeJSONError(w, http.StatusBadRequest, "record_id and resolution are required")
			return
		}
		status, err = h.appRuntime.ReconcileMigration(name, recordID, resolution)
	default:
		h.writeJSONError(w, http.StatusBadRequest, "unknown migration action")
		return
	}
	if err != nil {
		if errors.Is(err, appruntime.ErrMigrationExecutorUnavailable) {
			h.writeJSON(w, http.StatusConflict, map[string]any{
				"status":    "unsupported",
				"action":    action,
				"app":       name,
				"error":     err.Error(),
				"migration": mapMigrationStatus(status),
			})
			return
		}
		h.writeMigrationError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"action":    action,
		"app":       name,
		"migration": mapMigrationStatus(status),
	})
}

func (h *Handler) writeMigrationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, appruntime.ErrPackageNotFound):
		h.writeJSONError(w, http.StatusNotFound, err.Error())
	default:
		h.writeJSONError(w, http.StatusBadRequest, err.Error())
	}
}

func mapMigrationStatus(status appruntime.MigrationStatus) map[string]any {
	return map[string]any{
		"app":                 status.App,
		"version":             status.Version,
		"revision":            status.Revision,
		"steps_planned":       status.StepsPlanned,
		"steps_completed":     status.StepsCompleted,
		"verdict":             status.Verdict,
		"last_error":          status.LastError,
		"journal_path":        status.JournalPath,
		"reconciliation_path": status.ReconciliationPath,
		"executor_ready":      status.ExecutorReady,
	}
}

func parseDeviceIDs(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func normalizeDeviceIDs(deviceIDs []string) []string {
	out := make([]string, 0, len(deviceIDs))
	seen := make(map[string]struct{}, len(deviceIDs))
	for _, deviceID := range deviceIDs {
		deviceID = strings.TrimSpace(deviceID)
		if deviceID == "" {
			continue
		}
		if _, exists := seen[deviceID]; exists {
			continue
		}
		seen[deviceID] = struct{}{}
		out = append(out, deviceID)
	}
	return out
}

func parseSelectors(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func parseCSVValues(values []string) []string {
	out := make([]string, 0, len(values))
	for _, raw := range values {
		for _, part := range strings.Split(raw, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed == "" {
				continue
			}
			out = append(out, trimmed)
		}
	}
	return out
}

func (h *Handler) resolveCohortMembers(selectors []string) []string {
	devices := h.devices.List()
	members := make([]string, 0, len(devices))
	for _, d := range devices {
		if deviceMatchesSelectors(d, selectors) {
			members = append(members, d.DeviceID)
		}
	}
	sort.Strings(members)
	return members
}

func deviceMatchesSelectors(d device.Device, selectors []string) bool {
	for _, selector := range selectors {
		selector = strings.ToLower(strings.TrimSpace(selector))
		if selector == "" {
			continue
		}
		key, value, ok := strings.Cut(selector, ":")
		if !ok || strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			return false
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch key {
		case "id", "device":
			if !strings.EqualFold(d.DeviceID, value) {
				return false
			}
		case "zone":
			if !strings.EqualFold(d.Placement.Zone, value) {
				return false
			}
		case "role":
			matched := false
			for _, role := range d.Placement.Roles {
				if strings.EqualFold(role, value) {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		case "platform":
			if !strings.EqualFold(d.Platform, value) {
				return false
			}
		case "type":
			if !strings.EqualFold(d.DeviceType, value) {
				return false
			}
		case "state":
			if !strings.EqualFold(string(d.State), value) {
				return false
			}
		case "mobility":
			if !strings.EqualFold(d.Placement.Mobility, value) {
				return false
			}
		case "affinity":
			if !strings.EqualFold(d.Placement.Affinity, value) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func (h *Handler) writeJSONError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) withRequestLogging(next http.Handler) http.Handler {
	logger := eventlog.Component("admin.http")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ctx, end := eventlog.WithSpan(r.Context(), "admin:"+r.Method+":"+r.URL.Path)
		defer end()
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
		eventlog.Emit(ctx, "admin.http.request", slog.LevelInfo, "admin request",
			slog.String("component", "admin.http"),
			slog.Group("http",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			),
		)
		logger.Debug("admin request served", "event", "admin.http.request", "method", r.Method, "path", r.URL.Path)
	})
}

func (h *Handler) handleLogs(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	args := logFilterArgs(req)
	records, err := query.Search(h.cfg.LogDir, args, h.now().UTC())
	if err != nil {
		h.writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(records) > 200 {
		records = records[len(records)-200:]
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := logsTemplate.Execute(w, map[string]any{
		"Count":   len(records),
		"Filters": strings.Join(args, " "),
		"Rows":    records,
	}); err != nil {
		h.writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("render logs: %v", err))
	}
}

func (h *Handler) handleLogsJSONL(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	records, err := query.Search(h.cfg.LogDir, logFilterArgs(req), h.now().UTC())
	if err != nil {
		h.writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/x-ndjson")
	enc := json.NewEncoder(w)
	for _, record := range records {
		if err := enc.Encode(record); err != nil {
			return
		}
	}
}

func (h *Handler) handleLogsTrace(w http.ResponseWriter, req *http.Request) {
	traceID := strings.TrimSpace(strings.TrimPrefix(req.URL.Path, "/admin/logs/trace/"))
	if traceID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "trace id is required")
		return
	}
	records, err := query.ReadAll(h.cfg.LogDir)
	if err != nil {
		h.writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	events := query.Trace(records, traceID)
	if req.URL.Query().Get("format") == "json" {
		h.writeJSON(w, http.StatusOK, map[string]any{"trace_id": traceID, "events": events})
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := traceTemplate.Execute(w, map[string]any{
		"Title":  "Trace Timeline",
		"ID":     traceID,
		"Events": events,
		"Kind":   "trace",
	}); err != nil {
		h.writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("render trace logs: %v", err))
	}
}

func (h *Handler) handleLogsActivation(w http.ResponseWriter, req *http.Request) {
	activationID := strings.TrimSpace(strings.TrimPrefix(req.URL.Path, "/admin/logs/activation/"))
	if activationID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "activation id is required")
		return
	}
	records, err := query.ReadAll(h.cfg.LogDir)
	if err != nil {
		h.writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	events := query.Activation(records, activationID)
	if req.URL.Query().Get("format") == "json" {
		h.writeJSON(w, http.StatusOK, map[string]any{"activation_id": activationID, "events": events})
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := traceTemplate.Execute(w, map[string]any{
		"Title":  "Activation Timeline",
		"ID":     activationID,
		"Events": events,
		"Kind":   "activation",
	}); err != nil {
		h.writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("render activation logs: %v", err))
	}
}

func logFilterArgs(req *http.Request) []string {
	out := make([]string, 0)
	values := req.URL.Query()
	for key, items := range values {
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if key == "q" {
				out = append(out, item)
				continue
			}
			out = append(out, key+"="+item)
		}
	}
	return out
}

var dashboardTemplate = template.Must(template.New("admin").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Terminals Admin Dashboard</title>
  <style>
    :root { color-scheme: light; }
    body { margin: 0; font-family: ui-sans-serif, system-ui, sans-serif; background: #f5f7fb; color: #0f172a; }
    main { max-width: 1080px; margin: 0 auto; padding: 20px; }
    h1 { margin: 0 0 16px; }
    section { background: #fff; border: 1px solid #dbe2ee; border-radius: 12px; padding: 14px; margin-bottom: 14px; }
    pre { margin: 0; overflow: auto; max-height: 360px; background: #0f172a; color: #e2e8f0; padding: 12px; border-radius: 8px; }
    label { display: block; margin-bottom: 8px; }
    input { width: 100%; max-width: 420px; padding: 8px; border: 1px solid #cbd5e1; border-radius: 8px; }
    .row { display: flex; gap: 10px; flex-wrap: wrap; align-items: flex-end; }
    button { padding: 8px 14px; border: 1px solid #334155; border-radius: 8px; background: #1e293b; color: #fff; cursor: pointer; }
    .secondary { background: #fff; color: #1e293b; }
  </style>
</head>
<body>
<main>
  <h1>Terminals Admin Dashboard</h1>
  <p>Server: <strong>{{.ServerID}}</strong></p>

  <section>
    <h2>Scenario Control</h2>
    <div class="row">
      <label>Scenario name<input id="scenario" placeholder="terminal" /></label>
      <label>Device IDs (comma-separated)<input id="device_ids" placeholder="kitchen-1,hall-2" /></label>
      <button id="start_btn">Start</button>
      <button class="secondary" id="stop_btn">Stop</button>
    </div>
    <pre id="scenario_result">{}</pre>
  </section>

  <section>
    <h2>World Model</h2>
    <div class="row">
      <label>Device ID<input id="placement_device_id" placeholder="kitchen-1" /></label>
      <label>Zone<input id="placement_zone" placeholder="kitchen" /></label>
      <label>Roles (comma-separated)<input id="placement_roles" placeholder="kitchen_display,screen" /></label>
      <label>Mobility<input id="placement_mobility" placeholder="fixed" /></label>
      <label>Affinity<input id="placement_affinity" placeholder="home" /></label>
      <button id="placement_save_btn">Save placement</button>
    </div>
    <pre id="placement_result">{}</pre>
  </section>

  <section>
    <h2>Status</h2>
    <pre id="status">{}</pre>
  </section>

  <section>
    <h2>Devices</h2>
    <pre id="devices">[]</pre>
  </section>

  <section>
    <h2>Scenarios</h2>
    <pre id="scenarios">[]</pre>
  </section>

  <section>
    <h2>Activations</h2>
    <pre id="activations">[]</pre>
  </section>

  <section>
    <h2>Apps</h2>
    <div class="row">
      <label>App name<input id="app_name" placeholder="sound_watch" /></label>
      <button id="app_reload_btn">Reload</button>
      <button class="secondary" id="app_rollback_btn">Rollback</button>
    </div>
    <pre id="app_result">{}</pre>
    <pre id="apps">[]</pre>
  </section>
</main>
<script>
async function loadJSON(path) {
  const response = await fetch(path);
  return await response.json();
}
function format(json) {
  return JSON.stringify(json, null, 2);
}
async function refresh() {
  document.getElementById('status').textContent = format(await loadJSON('/admin/api/status'));
  document.getElementById('devices').textContent = format(await loadJSON('/admin/api/devices'));
  document.getElementById('scenarios').textContent = format(await loadJSON('/admin/api/scenarios'));
  document.getElementById('activations').textContent = format(await loadJSON('/admin/api/activations'));
  document.getElementById('apps').textContent = format(await loadJSON('/admin/api/apps'));
}
async function scenarioCommand(path) {
  const scenario = document.getElementById('scenario').value.trim();
  const deviceIDs = document.getElementById('device_ids').value.trim();
  const body = new URLSearchParams();
  body.set('scenario', scenario);
  body.set('device_ids', deviceIDs);
  const response = await fetch(path, { method: 'POST', body });
  const json = await response.json();
  document.getElementById('scenario_result').textContent = format(json);
  await refresh();
}
async function savePlacement() {
  const body = new URLSearchParams();
  body.set('device_id', document.getElementById('placement_device_id').value.trim());
  body.set('zone', document.getElementById('placement_zone').value.trim());
  body.set('roles', document.getElementById('placement_roles').value.trim());
  body.set('mobility', document.getElementById('placement_mobility').value.trim());
  body.set('affinity', document.getElementById('placement_affinity').value.trim());
  const response = await fetch('/admin/api/devices/placement', { method: 'POST', body });
  const json = await response.json();
  document.getElementById('placement_result').textContent = format(json);
  await refresh();
}
async function appCommand(path) {
  const body = new URLSearchParams();
  body.set('app', document.getElementById('app_name').value.trim());
  const response = await fetch(path, { method: 'POST', body });
  const json = await response.json();
  document.getElementById('app_result').textContent = format(json);
  await refresh();
}
document.getElementById('start_btn').addEventListener('click', () => scenarioCommand('/admin/api/scenarios/start'));
document.getElementById('stop_btn').addEventListener('click', () => scenarioCommand('/admin/api/scenarios/stop'));
document.getElementById('placement_save_btn').addEventListener('click', () => savePlacement());
document.getElementById('app_reload_btn').addEventListener('click', () => appCommand('/admin/api/apps/reload'));
document.getElementById('app_rollback_btn').addEventListener('click', () => appCommand('/admin/api/apps/rollback'));
refresh();
setInterval(refresh, 3000);
</script>
</body>
</html>`))

var logsTemplate = template.Must(template.New("logs").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Terminals Event Logs</title>
  <style>
    body { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; background: #0b1220; color: #e2e8f0; margin: 0; }
    main { padding: 16px; }
    a { color: #7dd3fc; }
    table { width: 100%; border-collapse: collapse; margin-top: 12px; }
    th, td { border-bottom: 1px solid #1e293b; text-align: left; padding: 6px; font-size: 13px; vertical-align: top; }
    th { color: #93c5fd; }
    .err { color: #fda4af; }
  </style>
</head>
<body>
<main>
  <h1>Event Logs</h1>
  <p>matching events: {{.Count}} | filters: {{.Filters}}</p>
  <p><a href="/admin">Back to dashboard</a></p>
  <table>
    <thead><tr><th>ts</th><th>level</th><th>event</th><th>component</th><th>msg</th><th>trace</th><th>activation</th></tr></thead>
    <tbody>
      {{range .Rows}}
      <tr>
        <td>{{index . "ts"}}</td>
        <td>{{index . "level"}}</td>
        <td>{{index . "event"}}</td>
        <td>{{index . "component"}}</td>
        <td>{{index . "msg"}}</td>
        <td><a href="/admin/logs/trace/{{index . "trace_id"}}">{{index . "trace_id"}}</a></td>
        <td><a href="/admin/logs/activation/{{index . "activation_id"}}">{{index . "activation_id"}}</a></td>
      </tr>
      {{end}}
    </tbody>
  </table>
</main>
</body>
</html>`))

var traceTemplate = template.Must(template.New("trace").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>{{.Title}}</title>
  <style>
    body { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; background: #0b1220; color: #e2e8f0; margin: 0; }
    main { padding: 16px; }
    a { color: #7dd3fc; }
    table { width: 100%; border-collapse: collapse; margin-top: 12px; }
    th, td { border-bottom: 1px solid #1e293b; text-align: left; padding: 6px; font-size: 13px; vertical-align: top; }
    th { color: #93c5fd; }
    .indent-1 { padding-left: 18px; }
    .indent-2 { padding-left: 36px; }
    .indent-3 { padding-left: 54px; }
  </style>
</head>
<body>
<main>
  <h1>{{.Title}}</h1>
  <p>{{.Kind}}: <strong>{{.ID}}</strong></p>
  <p><a href="/admin/logs">Back to logs</a></p>
  <table>
    <thead><tr><th>seq</th><th>ts</th><th>level</th><th>event</th><th>component</th><th>msg</th><th>span</th><th>parent</th></tr></thead>
    <tbody>
      {{range .Events}}
      <tr>
        <td>{{index . "seq"}}</td>
        <td>{{index . "ts"}}</td>
        <td>{{index . "level"}}</td>
        <td>{{index . "event"}}</td>
        <td>{{index . "component"}}</td>
        <td>{{index . "msg"}}</td>
        <td>{{index . "span_id"}}</td>
        <td>{{index . "parent_span_id"}}</td>
      </tr>
      {{end}}
    </tbody>
  </table>
</main>
</body>
</html>`))
