import { NextResponse } from "next/server";

import { ACCESS_COOKIE, REFRESH_COOKIE } from "@/lib/api";

/**
 * Exchanges the tokens a provider handed back for httpOnly session cookies.
 * The tokens arrive in a URL fragment, which only the browser can read, so a
 * tiny client page posts them here rather than writing a cookie by script
 * (which would make them readable by any script on the page).
 */
export async function POST(request: Request) {
  const body = (await request.json().catch(() => ({}))) as {
    accessToken?: string;
    refreshToken?: string;
    expiresIn?: number;
  };
  if (!body.accessToken) {
    return NextResponse.json(
      { error: { code: "invalid_argument", message: "Missing access token" } },
      { status: 400 },
    );
  }
  const response = NextResponse.json({ data: { ok: true } });
  response.cookies.set(ACCESS_COOKIE, body.accessToken, {
    httpOnly: true,
    sameSite: "lax",
    path: "/",
    maxAge: Math.max(60, body.expiresIn ?? 900),
  });
  if (body.refreshToken) {
    response.cookies.set(REFRESH_COOKIE, body.refreshToken, {
      httpOnly: true,
      sameSite: "lax",
      path: "/",
      maxAge: 60 * 60 * 24 * 30,
    });
  }
  return response;
}
