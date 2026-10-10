"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";

/** Opens a support ticket; guests supply an email so staff can answer. */
export function TicketForm({
  signedIn,
  labels,
}: {
  signedIn: boolean;
  labels: {
    title: string;
    subject: string;
    body: string;
    email: string;
    name: string;
    submit: string;
    sending: string;
    failed: string;
  };
}) {
  const router = useRouter();
  const [subject, setSubject] = useState("");
  const [body, setBody] = useState("");
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [pending, startTransition] = useTransition();

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      const res = await fetch("/api/tickets", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ subject, body, kind: "other", email, name }),
      });
      const payload = (await res.json().catch(() => ({}))) as {
        data?: { id?: string };
        error?: { message?: string };
      };
      if (!res.ok) throw new Error(payload.error?.message ?? labels.failed);
      setSubject("");
      setBody("");
      if (payload.data?.id) {
        router.push(`/support/${payload.data.id}`);
        return;
      }
      startTransition(() => router.refresh());
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : labels.failed);
    } finally {
      setBusy(false);
    }
  }

  return (
    <form onSubmit={submit} className="card-surface space-y-3 p-4">
      <p className="font-medium">{labels.title}</p>
      <input
        value={subject}
        onChange={(event) => setSubject(event.target.value)}
        placeholder={labels.subject}
        aria-label={labels.subject}
        required
        className="border-input h-9 w-full rounded-md border px-3"
      />
      {!signedIn ? (
        <div className="grid gap-3 sm:grid-cols-2">
          <input
            type="email"
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            placeholder={labels.email}
            aria-label={labels.email}
            required
            className="border-input h-9 w-full rounded-md border px-3"
          />
          <input
            value={name}
            onChange={(event) => setName(event.target.value)}
            placeholder={labels.name}
            aria-label={labels.name}
            className="border-input h-9 w-full rounded-md border px-3"
          />
        </div>
      ) : null}
      <textarea
        value={body}
        onChange={(event) => setBody(event.target.value)}
        placeholder={labels.body}
        aria-label={labels.body}
        required
        rows={4}
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
