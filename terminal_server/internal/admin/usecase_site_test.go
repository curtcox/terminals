package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUsecaseSiteRouteServesGeneratedIndexAndPages(t *testing.T) {
	handler := testHandler(t)

	indexReq := httptest.NewRequest(http.MethodGet, "/docs/usecases/", nil)
	indexRec := httptest.NewRecorder()
	handler.ServeHTTP(indexRec, indexReq)
	if indexRec.Code != http.StatusOK {
		t.Fatalf("GET /docs/usecases/ status=%d, want %d", indexRec.Code, http.StatusOK)
	}
	if body := indexRec.Body.String(); !strings.Contains(body, "Terminals Use Cases") || !strings.Contains(body, "UNTESTED") {
		t.Fatalf("index body missing generated site markers")
	}

	pageReq := httptest.NewRequest(http.MethodGet, "/docs/usecases/C1.html", nil)
	pageRec := httptest.NewRecorder()
	handler.ServeHTTP(pageRec, pageReq)
	if pageRec.Code != http.StatusOK {
		t.Fatalf("GET /docs/usecases/C1.html status=%d, want %d", pageRec.Code, http.StatusOK)
	}
	if body := pageRec.Body.String(); !strings.Contains(body, "Communication Use Cases") || !strings.Contains(body, "What it does") {
		t.Fatalf("C1 page body missing generated page markers")
	}
}

func TestUsecaseSiteRouteRejectsWrites(t *testing.T) {
	handler := testHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/docs/usecases/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /docs/usecases/ status=%d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}
