package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/config"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/replai"
	"github.com/curtcox/terminals/terminal_server/internal/replsession"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/terminal"
	"github.com/curtcox/terminals/terminal_server/internal/transport"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestReplSessionGetAndDeleteEndpoints(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	control := transport.NewControlService("HomeServer", devices)
	engine := scenario.NewEngine()
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})

	replSvc := replsession.NewService(terminal.NewManager())
	created, err := replSvc.CreateSession(context.Background(), replsession.CreateSessionRequest{
		DeviceID: "d1",
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	defer func() {
		_, _ = replSvc.TerminateSession(context.Background(), replsession.TerminateSessionRequest{
			SessionID: created.Session.ID,
		})
	}()

	h := NewHandler(control, runtime, replSvc, nil, nil, nil, devices, config.Config{MDNSName: "HomeServer"}, nil, nil)

	getReq := httptest.NewRequest(http.MethodGet, "/admin/api/repl/sessions/"+created.Session.ID, nil)
	getW := httptest.NewRecorder()
	h.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("GET repl session status = %d, want 200 body=%s", getW.Code, getW.Body.String())
	}

	delReq := httptest.NewRequest(http.MethodDelete, "/admin/api/repl/sessions/"+created.Session.ID, nil)
	delW := httptest.NewRecorder()
	h.ServeHTTP(delW, delReq)
	if delW.Code != http.StatusOK {
		t.Fatalf("DELETE repl session status = %d, want 200 body=%s", delW.Code, delW.Body.String())
	}

	missingReq := httptest.NewRequest(http.MethodGet, "/admin/api/repl/sessions/"+created.Session.ID, nil)
	missingW := httptest.NewRecorder()
	h.ServeHTTP(missingW, missingReq)
	if missingW.Code != http.StatusNotFound {
		t.Fatalf("GET after delete status = %d, want 404 body=%s", missingW.Code, missingW.Body.String())
	}
}

