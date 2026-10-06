// k6 load test for OpenShop.
//
//   k6 run scripts/loadtest.js
//   BASE=https://shop.example.com/api/v1 k6 run scripts/loadtest.js
//
// The scenario browses the catalog, then (for a share of users) logs in, adds
// an item and checks out. Every checkout sends a unique Idempotency-Key so
// retries never create duplicate orders.
import http from "k6/http";
import { check, group, sleep } from "k6";
import { Trend } from "k6/metrics";

const BASE = __ENV.BASE || "http://localhost:8080/api/v1";
const IDENTIFIER = __ENV.IDENTIFIER || "admin@openshop.local";
const PASSWORD = __ENV.PASSWORD || "admin12345";

const checkoutTrend = new Trend("checkout_duration", true);

export const options = {
  scenarios: {
    browsing: {
      executor: "ramping-vus",
      exec: "browse",
      startVUs: 5,
      stages: [
        { duration: "30s", target: 30 },
        { duration: "1m", target: 30 },
        { duration: "20s", target: 0 },
      ],
    },
    shopping: {
      executor: "constant-vus",
      exec: "shop",
      vus: 10,
      duration: "1m",
      startTime: "30s",
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.05"],
    http_req_duration: ["p(95)<800"],
    checkout_duration: ["p(95)<1500"],
  },
};

function login() {
  const res = http.post(
    `${BASE}/auth/login`,
    JSON.stringify({ identifier: IDENTIFIER, password: PASSWORD }),
    { headers: { "Content-Type": "application/json" } },
  );
  return res.json("data.accessToken");
}

export function browse() {
  group("browse", () => {
    const list = http.get(`${BASE}/products?pageSize=12`);
    check(list, { "products 200": (r) => r.status === 200 });
    const products = list.json("data") || [];
    if (products.length > 0) {
      const p = products[Math.floor(Math.random() * products.length)];
      const detail = http.get(`${BASE}/products/${p.id}`);
      check(detail, { "detail 200": (r) => r.status === 200 });
    }
  });
  sleep(Math.random() * 2);
}

export function shop() {
  const token = login();
  if (!token) return;
  const auth = { headers: { Authorization: `Bearer ${token}` } };

  const list = http.get(`${BASE}/products?pageSize=12`);
  const products = list.json("data") || [];
  if (products.length === 0) return;
  const product = products[Math.floor(Math.random() * products.length)];

  http.del(`${BASE}/cart`, null, auth);
  const add = http.post(
    `${BASE}/cart/items`,
    JSON.stringify({ productId: product.id, quantity: 1 }),
    { ...auth, headers: { ...auth.headers, "Content-Type": "application/json" } },
  );
  check(add, { "add to cart 200": (r) => r.status === 200 });

  const start = Date.now();
  const order = http.post(`${BASE}/orders`, null, {
    headers: {
      ...auth.headers,
      "Idempotency-Key": `k6-${__VU}-${__ITER}-${Date.now()}`,
    },
  });
  checkoutTrend.add(Date.now() - start);
  check(order, { "checkout 201": (r) => r.status === 201 });

  sleep(Math.random() * 2);
}
