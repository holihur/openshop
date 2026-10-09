"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import Link from "next/link";

type Mode = "login" | "register";

/**
 * Sign-in and sign-up forms. They post to this origin so the route handler can
 * put the tokens in httpOnly cookies, then refresh the router so the server
 * components re-render as the signed-in shopper.
 */
export function AuthForm({
  mode,
  allowRegistration,
  labels,
}: {
  mode: Mode;
  allowRegistration: boolean;
  labels: {
    email: string;
    password: string;
    name: string;
    submit: string;
    switchTo: string;
    switchHref: string;
    failed: string;
  };
}) {
  const router = useRouter();
  const [identifier, setIdentifier] = useState("");
  const [password, setPassword] = useState("");
  const [name, setName] = useState("");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const [pending, startTransition] = useTransition();

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    setError("");
    setSaving(true);
    try {
      const res = await fetch(`/api/auth/${mode}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(
          mode === "login"
            ? { identifier, password }
            : { email: identifier, password, name: name || undefined },
        ),
      });
      if (!res.ok) {
        const body = (await res.json().catch(() => ({}))) as { error?: { message?: string } };
        throw new Error(body.error?.message ?? labels.failed);
      }
      startTransition(() => {
        router.push("/account/orders");
        router.refresh();
      });
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : labels.failed);
    } finally {
      setSaving(false);
    }
  }

  return (
    <form onSubmit={submit} className="mx-auto max-w-sm space-y-3">
      <div className="space-y-1">
        <label htmlFor="identifier" className="text-sm font-medium">
          {labels.email}
        </label>
        <input
          id="identifier"
          type="email"
          autoComplete="email"
          value={identifier}
          onChange={(event) => setIdentifier(event.target.value)}
          required
          className="border-input h-9 w-full rounded-md border px-3"
        />
      </div>
      {mode === "register" ? (
        <div className="space-y-1">
          <label htmlFor="name" className="text-sm font-medium">
            {labels.name}
          </label>
          <input
            id="name"
            value={name}
            onChange={(event) => setName(event.target.value)}
            className="border-input h-9 w-full rounded-md border px-3"
          />
        </div>
      ) : null}
      <div className="space-y-1">
        <label htmlFor="password" className="text-sm font-medium">
          {labels.password}
        </label>
        <input
          id="password"
          type="password"
          autoComplete={mode === "login" ? "current-password" : "new-password"}
          value={password}
          onChange={(event) => setPassword(event.target.value)}
          required
          minLength={8}
          className="border-input h-9 w-full rounded-md border px-3"
        />
      </div>
      <button
        type="submit"
        disabled={saving || pending}
        className="bg-primary text-primary-foreground h-9 w-full rounded-md disabled:opacity-50"
      >
        {labels.submit}
      </button>
      {error ? <p className="text-sm text-red-600">{error}</p> : null}
      {allowRegistration || mode === "register" ? (
        <p className="text-muted-foreground text-center text-sm">
          <Link href={labels.switchHref} className="underline">
            {labels.switchTo}
          </Link>
        </p>
      ) : null}
    </form>
  );
}
