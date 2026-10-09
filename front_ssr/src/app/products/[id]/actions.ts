"use server";

import { revalidatePath, updateTag } from "next/cache";
import { cookies } from "next/headers";

import { ACCESS_COOKIE, apiSend } from "@/lib/api";

export interface ReviewState {
  ok: boolean;
  message?: string;
}

/**
 * Publishing a review is a server action rather than a fetch from the browser:
 * the write happens on the server with the session cookie, and the affected
 * cache tag is invalidated so the new review is visible on the very next render
 * (read-your-own-writes) instead of up to a revalidation window later.
 */
export async function submitReview(
  _previous: ReviewState,
  formData: FormData,
): Promise<ReviewState> {
  const productId = String(formData.get("productId") ?? "");
  const rating = Number.parseInt(String(formData.get("rating") ?? "5"), 10);
  const title = String(formData.get("title") ?? "").trim();
  const body = String(formData.get("body") ?? "").trim();

  const token = (await cookies()).get(ACCESS_COOKIE)?.value;
  if (!token) return { ok: false, message: "unauthenticated" };
  if (!productId) return { ok: false, message: "invalid_argument" };

  try {
    await apiSend(`/products/${encodeURIComponent(productId)}/reviews`, {
      method: "POST",
      token,
      body: { rating: Number.isFinite(rating) ? rating : 5, title, body },
    });
  } catch (cause) {
    const message = cause instanceof Error ? cause.message : "request_failed";
    return { ok: false, message };
  }

  // Invalidate exactly what changed: this product's reviews everywhere, and the
  // product page itself (its rating and review count change too).
  updateTag(`reviews:${productId}`);
  revalidatePath(`/products/${productId}`);
  return { ok: true };
}
