package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCapabilityClosureEndpoints(t *testing.T) {
	h := testHandler(t)

	getCases := []string{
		"/admin/api/identity",
		"/admin/api/identity/show?identity=system",
		"/admin/api/identity/groups",
		"/admin/api/identity/prefs?identity=system",
		"/admin/api/identity/resolve?audience=group:family",
		"/admin/api/identity/ack?subject_ref=message:msg-1",
		"/admin/api/session",
		"/admin/api/message/rooms",
		"/admin/api/message",
		"/admin/api/message/unread?identity_id=alice",
		"/admin/api/board",
		"/admin/api/artifact",
		"/admin/api/canvas",
		"/admin/api/search?q=hello",
		"/admin/api/search/timeline?scope=message",
		"/admin/api/search/related?subject=hello",
		"/admin/api/search/recent?scope=memory",
		"/admin/api/memory?q=hello",
		"/admin/api/memory/stream?scope=kitchen",
		"/admin/api/placement",
		"/admin/api/ui/views",
		"/admin/api/ui/snapshot?device_id=d1",
		"/admin/api/recent",
		"/admin/api/store/get?namespace=ns&key=k",
		"/admin/api/store/ns",
		"/admin/api/store/ls?namespace=ns",
		"/admin/api/store/watch?namespace=ns&prefix=k",
		"/admin/api/bus",
		"/admin/api/bus?kind=event&name=alarm&limit=1",
		"/admin/api/bus/replay?from=bus-1&to=bus-9&kind=event",
		"/admin/api/handlers",
		"/admin/api/scenarios/inline",
		"/admin/api/sim/devices",
	}
	for _, path := range getCases {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want 200 body=%s", path, w.Code, w.Body.String())
		}
	}

	postCases := []struct {
		path          string
		form          url.Values
		allowNotFound bool
		allowConflict bool
	}{
		{path: "/admin/api/identity/ack", form: url.Values{"subject_ref": {"message:msg-1"}, "actor": {"device:kitchen-screen"}, "mode": {"dismissed"}}},
		{path: "/admin/api/session/create", form: url.Values{"kind": {"help"}, "target": {"room"}}},
		{path: "/admin/api/session/attach", form: url.Values{"session_id": {"missing"}, "device_ref": {"device:screen-1"}}, allowNotFound: true},
		{path: "/admin/api/session/detach", form: url.Values{"session_id": {"missing"}, "device_ref": {"device:screen-1"}}, allowNotFound: true},
		{path: "/admin/api/session/control/request", form: url.Values{"session_id": {"missing"}, "participant": {"alice"}, "control_type": {"keyboard"}}, allowNotFound: true},
		{path: "/admin/api/session/control/grant", form: url.Values{"session_id": {"missing"}, "participant": {"alice"}, "granted_by": {"moderator"}, "control_type": {"keyboard"}}, allowNotFound: true},
		{path: "/admin/api/session/control/revoke", form: url.Values{"session_id": {"missing"}, "participant": {"alice"}, "revoked_by": {"moderator"}}, allowNotFound: true},
		{path: "/admin/api/message/room", form: url.Values{"name": {"kitchen"}}},
		{path: "/admin/api/message/post", form: url.Values{"room": {"room-1"}, "text": {"hello"}}},
		{path: "/admin/api/message/dm", form: url.Values{"target_ref": {"mom"}, "text": {"hello"}}},
		{path: "/admin/api/board/post", form: url.Values{"board": {"family"}, "text": {"note"}}},
		{path: "/admin/api/board/pin", form: url.Values{"board": {"family"}, "text": {"note"}}},
		{path: "/admin/api/artifact/create", form: url.Values{"kind": {"lesson"}, "title": {"math"}}},
		{path: "/admin/api/artifact/replace", form: url.Values{"artifact_id": {"missing"}, "title": {"ignored"}}, allowNotFound: true},
		{path: "/admin/api/artifact/template/save", form: url.Values{"name": {"base"}, "source_artifact_id": {"missing"}}, allowNotFound: true},
		{path: "/admin/api/artifact/template/apply", form: url.Values{"name": {"base"}, "target_artifact_id": {"missing"}}, allowNotFound: true},
		{path: "/admin/api/canvas/annotate", form: url.Values{"canvas": {"c1"}, "text": {"draw"}}},
		{path: "/admin/api/memory/remember", form: url.Values{"scope": {"kitchen"}, "text": {"milk"}}},
		{path: "/admin/api/ui/views/upsert", form: url.Values{"view_id": {"kitchen-home"}, "root_id": {"root-main"}, "descriptor": {`{"type":"stack"}`}}},
		{path: "/admin/api/ui/push", form: url.Values{"device_id": {"d1"}, "descriptor": {`{"type":"stack"}`}, "root_id": {"root-main"}}},
		{path: "/admin/api/ui/patch", form: url.Values{"device_id": {"d1"}, "component_id": {"banner"}, "descriptor": {`{"type":"text"}`}}},
		{path: "/admin/api/ui/transition", form: url.Values{"device_id": {"d1"}, "component_id": {"banner"}, "transition": {"fade"}, "duration_ms": {"150"}}},
		{path: "/admin/api/ui/subscribe", form: url.Values{"device_id": {"d1"}, "to": {"cohort:family-screens"}}},
		{path: "/admin/api/store/put", form: url.Values{"namespace": {"ns"}, "key": {"k"}, "value": {"v"}}},
		{path: "/admin/api/store/bind", form: url.Values{"namespace": {"ns"}, "key": {"k"}, "to": {"device-1:chat"}}},
		{path: "/admin/api/store/del", form: url.Values{"namespace": {"ns"}, "key": {"k"}}},
		{path: "/admin/api/bus/emit", form: url.Values{"kind": {"event"}, "name": {"alarm"}, "payload": {"ring"}}},
		{path: "/admin/api/handlers/on", form: url.Values{"selector": {"scenario=chat"}, "action": {"submit"}, "run": {"store put notes key value"}}},
		{path: "/admin/api/handlers/off", form: url.Values{"handler_id": {"handler-1"}}},
		{path: "/admin/api/scenarios/inline/define", form: url.Values{"name": {"red_alert"}, "match_intent": {"red alert"}, "on_start": {"ui broadcast all_screens banner"}}},
		{path: "/admin/api/scenarios/inline/undefine", form: url.Values{"name": {"red_alert"}}},
		{path: "/admin/api/sim/devices/new", form: url.Values{"device_id": {"sim-kitchen"}, "caps": {"display,keyboard"}}},
		{path: "/admin/api/sim/input", form: url.Values{"device_id": {"sim-kitchen"}, "component_id": {"chat_box"}, "action": {"submit"}, "value": {"hello"}}},
		{path: "/admin/api/sim/expect", form: url.Values{"device_id": {"sim-kitchen"}, "kind": {"ui"}, "selector": {"chat"}}, allowConflict: true},
		{path: "/admin/api/sim/record", form: url.Values{"device_id": {"sim-kitchen"}, "duration": {"1s"}}, allowNotFound: true},
		{path: "/admin/api/sim/devices/rm", form: url.Values{"device_id": {"sim-kitchen"}}},
		{path: "/admin/api/scripts/run", form: url.Values{"path": {"missing.term"}}, allowNotFound: true},
	}
	for _, tc := range postCases {
		req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if tc.allowNotFound && w.Code == http.StatusNotFound {
			continue
		}
		if tc.allowConflict && w.Code == http.StatusConflict {
			continue
		}
		if w.Code != http.StatusOK {
			t.Fatalf("POST %s status = %d, want 200 body=%s", tc.path, w.Code, w.Body.String())
		}
	}

	roomShowReq := httptest.NewRequest(http.MethodGet, "/admin/api/message/room?room=kitchen", nil)
	roomShowW := httptest.NewRecorder()
	h.ServeHTTP(roomShowW, roomShowReq)
	if roomShowW.Code != http.StatusOK {
		t.Fatalf("GET /admin/api/message/room status = %d, want 200 body=%s", roomShowW.Code, roomShowW.Body.String())
	}
	if !strings.Contains(roomShowW.Body.String(), `"name":"kitchen"`) {
		t.Fatalf("message room show missing kitchen payload: %s", roomShowW.Body.String())
	}

	createReq := httptest.NewRequest(http.MethodPost, "/admin/api/session/create", strings.NewReader(url.Values{
		"kind":   {"lesson"},
		"target": {"math-room"},
	}.Encode()))
	createReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	createW := httptest.NewRecorder()
	h.ServeHTTP(createW, createReq)
	if createW.Code != http.StatusOK {
		t.Fatalf("POST /admin/api/session/create status = %d, want 200 body=%s", createW.Code, createW.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(createW.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode session create response error = %v", err)
	}
	sessionMap, _ := created["session"].(map[string]any)
	sessionID, _ := sessionMap["id"].(string)
	if strings.TrimSpace(sessionID) == "" {
		t.Fatalf("missing created session id in response: %s", createW.Body.String())
	}

	for _, path := range []string{"/admin/api/session/join", "/admin/api/session/leave"} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(url.Values{
			"session_id":  {sessionID},
			"participant": {"alice"},
		}.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("POST %s status = %d, want 200 body=%s", path, w.Code, w.Body.String())
		}
	}

	attachReq := httptest.NewRequest(http.MethodPost, "/admin/api/session/attach", strings.NewReader(url.Values{
		"session_id": {sessionID},
		"device_ref": {"device:kitchen-display"},
	}.Encode()))
	attachReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	attachW := httptest.NewRecorder()
	h.ServeHTTP(attachW, attachReq)
	if attachW.Code != http.StatusOK {
		t.Fatalf("POST /admin/api/session/attach status = %d, want 200 body=%s", attachW.Code, attachW.Body.String())
	}
	if !strings.Contains(attachW.Body.String(), "device:kitchen-display") {
		t.Fatalf("attach response missing device ref: %s", attachW.Body.String())
	}

	for _, path := range []struct {
		url  string
		form url.Values
	}{
		{url: "/admin/api/session/control/request", form: url.Values{"session_id": {sessionID}, "participant": {"alice"}, "control_type": {"keyboard"}}},
		{url: "/admin/api/session/control/grant", form: url.Values{"session_id": {sessionID}, "participant": {"alice"}, "granted_by": {"moderator"}, "control_type": {"keyboard"}}},
		{url: "/admin/api/session/control/revoke", form: url.Values{"session_id": {sessionID}, "participant": {"alice"}, "revoked_by": {"moderator"}}},
		{url: "/admin/api/session/detach", form: url.Values{"session_id": {sessionID}, "device_ref": {"device:kitchen-display"}}},
	} {
		req := httptest.NewRequest(http.MethodPost, path.url, strings.NewReader(path.form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("POST %s status = %d, want 200 body=%s", path.url, w.Code, w.Body.String())
		}
	}

	for _, path := range []string{
		"/admin/api/session/show?session_id=" + url.QueryEscape(sessionID),
		"/admin/api/session/members?session_id=" + url.QueryEscape(sessionID),
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want 200 body=%s", path, w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), sessionID) {
			t.Fatalf("GET %s missing session id %q in body=%s", path, sessionID, w.Body.String())
		}
	}

	postMessageReq := httptest.NewRequest(http.MethodPost, "/admin/api/message/post", strings.NewReader(url.Values{
		"room": {"room-1"},
		"text": {"bring milk"},
	}.Encode()))
	postMessageReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postMessageW := httptest.NewRecorder()
	h.ServeHTTP(postMessageW, postMessageReq)
	if postMessageW.Code != http.StatusOK {
		t.Fatalf("POST /admin/api/message/post status = %d, want 200 body=%s", postMessageW.Code, postMessageW.Body.String())
	}
	var posted map[string]any
	if err := json.Unmarshal(postMessageW.Body.Bytes(), &posted); err != nil {
		t.Fatalf("decode message post response error = %v", err)
	}
	messageMap, _ := posted["message"].(map[string]any)
	messageID, _ := messageMap["id"].(string)
	if strings.TrimSpace(messageID) == "" {
		t.Fatalf("missing message id in response: %s", postMessageW.Body.String())
	}

	getMessageReq := httptest.NewRequest(http.MethodGet, "/admin/api/message/get?message_id="+url.QueryEscape(messageID), nil)
	getMessageW := httptest.NewRecorder()
	h.ServeHTTP(getMessageW, getMessageReq)
	if getMessageW.Code != http.StatusOK {
		t.Fatalf("GET /admin/api/message/get status = %d, want 200 body=%s", getMessageW.Code, getMessageW.Body.String())
	}
	if !strings.Contains(getMessageW.Body.String(), messageID) {
		t.Fatalf("message get response missing message id %q: %s", messageID, getMessageW.Body.String())
	}

	threadReq := httptest.NewRequest(http.MethodPost, "/admin/api/message/thread", strings.NewReader(url.Values{
		"root_ref": {messageID},
		"text":     {"thread follow-up"},
	}.Encode()))
	threadReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	threadW := httptest.NewRecorder()
	h.ServeHTTP(threadW, threadReq)
	if threadW.Code != http.StatusOK {
		t.Fatalf("POST /admin/api/message/thread status = %d, want 200 body=%s", threadW.Code, threadW.Body.String())
	}
	if !strings.Contains(threadW.Body.String(), `"thread_root_ref":"`+messageID+`"`) {
		t.Fatalf("thread response missing root ref %q: %s", messageID, threadW.Body.String())
	}
	var threaded map[string]any
	if err := json.Unmarshal(threadW.Body.Bytes(), &threaded); err != nil {
		t.Fatalf("decode message thread response error = %v", err)
	}
	threadMessageMap, _ := threaded["message"].(map[string]any)
	threadMessageID, _ := threadMessageMap["id"].(string)
	if strings.TrimSpace(threadMessageID) == "" {
		t.Fatalf("missing threaded message id in response: %s", threadW.Body.String())
	}

	directReq := httptest.NewRequest(http.MethodPost, "/admin/api/message/dm", strings.NewReader(url.Values{
		"target_ref": {"mom"},
		"text":       {"come downstairs"},
	}.Encode()))
	directReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	directW := httptest.NewRecorder()
	h.ServeHTTP(directW, directReq)
	if directW.Code != http.StatusOK {
		t.Fatalf("POST /admin/api/message/dm status = %d, want 200 body=%s", directW.Code, directW.Body.String())
	}
	if !strings.Contains(directW.Body.String(), "person:mom") {
		t.Fatalf("direct message response missing normalized target ref: %s", directW.Body.String())
	}

	boardPostReq := httptest.NewRequest(http.MethodPost, "/admin/api/board/post", strings.NewReader(url.Values{
		"board": {"family"},
		"text":  {"Need milk"},
	}.Encode()))
	boardPostReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	boardPostW := httptest.NewRecorder()
	h.ServeHTTP(boardPostW, boardPostReq)
	if boardPostW.Code != http.StatusOK {
		t.Fatalf("POST /admin/api/board/post status = %d, want 200 body=%s", boardPostW.Code, boardPostW.Body.String())
	}
	if strings.Contains(boardPostW.Body.String(), `"pinned":true`) {
		t.Fatalf("board post response should not be pinned: %s", boardPostW.Body.String())
	}

	ackReq := httptest.NewRequest(http.MethodPost, "/admin/api/message/ack", strings.NewReader(url.Values{
		"identity_id": {"alice"},
		"message_id":  {messageID},
	}.Encode()))
	ackReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ackW := httptest.NewRecorder()
	h.ServeHTTP(ackW, ackReq)
	if ackW.Code != http.StatusOK {
		t.Fatalf("POST /admin/api/message/ack status = %d, want 200 body=%s", ackW.Code, ackW.Body.String())
	}

	ackThreadReq := httptest.NewRequest(http.MethodPost, "/admin/api/message/ack", strings.NewReader(url.Values{
		"identity_id": {"alice"},
		"message_id":  {threadMessageID},
	}.Encode()))
	ackThreadReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ackThreadW := httptest.NewRecorder()
	h.ServeHTTP(ackThreadW, ackThreadReq)
	if ackThreadW.Code != http.StatusOK {
		t.Fatalf("POST /admin/api/message/ack (thread) status = %d, want 200 body=%s", ackThreadW.Code, ackThreadW.Body.String())
	}

	unreadReq := httptest.NewRequest(http.MethodGet, "/admin/api/message/unread?identity_id=alice&room=room-1", nil)
	unreadW := httptest.NewRecorder()
	h.ServeHTTP(unreadW, unreadReq)
	if unreadW.Code != http.StatusOK {
		t.Fatalf("GET /admin/api/message/unread status = %d, want 200 body=%s", unreadW.Code, unreadW.Body.String())
	}
	if strings.Contains(unreadW.Body.String(), messageID) {
		t.Fatalf("expected message %q to be acknowledged, unread body=%s", messageID, unreadW.Body.String())
	}

	createArtifactReq := httptest.NewRequest(http.MethodPost, "/admin/api/artifact/create", strings.NewReader(url.Values{
		"kind":  {"lesson"},
		"title": {"math basics"},
	}.Encode()))
	createArtifactReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	createArtifactW := httptest.NewRecorder()
	h.ServeHTTP(createArtifactW, createArtifactReq)
	if createArtifactW.Code != http.StatusOK {
		t.Fatalf("POST /admin/api/artifact/create status = %d, want 200 body=%s", createArtifactW.Code, createArtifactW.Body.String())
	}
	var createdArtifact map[string]any
	if err := json.Unmarshal(createArtifactW.Body.Bytes(), &createdArtifact); err != nil {
		t.Fatalf("decode artifact create response error = %v", err)
	}
	artifactMap, _ := createdArtifact["artifact"].(map[string]any)
	artifactID, _ := artifactMap["id"].(string)
	if strings.TrimSpace(artifactID) == "" {
		t.Fatalf("missing artifact id in response: %s", createArtifactW.Body.String())
	}

	patchArtifactReq := httptest.NewRequest(http.MethodPost, "/admin/api/artifact/patch", strings.NewReader(url.Values{
		"artifact_id": {artifactID},
		"title":       {"math mastery"},
	}.Encode()))
	patchArtifactReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	patchArtifactW := httptest.NewRecorder()
	h.ServeHTTP(patchArtifactW, patchArtifactReq)
	if patchArtifactW.Code != http.StatusOK {
		t.Fatalf("POST /admin/api/artifact/patch status = %d, want 200 body=%s", patchArtifactW.Code, patchArtifactW.Body.String())
	}
	if !strings.Contains(patchArtifactW.Body.String(), "math mastery") {
		t.Fatalf("patched artifact missing updated title: %s", patchArtifactW.Body.String())
	}

	replaceArtifactReq := httptest.NewRequest(http.MethodPost, "/admin/api/artifact/replace", strings.NewReader(url.Values{
		"artifact_id": {artifactID},
		"title":       {"math replacement"},
	}.Encode()))
	replaceArtifactReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	replaceArtifactW := httptest.NewRecorder()
	h.ServeHTTP(replaceArtifactW, replaceArtifactReq)
	if replaceArtifactW.Code != http.StatusOK {
		t.Fatalf("POST /admin/api/artifact/replace status = %d, want 200 body=%s", replaceArtifactW.Code, replaceArtifactW.Body.String())
	}
	if !strings.Contains(replaceArtifactW.Body.String(), "math replacement") {
		t.Fatalf("replaced artifact missing updated title: %s", replaceArtifactW.Body.String())
	}

	saveTemplateReq := httptest.NewRequest(http.MethodPost, "/admin/api/artifact/template/save", strings.NewReader(url.Values{
		"name":               {"lesson-base"},
		"source_artifact_id": {artifactID},
	}.Encode()))
	saveTemplateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	saveTemplateW := httptest.NewRecorder()
	h.ServeHTTP(saveTemplateW, saveTemplateReq)
	if saveTemplateW.Code != http.StatusOK {
		t.Fatalf("POST /admin/api/artifact/template/save status = %d, want 200 body=%s", saveTemplateW.Code, saveTemplateW.Body.String())
	}
	if !strings.Contains(saveTemplateW.Body.String(), "lesson-base") {
		t.Fatalf("template save response missing template name: %s", saveTemplateW.Body.String())
	}

	applyTemplateReq := httptest.NewRequest(http.MethodPost, "/admin/api/artifact/template/apply", strings.NewReader(url.Values{
		"name":               {"lesson-base"},
		"target_artifact_id": {artifactID},
	}.Encode()))
	applyTemplateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	applyTemplateW := httptest.NewRecorder()
	h.ServeHTTP(applyTemplateW, applyTemplateReq)
	if applyTemplateW.Code != http.StatusOK {
		t.Fatalf("POST /admin/api/artifact/template/apply status = %d, want 200 body=%s", applyTemplateW.Code, applyTemplateW.Body.String())
	}
	if !strings.Contains(applyTemplateW.Body.String(), "lesson-base") && !strings.Contains(applyTemplateW.Body.String(), "math replacement") {
		t.Fatalf("template apply response missing expected payload: %s", applyTemplateW.Body.String())
	}

	getArtifactReq := httptest.NewRequest(http.MethodGet, "/admin/api/artifact/get?artifact_id="+url.QueryEscape(artifactID), nil)
	getArtifactW := httptest.NewRecorder()
	h.ServeHTTP(getArtifactW, getArtifactReq)
	if getArtifactW.Code != http.StatusOK {
		t.Fatalf("GET /admin/api/artifact/get status = %d, want 200 body=%s", getArtifactW.Code, getArtifactW.Body.String())
	}
	if !strings.Contains(getArtifactW.Body.String(), `"version":4`) {
		t.Fatalf("artifact get body missing updated version: %s", getArtifactW.Body.String())
	}

	historyReq := httptest.NewRequest(http.MethodGet, "/admin/api/artifact/history?artifact_id="+url.QueryEscape(artifactID), nil)
	historyW := httptest.NewRecorder()
	h.ServeHTTP(historyW, historyReq)
	if historyW.Code != http.StatusOK {
		t.Fatalf("GET /admin/api/artifact/history status = %d, want 200 body=%s", historyW.Code, historyW.Body.String())
	}
	if !strings.Contains(historyW.Body.String(), `"action":"create"`) || !strings.Contains(historyW.Body.String(), `"action":"patch"`) || !strings.Contains(historyW.Body.String(), `"action":"replace"`) {
		t.Fatalf("artifact history missing create/patch/replace entries: %s", historyW.Body.String())
	}
}

func TestIdentityResolveEndpointRequiresGET(t *testing.T) {
	h := testHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/admin/api/identity/resolve", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405 body=%s", w.Code, w.Body.String())
	}
}
