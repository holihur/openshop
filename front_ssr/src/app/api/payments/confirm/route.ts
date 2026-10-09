import { NextResponse } from "next/server";
import { cookies } from "next/headers";

import { API_BASE, ACCESS_COOKIE, GUEST_COOKIE } from "@/lib/api";

/**
 * Confirms a sandbox payment. A real provider calls a webhook instead; this
 * mirrors what the single-page storefront does so the demo flow completes.
 */
export async function POST(request: Request) {
  const jar = await cookies();
  const token = jar.get(ACCESS_COOKIE)?.value;
  const guestId = jar.get(GUEST_COOKIE)?.value;

  const body = (await request.json().catch(() => ({}))) as {
    providerRef?: string;
    provider?: string;
  };

  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (token) headers.Authorization = `Bearer ${token}`;
  if (guestId) headers["X-Guest-Id"] = guestId;

  const path = token ? "/payments/simulate" : "/guest/payments/simulate";
  const res = await fetch(`${API_BASE}/api/v1${path}`, {
    method: "POST",
    headers,
    body: JSON.stringify({ providerRef: body.providerRef, provider: body.provider ?? "mock" }),
    cache: "no-store",
  });
  return new NextResponse(await res.text(), {
    status: res.status,
    headers: { "Content-Type": "application/json" },
  });
}
