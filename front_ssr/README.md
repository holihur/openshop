# openshop storefront (SSR)

A server-rendered storefront built with Next.js, deployed independently of the
Vite storefront in `../front`. Both consume the same Go API, so they can evolve
in parallel: this one optimises for first paint, SEO and resilience, the other
for a rich client-side experience.

## Why a second storefront

The Vite app is a single-page application: the browser downloads an empty shell,
then fetches data. That is fine for a signed-in account area, but the first
paint of a product page — the page that search engines and first-time shoppers
see — is empty until JavaScript runs.

This app renders the HTML on the server and sends it complete.

## How it talks to the API

* Server components call the Go API directly at `OPENSHOP_API_BASE` (default
  `http://localhost:8080`), using Next's data cache with short revalidation
  windows and tags.
* Mutations go through **route handlers on this origin** (`/api/auth/*`,
  `/api/cart/*`). The handler reads the shopper's cookies, attaches them as the
  API credentials and relays the response.
* Tokens live in `httpOnly` cookies, never in JavaScript-reachable storage, so
  no client-side auth state exists at all.
* `middleware.ts` issues a guest id for anonymous shoppers (the API keys a guest
  cart by `X-Guest-Id`) and silently refreshes an expired access token, because
  a server component cannot write cookies itself.

## Pages

| Route | Rendering |
| --- | --- |
| `/` | Server: hero and featured products from the ops configuration |
| `/products` | Server: filters and paging live in the URL, so pages are cacheable and shareable |
| `/products/[id]` | Server: metadata, JSON-LD, gallery, delivery expectation, FAQ; one client island for the cart |
| `/cart` | Server totals, client islands for quantity/remove |
| `/login`, `/register` | Server shell, tokens stored as cookies by a route handler |
| `/account/orders`, `/account/orders/[id]` | Server, reads the token cookie |

## Server actions

Writing a review is a **server action**, not a fetch from the browser. That
keeps the write on the server, and it lets the action invalidate exactly what
changed (`updateTag("reviews:<productId>")` plus `revalidatePath`), so the new
review is visible on the next render rather than one revalidation window later.
The form is a plain `<form action={...}>`, which the state hook only decorates
with inline feedback.

## Environment

| Variable | Purpose |
| --- | --- |
| `OPENSHOP_API_BASE` | Server-side API address (never exposed to the browser) |
| `NEXT_PUBLIC_API_BASE` | Only used by the rewrite; defaults to `/api/v1` |

## Development

```sh
pnpm --filter @openshop/front-ssr dev      # http://localhost:3000
pnpm --filter @openshop/front-ssr build
pnpm --filter @openshop/front-ssr lint
```

`OPENSHOP_API_BASE=http://localhost:8080` is the default, matching a locally
running `openshop` server.
