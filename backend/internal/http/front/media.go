package front

import (
	"bytes"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// mediaHandler serves locally stored uploads. When a raster image is requested
// with ?w=NNN it resizes on the fly and caches the result on disk, so the
// storefront can use responsive srcset without an external image service. SVGs
// (vector) and unsupported formats are served unchanged.
func mediaHandler(dir string) http.Handler {
	root := filepath.Clean(dir)
	fileServer := http.FileServer(http.Dir(root))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := strings.TrimPrefix(r.URL.Path, "/uploads/")
		clean := filepath.Clean("/" + rel)
		full, ok := safePath(root, clean)
		if !ok {
			http.NotFound(w, r)
			return
		}

		width := parseWidth(r.URL.Query().Get("w"))
		serveOriginal := func() {
			// Uploads are attacker-influenced content served from the storefront
			// origin: never let a browser sniff or execute them.
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
			if !isRaster(full) {
				w.Header().Set("Content-Disposition", "attachment")
			}
			req := r.Clone(r.Context())
			req.URL.Path = "/" + strings.TrimPrefix(clean, "/")
			fileServer.ServeHTTP(w, req)
		}
		if width == 0 || !isRaster(full) {
			serveOriginal()
			return
		}
		data, ok := resizeCached(root, clean, full, width)
		if !ok {
			// Decode failed or the image is already small enough: serve original.
			serveOriginal()
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		http.ServeContent(w, r, filepath.Base(full), modTime(full), bytes.NewReader(data))
	})
}

// safePath resolves an untrusted request path against the storage root and
// reports whether the result stays inside it. It is the single place where a
// URL becomes a filesystem path, so the traversal check cannot be bypassed by
// a later caller.
func safePath(root, clean string) (string, bool) {
	full := filepath.Join(root, clean)
	if full != root && !strings.HasPrefix(full, root+string(os.PathSeparator)) {
		return "", false
	}
	return full, true
}

func parseWidth(raw string) int {
	if raw == "" {
		return 0
	}
	w, err := strconv.Atoi(raw)
	if err != nil || w < 16 || w > 4000 {
		return 0
	}
	return w
}

func isRaster(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	default:
		return false
	}
}

// resizeCached returns the JPEG-encoded image scaled to width. It reads from and
// writes to <root>/.cache so repeated requests are cheap.
func resizeCached(root, clean, full string, width int) ([]byte, bool) {
	cachePath := filepath.Join(root, ".cache", strings.TrimPrefix(clean, "/")+"_"+strconv.Itoa(width)+".jpg")
	// The cache path is derived from an already-validated path, so it is safe to
	// read and write directly.
	if data, err := os.ReadFile(cachePath); err == nil { // #nosec G304 G703 -- derived from a validated path
		return data, true
	}

	f, err := os.Open(full) // #nosec G304 -- validated by safePath
	if err != nil {
		return nil, false
	}
	defer f.Close()

	src, _, err := image.Decode(f)
	if err != nil {
		return nil, false
	}
	b := src.Bounds()
	if b.Dx() <= width {
		return nil, false // never upscale
	}
	height := b.Dy() * width / b.Dx()
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 82}); err != nil {
		return nil, false
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o750); err == nil { // #nosec G703 -- derived from a validated path
		_ = os.WriteFile(cachePath, buf.Bytes(), 0o600) // #nosec G304 G703 -- derived from a validated path
	}
	return buf.Bytes(), true
}

func modTime(path string) time.Time {
	if info, err := os.Stat(path); err == nil { // #nosec G304 -- validated by safePath
		return info.ModTime()
	}
	return time.Time{}
}
