#!/usr/bin/env sh
# Boots a server for the Playwright suite.
#   e2e-server.sh server  -> storefront binary on :18081 (migrates + seeds)
#   e2e-server.sh ops     -> admin binary on :18082
#   e2e-server.sh ssr     -> Next.js storefront on :18083, API at $E2E_API_BASE
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

# The suite gets its own database so it is unaffected by (and does not pollute)
# whatever is in the development database. Without this the console pages churn
# through a developer's accumulated orders and the first navigation can time out.
if [ -z "${POSTGRES_DSN:-}" ]; then
  E2E_DB="${E2E_DB:-openshop_e2e}"
  export POSTGRES_DSN="host=localhost port=5432 user=${POSTGRES_USER:-openshop} password=${POSTGRES_PASSWORD:-openshop} dbname=$E2E_DB sslmode=disable TimeZone=UTC"
  if command -v psql >/dev/null 2>&1; then
    if ! PGPASSWORD="${POSTGRES_PASSWORD:-openshop}" psql -h localhost -U "${POSTGRES_USER:-openshop}" -d postgres -tAc \
        "SELECT 1 FROM pg_database WHERE datname='$E2E_DB'" | grep -q 1; then
      PGPASSWORD="${POSTGRES_PASSWORD:-openshop}" psql -h localhost -U "${POSTGRES_USER:-openshop}" -d postgres \
        -c "CREATE DATABASE $E2E_DB" >/dev/null 2>&1 || true
    fi
  fi
fi

if [ "$MODE" = "ssr" ]; then
  # Server-rendered storefront. It needs the API (start it with the "server"
  # mode); the browser only ever talks to this origin.
  export OPENSHOP_API_BASE="${E2E_API_BASE:-http://localhost:18081}"
  export PORT="${SSR_PORT:-18083}"
  cd "$ROOT/front_ssr"
  pnpm build
  exec pnpm exec next start -p "$PORT"
fi

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
