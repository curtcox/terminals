package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSimAndScriptsEndpoints(t *testing.T) {
	h := testHandler(t)

	newReq := httptest.NewRequest(http.MethodPost, "/admin/api/sim/devices/new", strings.NewReader(url.Values{
		"device_id": {"sim-lab"},
		"caps":      {"display,keyboard"},
	}.Encode()))
	newReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	newW := httptest.NewRecorder()
	h.ServeHTTP(newW, newReq)
	if newW.Code != http.StatusOK {
		t.Fatalf("sim device new status = %d, want 200 body=%s", newW.Code, newW.Body.String())
	}

	inputReq := httptest.NewRequest(http.MethodPost, "/admin/api/sim/input", strings.NewReader(url.Values{
		"device_id":    {"sim-lab"},
		"component_id": {"banner"},
		"action":       {"tap"},
		"value":        {"ack"},
	}.Encode()))
	inputReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	inputW := httptest.NewRecorder()
	h.ServeHTTP(inputW, inputReq)
	if inputW.Code != http.StatusOK {
		t.Fatalf("sim input status = %d, want 200 body=%s", inputW.Code, inputW.Body.String())
	}

	uiReq := httptest.NewRequest(http.MethodGet, "/admin/api/sim/ui?device_id=sim-lab", nil)
	uiW := httptest.NewRecorder()
	h.ServeHTTP(uiW, uiReq)
	if uiW.Code != http.StatusOK {
		t.Fatalf("sim ui status = %d, want 200 body=%s", uiW.Code, uiW.Body.String())
	}
	if !strings.Contains(uiW.Body.String(), `"device_id":"sim-lab"`) {
		t.Fatalf("sim ui body missing device id: %s", uiW.Body.String())
	}

	expectReq := httptest.NewRequest(http.MethodPost, "/admin/api/sim/expect", strings.NewReader(url.Values{
		"device_id": {"sim-lab"},
		"kind":      {"ui"},
		"selector":  {"banner"},
	}.Encode()))
	expectReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	expectW := httptest.NewRecorder()
	h.ServeHTTP(expectW, expectReq)
	if expectW.Code != http.StatusConflict {
		t.Fatalf("sim expect status = %d, want 409 body=%s", expectW.Code, expectW.Body.String())
	}

	recordReq := httptest.NewRequest(http.MethodPost, "/admin/api/sim/record", strings.NewReader(url.Values{
		"device_id": {"sim-lab"},
		"duration":  {"5s"},
	}.Encode()))
	recordReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recordW := httptest.NewRecorder()
	h.ServeHTTP(recordW, recordReq)
	if recordW.Code != http.StatusOK {
		t.Fatalf("sim record status = %d, want 200 body=%s", recordW.Code, recordW.Body.String())
	}
	if !strings.Contains(recordW.Body.String(), `"duration":"5s"`) {
		t.Fatalf("sim record body missing duration: %s", recordW.Body.String())
	}

	scriptPath := filepath.Join(t.TempDir(), "smoke.term")
	if err := os.WriteFile(scriptPath, []byte("# comment\n\nstore put notes k v\nui push d1 banner\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(script) error = %v", err)
	}
	dryRunReq := httptest.NewRequest(http.MethodPost, "/admin/api/scripts/dry-run", strings.NewReader(url.Values{
		"path": {scriptPath},
	}.Encode()))
	dryRunReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	dryRunW := httptest.NewRecorder()
	h.ServeHTTP(dryRunW, dryRunReq)
	if dryRunW.Code != http.StatusOK {
		t.Fatalf("scripts dry-run status = %d, want 200 body=%s", dryRunW.Code, dryRunW.Body.String())
	}
	if !strings.Contains(dryRunW.Body.String(), `"command_count":2`) {
		t.Fatalf("scripts dry-run body missing command count: %s", dryRunW.Body.String())
	}

	runReq := httptest.NewRequest(http.MethodPost, "/admin/api/scripts/run", strings.NewReader(url.Values{
		"path": {scriptPath},
	}.Encode()))
	runReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	runW := httptest.NewRecorder()
	h.ServeHTTP(runW, runReq)
	if runW.Code != http.StatusOK {
		t.Fatalf("scripts run status = %d, want 200 body=%s", runW.Code, runW.Body.String())
	}
	if !strings.Contains(runW.Body.String(), `"executed_count":2`) {
		t.Fatalf("scripts run body missing executed count: %s", runW.Body.String())
	}
}

