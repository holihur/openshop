#!/usr/bin/env sh
# Builds the front and ops SPAs and copies their output into the Go module so the
# //go:embed directives pick them up. Used by `make fe-build`, the Docker image
# and the GoReleaser pipeline. Safe to run when pnpm is missing: it then leaves
# the placeholder in place and the server serves a "not built" page.
set -eu

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

if ! command -v pnpm >/dev/null 2>&1; then
  echo "pnpm not found; skipping frontend embed (server will serve a placeholder)" >&2
  exit 0
fi

embed_app() {
  app="$1"
  out="$ROOT/backend/internal/http/$app/assets/dist"
  echo "building $app…"
  (cd "$ROOT" && pnpm --filter "@openshop/$app" build)
  rm -rf "$out"
  mkdir -p "$out"
  cp -R "$ROOT/$app/dist/." "$out"/
  # Keep the placeholder so the working tree stays clean.
  : > "$out/.gitkeep"
}

(cd "$ROOT" && pnpm install --frozen-lockfile)
embed_app front
embed_app ops

echo "embedded front and ops into $ROOT/backend/internal/http"
