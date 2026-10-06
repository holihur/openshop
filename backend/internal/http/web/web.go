// Package web embeds the two compiled single-page applications and serves them
// from the Go binary: the customer storefront at the site root and the admin
// console under /ops. One process therefore serves the API and both frontends,
// which is what makes the release binary a self-contained install.
//
// The front/dist and ops/dist directories are populated by
// scripts/embed-frontend.sh (or the release pipeline) and are gitignored apart
// from a placeholder, so a plain `go build` still compiles and serves a helpful
// placeholder page instead of failing.
package web

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:front/dist
var frontFS embed.FS

//go:embed all:ops/dist
var opsFS embed.FS

// Front returns the handler for the customer storefront, mounted at "/".
func Front() http.Handler { return spa(frontFS, "front/dist", "") }

// Ops returns the handler for the admin console, mounted under "/ops".
func Ops() http.Handler { return spa(opsFS, "ops/dist", "/ops") }

// FrontBuilt reports whether a real storefront build is embedded.
func FrontBuilt() bool { return built(frontFS, "front/dist") }

// OpsBuilt reports whether a real admin console build is embedded.
func OpsBuilt() bool { return built(opsFS, "ops/dist") }

func built(efs embed.FS, root string) bool {
	sub, err := fs.Sub(efs, root)
	if err != nil {
		return false
	}
	_, err = fs.Stat(sub, "index.html")
	return err == nil
}

// spa serves a single-page app from an embedded directory. Hashed assets get
// long-lived cache headers; unknown paths fall back to index.html so
// client-side routing survives a hard refresh. When no build is embedded it
// serves a placeholder page.
func spa(efs embed.FS, root, prefix string) http.Handler {
	sub, err := fs.Sub(efs, root)
	if err != nil {
		return http.HandlerFunc(placeholder)
	}
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		path := r.URL.Path
		if prefix != "" {
			switch {
			case path == prefix:
				path = "/"
			case strings.HasPrefix(path, prefix+"/"):
				path = strings.TrimPrefix(path, prefix)
			default:
				http.NotFound(w, r)
				return
			}
		}
		clean := strings.TrimPrefix(path, "/")
		if clean == "" {
			serveIndex(w, sub)
			return
		}
		if info, err := fs.Stat(sub, clean); err == nil && !info.IsDir() {
			if strings.HasPrefix(clean, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			req := r.Clone(r.Context())
			req.URL.Path = "/" + clean
			fileServer.ServeHTTP(w, req)
			return
		}
		serveIndex(w, sub)
	})
}

func serveIndex(w http.ResponseWriter, sub fs.FS) {
	index, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		placeholder(w, nil)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(index)
}

func placeholder(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, placeholderHTML)
}

const placeholderHTML = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><title>OpenShop</title>
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>body{font-family:system-ui,sans-serif;max-width:40rem;margin:15vh auto;padding:0 1.5rem;line-height:1.6;color:#111}
code{background:#f2f2f2;padding:.15rem .4rem;border-radius:.25rem}</style></head>
<body><h1>OpenShop API is running</h1>
<p>The frontend has not been embedded in this build.</p>
<p>Build it with <code>make fe-build</code> (which builds the <code>front</code> and
<code>ops</code> apps and copies their <code>dist/</code> into the backend), then rebuild
the server. Release binaries from GitHub already include both frontends.</p>
<p>The API lives under <code>/api/v1</code>, the storefront at <code>/</code>, the admin
console at <code>/ops</code> and the docs at <code>/docs</code>.</p>
</body></html>
`
