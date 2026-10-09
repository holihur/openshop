"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

import type { Locale } from "@/lib/i18n";

/**
 * Confirms the provider's return. The confirmation is a POST, so it cannot run
 * during server rendering: this island performs it once on mount.
 */
export function PaymentConfirmer({
  providerRef,
  provider,
  orderNo,
  locale,
  labels,
}: {
  providerRef: string;
  provider: string;
  orderNo: string;
  locale: Locale;
  labels: { confirming: string; paid: string; failed: string; failReason: string };
}) {
  const [state, setState] = useState<"pending" | "paid" | "failed">(
    providerRef ? "pending" : "failed",
  );
  const [reason, setReason] = useState(providerRef ? "" : labels.failReason);

  useEffect(() => {
    if (!providerRef) return;
    let cancelled = false;
    fetch("/api/payments/confirm", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ providerRef, provider }),
    })
      .then(async (res) => {
        if (cancelled) return;
        if (res.ok) {
          setState("paid");
          return;
        }
        const body = (await res.json().catch(() => ({}))) as { error?: { message?: string } };
        setReason(body.error?.message ?? labels.failed);
        setState("failed");
      })
      .catch(() => {
        if (!cancelled) setState("failed");
      });
    return () => {
      cancelled = true;
    };
  }, [providerRef, provider, labels.failed, labels.failReason]);

  return (
    <div className="mx-auto max-w-md space-y-4 py-16 text-center">
      <h1 className="text-xl font-semibold">
        {state === "pending" ? labels.confirming : state === "paid" ? labels.paid : labels.failed}
      </h1>
      {state === "failed" && reason ? <p className="text-muted-foreground text-sm">{reason}</p> : null}
      {orderNo ? (
        <p className="text-muted-foreground text-sm">
          {orderNo}
        </p>
      ) : null}
      <Link href="/account/orders" className="inline-block underline">
        {locale === "zh" ? "查看订单" : "View your orders"}
      </Link>
    </div>
  );
}
