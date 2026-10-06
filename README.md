# OpenShop

A production-grade, **horizontally scalable** e-commerce storefront.

- **Backend** — Go · Gin · GORM · PostgreSQL · NATS (JetStream) · Redis
- **Frontends** — two React 19 + TypeScript + Vite + TailwindCSS v4 + shadcn/ui
  SPAs: `front` (storefront) and `ops` (admin console). Both are compiled and
  **embedded into the Go binary**, so one process serves the API and both UIs.
  Package management is pnpm (workspace).

Every third-party integration is reached through a **port interface**, so the
business logic never depends on a concrete database, cache, broker, payment
gateway, object store or mail provider. Adapters are wired once, in the
composition root (`internal/bootstrap`).

## What's included

**Storefront** — catalog with categories, product variants/SKUs, image gallery,
full-text search, reviews with ratings, cart, coupons, shipping methods and tax,
saved addresses, checkout, order tracking and self-service cancellation.

**Operations** — a routed admin console (`/ops`) with a sidebar: merchant
dashboard (revenue, order counts, low stock), product/variant management with
image upload, order fulfilment (ship with tracking, complete, refund), coupon
and shipping-method/zone management, exchange rates, an audit trail (including
failed logins), and review moderation.

**Platform** — JWT auth with refresh rotation, theft detection, email
verification and password reset; per-IP and per-user sliding-window rate
limiting; payments
behind a provider port (sandbox included); guest checkout via an access token;
multi-currency settlement; transactional outbox for reliable events; idempotent
checkout; distributed locks; NATS queue-group consumers; Redis cache/locks/carts;
Prometheus metrics; OpenTelemetry tracing across the async boundary; versioned
migrations; SEO (robots/sitemap/JSON-LD); Docker Compose, Kubernetes manifests,
CI, a smoke test and a k6 load test.

**Delivery** — a single self-contained binary that serves the API, the storefront
(`/`) and the admin console (`/ops`); GoReleaser publishes signed-checksum
binaries for Linux/macOS/Windows to GitHub Releases, and a one-line install
script downloads the right build for the host.

---

## Table of contents

