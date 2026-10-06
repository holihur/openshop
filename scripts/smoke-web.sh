#!/usr/bin/env bash
#
# Verifies that the running binaries serve their embedded SPAs and assets.
#
#   WEB_BASE=http://localhost:8080 OPS_BASE=http://localhost:8081 ./scripts/smoke-web.sh
#
# WEB_BASE is the public storefront binary; OPS_BASE is the internal ops binary.
# Set OPS_BASE="" to skip the ops checks. Requires servers built with the
# frontends embedded (`make fe-build` first). Requires curl.

set -euo pipefail

WEB_BASE="${WEB_BASE:-http://localhost:8080}"
OPS_BASE="${OPS_BASE-http://localhost:8081}"
API_BASE="${API_BASE:-$WEB_BASE/api/v1}"

pass() { printf '\033[32m✓\033[0m %s\n' "$*"; }
fail() { printf '\033[31m✗\033[0m %s\n' "$*"; exit 1; }

# Fetch a path from a base URL, assert 200 and return the body on stdout.
fetch_ok() {
  local base="$1" path="$2" body code
  body="$(mktemp)"
  code="$(curl -s -o "$body" -w '%{http_code}' "$base$path")"
  [ "$code" = "200" ] || fail "$base$path -> HTTP $code"
  cat "$body"
  rm -f "$body"
}

# Assert the JS asset referenced by a shell is served.
check_asset() {
  local name="$1" base="$2" html="$3" asset code
  asset="$(printf '%s' "$html" | grep -oE '/assets/[^"]+\.js' | head -1)"
  [ -n "$asset" ] || fail "$name shell references no JS asset"
  code="$(curl -s -o /dev/null -w '%{http_code}' "$base$asset")"
  [ "$code" = "200" ] || fail "$name asset $asset -> HTTP $code"
  pass "$name asset $asset (200)"
}

echo "→ Storefront: $WEB_BASE"
HOME_HTML="$(fetch_ok "$WEB_BASE" "/")"
echo "$HOME_HTML" | grep -q 'id="root"' || fail "storefront shell missing #root"
pass "storefront / (200, #root present)"
fetch_ok "$WEB_BASE" "/products" >/dev/null
pass "storefront /products (SPA fallback)"
check_asset "storefront" "$WEB_BASE" "$HOME_HTML"

if [ -n "$OPS_BASE" ]; then
  echo "→ Ops console: $OPS_BASE"
  OPS_HTML="$(fetch_ok "$OPS_BASE" "/")"
  echo "$OPS_HTML" | grep -q 'id="root"' || fail "ops shell missing #root"
  pass "ops / (200, #root present)"
  fetch_ok "$OPS_BASE" "/login" >/dev/null
  pass "ops /login (SPA fallback)"
  check_asset "ops" "$OPS_BASE" "$OPS_HTML"
fi

curl -fsS "$API_BASE/version" | grep -q '"version"' || fail "api version"
pass "storefront api /version"

echo
printf '\033[32mAll web smoke checks passed.\033[0m\n'
