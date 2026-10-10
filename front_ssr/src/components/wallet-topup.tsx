"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";

/**
 * Tops the wallet up. The credit is applied by the API, so the page only has to
 * re-render to show the new balance.
 */
export function WalletTopUp({
  minCents,
  maxCents,
  labels,
}: {
  minCents: number;
  maxCents: number;
  labels: { title: string; amount: string; submit: string; sending: string; failed: string };
}) {
  const router = useRouter();
  const [amount, setAmount] = useState((minCents / 100).toFixed(2));
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [pending, startTransition] = useTransition();

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      const cents = Math.round(Number.parseFloat(amount || "0") * 100);
      const res = await fetch("/api/wallet/topup", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ amountCents: cents }),
      });
      const payload = (await res.json().catch(() => ({}))) as {
        error?: { message?: string };
      };
      if (!res.ok) throw new Error(payload.error?.message ?? labels.failed);
      // The credit is applied by the API; re-rendering shows the new balance.
      startTransition(() => router.refresh());
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : labels.failed);
    } finally {
      setBusy(false);
    }
  }

  return (
    <form onSubmit={submit} className="card-surface space-y-2 p-4">
      <p className="font-medium">{labels.title}</p>
      <div className="flex items-end gap-2">
        <label className="flex-1 space-y-1">
          <span className="text-muted-foreground text-xs">{labels.amount}</span>
          <input
            type="number"
            min={minCents / 100}
            max={maxCents / 100}
            step="0.01"
            value={amount}
            onChange={(event) => setAmount(event.target.value)}
            className="border-input h-9 w-full rounded-md border px-3"
          />
        </label>
        <button
          type="submit"
          disabled={busy || pending}
          className="bg-primary text-primary-foreground h-9 rounded-md px-4 disabled:opacity-50"
        >
          {busy || pending ? labels.sending : labels.submit}
        </button>
      </div>
      {error ? <p className="text-sm text-red-600">{error}</p> : null}
    </form>
  );
}
