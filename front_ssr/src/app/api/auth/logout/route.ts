import { NextResponse } from "next/server";
import { cookies } from "next/headers";

import { API_BASE, ACCESS_COOKIE, REFRESH_COOKIE } from "@/lib/api";

/**
 * Clears the session cookies. The refresh token is revoked at the API first so
 * a stolen cookie cannot be replayed after signing out.
 */
export async function POST() {
  const jar = await cookies();
  const token = jar.get(ACCESS_COOKIE)?.value;
  const refresh = jar.get(REFRESH_COOKIE)?.value;

  if (refresh) {
    await fetch(`${API_BASE}/api/v1/auth/logout`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
      body: JSON.stringify({ refreshToken: refresh }),
      cache: "no-store",
    }).catch(() => undefined);
  }

  const response = NextResponse.json({ data: { ok: true } });
  response.cookies.delete(ACCESS_COOKIE);
  response.cookies.delete(REFRESH_COOKIE);
  return response;
}
