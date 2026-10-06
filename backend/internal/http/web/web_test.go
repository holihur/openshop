package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFrontAndOpsServeOK(t *testing.T) {
	cases := map[string]struct {
		handler http.Handler
		path    string
	}{
		"front": {Front(), "/"},
		"ops":   {Ops(), "/ops"},
	}
	for name, tc := range cases {
		rec := httptest.NewRecorder()
		tc.handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200", name, rec.Code)
		}
		if rec.Body.Len() == 0 {
			t.Errorf("%s: empty body", name)
		}
	}
}

func TestOpsRejectsNonOpsPath(t *testing.T) {
	rec := httptest.NewRecorder()
	Ops().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/not-ops", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestOpsStripsPrefix(t *testing.T) {
	// A client-side route under /ops must fall back to index (200), not 404.
	rec := httptest.NewRecorder()
	Ops().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ops/products", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestUnbuiltServesPlaceholder(t *testing.T) {
	if FrontBuilt() && OpsBuilt() {
		t.Skip("frontends are embedded; placeholder path not exercised")
	}
	rec := httptest.NewRecorder()
	Front().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("content-type = %q", got)
	}
}
