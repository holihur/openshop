import { cookies } from "next/headers";

import { ACCESS_COOKIE, GUEST_COOKIE, REFRESH_COOKIE, apiGet, apiSend } from "@/lib/api";
import type { AuthSession, Category, Product, SiteConfig } from "@/lib/types";

/**
 * Request-scoped view of the shopper. Every server-rendered page reads this so
 * the same code works for a guest (a guest id cookie owns the cart) and for a
 * signed-in customer (a bearer token).
 */
export interface Shopper {
  token?: string;
  guestId?: string;
}

export async function getShopper(): Promise<Shopper> {
  const jar = await cookies();
  return {
    token: jar.get(ACCESS_COOKIE)?.value,
    guestId: jar.get(GUEST_COOKIE)?.value,
  };
}

export async function isSignedIn(): Promise<boolean> {
  return Boolean((await getShopper()).token);
}

/**
 * Refreshes the access token using the refresh cookie. Route handlers call this
 * so a shopper is never bounced to the sign-in page while their refresh token is
 * still valid.
 */
export async function refreshSession(): Promise<AuthSession | null> {
  const jar = await cookies();
  const refresh = jar.get(REFRESH_COOKIE)?.value;
  if (!refresh) return null;
  try {
    return await apiSend<AuthSession>("/auth/refresh", {
      method: "POST",
      body: { refreshToken: refresh },
    });
  } catch {
    return null;
  }
}

/**
 * Storefront configuration. It changes rarely and is written by the ops console,
 * so it is cached and tagged for revalidation.
 */
export function getSiteConfig(): Promise<SiteConfig> {
  return apiGet<SiteConfig>("/site", { revalidate: 60, tags: ["site"] });
}

export function getCategories(): Promise<Category[]> {
  return apiGet<Category[]>("/categories", { revalidate: 60, tags: ["catalog"] });
}
