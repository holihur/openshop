"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";

import type { Locale } from "@/lib/i18n";

/**
 * Saves or removes a product. The write goes through this origin, so the API
 * credentials stay in the cookie.
 */
export function WishlistButton({
  productId,
  saved,
  locale,
}: {
  productId: string;
  saved: boolean;
  locale: Locale;
}) {
  const router = useRouter();
  const [on, setOn] = useState(saved);
  const [busy, setBusy] = useState(false);
  const [pending, startTransition] = useTransition();
  const label = on
    ? locale === "zh"
      ? "已收藏，点击取消"
      : "Saved — click to remove"
    : locale === "zh"
      ? "收藏"
      : "Save for later";

  async function toggle() {
    setBusy(true);
    try {
      const res = on
        ? await fetch(`/api/wishlist/${productId}`, { method: "DELETE" })
        : await fetch("/api/wishlist", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ productId }),
          });
      if (res.status === 401) {
        // Saving needs an account; send the shopper to sign in rather than
        // failing silently.
        router.push("/login");
        return;
      }
      if (res.ok) {
        setOn(!on);
        startTransition(() => router.refresh());
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <button
      type="button"
      onClick={() => void toggle()}
      disabled={busy || pending}
      aria-pressed={on}
      title={label}
      className={
        on
          ? "border-primary text-primary rounded-md border px-3 py-1.5 text-sm"
          : "text-muted-foreground rounded-md border px-3 py-1.5 text-sm hover:border-current"
      }
    >
      {on ? "♥" : "♡"} {label}
    </button>
  );
}
