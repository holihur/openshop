import { NextResponse } from "next/server";

import { API_BASE } from "@/lib/api";

/**
 * Starts a social sign-in on this origin. The API's authorization endpoint
 * answers with the provider redirect; this handler relays it, so the browser
 * never needs to know the API address and no rewrite has to be baked into the
 * build.
 */
export async function GET(
  _request: Request,
  { params }: { params: Promise<{ provider: string }> },
) {
  const { provider } = await params;
  const res = await fetch(`${API_BASE}/api/v1/auth/social/${encodeURIComponent(provider)}/start`, {
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
