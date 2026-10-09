import { NextResponse, type NextRequest } from "next/server";

import { ACCESS_COOKIE, GUEST_COOKIE, REFRESH_COOKIE } from "@/lib/api";

const API_BASE = process.env.OPENSHOP_API_BASE ?? "http://localhost:8080";

/**
 * Next 16 calls this the proxy (it replaced the middleware convention). It runs
 * before any page is rendered and has two jobs:
 *
 * 1. Give an anonymous shopper a stable guest id, because the API owns the cart
 *    by that id. It is set as a cookie so the server can forward it as an
 *    X-Guest-Id header.
 * 2. Refresh an expired access token transparently. A server component cannot
 *    write cookies, so the refresh has to happen here; without it the shopper
 *    would be signed out the moment the access token expired.
 */
export default async function proxy(request: NextRequest) {
  // Cloning the request headers is what forwards the cookies set below to the
  // server components in this same render.
  const response = NextResponse.next({ request: { headers: request.headers } });

  if (!request.cookies.get(GUEST_COOKIE)?.value) {
    const guestId = crypto.randomUUID();
    request.cookies.set(GUEST_COOKIE, guestId);
    response.cookies.set(GUEST_COOKIE, guestId, {
      httpOnly: true,
      sameSite: "lax",
      path: "/",
      maxAge: 60 * 60 * 24 * 365,
    });
  }

  const access = request.cookies.get(ACCESS_COOKIE)?.value;
  const refresh = request.cookies.get(REFRESH_COOKIE)?.value;
  if (!access && refresh) {
    try {
      const res = await fetch(`${API_BASE}/api/v1/auth/refresh`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refreshToken: refresh }),
        cache: "no-store",
      });
      if (res.ok) {
        const payload = (await res.json()) as {
          data?: { accessToken: string; refreshToken: string; expiresIn: number };
        };
        const session = payload.data;
        if (session) {
          request.cookies.set(ACCESS_COOKIE, session.accessToken);
          response.cookies.set(ACCESS_COOKIE, session.accessToken, {
            httpOnly: true,
            sameSite: "lax",
            path: "/",
            maxAge: Math.max(60, session.expiresIn),
          });
          response.cookies.set(REFRESH_COOKIE, session.refreshToken, {
            httpOnly: true,
            sameSite: "lax",
            path: "/",
            maxAge: 60 * 60 * 24 * 30,
          });
        }
      } else {
        // The refresh token is dead: drop both cookies so the header stops
        // claiming the shopper is signed in.
        response.cookies.delete(ACCESS_COOKIE);
        response.cookies.delete(REFRESH_COOKIE);
      }
    } catch {
      // The API being unreachable must not break page rendering.
    }
  }

  return response;
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
