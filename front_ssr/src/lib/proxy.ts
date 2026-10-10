import { NextResponse } from "next/server";
import { cookies } from "next/headers";

import { API_BASE, ACCESS_COOKIE, GUEST_COOKIE } from "@/lib/api";

/**
 * Forwards a browser request to the API with the shopper's cookies attached.
 *
 * Mutations run through this origin so the access token never reaches the
 * browser: pages and the route handlers that back them share one credential
 * source, and neither the API address nor the token appears in client code.
 */
export async function forward(
  request: Request,
  path: string,
  method: "GET" | "POST" | "PATCH" | "PUT" | "DELETE",
  body?: string,
): Promise<NextResponse> {
  const jar = await cookies();
  const headers: Record<string, string> = {};
  const payload = body ?? (method === "GET" || method === "DELETE" ? undefined : await request.text());
  if (payload !== undefined && payload !== "") headers["Content-Type"] = "application/json";

  const token = jar.get(ACCESS_COOKIE)?.value;
  if (token) headers.Authorization = `Bearer ${token}`;
  const guestId = jar.get(GUEST_COOKIE)?.value;
  if (guestId) headers["X-Guest-Id"] = guestId;

  const res = await fetch(`${API_BASE}/api/v1${path}`, {
    method,
    headers,
    body: payload === "" ? undefined : payload,
    cache: "no-store",
  });
  const reply = await res.text();
  // A 204 (or any empty reply) must not carry a body: Next rejects one.
  if (!reply) return new NextResponse(null, { status: res.status });
  return new NextResponse(reply, {
    status: res.status,
    headers: { "Content-Type": "application/json" },
  });
}

/** true when the API call succeeded, used to decide whether to re-render. */
export function isOk(res: NextResponse): boolean {
  return res.status >= 200 && res.status < 300;
}
