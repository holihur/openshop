#!/usr/bin/env bash
#
# Verifies that the running server serves both embedded SPAs and their assets.
#
#   WEB_BASE=http://localhost:8080 ./scripts/smoke-web.sh
#
# Requires a server built with the frontends embedded (`make fe-build` first).
# Requires curl.

set -euo pipefail

WEB_BASE="${WEB_BASE:-http://localhost:8080}"
API_BASE="${API_BASE:-$WEB_BASE/api/v1}"

pass() { printf '\033[32m✓\033[0m %s\n' "$*"; }
fail() { printf '\033[31m✗\033[0m %s\n' "$*"; exit 1; }

echo "→ Web: $WEB_BASE"

# Fetch a path, assert 200 and return the body on stdout.
fetch_ok() {
  local path="$1" body
  local code
  body="$(mktemp)"
  code="$(curl -s -o "$body" -w '%{http_code}' "$WEB_BASE$path")"
  [ "$code" = "200" ] || fail "$path -> HTTP $code"
  cat "$body"
  rm -f "$body"
}

# 1. Storefront shell and a client-side route (SPA fallback).
HOME_HTML="$(fetch_ok "/")"
echo "$HOME_HTML" | grep -q 'id="root"' || fail "storefront shell missing #root"
pass "storefront / (200, #root present)"

fetch_ok "/products" >/dev/null
pass "storefront /products (SPA fallback)"

# 2. Admin console shell and a client-side route.
OPS_HTML="$(fetch_ok "/ops")"
echo "$OPS_HTML" | grep -q 'id="root"' || fail "ops shell missing #root"
pass "ops /ops (200, #root present)"

fetch_ok "/ops/products" >/dev/null
pass "ops /ops/products (SPA fallback)"

# 3. The JS asset referenced by each shell must be served.
for pair in "storefront:$HOME_HTML" "ops:$OPS_HTML"; do
  name="${pair%%:*}"
  html="${pair#*:}"
  asset="$(printf '%s' "$html" | grep -oE '(/(ops)?)?/assets/[^"]+\.js' | head -1)"
  [ -n "$asset" ] || fail "$name shell references no JS asset"
  code="$(curl -s -o /dev/null -w '%{http_code}' "$WEB_BASE$asset")"
  [ "$code" = "200" ] || fail "$name asset $asset -> HTTP $code"
  pass "$name asset $asset (200)"
done

# 4. API is reachable and reports a version.
curl -fsS "$API_BASE/version" | grep -q '"version"' || fail "api version"
pass "api /version"

echo
printf '\033[32mAll web smoke checks passed.\033[0m\n'
