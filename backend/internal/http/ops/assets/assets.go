// Package assets embeds the compiled admin console SPA. It lives in the ops
// surface so the openshop (storefront) binary never contains these bytes.
package assets

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/holihur/openshop/internal/http/shared/web"
)

//go:embed all:dist
var embedded embed.FS

func dist() fs.FS {
	sub, err := fs.Sub(embedded, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}

// Handler serves the admin console SPA.
func Handler() http.Handler { return web.Handler(dist(), "admin console") }

// Built reports whether the admin console SPA is embedded.
func Built() bool { return web.Built(dist()) }
