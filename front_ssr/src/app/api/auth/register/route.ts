import { NextResponse } from "next/server";

import { API_BASE, ACCESS_COOKIE, REFRESH_COOKIE } from "@/lib/api";

/** Registration also establishes a session, exactly like the API does. */
export async function POST(request: Request) {
  const body = (await request.json().catch(() => ({}))) as Record<string, unknown>;

  const res = await fetch(`${API_BASE}/api/v1/auth/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
    cache: "no-store",
  });
  const payload = (await res.json().catch(() => ({}))) as {
    data?: { accessToken: string; refreshToken: string; expiresIn: number };
    error?: { code: string; message: string };
  };

  if (!res.ok || !payload.data) {
    return NextResponse.json(
      { error: payload.error ?? { code: "register_failed", message: "Registration failed" } },
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
