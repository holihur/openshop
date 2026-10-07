package front

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func writePNG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("write png: %v", err)
	}
}

func TestMediaHandlerResizesRaster(t *testing.T) {
	dir := t.TempDir()
	writePNG(t, filepath.Join(dir, "big.png"), 1200, 600)

	h := mediaHandler(dir)

	// Original is served untouched.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/uploads/big.png", nil))
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("original: code=%d type=%q", rec.Code, rec.Header().Get("Content-Type"))
	}

	// ?w=400 returns a JPEG scaled to 400px wide.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/uploads/big.png?w=400", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("resized: code=%d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "image/jpeg" {
		t.Fatalf("resized content-type = %q", got)
	}
	img, _, err := image.Decode(rec.Body)
	if err != nil {
		t.Fatalf("decode resized: %v", err)
	}
	if w := img.Bounds().Dx(); w != 400 {
		t.Fatalf("resized width = %d, want 400", w)
	}

	// The resized variant is cached on disk.
	if _, err := os.Stat(filepath.Join(dir, ".cache", "big.png_400.jpg")); err != nil {
		t.Fatalf("expected cached variant: %v", err)
	}
}

func TestMediaHandlerServesSVGUnchanged(t *testing.T) {
	dir := t.TempDir()
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"></svg>`)
	if err := os.WriteFile(filepath.Join(dir, "logo.svg"), svg, 0o644); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	mediaHandler(dir).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/uploads/logo.svg?w=100", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("svg: code=%d", rec.Code)
	}
	if !bytes.Equal(rec.Body.Bytes(), svg) {
		t.Fatalf("svg body changed")
	}
}

func TestMediaHandlerBlocksTraversal(t *testing.T) {
	dir := t.TempDir()
	rec := httptest.NewRecorder()
	mediaHandler(dir).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/uploads/../../etc/passwd", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("traversal: code=%d, want 404", rec.Code)
	}
}
