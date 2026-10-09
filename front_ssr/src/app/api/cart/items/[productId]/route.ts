import { NextResponse } from "next/server";
import { cookies } from "next/headers";

import { API_BASE, ACCESS_COOKIE, GUEST_COOKIE } from "@/lib/api";

/**
 * Cart line mutations are proxied so the browser never receives the access
 * token. The API keys a cart line by product, with the variant in the body
 * (PATCH) or query (DELETE).
 */
async function forward(
  request: Request,
  productId: string,
  method: "PATCH" | "DELETE",
  body?: string,
) {
  const jar = await cookies();
  const token = jar.get(ACCESS_COOKIE)?.value;
  const guestId = jar.get(GUEST_COOKIE)?.value;

  const headers: Record<string, string> = {};
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (token) headers.Authorization = `Bearer ${token}`;
  if (guestId) headers["X-Guest-Id"] = guestId;

  const query = new URL(request.url).search;
  const res = await fetch(
    `${API_BASE}/api/v1/cart/items/${encodeURIComponent(productId)}${query}`,
    { method, headers, body, cache: "no-store" },
  );
  return new NextResponse(await res.text(), {
    status: res.status,
    headers: { "Content-Type": "application/json" },
  });
}

export async function PATCH(
  request: Request,
  { params }: { params: Promise<{ productId: string }> },
) {
  const { productId } = await params;
  return forward(request, productId, "PATCH", await request.text());
}

export async function DELETE(
  request: Request,
  { params }: { params: Promise<{ productId: string }> },
) {
  const { productId } = await params;
  return forward(request, productId, "DELETE");
}
