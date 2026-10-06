// Command server starts the OpenShop HTTP API, event consumers and background
// workers. The same binary runs on every replica; scaling out is just running
// more copies behind a load balancer.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"github.com/holihur/openshop/internal/bootstrap"
	"github.com/holihur/openshop/internal/config"
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

	app, err := bootstrap.New(ctx, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bootstrap error:", err)
		os.Exit(1)
	}

	if err := app.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, "runtime error:", err)
		os.Exit(1)
	}
}
