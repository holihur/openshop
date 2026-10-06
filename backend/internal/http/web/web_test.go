package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFrontAndOpsServeOK(t *testing.T) {
	cases := map[string]http.Handler{"front": Front(), "ops": Ops()}
	for name, handler := range cases {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		if rec.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200", name, rec.Code)
		}
		if rec.Body.Len() == 0 {
			t.Errorf("%s: empty body", name)
		}
	}
}

func TestServesClientRoutes(t *testing.T) {
	// A client-side route must fall back to index (200), not 404.
	rec := httptest.NewRecorder()
	Front().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/products/abc", nil))
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
