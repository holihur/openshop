import { NextResponse } from "next/server";

import { API_BASE, ACCESS_COOKIE, REFRESH_COOKIE } from "@/lib/api";

/**
 * Sign-in is handled by a route handler so the access and refresh tokens are
 * stored in httpOnly cookies. The browser never sees them, which is what makes
 * server rendering possible: every page reads the cookie on the server.
 */
export async function POST(request: Request) {
  const body = (await request.json().catch(() => ({}))) as {
    identifier?: string;
    password?: string;
  };

  const res = await fetch(`${API_BASE}/api/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json", "Accept-Language": "en" },
    body: JSON.stringify({ identifier: body.identifier, password: body.password }),
    cache: "no-store",
  });
  const payload = (await res.json().catch(() => ({}))) as {
    data?: { accessToken: string; refreshToken: string; expiresIn: number };
    error?: { code: string; message: string };
  };

  if (!res.ok || !payload.data) {
    return NextResponse.json(
      { error: payload.error ?? { code: "login_failed", message: "Sign in failed" } },
      { status: res.status || 400 },
    );
  }

  const response = NextResponse.json({ data: { ok: true } });
  response.cookies.set(ACCESS_COOKIE, payload.data.accessToken, {
    httpOnly: true,
    sameSite: "lax",
    path: "/",
    maxAge: Math.max(60, payload.data.expiresIn),
  });
  response.cookies.set(REFRESH_COOKIE, payload.data.refreshToken, {
    httpOnly: true,
    sameSite: "lax",
    path: "/",
    maxAge: 60 * 60 * 24 * 30,
  });
  return response;
}
