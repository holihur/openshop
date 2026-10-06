#!/usr/bin/env bash
#
# Idempotent local development setup for OpenShop.
#
# It prepares everything needed to run the backend without Docker:
#   * PostgreSQL role + database
#   * Redis (already running as a system service on most machines)
#   * NATS server binary (downloaded on demand, started with JetStream)
#
# The script is safe to re-run and never overwrites existing data.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_DIR="$ROOT/bin"
DEV_DIR="$ROOT/.dev"
LOG_DIR="$DEV_DIR/logs"
mkdir -p "$BIN_DIR" "$DEV_DIR" "$LOG_DIR"

NATS_VERSION="${NATS_VERSION:-2.10.22}"
PG_USER="${PG_USER:-openshop}"
PG_PASSWORD="${PG_PASSWORD:-openshop}"
PG_DB="${PG_DB:-openshop}"

log() { printf '\033[36m[dev-setup]\033[0m %s\n' "$*"; }
warn() { printf '\033[33m[dev-setup]\033[0m %s\n' "$*"; }

# ---------------------------------------------------------------------------
# PostgreSQL
# ---------------------------------------------------------------------------
ensure_postgres() {
  if ! command -v psql >/dev/null 2>&1; then
    warn "psql not found; install PostgreSQL or use docker compose instead"
    return 1
  fi

  if ! pg_isready -q 2>/dev/null; then
    warn "PostgreSQL is not accepting connections on the default port"
    return 1
  fi

  # Use the postgres superuser via peer auth. `cd /` avoids the
  # "could not change directory" warning.
  local psql_super="sudo -n -u postgres psql -d postgres -tAc"

  if (cd / && $psql_super "SELECT 1 FROM pg_roles WHERE rolname='${PG_USER}'") | grep -q 1; then
    log "postgres role '${PG_USER}' already exists"
  else
    (cd / && sudo -n -u postgres psql -d postgres -c \
      "CREATE ROLE ${PG_USER} LOGIN PASSWORD '${PG_PASSWORD}';") >/dev/null
    log "created postgres role '${PG_USER}'"
  fi

  if (cd / && $psql_super "SELECT 1 FROM pg_database WHERE datname='${PG_DB}'") | grep -q 1; then
    log "postgres database '${PG_DB}' already exists"
  else
    (cd / && sudo -n -u postgres createdb -O "${PG_USER}" "${PG_DB}") >/dev/null
    log "created postgres database '${PG_DB}'"
  fi
}

# ---------------------------------------------------------------------------
# Redis
# ---------------------------------------------------------------------------
ensure_redis() {
  if ! command -v redis-cli >/dev/null 2>&1; then
    warn "redis-cli not found; install Redis or use docker compose instead"
    return 1
  fi
  if redis-cli ping >/dev/null 2>&1; then
    log "redis is up"
  else
    warn "redis is not running; try: sudo systemctl start redis-server"
    return 1
  fi
}

# ---------------------------------------------------------------------------
# NATS
# ---------------------------------------------------------------------------
download_nats() {
  local dest="$BIN_DIR/nats-server"
  if [[ -x "$dest" ]]; then
    log "nats-server already present at $dest"
    return 0
  fi

  local arch
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64) arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    *) warn "unsupported architecture: $arch"; return 1 ;;
  esac

  local url="https://github.com/nats-io/nats-server/releases/download/v${NATS_VERSION}/nats-server-v${NATS_VERSION}-linux-${arch}.tar.gz"
  local tmp="$DEV_DIR/nats-download.tar.gz"

  log "downloading nats-server v${NATS_VERSION} (${arch})…"
  if ! curl -fsSL --retry 3 --retry-delay 2 -o "$tmp" "$url"; then
    warn "download failed; falling back to Docker"
    return 1
  fi
  tar -xzf "$tmp" -C "$DEV_DIR"
  mv "$DEV_DIR/nats-server-v${NATS_VERSION}-linux-${arch}/nats-server" "$dest"
  rm -rf "$tmp" "$DEV_DIR/nats-server-v${NATS_VERSION}-linux-${arch}"
  log "installed nats-server to $dest"
}

start_nats() {
  if curl -fsS http://127.0.0.1:8222/varz >/dev/null 2>&1; then
    log "nats is already running"
    return 0
  fi

  local bin=""
  if [[ -x "$BIN_DIR/nats-server" ]]; then
    bin="$BIN_DIR/nats-server"
  elif command -v nats-server >/dev/null 2>&1; then
    bin="$(command -v nats-server)"
  fi

  if [[ -z "$bin" ]]; then
    if download_nats; then
      bin="$BIN_DIR/nats-server"
    elif command -v docker >/dev/null 2>&1; then
      log "starting nats via docker"
      docker rm -f openshop-nats >/dev/null 2>&1 || true
      docker run -d --name openshop-nats -p 4222:4222 -p 8222:8222 nats:2-alpine -js -m 8222 >/dev/null
      log "nats container started"
      return 0
    else
      warn "could not obtain nats-server"
      return 1
    fi
  fi

  log "starting nats-server with JetStream"
  nohup "$bin" -js -sd "$DEV_DIR/nats-data" -m 8222 \
    >"$LOG_DIR/nats.log" 2>&1 &
  echo $! > "$DEV_DIR/nats.pid"

  for _ in $(seq 1 30); do
    if curl -fsS http://127.0.0.1:8222/varz >/dev/null 2>&1; then
      log "nats is up (pid $(cat "$DEV_DIR/nats.pid"))"
      return 0
    fi
    sleep 0.5
  done
  warn "nats did not become ready; see $LOG_DIR/nats.log"
  return 1
}

main() {
  ensure_postgres || true
  ensure_redis || true
  start_nats || true
  log "done. Next: make migrate && make seed && make run"
}

main "$@"
