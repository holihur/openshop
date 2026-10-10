"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";

/** Adds a message to a ticket thread. */
export function TicketReply({
  ticketId,
  labels,
}: {
  ticketId: string;
  labels: { title: string; placeholder: string; submit: string; sending: string; failed: string };
}) {
  const router = useRouter();
  const [body, setBody] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [pending, startTransition] = useTransition();

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      const res = await fetch(`/api/tickets/${ticketId}/messages`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ body }),
      });
      if (!res.ok) {
        const payload = (await res.json().catch(() => ({}))) as { error?: { message?: string } };
        throw new Error(payload.error?.message ?? labels.failed);
      }
      setBody("");
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
      <textarea
        value={body}
        onChange={(event) => setBody(event.target.value)}
        placeholder={labels.placeholder}
        aria-label={labels.placeholder}
        required
        rows={3}
        className="border-input w-full rounded-md border px-3 py-2"
      />
      <button
        type="submit"
        disabled={busy || pending}
        className="bg-primary text-primary-foreground h-9 rounded-md px-4 disabled:opacity-50"
      >
        {busy || pending ? labels.sending : labels.submit}
      </button>
      {error ? <p className="text-sm text-red-600">{error}</p> : null}
    </form>
  );
}
