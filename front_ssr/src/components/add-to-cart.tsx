"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";

interface VariantOption {
  id: string;
  name: string;
  priceCents: number;
  stock: number;
}

/**
 * The only interactive part of the product page. Adding to the cart goes
 * through a route handler on this origin, which attaches the guest or access
 * token cookie — the browser never handles an API token itself.
 */
export function AddToCartButton({
  productId,
  variants,
  disabled,
  labels,
}: {
  productId: string;
  variants: VariantOption[];
  disabled: boolean;
  labels: { add: string; adding: string; quantity: string; variant: string; failed: string };
}) {
  const router = useRouter();
  const [variantId, setVariantId] = useState(variants[0]?.id ?? "");
  const [quantity, setQuantity] = useState(1);
  const [error, setError] = useState("");
  const [pending, startTransition] = useTransition();
  const [saving, setSaving] = useState(false);

  async function add() {
    setError("");
    setSaving(true);
    try {
      const res = await fetch("/api/cart/items", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ productId, variantId: variantId || undefined, quantity }),
      });
      if (!res.ok) {
        const body = (await res.json().catch(() => ({}))) as { error?: { message?: string } };
        throw new Error(body.error?.message ?? labels.failed);
      }
      startTransition(() => router.refresh());
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : labels.failed);
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="space-y-2">
      {variants.length > 0 ? (
        <select
          value={variantId}
          onChange={(event) => setVariantId(event.target.value)}
          aria-label={labels.variant}
          className="border-input h-9 w-full rounded-md border px-3"
        >
          {variants.map((variant) => (
            <option key={variant.id} value={variant.id} disabled={variant.stock <= 0}>
              {variant.name}
              {variant.stock <= 0 ? " (sold out)" : ""}
            </option>
          ))}
        </select>
      ) : null}
      <div className="flex items-center gap-2">
        <input
          type="number"
          min={1}
          value={quantity}
          onChange={(event) => setQuantity(Math.max(1, Number.parseInt(event.target.value, 10) || 1))}
          aria-label={labels.quantity}
          className="border-input h-9 w-20 rounded-md border px-3"
        />
        <button
          type="button"
          onClick={add}
          disabled={disabled || saving || pending}
          className="bg-primary text-primary-foreground h-9 flex-1 rounded-md px-4 disabled:opacity-50"
        >
          {saving ? labels.adding : labels.add}
        </button>
      </div>
      {error ? <p className="text-sm text-red-600">{error}</p> : null}
    </div>
  );
}