func TestScriptsRunCrossUsecaseSimulationFixture(t *testing.T) {
	h := testHandler(t)
	fixturePath := filepath.Join("..", "..", "testdata", "repl", "phase12-cross-usecase.term")

	dryRunReq := httptest.NewRequest(http.MethodPost, "/admin/api/scripts/dry-run", strings.NewReader(url.Values{
		"path": {fixturePath},
	}.Encode()))
	dryRunReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	dryRunW := httptest.NewRecorder()
	h.ServeHTTP(dryRunW, dryRunReq)
	if dryRunW.Code != http.StatusOK {
		t.Fatalf("fixture scripts dry-run status = %d, want 200 body=%s", dryRunW.Code, dryRunW.Body.String())
	}
	if !strings.Contains(dryRunW.Body.String(), `"command_count":25`) {
		t.Fatalf("fixture scripts dry-run body missing command count: %s", dryRunW.Body.String())
	}

	runReq := httptest.NewRequest(http.MethodPost, "/admin/api/scripts/run", strings.NewReader(url.Values{
		"path": {fixturePath},
	}.Encode()))
	runReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	runW := httptest.NewRecorder()
	h.ServeHTTP(runW, runReq)
	if runW.Code != http.StatusOK {
		t.Fatalf("fixture scripts run status = %d, want 200 body=%s", runW.Code, runW.Body.String())
	}
	body := runW.Body.String()
	if !strings.Contains(body, `"executed_count":25`) || !strings.Contains(body, `"failed_count":0`) {
		t.Fatalf("fixture scripts run body missing execution counters: %s", body)
	}
	if !strings.Contains(body, `"memory recall fixture-memory-mutating"`) {
		t.Fatalf("fixture scripts run body missing memory recall command trace: %s", body)
	}

	storeReq := httptest.NewRequest(http.MethodGet, "/admin/api/store/get?namespace=phase12&key=status", nil)
	storeW := httptest.NewRecorder()
	h.ServeHTTP(storeW, storeReq)
	if storeW.Code != http.StatusOK {
		t.Fatalf("fixture store get status = %d, want 200 body=%s", storeW.Code, storeW.Body.String())
	}
	if !strings.Contains(storeW.Body.String(), `"namespace":"phase12"`) || !strings.Contains(storeW.Body.String(), `"value":"seeded"`) {
		t.Fatalf("fixture store get body missing seeded record: %s", storeW.Body.String())
	}

	messageReq := httptest.NewRequest(http.MethodGet, "/admin/api/message?room=phase12-room", nil)
	messageW := httptest.NewRecorder()
	h.ServeHTTP(messageW, messageReq)
	if messageW.Code != http.StatusOK {
		t.Fatalf("fixture message ls status = %d, want 200 body=%s", messageW.Code, messageW.Body.String())
	}
	if !strings.Contains(messageW.Body.String(), `"room":"phase12-room"`) || !strings.Contains(messageW.Body.String(), `"text":"fixture-layer2-mutating"`) {
		t.Fatalf("fixture message ls body missing layer2 message side effect: %s", messageW.Body.String())
	}

	boardReq := httptest.NewRequest(http.MethodGet, "/admin/api/board?board=phase12-board", nil)
	boardW := httptest.NewRecorder()
	h.ServeHTTP(boardW, boardReq)
	if boardW.Code != http.StatusOK {
		t.Fatalf("fixture board ls status = %d, want 200 body=%s", boardW.Code, boardW.Body.String())
	}
	if !strings.Contains(boardW.Body.String(), `"board":"phase12-board"`) || !strings.Contains(boardW.Body.String(), `"text":"fixture-board-mutating"`) {
		t.Fatalf("fixture board ls body missing layer2 board side effect: %s", boardW.Body.String())
	}

	artifactsReq := httptest.NewRequest(http.MethodGet, "/admin/api/artifact", nil)
	artifactsW := httptest.NewRecorder()
	h.ServeHTTP(artifactsW, artifactsReq)
	if artifactsW.Code != http.StatusOK {
		t.Fatalf("fixture artifact ls status = %d, want 200 body=%s", artifactsW.Code, artifactsW.Body.String())
	}
	var artifactsBody map[string]any
	if err := json.Unmarshal(artifactsW.Body.Bytes(), &artifactsBody); err != nil {
		t.Fatalf("decode fixture artifact ls body error = %v body=%s", err, artifactsW.Body.String())
	}
	artifactItems, _ := artifactsBody["artifacts"].([]any)
	artifactID := ""
	for _, item := range artifactItems {
		artifactMap, _ := item.(map[string]any)
		if artifactMap == nil {
			continue
		}
		if title, _ := artifactMap["title"].(string); title == "fixture-artifact-mutating" {
			artifactID, _ = artifactMap["id"].(string)
			break
		}
	}
	if strings.TrimSpace(artifactID) == "" {
		t.Fatalf("fixture artifact ls missing layer2 artifact side effect: %s", artifactsW.Body.String())
	}

	artifactReq := httptest.NewRequest(http.MethodGet, "/admin/api/artifact/history?artifact_id="+url.QueryEscape(artifactID), nil)
	artifactW := httptest.NewRecorder()
	h.ServeHTTP(artifactW, artifactReq)
	if artifactW.Code != http.StatusOK {
		t.Fatalf("fixture artifact history status = %d, want 200 body=%s", artifactW.Code, artifactW.Body.String())
	}
	if !strings.Contains(artifactW.Body.String(), `"artifact_id":"`+artifactID+`"`) || !strings.Contains(artifactW.Body.String(), `"action":"create"`) || !strings.Contains(artifactW.Body.String(), `"title":"fixture-artifact-mutating"`) {
		t.Fatalf("fixture artifact history missing layer2 artifact side effect: %s", artifactW.Body.String())
	}

	canvasReq := httptest.NewRequest(http.MethodGet, "/admin/api/canvas?canvas=phase12-canvas", nil)
	canvasW := httptest.NewRecorder()
	h.ServeHTTP(canvasW, canvasReq)
	if canvasW.Code != http.StatusOK {
		t.Fatalf("fixture canvas ls status = %d, want 200 body=%s", canvasW.Code, canvasW.Body.String())
	}
	if !strings.Contains(canvasW.Body.String(), `"canvas":"phase12-canvas"`) || !strings.Contains(canvasW.Body.String(), `"text":"fixture-canvas-mutating"`) {
		t.Fatalf("fixture canvas ls body missing layer2 canvas side effect: %s", canvasW.Body.String())
	}

	sessionsReq := httptest.NewRequest(http.MethodGet, "/admin/api/session", nil)
	sessionsW := httptest.NewRecorder()
	h.ServeHTTP(sessionsW, sessionsReq)
	if sessionsW.Code != http.StatusOK {
		t.Fatalf("fixture session ls status = %d, want 200 body=%s", sessionsW.Code, sessionsW.Body.String())
	}
	var sessionsBody map[string]any
	if err := json.Unmarshal(sessionsW.Body.Bytes(), &sessionsBody); err != nil {
		t.Fatalf("decode fixture session ls body error = %v body=%s", err, sessionsW.Body.String())
	}
	sessionItems, _ := sessionsBody["sessions"].([]any)
	sessionID := ""
	for _, item := range sessionItems {
		sessionMap, _ := item.(map[string]any)
		if sessionMap == nil {
			continue
		}
		if kind, _ := sessionMap["kind"].(string); kind != "lesson" {
			continue
		}
		if target, _ := sessionMap["target"].(string); target != "phase12-session" {
			continue
		}
		sessionID, _ = sessionMap["id"].(string)
		break
	}
	if strings.TrimSpace(sessionID) == "" {
		t.Fatalf("fixture session ls missing layer2 session side effect: %s", sessionsW.Body.String())
	}

	sessionReq := httptest.NewRequest(http.MethodGet, "/admin/api/session/members?session_id="+url.QueryEscape(sessionID), nil)
	sessionW := httptest.NewRecorder()
	h.ServeHTTP(sessionW, sessionReq)
	if sessionW.Code != http.StatusOK {
		t.Fatalf("fixture session members status = %d, want 200 body=%s", sessionW.Code, sessionW.Body.String())
	}
	if !strings.Contains(sessionW.Body.String(), `"session_id":"`+sessionID+`"`) || !strings.Contains(sessionW.Body.String(), `"identity_id":"fixture-session-member"`) {
		t.Fatalf("fixture session members body missing session side effect: %s", sessionW.Body.String())
	}

	ackReq := httptest.NewRequest(http.MethodGet, "/admin/api/identity/ack?subject_ref=phase12-identity-subject", nil)
	ackW := httptest.NewRecorder()
	h.ServeHTTP(ackW, ackReq)
	if ackW.Code != http.StatusOK {
		t.Fatalf("fixture identity ack show status = %d, want 200 body=%s", ackW.Code, ackW.Body.String())
	}
	if !strings.Contains(ackW.Body.String(), `"subject_ref":"phase12-identity-subject"`) ||
		!strings.Contains(ackW.Body.String(), `"actor_ref":"person:fixture-identity"`) ||
		!strings.Contains(ackW.Body.String(), `"mode":"confirmed"`) {
		t.Fatalf("fixture identity ack show body missing identity ack side effect: %s", ackW.Body.String())
	}

	memoryReq := httptest.NewRequest(http.MethodGet, "/admin/api/memory?q=fixture-memory-mutating", nil)
	memoryW := httptest.NewRecorder()
	h.ServeHTTP(memoryW, memoryReq)
	if memoryW.Code != http.StatusOK {
		t.Fatalf("fixture memory recall status = %d, want 200 body=%s", memoryW.Code, memoryW.Body.String())
	}
	if !strings.Contains(memoryW.Body.String(), `"scope":"phase12-memory"`) ||
		!strings.Contains(memoryW.Body.String(), `"text":"fixture-memory-mutating"`) {
		t.Fatalf("fixture memory recall body missing memory side effect: %s", memoryW.Body.String())
	}

	simReq := httptest.NewRequest(http.MethodGet, "/admin/api/sim/ui?device_id=sim-fixture", nil)
	simW := httptest.NewRecorder()
	h.ServeHTTP(simW, simReq)
	if simW.Code != http.StatusNotFound {
		t.Fatalf("fixture sim ui status after cleanup = %d, want 404 body=%s", simW.Code, simW.Body.String())
	}
}
