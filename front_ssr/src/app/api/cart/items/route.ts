import { NextResponse } from "next/server";

import { API_BASE, ACCESS_COOKIE, GUEST_COOKIE } from "@/lib/api";
import { cookies } from "next/headers";

/**
 * Cart mutation proxy. The browser calls this origin, the handler attaches the
 * shopper's cookies as API credentials, and the response is relayed. Keeping the
 * token out of the browser is the reason a server-rendered storefront needs no
 * client-side auth state at all.
 */
export async function POST(request: Request) {
  const jar = await cookies();
  const token = jar.get(ACCESS_COOKIE)?.value;
  const guestId = jar.get(GUEST_COOKIE)?.value;

  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (token) headers.Authorization = `Bearer ${token}`;
  if (guestId) headers["X-Guest-Id"] = guestId;

  const res = await fetch(`${API_BASE}/api/v1/cart/items`, {
    method: "POST",
    headers,
    body: await request.text(),
    cache: "no-store",
  });
  return new NextResponse(await res.text(), {
    status: res.status,
    headers: { "Content-Type": "application/json" },
  });
}
