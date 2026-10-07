#!/usr/bin/env bash
#
# End-to-end smoke test against a running OpenShop API.
#
#   API_BASE=http://localhost:8080/api/v1 ./scripts/smoke.sh
#
# It exercises: login, catalog, cart, checkout, payment creation, the sandbox
# payment confirmation and the resulting paid order. Requires curl and jq.

set -euo pipefail

API_BASE="${API_BASE:-http://localhost:8080/api/v1}"
# The storefront realm only signs in customers; admins use the ops binary.
IDENTIFIER="${IDENTIFIER:-customer@openshop.local}"
PASSWORD="${PASSWORD:-customer12345}"

command -v jq >/dev/null || { echo "jq is required"; exit 1; }

pass() { printf '\033[32m✓\033[0m %s\n' "$*"; }
fail() { printf '\033[31m✗\033[0m %s\n' "$*"; exit 1; }

echo "→ API: $API_BASE"

# 1. Liveness / readiness
curl -fsS "${API_BASE%/api/v1}/healthz" >/dev/null || fail "healthz"
pass "healthz"

READY=$(curl -fsS "${API_BASE%/api/v1}/readyz")
echo "$READY" | jq -e '.status == "ok"' >/dev/null || fail "readyz: $READY"
pass "readyz (postgres, redis, nats)"

# 2. Login
TOKEN=$(curl -fsS "$API_BASE/auth/login" -H 'Content-Type: application/json' \
  -d "{\"identifier\":\"$IDENTIFIER\",\"password\":\"$PASSWORD\"}" | jq -r .data.accessToken)
[ -n "$TOKEN" ] && [ "$TOKEN" != "null" ] || fail "login"
pass "login"

AUTH=(-H "Authorization: Bearer $TOKEN")

# 3. Catalog — pick a simple product (no variants) that is in stock.
PRODUCT_ID=$(curl -fsS "$API_BASE/products?pageSize=50" | jq -r '[.data[] | select(.stock > 1)][0].id')
[ -n "$PRODUCT_ID" ] && [ "$PRODUCT_ID" != "null" ] || fail "no product with stock > 1 (run: make seed)"
STOCK_BEFORE=$(curl -fsS "$API_BASE/products/$PRODUCT_ID" | jq -r .data.stock)
pass "catalog (product $PRODUCT_ID, stock $STOCK_BEFORE)"

# 4. Cart
curl -fsS -X DELETE "$API_BASE/cart" "${AUTH[@]}" >/dev/null
curl -fsS -X POST "$API_BASE/cart/items" "${AUTH[@]}" -H 'Content-Type: application/json' \
  -d "{\"productId\":\"$PRODUCT_ID\",\"quantity\":2}" >/dev/null
pass "add to cart"

# 5. Checkout (idempotent)
IDEM="smoke-$(date +%s%N)"
ORDER=$(curl -fsS -X POST "$API_BASE/orders" "${AUTH[@]}" -H "Idempotency-Key: $IDEM")
ORDER_ID=$(echo "$ORDER" | jq -r .data.id)
ORDER_NO=$(echo "$ORDER" | jq -r .data.orderNo)
[ "$(echo "$ORDER" | jq -r .data.status)" = "pending_payment" ] || fail "checkout status"
STOCK_MID=$(curl -fsS "$API_BASE/products/$PRODUCT_ID" | jq -r .data.stock)
[ "$STOCK_MID" -eq "$((STOCK_BEFORE - 2))" ] || fail "stock not reserved ($STOCK_BEFORE → $STOCK_MID)"
pass "checkout ($ORDER_NO, stock $STOCK_BEFORE → $STOCK_MID)"

# 5b. Replaying the same idempotency key must return the same order.
REPLAY=$(curl -fsS -X POST "$API_BASE/orders" "${AUTH[@]}" -H "Idempotency-Key: $IDEM" | jq -r .data.orderNo)
[ "$REPLAY" = "$ORDER_NO" ] || fail "idempotency replay created a different order"
pass "idempotent replay returned the same order"

# 6. Payment + sandbox confirmation
PAY=$(curl -fsS -X POST "$API_BASE/payments" "${AUTH[@]}" -H 'Content-Type: application/json' \
  -d "{\"orderId\":\"$ORDER_ID\",\"provider\":\"mock\",\"returnUrl\":\"http://localhost:5173/payment/result\"}")
PROVIDER_REF=$(echo "$PAY" | jq -r .data.redirectUrl | sed -E 's/.*payment_ref=([^&]+).*/\1/')
[ -n "$PROVIDER_REF" ] || fail "payment redirect"
pass "payment created ($PROVIDER_REF)"

curl -fsS -X POST "$API_BASE/payments/simulate" "${AUTH[@]}" -H 'Content-Type: application/json' \
  -d "{\"providerRef\":\"$PROVIDER_REF\",\"provider\":\"mock\"}" >/dev/null
pass "sandbox payment confirmed"

# 7. Order is paid (allow a moment for the async webhook + outbox relay)
for _ in $(seq 1 10); do
  STATUS=$(curl -fsS "$API_BASE/orders/$ORDER_ID" "${AUTH[@]}" | jq -r .data.status)
  [ "$STATUS" = "paid" ] && break
  sleep 0.5
done
[ "$STATUS" = "paid" ] || fail "order not paid (status=$STATUS)"
pass "order $ORDER_NO is paid"

echo
printf '\033[32mAll smoke checks passed.\033[0m\n'
