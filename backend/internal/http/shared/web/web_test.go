package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func builtFS() fstest.MapFS {
	return fstest.MapFS{
		"index.html":    &fstest.MapFile{Data: []byte("<html>app</html>")},
		"assets/app.js": &fstest.MapFile{Data: []byte("console.log(1)")},
	}
}

func TestHandlerServesIndexAssetAndFallback(t *testing.T) {
	h := Handler(builtFS(), "storefront")

	// Root serves index.html.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "app") {
		t.Fatalf("root: code=%d body=%q", rec.Code, rec.Body.String())
	}

	// A real asset is served with an immutable cache header.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/assets/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("asset: code=%d", rec.Code)
	}
	if rec.Header().Get("Cache-Control") == "" {
		t.Fatal("asset: missing Cache-Control")
	}

	// An unknown client route falls back to index.html.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/products/abc", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "app") {
		t.Fatalf("fallback: code=%d body=%q", rec.Code, rec.Body.String())
	}

	// Non-GET is rejected.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("post: code=%d, want 405", rec.Code)
	}
}

func TestBuiltAndPlaceholder(t *testing.T) {
	if Built(fstest.MapFS{}) {
		t.Fatal("empty fs reported as built")
	}
	h := Handler(fstest.MapFS{}, "admin console")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("placeholder: code=%d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "admin console") {
		t.Fatalf("placeholder missing label: %q", rec.Body.String())
	}
}
