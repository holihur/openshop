"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";

/** Marks every notification read, then re-renders the list on the server. */
export function MarkAllReadButton({ label }: { label: string }) {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [pending, startTransition] = useTransition();

  return (
    <button
      type="button"
      disabled={busy || pending}
      onClick={async () => {
        setBusy(true);
        try {
          const res = await fetch("/api/notifications/read-all", { method: "POST" });
          if (res.ok) startTransition(() => router.refresh());
        } finally {
          setBusy(false);
        }
      }}
      className="rounded-md border px-3 py-1.5 text-sm"
    >
      {label}
    </button>
  );
}