- [Why it scales horizontally](#why-it-scales-horizontally)
- [Architecture](#architecture)
- [Ports & adapters](#ports--adapters)
- [Directory layout](#directory-layout)
- [Quick start](#quick-start)
- [Configuration](#configuration)
- [API reference](#api-reference)
- [Testing](#testing)
- [Design decisions](#design-decisions)

---

## Why it scales horizontally

Nothing that matters lives in a single process. You can run `N` API replicas
behind a load balancer and they will behave as one system.

| Concern | Mechanism |
| --- | --- |
| **Authentication** | Stateless JWTs verified locally by any replica; refresh tokens and a logout deny-list live in shared Redis. |
| **Carts** | Stored in Redis, not process memory, so any replica serves any user. |
| **Cache** | A single shared Redis cache; invalidations are visible to all replicas instantly. |
| **Checkout** | A per-user **distributed lock** (`Redis SET NX` with owner token) prevents duplicate submissions across replicas. |
| **Inventory** | Atomic compare-and-set (`UPDATE ... WHERE stock >= ?`) inside a DB transaction; overselling is impossible under concurrency. |
| **Events** | NATS JetStream with **durable queue groups** — adding replicas adds throughput, and each event is handled once per group. |
| **Reliable events** | A **transactional outbox**: events are committed in the same DB transaction as the state change, then a relay publishes them (SKIP LOCKED, at-least-once). A crash can never lose an event. |
| **Safe retries** | `Idempotency-Key` on checkout/payment: the first response is cached and replayed, so network retries never create duplicate orders. |
| **Coupon limits** | Global usage caps enforced by an atomic conditional `UPDATE … WHERE used_count < usage_limit`, so the cap holds across replicas. |
| **Scheduled jobs** | The order-expiry sweeper takes a Redis **leader lock**, so only one replica sweeps per tick. If it dies, the lock expires and another takes over. |
| **Readiness** | `/readyz` probes PostgreSQL, Redis and NATS so the load balancer can drain unhealthy replicas. |
| **Graceful shutdown** | In-flight requests drain, workers stop, and connections close cleanly on `SIGTERM`. |

```mermaid
flowchart LR
    LB[Load balancer] --> A1[API replica 1]
    LB --> A2[API replica 2]
    LB --> A3[API replica N]
    A1 & A2 & A3 --> PG[(PostgreSQL)]
    A1 & A2 & A3 --> R[(Redis\ncache · locks · carts)]
    A1 & A2 & A3 --> N[(NATS JetStream)]
    PG -. outbox relay .-> N
    N --> W1[Consumers / workers]
```

### Reliability: the transactional outbox

Publishing to a broker *after* committing a database write has a classic failure
window: the process can crash in between, silently losing the event. OpenShop
avoids this with the **transactional outbox**:

1. Checkout writes the order **and** an `outbox_events` row in one transaction.
2. A relay worker leases pending rows with `SELECT … FOR UPDATE SKIP LOCKED`,
   publishes them to NATS, then marks them published.
3. Failed publishes are retried with exponential backoff; a lease that expires
   (crashed relay) is reclaimed automatically.

Any number of relays can run concurrently — `SKIP LOCKED` guarantees each
message is leased by exactly one — so event delivery scales with the fleet.

---

## Architecture

OpenShop uses **hexagonal (ports & adapters) architecture**.

```
              ┌─────────────────────────────────────────────┐
 HTTP (Gin) → │  service (business logic, depends on ports) │ → port interfaces
              └─────────────────────────────────────────────┘
                                   ▲
                                   │ implemented by
              ┌────────────────────┴────────────────────────┐
              │  adapter: postgres · redis · nats · payment │
              │           storage · mail · sms · security   │
              └─────────────────────────────────────────────┘
```

- `internal/domain` — entities and business rules. Imports **nothing** external.
- `internal/port` — interfaces for every external capability.
- `internal/service` — use cases. Depends only on `domain` + `port`.
- `internal/adapter/*` — concrete implementations (the only place GORM, Redis,
  NATS, AWS SDK, bcrypt, JWT, etc. are imported).
- `internal/http` — Gin transport; translates HTTP ⇄ service.
- `internal/bootstrap` — composition root; the single place adapters are chosen.

This is why the entire service layer is tested with **zero infrastructure**
running (see [Testing](#testing)).

---

## Ports & adapters

| Port (`internal/port`) | Production adapter | Local / dev adapter |
| --- | --- | --- |
| `UserRepository`, `ProductRepository`, `OrderRepository`, `PaymentRepository`, `CategoryRepository` | GORM + PostgreSQL | in-memory fakes (tests) |
| `CartRepository` | Redis | in-memory fake (tests) |
| `CouponRepository`, `ReviewRepository` | GORM + PostgreSQL | in-memory fakes (tests) |
| `VariantRepository` | GORM + PostgreSQL (SKU inventory) | in-memory fake (tests) |
| `Cache` | Redis | in-memory fake (tests) |
| `Locker` | Redis (`SET NX` + Lua release) | in-memory fake (tests) |
| `EventBus` | NATS JetStream | in-memory fake (tests) |
| `Outbox` | PostgreSQL (`SKIP LOCKED` relay) | in-memory fake (tests) |
| `PaymentProvider` / `PaymentRegistry` | `mock` (sandbox) | plug in Stripe/Alipay/WeChat |
| `ObjectStorage` | S3 / MinIO (`s3`) | local disk (`local`) |
| `Mailer` | SMTP | log |
| `SMS` | Aliyun (stub) | log |
| `PasswordHasher` | bcrypt | fake (tests) |
| `TokenIssuer` | JWT (HS256) | fake (tests) |
| `IDGenerator` | UUID v4 | sequential (tests) |
| `Clock` | system clock | fixed (tests) |
| `TxManager` | GORM transaction | pass-through (tests) |
| `Logger`, `Metrics` | `log/slog`, Prometheus (`/metrics`) | no-op |

Swap an adapter by changing one line in `internal/bootstrap/app.go`.

---

## Directory layout

```
openshop/
├── backend/
│   ├── cmd/
│   │   ├── server/         # API + workers
│   │   ├── migrate/        # versioned SQL migrations (advisory-locked)
│   │   └── seed/           # demo admin, categories, products
│   ├── internal/
│   │   ├── domain/         # entities, business rules, errors
│   │   ├── port/           # interfaces (the dependency boundary)
│   │   ├── service/        # use cases + tests with fakes
│   │   ├── adapter/        # postgres, redis, nats, payment, storage, mail, sms, security, logger, metrics
│   │   ├── http/           # router, middleware, handlers, response, docs (OpenAPI), web (embedded SPAs)
│   │   ├── worker/         # event consumers, order sweeper, outbox relay
│   │   ├── config/         # 12-factor env configuration
│   │   ├── version/        # build metadata injected by GoReleaser
│   │   └── bootstrap/      # composition root
│   └── migrations/         # *.sql up + down/ (reversible)
├── lib/                    # shared frontend code: api client, auth, types, ui, hooks
├── front/                  # storefront SPA (Vite, built to front/dist)
├── ops/                    # admin console SPA (Vite base /ops, built to ops/dist)
├── deploy/k8s/             # namespace/config, infra, backend (HPA+PDB), ingress
├── scripts/                # dev-setup.sh, embed-frontend.sh, install.sh, smoke.sh, loadtest.js
├── .github/workflows/      # CI (backend, frontends, docker) + release (GoReleaser)
├── .goreleaser.yaml        # release binaries for GitHub Releases
├── docker-compose.yml
└── Makefile
```

---

## Quick start

### Option 0 — Install the release binary (one line)

No Go or Node toolchain required; the binaries already embed both SPAs.

```bash
curl -fsSL https://raw.githubusercontent.com/holihur/openshop/main/scripts/install.sh | sh
```

The installer detects the OS/arch, verifies the SHA-256 checksum, installs
`openshop`, `openshop-migrate` and `openshop-seed` (plus the SQL migrations) and
prints the next steps. Then point it at PostgreSQL/Redis/NATS, migrate and run:

```bash
export POSTGRES_DSN='host=localhost port=5432 user=openshop password=openshop dbname=openshop sslmode=disable TimeZone=UTC'
export REDIS_ADDR='localhost:6379'
export NATS_URL='nats://localhost:4222'
export JWT_SECRET="$(head -c 32 /dev/urandom | base64)"
openshop-migrate -dir ~/.local/share/openshop/migrations
openshop
# storefront http://localhost:8080/ · admin http://localhost:8080/ops · docs /docs
```

### Option A — Docker Compose (everything)

One image contains the API and both embedded SPAs.

```bash
docker compose up --build -d          # builds the image, migrates, starts backend
docker compose run --rm seed          # optional demo data (profile: seed)
```

- Storefront: <http://localhost:8080/>
- Admin console (ops): <http://localhost:8080/ops>
- API: <http://localhost:8080/api/v1> · docs <http://localhost:8080/docs>
- NATS monitoring: <http://localhost:8222>

A root `.env` (generated by `install.sh`, or copy `backend/.env.example`) can set
`POSTGRES_PASSWORD`, `JWT_SECRET`, `HTTP_PORT`, `APP_CURRENCY`, …

Run several API replicas to see horizontal scaling:

```bash
docker compose up --scale backend=3
```

### Option B — Local toolchain

```bash
# 1. Prepare Postgres, Redis and NATS (idempotent)
./scripts/dev-setup.sh

# 2. Backend
cd backend
cp .env.example .env
go run ./cmd/migrate -dir migrations
# Roll back the last N migrations (paired *.down.sql files live in migrations/down/):
# go run ./cmd/migrate -dir migrations -down 1
go run ./cmd/seed
go run ./cmd/server

# 3. Frontends (separate terminals)
pnpm install
pnpm --filter @openshop/front dev   # storefront on :5173
pnpm --filter @openshop/ops dev     # admin console on :5174
```

To serve the built SPAs from the Go binary locally, run `make fe-build` before
`go run ./cmd/server`; it builds `front`/`ops` and copies them into the module.

Demo credentials created by the seed: `admin@openshop.local` / `admin12345`.

### Make targets

```bash
make help        # list everything
make infra       # start postgres/redis/nats via docker
make migrate     # apply migrations
make migrate-down N=1   # revert the last migration
make seed        # insert demo data
make run         # run the API
make test        # go test ./... -race
make fe-build    # build front + ops and embed them into the backend
make fe-dev      # storefront dev server
make ops-dev     # admin console dev server
make release     # local GoReleaser snapshot build
```

---

## Configuration

All configuration is environment-based (see `backend/.env.example`). Highlights:

| Variable | Default | Purpose |
| --- | --- | --- |
| `POSTGRES_DSN` | local DSN | PostgreSQL connection string |
| `REDIS_ADDR` | `localhost:6379` | shared cache / locks / carts |
| `NATS_URL` | `nats://localhost:4222` | event bus (JetStream required) |
| `JWT_SECRET` | dev value | **must** be set in production |
| `ORDER_TTL` | `30m` | unpaid-order expiry window |
| `PAYMENT_PROVIDER` | `mock` | active payment gateway |
| `STORAGE_DRIVER` | `local` | `local` or `s3` |
| `WORKER_ENABLED` | `true` | run event consumers + sweeper |
| `HTTP_CORS_ORIGINS` | `http://localhost:5173,http://localhost:5174` | allowed SPA dev origins |

---

## API reference

Base path: `/api/v1`. Responses are enveloped as `{ "data": … }` or
`{ "error": { "code", "message" } }`; list endpoints also return `meta`.

- Interactive docs: `GET /docs`
- OpenAPI document: `GET /api/v1/openapi.yaml`
- Prometheus metrics: `GET /metrics`

Send `Idempotency-Key: <uuid>` on `POST /orders` and `POST /payments` to make
retries safe.

### Auth

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `POST` | `/auth/register` | — | Create an account |
| `POST` | `/auth/login` | — | Sign in (email or phone) |
| `POST` | `/auth/refresh` | — | Rotate the refresh token |
| `POST` | `/auth/email/verify` | — | Verify an email with a single-use token |
| `POST` | `/auth/email/resend` | ✔ | Re-send the verification email |
| `POST` | `/auth/password/forgot` | — | Email a reset link (no enumeration) |
| `POST` | `/auth/password/reset` | — | Reset with a single-use token (revokes sessions) |
| `POST` | `/auth/password/change` | ✔ | Change password (revokes sessions) |
| `POST` | `/auth/logout` | ✔ | Revoke the current tokens |
| `GET` | `/auth/me` | ✔ | Current profile |
| `GET` | `/auth/me/export` | ✔ | Export all personal data (GDPR) |
| `DELETE` | `/auth/me` | ✔ | Erase account and personal data |

### Catalog

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `GET` | `/categories` | — | List categories |
| `GET` | `/products` | — | List/search (`categoryId`, `keyword`, `sort`, `page`, `pageSize`) |
| `GET` | `/products/:id` | — | Product detail (cached) |

### Cart

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `GET` | `/cart` | ✔ | Get the cart |
| `POST` | `/cart/items` | ✔ | Add an item |
| `PATCH` | `/cart/items/:productId` | ✔ | Set quantity |
| `DELETE` | `/cart/items/:productId` | ✔ | Remove an item |
| `DELETE` | `/cart` | ✔ | Clear the cart |

### Reviews & coupons

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `GET` | `/products/:id/reviews` | — | List a product's reviews |
| `POST` | `/products/:id/reviews` | ✔ | Create a review (one per user; marked verified after a paid order) |
| `PATCH` | `/reviews/:id` | ✔ | Update your review |
| `DELETE` | `/reviews/:id` | ✔ | Delete a review (owner or admin) |
| `POST` | `/coupons/preview` | ✔ | Validate a coupon and preview the discount |
| `GET` | `/shipping-methods` | — | List active shipping methods |
| `GET` | `/currencies` | — | Base currency and exchange rates |
| `GET` | `/wishlist` | ✔ | List saved products |
| `POST` | `/wishlist` | ✔ | Save a product |
| `DELETE` | `/wishlist/:productId` | ✔ | Remove a saved product |

### Orders & payments

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `POST` | `/orders` | ✔ | Checkout (reserves stock atomically) |
| `GET` | `/orders` | ✔ | List orders (admins see all) |
| `GET` | `/orders/:id` | ✔ | Order detail |
| `GET` | `/orders/:id/invoice` | ✔ | Download the order invoice as PDF |
| `POST` | `/orders/:id/cancel` | ✔ | Cancel & release stock |
| `POST` | `/orders/:id/complete` | ✔ | Confirm receipt of a shipped order |
| `GET` | `/addresses` | ✔ | List shipping addresses |
| `POST` | `/addresses` | ✔ | Create an address |
| `PATCH` | `/addresses/:id` | ✔ | Update an address |
| `DELETE` | `/addresses/:id` | ✔ | Delete an address |
| `POST` | `/addresses/:id/default` | ✔ | Set the default address |
| `POST` | `/payments` | ✔ | Create a payment session |
| `POST` | `/payments/simulate` | ✔ | Sandbox: confirm a mock payment |
| `POST` | `/webhooks/payments/:provider` | — | Provider callback (signature-verified) |
| `GET` | `/guest/orders/:token` | — | View a guest order by access token |
| `POST` | `/guest/orders/:token/pay` | — | Pay a guest order |
| `POST` | `/guest/orders/:token/cancel` | — | Cancel a guest order |
| `POST` | `/guest/orders/:token/complete` | — | Confirm receipt of a guest order |

### Admin

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/admin/categories` | Create a category |
| `POST` | `/admin/products` | Create a product |
| `PATCH` | `/admin/products/:id` | Update a product |
| `POST` | `/admin/uploads` | Upload an image (multipart) |
| `GET` | `/admin/orders` | List all orders |
| `GET` | `/admin/stats` | Dashboard: revenue, order counts, recent orders, low stock |
| `PATCH` | `/admin/coupons/:id` | Update or deactivate a coupon |
| `GET` | `/admin/reviews` | List reviews for moderation |
| `GET` | `/admin/inventory/low-stock` | Products/variants at or below the threshold |
| `GET` | `/admin/audit-logs` | Audit trail (security and admin actions) |
| `PUT` | `/admin/currencies/:code` | Set an exchange rate |
| `POST` | `/admin/orders/:id/refund` | Refund a paid order (restores stock) |
| `POST` | `/admin/orders/:id/ship` | Mark a paid order shipped (tracking number) |
| `POST` | `/admin/orders/:id/complete` | Mark a shipped order completed |
| `GET` | `/admin/orders/:id/invoice` | Download any order's invoice as PDF |
| `GET` | `/admin/shipping-methods` | List shipping methods |
| `POST` | `/admin/shipping-methods` | Create a shipping method |
| `PATCH` | `/admin/shipping-methods/:id` | Update a shipping method |
| `GET` | `/admin/shipping-zones` | List shipping zones |
| `POST` | `/admin/shipping-zones` | Create a shipping zone |
| `PATCH` | `/admin/shipping-zones/:id` | Update a shipping zone |
| `PUT` | `/admin/shipping-zones/:zoneId/rates/:methodId` | Set a zone rate (flat + per-kg) |
| `GET` | `/admin/coupons` | List coupons |
| `POST` | `/admin/coupons` | Create a coupon |
| `GET` | `/admin/products/:id/variants` | List a product's variants |
| `POST` | `/admin/products/:id/variants` | Create a variant (SKU) |
| `PATCH` | `/admin/variants/:id` | Update a variant |

### Ops

`GET /healthz` (liveness) · `GET /readyz` (dependency readiness) ·
`GET /api/v1/version`

### Example: checkout

```bash
TOKEN=$(curl -s localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"identifier":"admin@openshop.local","password":"admin12345"}' \
  | jq -r .data.accessToken)

curl -s localhost:8080/api/v1/cart/items \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"productId":"<id>","quantity":2}'

curl -s -X POST localhost:8080/api/v1/orders -H "Authorization: Bearer $TOKEN"
```

---

## Testing

The service layer depends only on ports, so it is tested with **in-memory fakes
and no infrastructure**:

```bash
cd backend && go test ./... -race
```

Covered scenarios include:

- checkout reserves stock, clears the cart and enqueues `order.created`
- insufficient stock is rejected without side effects
- cancellation restores stock and cannot double-refund it
- `MarkPaid` is idempotent (duplicate webhooks are no-ops)
- expired orders are listed for the sweeper
- registration/login/refresh and **refresh-token theft detection**
- the outbox relay publishes, retries and caps backoff

### Integration tests (real PostgreSQL)

Adapter behaviour that depends on the database (e.g. `SKIP LOCKED` leasing) is
covered by integration tests that run when a DSN is provided:

```bash
cd backend
TEST_DATABASE_URL='host=localhost user=openshop password=openshop dbname=openshop sslmode=disable' \
  go test ./internal/adapter/postgres/ -run TestOutboxClaimSemantics
```

### End-to-end smoke test

Against a running API:

```bash
API_BASE=http://localhost:8080/api/v1 ./scripts/smoke.sh
```

It verifies login, catalog, stock reservation, **idempotent replay**, payment
and the resulting paid order.

### Load test

```bash
k6 run scripts/loadtest.js            # browse + shop scenarios, thresholds enforced
```

---

## Observability

Every process exposes Prometheus metrics at `GET /metrics`:

| Metric | Type | Meaning |
| --- | --- | --- |
| `openshop_http_requests_total` | counter | requests by method, route, status |
| `openshop_http_request_duration_seconds` | histogram | request latency |
| `openshop_orders_created_total` | counter | orders created by currency |
| `openshop_orders_paid_total` | counter | orders paid |
| `openshop_payments_succeeded_total` | counter | successful payments by provider |
| `openshop_payments_refunded_total` | counter | refunded payments by provider |
| `openshop_orders_refunded_total` | counter | refunded orders |
| `openshop_outbox_published_total` | counter | events relayed by subject |
| `openshop_outbox_publish_failures_total` | counter | relay failures (retried) |

Logging is structured (`log/slog`, JSON in production) with a per-request
correlation id propagated from `X-Request-Id`. The `Metrics` and `Logger` ports
keep both swappable and no-op in tests.

### Distributed tracing

Tracing is behind the `port.Tracer` port with an OpenTelemetry adapter. When
`OTEL_EXPORTER_OTLP_ENDPOINT` is set, spans are exported over OTLP/HTTP;
otherwise a no-op tracer is used, so the binary carries no runtime vendor
dependency. Trace context is propagated with W3C `traceparent`:

- the HTTP middleware continues an inbound trace (or starts one),
- checkout injects the `traceparent` into the outbox event,
- the relay extracts it when publishing, and consumers extract it again when
  handling — so **checkout → outbox → relay → consumer is a single trace**,
  including the asynchronous hop.

```bash
# point at any OTLP collector (Jaeger, Tempo, Grafana Agent, …)
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4318 OTEL_INSECURE=true go run ./cmd/server
```

## Deployment

### Kubernetes

`deploy/k8s/` contains a runnable manifest set:

- `00-namespace-config.yaml` — namespace, ConfigMap, Secret
- `10-infra.yaml` — PostgreSQL / Redis / NATS StatefulSets (swap for managed services in production)
- `20-backend.yaml` — Deployment (3 replicas, rolling update, anti-affinity,
  topology spread), readiness/liveness probes, **HPA** (3→20 on CPU/memory), **PDB**,
  and an **init container** that runs migrations (safe on every pod via the advisory lock)
- `30-ingress.yaml` — Ingress routing everything to the backend, which serves the
  API and both embedded SPAs

```bash
kubectl apply -f deploy/k8s/
kubectl -n openshop rollout status deploy/openshop-backend
kubectl -n openshop scale deploy/openshop-backend --replicas=6
```

### Release binaries (GoReleaser)

Tag a release and GitHub Actions builds and publishes the binaries:

```bash
git tag v1.0.0 && git push origin v1.0.0
```

`.goreleaser.yaml` cross-compiles `openshop`, `openshop-migrate` and
`openshop-seed` for Linux/macOS/Windows (amd64 + arm64), embeds both SPAs into
the server binary, injects the version/commit/date, and ships the SQL migrations
in each archive. `scripts/install.sh` installs the right build for the host and
verifies its checksum.

### CI

`.github/workflows/ci.yml` runs on every push/PR:

- **backend**: `gofmt` check, `go vet`, `go build`, `go test -race` with real
  PostgreSQL + Redis services (integration tests included), coverage summary
- **frontends**: `pnpm install --frozen-lockfile`, type-check and production build
  of both `front` and `ops`
- **docker**: builds the single backend image (API + embedded SPAs)
- **release-config**: `goreleaser check`

`.github/workflows/release.yml` publishes a GitHub Release on every `v*` tag.

## Known limitations

Honest gaps a buyer should know about:

- **No server-side rendering.** Per-page SEO meta is set client-side; crawlers
  that do not execute JavaScript rely on `/sitemap.xml` and the base tags. Full
  SEO would need SSR/SSG.
- **Single base currency for pricing.** Products are priced in the store base
  currency; other currencies are converted at checkout. Per-currency price lists
  are not supported.
- **Refunds always return stock.** Refunding a shipped order restores inventory,
  which assumes the goods came back. A returns workflow is not modelled.
- **No automated end-to-end or frontend tests.** Coverage is service unit tests,
  one adapter integration test and a shell smoke test; there is no browser E2E
  suite.
- **Email/SMS default to log drivers.** Real delivery needs SMTP/SMS
  credentials; the sandbox payment provider must be replaced for real charges.
- **Invoice PDFs use built-in fonts.** The dependency-free renderer uses
  Helvetica/WinAnsi, so non-Latin glyphs (for example CJK) are sanitised;
  shipping an embedded Unicode font would fix this.

## Design decisions

- **Money as integers.** Prices are stored in minor units (`price_cents`) to
  avoid floating-point drift.
- **UUID primary keys.** Any replica can generate ids without a central
  sequence, which removes a scaling bottleneck and simplifies merges.
- **Versioned migrations with an advisory lock.** `make migrate` is safe to run
  from every replica at deploy time; each migration has a paired
  `migrations/down/*.down.sql` so `make migrate-down N=1` reverts the last N.
- **Sandbox payments.** The `mock` provider implements an extra
  `SandboxProvider` port so the full checkout can be exercised locally; real
  providers simply don't implement it, and `POST /payments/simulate` rejects
  them.
- **Error envelopes.** Domain sentinel errors are mapped to HTTP status codes in
  one place (`internal/http/response`), keeping handlers thin.
- **At-least-once, never lost.** Events go through the transactional outbox, so
  they are committed with the business write and relayed with retries; consumers
  are idempotent.
- **Idempotent writes.** Checkout and payment accept an `Idempotency-Key`; the
  first outcome is cached and replayed, and concurrent duplicates get `409`.
- **Read-your-writes for stock.** Checkout evicts the product cache
  synchronously (and the event consumer does so again as a safety net), so the
  catalog reflects reservations immediately.
- **Least privilege in the catalog.** Public product listings are forced to
  `published`; only admins can list drafts via `/admin/products`.
- **Discounts are integers too.** Percent coupons use integer math and caps so
  rounding never produces fractional money; the discount can never exceed the
  subtotal.
- **Indexed search with a fallback.** Products carry a generated `tsvector`
  column with a GIN index; queries also fall back to a substring match so CJK
  and partial words still work.
- **One checkout path for simple and variant products.** When a product has
  variants, inventory and price live on the variant; otherwise on the product.
  Checkout resolves the purchasable unit, so both share the same atomic
  reservation, cancellation and refund logic. Cart lines are keyed by
  `(product, variant)`.
- **Addresses are snapshotted.** Checkout copies the chosen address onto the
  order, so editing the address book never rewrites order history.
- **Explicit fulfilment state machine.** `pending_payment → paid → shipped →
  completed`, with `cancelled`/`refunded` as terminal branches. Every transition
  is idempotent, lock-protected and emits an event.
- **Money is computed server-side.** Shipping (flat rate with a free threshold,
  plus optional per-zone rates and per-kilogram weight surcharges) and tax
  (configurable basis-point rate) are calculated at checkout on the discounted
  subtotal; the client only previews them.
- **Guest checkout shares one cart path.** The cart owner is a *subject* — a
  user id or an `X-Guest-Id` — and guest orders are authorised by an unguessable
  access token, so guests reuse the same inventory, coupon and payment logic.
- **Multi-currency settles server-side.** Catalog prices are previewed in the
  chosen currency from exchange rates; checkout converts unit prices, fixed
  coupons and shipping, then stores the order in that currency.
- **Everything auditable.** Security and admin actions are appended to an audit
  trail with actor, resource, metadata and IP.
