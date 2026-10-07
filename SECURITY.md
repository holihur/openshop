# Security model and audit

This document records how openshop authenticates and authorises callers, and the
outcome of the security audit performed across the backend, both SPAs and the
deployment configuration.

## Authentication

| Credential | Used by | Notes |
| --- | --- | --- |
| Session JWT | browsers | HMAC-SHA256, 2 h access / 30 d refresh, deny-list on logout |
| Personal access token | scripts / CI | `osp_…`, SHA-256 hashed at rest, scoped, CIDR-bound |

* **JWT verification** accepts only HMAC algorithms (the key function rejects any
  other `alg`, which blocks algorithm-confusion and `none` tokens) and validates
  `iss` and `aud`. The audience binds a token to a surface, so a storefront token
  is rejected by the operations binary and vice versa.
* **Personal access tokens** are bound to a realm (`front` / `ops`), carry an
  explicit scope list, an optional CIDR allow-list, an optional expiry and a
  revocation timestamp. Only the SHA-256 hash is stored, so a database leak
  cannot be replayed. Managing tokens requires a browser session, so a leaked
  token cannot mint or revoke tokens.
* **Passwords** are hashed with bcrypt (default cost). Password reset and email
  verification use single-use random tokens.
* **Production guard**: `JWT_SECRET` must be a random value of at least 32
  characters, and CORS must not be `*`.

## Authorisation

* Every operations route is behind `RequireOps` plus a fine-grained
  `RequirePermission`. A personal access token must additionally grant the
  matching scope, so a token can never do more than its scope list, and never
  more than its owner's role.
* Storefront routes derive the required scope from the route and the HTTP method
  (`GET` reads, anything else writes); unmapped resources require the wildcard
  scope, so a new endpoint is closed to tokens until it is mapped deliberately.
* Object access is scoped to the caller: orders, tickets, addresses, returns,
  withdrawals, notifications and tokens are all filtered by the authenticated
  user id (or by an explicit admin flag).

## Audit findings

All of the following were found during the audit and have been fixed.

| # | Severity | Finding | Fix |
| --- | --- | --- | --- |
| 1 | High | Image upload fell back to the client-supplied file extension with no allow-list, so an operator could store an `.html` or `.svg` file that the media handler then served from the storefront origin (stored XSS). | The extension is now the allow-list, SVG is rejected, and the leading bytes must match the declared image type. |
| 2 | High | A broadcast notification accepted an arbitrary `link`, which the storefront rendered as a navigation target — a `javascript:` URL was stored XSS for every customer. | Links are restricted to same-origin relative paths. |
| 3 | Medium | With `HTTP_CORS_ORIGINS=*` the middleware reflected any origin together with `Access-Control-Allow-Credentials: true`. | A wildcard never carries credentials, and `*` is rejected in production. |
| 4 | Medium | Stripe webhook signatures had no timestamp tolerance, so a captured event could be replayed indefinitely. | Signatures outside a 5-minute window are rejected. |
| 5 | Medium | Uploads were served without `X-Content-Type-Options` or a content policy. | Uploads are served with `nosniff` and a sandboxing CSP; non-raster files download instead of rendering. |
| 6 | Medium | Sign-in had no limit beyond the global per-IP one (50 rps), leaving room for credential stuffing. | A dedicated per-IP limit (`security.auth_rate_limit_rps`, default 10 rps) guards the sign-in and password endpoints on both surfaces. |
| 7 | Low | A short or default `JWT_SECRET` was only rejected when it equalled the literal default. | Production now requires at least 32 characters. |

## Verified properties

* No SQL is built by string formatting; every query is parameterised.
* The media handler normalises the path and rejects anything outside the upload
  root (no traversal).
* No `dangerouslySetInnerHTML`, `innerHTML` or `eval` in either SPA.
* Webhook signatures are compared in constant time (`hmac.Equal`).
* The idempotency key is scoped to the caller, so a colliding key cannot leak
  another user's response.
* Request bodies on public webhook endpoints are read through a 1 MiB limit, and
  uploads are capped at 10 MiB.

## Accepted risks and recommendations

These are known and deliberate at the current stage; they are listed so they are
not mistaken for oversights.

1. **No account lockout.** Repeated failures are throttled per IP only. A
   distributed attack from many addresses can still grind passwords. Adding a
   per-account failure counter is recommended before high-value traffic.
2. **Rate limiting trusts the proxy configuration.** `HTTP_TRUSTED_PROXIES`
   must list the ingress addresses in production. If it is left empty behind a
   proxy, every client shares one bucket; if it is too broad, `X-Forwarded-For`
   can be spoofed to bypass limits.
3. **Session tokens live in `localStorage`.** Any XSS in the app would expose
   them. The app sets no cookies, which removes CSRF risk; moving sessions to
   `HttpOnly` cookies with a CSRF token is the stronger option if XSS risk
   grows.
4. **OIDC issuer URLs are operator-supplied**, so an operator can make the
   server perform an outbound request during discovery. Consider an issuer
   allow-list and blocking link-local / metadata addresses.
5. **The audit log is append-only but not tamper-evident.** A hash chain would
   make deletion or alteration detectable. Entries are pruned after
   `AUDIT_RETENTION` (90 days by default).
6. **No WAF or bot protection.** Large-scale abuse relies on the shared rate
   limiter and the deployment's edge controls.