func TestReplAIEndpoints(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	control := transport.NewControlService("HomeServer", devices)
	engine := scenario.NewEngine()
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})

	replSvc := replsession.NewService(terminal.NewManager())
	created, err := replSvc.CreateSession(context.Background(), replsession.CreateSessionRequest{
		DeviceID: "d1",
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if err := replSvc.SetThread(created.Session.ID, "thread-99"); err != nil {
		t.Fatalf("SetThread() error = %v", err)
	}
	if err := replSvc.SetHistory(created.Session.ID, []string{"user: why suspended?", "assistant: preempted by red_alert"}); err != nil {
		t.Fatalf("SetHistory() error = %v", err)
	}
	defer func() {
		_, _ = replSvc.TerminateSession(context.Background(), replsession.TerminateSessionRequest{
			SessionID: created.Session.ID,
		})
	}()
	aiSvc := replai.NewService(replSvc, replai.Config{
		DefaultProvider: "ollama",
		DefaultModel:    "llama3.1",
		Providers: []replai.ProviderConfig{
			{Name: "openrouter", Models: []string{"anthropic/claude-sonnet-4-6"}},
			{Name: "ollama", Models: []string{"llama3.1"}},
		},
	})
	threadSnapshot, err := aiSvc.GetThread(context.Background(), replai.GetThreadRequest{SessionID: created.Session.ID})
	if err != nil {
		t.Fatalf("GetThread() error = %v", err)
	}
	if threadSnapshot.Thread != "thread-99" || len(threadSnapshot.History) != 2 {
		t.Fatalf("thread snapshot = %#v, want thread-99 + 2 entries", threadSnapshot)
	}

	h := NewHandler(control, runtime, replSvc, aiSvc, nil, nil, devices, config.Config{MDNSName: "HomeServer"}, nil, nil)

	providersReq := httptest.NewRequest(http.MethodGet, "/admin/api/repl/ai/providers", nil)
	providersW := httptest.NewRecorder()
	h.ServeHTTP(providersW, providersReq)
	if providersW.Code != http.StatusOK {
		t.Fatalf("providers status = %d, want 200 body=%s", providersW.Code, providersW.Body.String())
	}
	if !strings.Contains(providersW.Body.String(), "openrouter") || !strings.Contains(providersW.Body.String(), "ollama") {
		t.Fatalf("providers body = %s", providersW.Body.String())
	}

	modelsReq := httptest.NewRequest(http.MethodGet, "/admin/api/repl/ai/models?provider=ollama", nil)
	modelsW := httptest.NewRecorder()
	h.ServeHTTP(modelsW, modelsReq)
	if modelsW.Code != http.StatusOK {
		t.Fatalf("models status = %d, want 200 body=%s", modelsW.Code, modelsW.Body.String())
	}
	if !strings.Contains(modelsW.Body.String(), "llama3.1") {
		t.Fatalf("models body = %s", modelsW.Body.String())
	}

	getSelectionReq := httptest.NewRequest(http.MethodGet, "/admin/api/repl/ai/selection?session_id="+created.Session.ID, nil)
	getSelectionW := httptest.NewRecorder()
	h.ServeHTTP(getSelectionW, getSelectionReq)
	if getSelectionW.Code != http.StatusOK {
		t.Fatalf("selection GET status = %d, want 200 body=%s", getSelectionW.Code, getSelectionW.Body.String())
	}
	if !strings.Contains(getSelectionW.Body.String(), "\"provider\":\"ollama\"") {
		t.Fatalf("selection GET body = %s", getSelectionW.Body.String())
	}

	setSelectionReq := httptest.NewRequest(http.MethodPost, "/admin/api/repl/ai/selection", strings.NewReader(url.Values{
		"session_id": {created.Session.ID},
		"provider":   {"openrouter"},
		"model":      {"anthropic/claude-sonnet-4-6"},
	}.Encode()))
	setSelectionReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	setSelectionW := httptest.NewRecorder()
	h.ServeHTTP(setSelectionW, setSelectionReq)
	if setSelectionW.Code != http.StatusOK {
		t.Fatalf("selection POST status = %d, want 200 body=%s", setSelectionW.Code, setSelectionW.Body.String())
	}
	if !strings.Contains(setSelectionW.Body.String(), "\"provider\":\"openrouter\"") {
		t.Fatalf("selection POST body = %s", setSelectionW.Body.String())
	}

	askReq := httptest.NewRequest(http.MethodPost, "/admin/api/repl/ai/ask", strings.NewReader(url.Values{
		"session_id": {created.Session.ID},
		"prompt":     {"why is act_42 suspended?"},
	}.Encode()))
	askReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	askW := httptest.NewRecorder()
	h.ServeHTTP(askW, askReq)
	if askW.Code != http.StatusOK {
		t.Fatalf("ask POST status = %d, want 200 body=%s", askW.Code, askW.Body.String())
	}
	if !strings.Contains(askW.Body.String(), "\"answer\"") {
		t.Fatalf("ask POST body = %s", askW.Body.String())
	}

	genReq := httptest.NewRequest(http.MethodPost, "/admin/api/repl/ai/gen", strings.NewReader(url.Values{
		"session_id":  {created.Session.ID},
		"description": {"a tal app that rings a chime when the dryer beeps"},
	}.Encode()))
	genReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	genW := httptest.NewRecorder()
	h.ServeHTTP(genW, genReq)
	if genW.Code != http.StatusOK {
		t.Fatalf("gen POST status = %d, want 200 body=%s", genW.Code, genW.Body.String())
	}
	if !strings.Contains(genW.Body.String(), "\"output\"") {
		t.Fatalf("gen POST body = %s", genW.Body.String())
	}

	pinContextReq := httptest.NewRequest(http.MethodPost, "/admin/api/repl/ai/context/pin", strings.NewReader(url.Values{
		"session_id": {created.Session.ID},
		"ref":        {"claims:tree"},
	}.Encode()))
	pinContextReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	pinContextW := httptest.NewRecorder()
	h.ServeHTTP(pinContextW, pinContextReq)
	if pinContextW.Code != http.StatusOK {
		t.Fatalf("context pin status = %d, want 200 body=%s", pinContextW.Code, pinContextW.Body.String())
	}
	if !strings.Contains(pinContextW.Body.String(), "claims:tree") {
		t.Fatalf("context pin body = %s", pinContextW.Body.String())
	}

	getContextReq := httptest.NewRequest(http.MethodGet, "/admin/api/repl/ai/context?session_id="+created.Session.ID, nil)
	getContextW := httptest.NewRecorder()
	h.ServeHTTP(getContextW, getContextReq)
	if getContextW.Code != http.StatusOK {
		t.Fatalf("context GET status = %d, want 200 body=%s", getContextW.Code, getContextW.Body.String())
	}
	if !strings.Contains(getContextW.Body.String(), "claims:tree") {
		t.Fatalf("context GET body = %s", getContextW.Body.String())
	}

	setPolicyReq := httptest.NewRequest(http.MethodPost, "/admin/api/repl/ai/policy", strings.NewReader(url.Values{
		"session_id": {created.Session.ID},
		"policy":     {"prompt-all"},
	}.Encode()))
	setPolicyReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	setPolicyW := httptest.NewRecorder()
	h.ServeHTTP(setPolicyW, setPolicyReq)
	if setPolicyW.Code != http.StatusOK {
		t.Fatalf("policy POST status = %d, want 200 body=%s", setPolicyW.Code, setPolicyW.Body.String())
	}
	if !strings.Contains(setPolicyW.Body.String(), "\"policy\":\"prompt-all\"") {
		t.Fatalf("policy POST body = %s", setPolicyW.Body.String())
	}

	getPolicyReq := httptest.NewRequest(http.MethodGet, "/admin/api/repl/ai/policy?session_id="+created.Session.ID, nil)
	getPolicyW := httptest.NewRecorder()
	h.ServeHTTP(getPolicyW, getPolicyReq)
	if getPolicyW.Code != http.StatusOK {
		t.Fatalf("policy GET status = %d, want 200 body=%s", getPolicyW.Code, getPolicyW.Body.String())
	}
	if !strings.Contains(getPolicyW.Body.String(), "\"policy\":\"prompt-all\"") {
		t.Fatalf("policy GET body = %s", getPolicyW.Body.String())
	}

	historyReq := httptest.NewRequest(http.MethodGet, "/admin/api/repl/ai/history?session_id="+created.Session.ID, nil)
	historyW := httptest.NewRecorder()
	h.ServeHTTP(historyW, historyReq)
	if historyW.Code != http.StatusOK {
		t.Fatalf("history GET status = %d, want 200 body=%s", historyW.Code, historyW.Body.String())
	}
	if !strings.Contains(historyW.Body.String(), "\"thread\":\"thread-99\"") || !strings.Contains(historyW.Body.String(), "preempted by red_alert") {
		t.Fatalf("history GET body = %s", historyW.Body.String())
	}

	resetReq := httptest.NewRequest(http.MethodPost, "/admin/api/repl/ai/reset", strings.NewReader(url.Values{
		"session_id": {created.Session.ID},
	}.Encode()))
	resetReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resetW := httptest.NewRecorder()
	h.ServeHTTP(resetW, resetReq)
	if resetW.Code != http.StatusOK {
		t.Fatalf("reset POST status = %d, want 200 body=%s", resetW.Code, resetW.Body.String())
	}
	if !strings.Contains(resetW.Body.String(), "\"thread\":\"\"") || !strings.Contains(resetW.Body.String(), "\"history\":[]") {
		t.Fatalf("reset POST body = %s", resetW.Body.String())
	}
}
