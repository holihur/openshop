// Package web serves a single-page application from any fs.FS. It is shared by
// the front and ops surfaces, but it embeds nothing itself: each surface owns
// its own //go:embed, so a binary only contains the SPA it serves.
package web

import (
	"io"
	"io/fs"
	"net/http"
	"strings"
)

// Handler serves an SPA rooted at fsys (the build output directory). Hashed
// assets get long-lived cache headers; unknown paths fall back to index.html so
// client-side routing survives a hard refresh. When no index.html is present it
// serves a placeholder page naming the surface.
func Handler(fsys fs.FS, label string) http.Handler {
	fileServer := http.FileServer(http.FS(fsys))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		clean := strings.TrimPrefix(r.URL.Path, "/")
		if clean == "" {
			serveIndex(w, fsys, label)
			return
		}
		if info, err := fs.Stat(fsys, clean); err == nil && !info.IsDir() {
			if strings.HasPrefix(clean, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			req := r.Clone(r.Context())
			req.URL.Path = "/" + clean
			fileServer.ServeHTTP(w, req)
			return
		}
		serveIndex(w, fsys, label)
	})
}

// Built reports whether fsys contains a real build (index.html present).
func Built(fsys fs.FS) bool {
	_, err := fs.Stat(fsys, "index.html")
	return err == nil
}

func serveIndex(w http.ResponseWriter, fsys fs.FS, label string) {
	index, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		placeholder(w, label)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(index)
}

func placeholder(w http.ResponseWriter, label string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, strings.ReplaceAll(placeholderHTML, "{{LABEL}}", label))
}

const placeholderHTML = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><title>OpenShop</title>
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>body{font-family:system-ui,sans-serif;max-width:40rem;margin:15vh auto;padding:0 1.5rem;line-height:1.6;color:#111}
code{background:#f2f2f2;padding:.15rem .4rem;border-radius:.25rem}</style></head>
<body><h1>OpenShop API is running</h1>
<p>The {{LABEL}} has not been embedded in this build.</p>
<p>Build it with <code>make fe-build</code> (which builds the SPAs and copies their
<code>dist/</code> into the Go module), then rebuild. Release binaries from GitHub
already include the SPA.</p>
</body></html>
`
