package admin

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestStorePutTTLValidation(t *testing.T) {
	h := testHandler(t)

	badTTLReq := httptest.NewRequest(http.MethodPost, "/admin/api/store/put", strings.NewReader(url.Values{
		"namespace": {"ns"},
		"key":       {"k"},
		"value":     {"v"},
		"ttl":       {"invalid"},
	}.Encode()))
	badTTLReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	badTTLW := httptest.NewRecorder()
	h.ServeHTTP(badTTLW, badTTLReq)
	if badTTLW.Code != http.StatusBadRequest {
		t.Fatalf("invalid ttl status = %d, want 400 body=%s", badTTLW.Code, badTTLW.Body.String())
	}

	zeroTTLReq := httptest.NewRequest(http.MethodPost, "/admin/api/store/put", strings.NewReader(url.Values{
		"namespace": {"ns"},
		"key":       {"k"},
		"value":     {"v"},
		"ttl":       {"0s"},
	}.Encode()))
	zeroTTLReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	zeroTTLW := httptest.NewRecorder()
	h.ServeHTTP(zeroTTLW, zeroTTLReq)
	if zeroTTLW.Code != http.StatusBadRequest {
		t.Fatalf("zero ttl status = %d, want 400 body=%s", zeroTTLW.Code, zeroTTLW.Body.String())
	}

	goodTTLReq := httptest.NewRequest(http.MethodPost, "/admin/api/store/put", strings.NewReader(url.Values{
		"namespace": {"ns"},
		"key":       {"k"},
		"value":     {"v"},
		"ttl":       {"1m"},
	}.Encode()))
	goodTTLReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	goodTTLW := httptest.NewRecorder()
	h.ServeHTTP(goodTTLW, goodTTLReq)
	if goodTTLW.Code != http.StatusOK {
		t.Fatalf("valid ttl status = %d, want 200 body=%s", goodTTLW.Code, goodTTLW.Body.String())
	}
	if !strings.Contains(goodTTLW.Body.String(), "expires_at") {
		t.Fatalf("valid ttl response missing expires_at: %s", goodTTLW.Body.String())
	}
}

func TestStoreBindAndBusTailLimitValidation(t *testing.T) {
	h := testHandler(t)

	putReq := httptest.NewRequest(http.MethodPost, "/admin/api/store/put", strings.NewReader(url.Values{
		"namespace": {"ns"},
		"key":       {"k"},
		"value":     {"v"},
	}.Encode()))
	putReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	putW := httptest.NewRecorder()
	h.ServeHTTP(putW, putReq)
	if putW.Code != http.StatusOK {
		t.Fatalf("store put status = %d, want 200 body=%s", putW.Code, putW.Body.String())
	}

	badBindReq := httptest.NewRequest(http.MethodPost, "/admin/api/store/bind", strings.NewReader(url.Values{
		"namespace": {"ns"},
		"key":       {"k"},
		"to":        {"badbinding"},
	}.Encode()))
	badBindReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	badBindW := httptest.NewRecorder()
	h.ServeHTTP(badBindW, badBindReq)
	if badBindW.Code != http.StatusBadRequest {
		t.Fatalf("bad bind status = %d, want 400 body=%s", badBindW.Code, badBindW.Body.String())
	}

	goodBindReq := httptest.NewRequest(http.MethodPost, "/admin/api/store/bind", strings.NewReader(url.Values{
		"namespace": {"ns"},
		"key":       {"k"},
		"to":        {"device-1:chat"},
	}.Encode()))
	goodBindReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	goodBindW := httptest.NewRecorder()
	h.ServeHTTP(goodBindW, goodBindReq)
	if goodBindW.Code != http.StatusOK {
		t.Fatalf("good bind status = %d, want 200 body=%s", goodBindW.Code, goodBindW.Body.String())
	}

	badLimitReq := httptest.NewRequest(http.MethodGet, "/admin/api/bus?limit=zero", nil)
	badLimitW := httptest.NewRecorder()
	h.ServeHTTP(badLimitW, badLimitReq)
	if badLimitW.Code != http.StatusBadRequest {
		t.Fatalf("bad bus limit status = %d, want 400 body=%s", badLimitW.Code, badLimitW.Body.String())
	}
}
