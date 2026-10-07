// Command ops runs the operations (admin) console and its API. It is a separate
// binary from the storefront server so it can be deployed on an internal
// network; it serves the admin SPA at / and the admin API under /api/v1/ops.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"github.com/holihur/openshop/internal/bootstrap"
	"github.com/holihur/openshop/internal/config"
	apphttpops "github.com/holihur/openshop/internal/http/ops"
	"github.com/holihur/openshop/internal/version"
)

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println(version.String())
		return
	}

	// .env is optional; real deployments inject environment variables.
	_ = godotenv.Load(".env", "../.env")

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := bootstrap.New(ctx, cfg, bootstrap.Options{
		Addr:       cfg.Ops.Addr,
		Surface:    bootstrap.SurfaceOps,
		RunWorkers: false,
		BuildHTTP: func(d bootstrap.HTTPDeps) http.Handler {
			return apphttpops.New(d.Config, d.Tokens, d.Auth, d.Limiter, d.Cache, d.Settings, d.Metrics, d.Tracer, d.Handler, d.Analytics)
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "bootstrap error:", err)
		os.Exit(1)
	}

	if err := app.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, "runtime error:", err)
		os.Exit(1)
	}
}
