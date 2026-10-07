#!/usr/bin/env sh
# Boots a server for the Playwright suite.
#   e2e-server.sh server  -> storefront binary on :18081 (migrates + seeds)
#   e2e-server.sh ops     -> admin binary on :18082
# PostgreSQL, Redis and NATS must be reachable (the CI e2e job provides them).
set -eu

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MODE="${1:-server}"

if [ ! -f "$ROOT/backend/internal/http/front/assets/dist/index.html" ]; then
  "$ROOT/scripts/embed-frontend.sh"
fi

cd "$ROOT/backend"
if [ "$MODE" = "server" ]; then
  export HTTP_ADDR="${HTTP_ADDR:-:18081}"
  export INSTANCE_ID="${INSTANCE_ID:-e2e}"
  export STORAGE_PUBLIC_URL="${STORAGE_PUBLIC_URL:-http://localhost:18081/uploads}"
  go run ./cmd/migrate -dir migrations
  go run ./cmd/seed
  exec go run ./cmd/server
else
  export OPS_ADDR="${OPS_ADDR:-:18082}"
  export INSTANCE_ID="${INSTANCE_ID:-e2e-ops}"
  export WORKER_ENABLED=false
  exec go run ./cmd/ops
fi
