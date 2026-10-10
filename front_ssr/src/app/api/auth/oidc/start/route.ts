import { NextResponse } from "next/server";

import { API_BASE } from "@/lib/api";

/**
 * Starts single sign-on on this origin. The API answers with the provider
 * redirect, which is relayed so the browser never needs the API address.
 */
export async function GET(request: Request) {
  const provider = new URL(request.url).searchParams.get("provider") ?? "";
  const query = provider ? `?provider=${encodeURIComponent(provider)}` : "";
  const res = await fetch(`${API_BASE}/api/v1/auth/oidc/start${query}`, {
    redirect: "manual",
    cache: "no-store",
  });
  const location = res.headers.get("location");
  if (!location) {
    return NextResponse.json(
      { error: { code: "unavailable", message: "This sign-in method is not available" } },
      { status: res.status === 200 ? 502 : res.status },
    );
  }
  return NextResponse.redirect(location, 302);
}
