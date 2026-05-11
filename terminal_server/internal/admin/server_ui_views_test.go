package admin

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestUIViewEndpointsCRUD(t *testing.T) {
	h := testHandler(t)

	upsertReq := httptest.NewRequest(http.MethodPost, "/admin/api/ui/views/upsert", strings.NewReader(url.Values{
		"view_id":    {"Kitchen-Home"},
		"root_id":    {"root-main"},
		"descriptor": {`{"type":"stack","children":[]}`},
	}.Encode()))
	upsertReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	upsertW := httptest.NewRecorder()
	h.ServeHTTP(upsertW, upsertReq)
	if upsertW.Code != http.StatusOK {
		t.Fatalf("ui view upsert status = %d, want 200 body=%s", upsertW.Code, upsertW.Body.String())
	}
	if !strings.Contains(upsertW.Body.String(), `"view_id":"kitchen-home"`) {
		t.Fatalf("ui view upsert body missing normalized view_id: %s", upsertW.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/api/ui/views", nil)
	listW := httptest.NewRecorder()
	h.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("ui views list status = %d, want 200 body=%s", listW.Code, listW.Body.String())
	}
	if !strings.Contains(listW.Body.String(), `"kitchen-home"`) {
		t.Fatalf("ui views list missing kitchen-home: %s", listW.Body.String())
	}

	showReq := httptest.NewRequest(http.MethodGet, "/admin/api/ui/views?view_id=kitchen-home", nil)
	showW := httptest.NewRecorder()
	h.ServeHTTP(showW, showReq)
	if showW.Code != http.StatusOK {
		t.Fatalf("ui views show status = %d, want 200 body=%s", showW.Code, showW.Body.String())
	}
	if !strings.Contains(showW.Body.String(), `"root_id":"root-main"`) {
		t.Fatalf("ui views show missing root_id: %s", showW.Body.String())
	}

	delReq := httptest.NewRequest(http.MethodPost, "/admin/api/ui/views/del", strings.NewReader(url.Values{
		"view_id": {"kitchen-home"},
	}.Encode()))
	delReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	delW := httptest.NewRecorder()
	h.ServeHTTP(delW, delReq)
	if delW.Code != http.StatusOK {
		t.Fatalf("ui views delete status = %d, want 200 body=%s", delW.Code, delW.Body.String())
	}
	if !strings.Contains(delW.Body.String(), `"deleted":true`) {
		t.Fatalf("ui views delete body missing deleted=true: %s", delW.Body.String())
	}

	placementReq := httptest.NewRequest(http.MethodPost, "/admin/api/devices/placement", strings.NewReader(url.Values{
		"device_id": {"d1"},
		"zone":      {"kitchen"},
		"roles":     {"screen"},
	}.Encode()))
	placementReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	placementW := httptest.NewRecorder()
	h.ServeHTTP(placementW, placementReq)
	if placementW.Code != http.StatusOK {
		t.Fatalf("placement status = %d, want 200 body=%s", placementW.Code, placementW.Body.String())
	}

	cohortReq := httptest.NewRequest(http.MethodPost, "/admin/api/cohort/upsert", strings.NewReader(url.Values{
		"name":      {"family-screens"},
		"selectors": {"zone:kitchen,role:screen"},
	}.Encode()))
	cohortReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	cohortW := httptest.NewRecorder()
	h.ServeHTTP(cohortW, cohortReq)
	if cohortW.Code != http.StatusOK {
		t.Fatalf("cohort upsert status = %d, want 200 body=%s", cohortW.Code, cohortW.Body.String())
	}

	pushReq := httptest.NewRequest(http.MethodPost, "/admin/api/ui/push", strings.NewReader(url.Values{
		"device_id":  {"d1"},
		"root_id":    {"root-main"},
		"descriptor": {`{"type":"stack"}`},
	}.Encode()))
	pushReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	pushW := httptest.NewRecorder()
	h.ServeHTTP(pushW, pushReq)
	if pushW.Code != http.StatusOK {
		t.Fatalf("ui push status = %d, want 200 body=%s", pushW.Code, pushW.Body.String())
	}
	if !strings.Contains(pushW.Body.String(), `"device_id":"d1"`) {
		t.Fatalf("ui push body missing device id: %s", pushW.Body.String())
	}

	patchReq := httptest.NewRequest(http.MethodPost, "/admin/api/ui/patch", strings.NewReader(url.Values{
		"device_id":    {"d1"},
		"component_id": {"banner"},
		"descriptor":   {`{"type":"text"}`},
	}.Encode()))
	patchReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	patchW := httptest.NewRecorder()
	h.ServeHTTP(patchW, patchReq)
	if patchW.Code != http.StatusOK {
		t.Fatalf("ui patch status = %d, want 200 body=%s", patchW.Code, patchW.Body.String())
	}
	if !strings.Contains(patchW.Body.String(), `"last_patch_component_id":"banner"`) {
		t.Fatalf("ui patch body missing patch component id: %s", patchW.Body.String())
	}

	transitionReq := httptest.NewRequest(http.MethodPost, "/admin/api/ui/transition", strings.NewReader(url.Values{
		"device_id":    {"d1"},
		"component_id": {"banner"},
		"transition":   {"fade"},
		"duration_ms":  {"150"},
	}.Encode()))
	transitionReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	transitionW := httptest.NewRecorder()
	h.ServeHTTP(transitionW, transitionReq)
	if transitionW.Code != http.StatusOK {
		t.Fatalf("ui transition status = %d, want 200 body=%s", transitionW.Code, transitionW.Body.String())
	}
	if !strings.Contains(transitionW.Body.String(), `"last_transition":"fade"`) {
		t.Fatalf("ui transition body missing transition name: %s", transitionW.Body.String())
	}

	broadcastReq := httptest.NewRequest(http.MethodPost, "/admin/api/ui/broadcast", strings.NewReader(url.Values{
		"cohort":     {"family-screens"},
		"descriptor": {`{"type":"banner"}`},
		"patch_id":   {"alert-banner"},
	}.Encode()))
	broadcastReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	broadcastW := httptest.NewRecorder()
	h.ServeHTTP(broadcastW, broadcastReq)
	if broadcastW.Code != http.StatusOK {
		t.Fatalf("ui broadcast status = %d, want 200 body=%s", broadcastW.Code, broadcastW.Body.String())
	}
	if !strings.Contains(broadcastW.Body.String(), `"members":["d1"]`) {
		t.Fatalf("ui broadcast body missing resolved members: %s", broadcastW.Body.String())
	}

	subscribeReq := httptest.NewRequest(http.MethodPost, "/admin/api/ui/subscribe", strings.NewReader(url.Values{
		"device_id": {"d1"},
		"to":        {"cohort:family-screens"},
	}.Encode()))
	subscribeReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	subscribeW := httptest.NewRecorder()
	h.ServeHTTP(subscribeW, subscribeReq)
	if subscribeW.Code != http.StatusOK {
		t.Fatalf("ui subscribe status = %d, want 200 body=%s", subscribeW.Code, subscribeW.Body.String())
	}
	if !strings.Contains(subscribeW.Body.String(), `"subscriptions":["cohort:family-screens"]`) {
		t.Fatalf("ui subscribe body missing subscription target: %s", subscribeW.Body.String())
	}

	snapshotReq := httptest.NewRequest(http.MethodGet, "/admin/api/ui/snapshot?device_id=d1", nil)
	snapshotW := httptest.NewRecorder()
	h.ServeHTTP(snapshotW, snapshotReq)
	if snapshotW.Code != http.StatusOK {
		t.Fatalf("ui snapshot status = %d, want 200 body=%s", snapshotW.Code, snapshotW.Body.String())
	}
	if !strings.Contains(snapshotW.Body.String(), `"device_id":"d1"`) {
		t.Fatalf("ui snapshot body missing device id: %s", snapshotW.Body.String())
	}
}
