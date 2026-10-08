#!/usr/bin/env sh
# Boots a server for the Playwright suite.
#   e2e-server.sh server  -> storefront binary on :18081 (migrates + seeds)
#   e2e-server.sh ops     -> admin binary on :18082
# PostgreSQL, Redis and NATS must be reachable (the CI e2e job provides them).
#
# The binaries are built once into .dev/e2e-bin and reused, so a re-run does not
# pay for another compile and the server starts deterministically.
set -eu

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MODE="${1:-server}"
BIN_DIR="$ROOT/.dev/e2e-bin"

if [ ! -f "$ROOT/backend/internal/http/front/assets/dist/index.html" ]; then
  "$ROOT/scripts/embed-frontend.sh"
fi

mkdir -p "$BIN_DIR"
cd "$ROOT/backend"

build() { # build <package-dir> <binary>
  go build -o "$BIN_DIR/$2" "./cmd/$1"
}

if [ "$MODE" = "server" ]; then
  export HTTP_ADDR="${HTTP_ADDR:-:18081}"
  export INSTANCE_ID="${INSTANCE_ID:-e2e}"
  export STORAGE_PUBLIC_URL="${STORAGE_PUBLIC_URL:-http://localhost:18081/uploads}"
  # Every test client shares one IP, so the per-IP limiters would throttle the
  # suite rather than the app. They stay enforced in production.
  export HTTP_RATE_LIMIT_RPS="${HTTP_RATE_LIMIT_RPS:-100000}"
  export HTTP_RATE_LIMIT_USER_RPS="${HTTP_RATE_LIMIT_USER_RPS:-100000}"
  export HTTP_AUTH_RATE_LIMIT_RPS="${HTTP_AUTH_RATE_LIMIT_RPS:-100000}"
  build migrate migrate
  build seed seed
  "$BIN_DIR/migrate" -dir migrations
  "$BIN_DIR/seed"
  build server server
  exec "$BIN_DIR/server"
else
  export OPS_ADDR="${OPS_ADDR:-:18082}"
  export INSTANCE_ID="${INSTANCE_ID:-e2e-ops}"
  export WORKER_ENABLED=false
  export HTTP_RATE_LIMIT_RPS="${HTTP_RATE_LIMIT_RPS:-100000}"
  export HTTP_AUTH_RATE_LIMIT_RPS="${HTTP_AUTH_RATE_LIMIT_RPS:-100000}"
  build ops ops
  exec "$BIN_DIR/ops"
fi
