"use client";

import { useRouter } from "next/navigation";
import { useTransition } from "react";

/**
 * Writes the chosen display currency to a cookie and re-renders on the server,
 * so every price in the page is converted consistently.
 */
export function CurrencySwitcher({
  current,
  available,
}: {
  current: string;
  available: string[];
}) {
  const router = useRouter();
  const [pending, startTransition] = useTransition();
  if (available.length < 2) return null;

  return (
    <select
      value={current}
      aria-label="Display currency"
      disabled={pending}
      onChange={(event) => {
        document.cookie = `ssr_currency=${event.target.value}; path=/; max-age=${60 * 60 * 24 * 365}`;
        startTransition(() => router.refresh());
      }}
      className="border-input bg-background h-7 rounded-md border px-2 text-xs"
    >
      {available.map((code) => (
        <option key={code} value={code}>
          {code}
        </option>
      ))}
    </select>
  );
}
