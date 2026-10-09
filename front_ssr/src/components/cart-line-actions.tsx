"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";

/**
 * Quantity and remove controls for a cart line. Both call the route handler on
 * this origin, then refresh the server component so the totals come back from
 * the API rather than being recomputed in the browser.
 */
export function CartLineActions({
  productId,
  variantId,
  quantity,
  labels,
}: {
  productId: string;
  variantId?: string;
  quantity: number;
  labels: { quantity: string; remove: string; updating: string };
}) {
  const router = useRouter();
  const [value, setValue] = useState(quantity);
  const [busy, setBusy] = useState(false);
  const [pending, startTransition] = useTransition();
  const suffix = variantId ? `?variantId=${encodeURIComponent(variantId)}` : "";

  async function send(next: number) {
    setBusy(true);
    try {
      const res =
        next <= 0
          ? await fetch(`/api/cart/items/${productId}${suffix}`, { method: "DELETE" })
          : await fetch(`/api/cart/items/${productId}`, {
              method: "PATCH",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify({ variantId, quantity: next }),
            });
      if (res.ok) {
        setValue(Math.max(0, next));
        startTransition(() => router.refresh());
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="flex items-center gap-2">
      <input
        type="number"
        min={0}
        value={value}
        aria-label={labels.quantity}
        onChange={(event) => setValue(Math.max(0, Number.parseInt(event.target.value, 10) || 0))}
        onBlur={() => {
          if (value !== quantity) void send(value);
        }}
        onKeyDown={(event) => {
          if (event.key === "Enter") {
            event.preventDefault();
            void send(value);
          }
        }}
        className="border-input h-9 w-16 rounded-md border px-2"
      />
      <button
        type="button"
        onClick={() => void send(0)}
        disabled={busy || pending}
        className="text-muted-foreground text-sm underline"
      >
        {busy || pending ? labels.updating : labels.remove}
      </button>
    </div>
  );
}
