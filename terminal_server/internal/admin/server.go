// Package admin serves a lightweight web dashboard and JSON admin APIs.
package admin

import (
	"context"
	"net/http"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/appruntime"
	"github.com/curtcox/terminals/terminal_server/internal/capability"
	"github.com/curtcox/terminals/terminal_server/internal/config"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/diagnostics/bugreport"
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
	mux.Handle("/artifacts/", newArtifactHandler())
	mux.Handle("/docs/usecases/", newUsecaseSiteHandler())
	mux.HandleFunc("/docs/usecases", redirectUsecaseSiteIndex)
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
	mux.HandleFunc("/admin/api/apps/migrate/logs", h.handleAppMigrationLogs)
	mux.HandleFunc("/admin/api/apps/migrate/retry", h.handleAppMigrationRetry)
	mux.HandleFunc("/admin/api/apps/migrate/abort", h.handleAppMigrationAbort)
	mux.HandleFunc("/admin/api/apps/migrate/drain-ready", h.handleAppMigrationDrainReady)
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
