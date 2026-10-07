// Package assets embeds the compiled storefront SPA. It lives in the front
// surface so the openshop-ops binary never contains these bytes.
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

// Handler serves the storefront SPA.
func Handler() http.Handler { return web.Handler(dist(), "storefront") }

// Built reports whether the storefront SPA is embedded.
func Built() bool { return web.Built(dist()) }
