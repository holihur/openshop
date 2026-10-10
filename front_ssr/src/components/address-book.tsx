"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";

interface AddressLabels {
  add: string;
  recipient: string;
  phone: string;
  province: string;
  city: string;
  district: string;
  line1: string;
  postalCode: string;
  save: string;
  saving: string;
  remove: string;
  makeDefault: string;
  isDefault: string;
  empty: string;
  failed: string;
}

const empty = {
  recipient: "",
  phone: "",
  province: "",
  city: "",
  district: "",
  line1: "",
  postalCode: "",
};

/** Add, remove and default the delivery addresses. */
export function AddressBook({
  addresses,
  labels,
}: {
  addresses: {
    id: string;
    recipient: string;
    phone: string;
    province: string;
    city: string;
    district: string;
    line1: string;
    postalCode: string;
    default: boolean;
  }[];
  labels: AddressLabels;
}) {
  const router = useRouter();
  const [draft, setDraft] = useState(empty);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [pending, startTransition] = useTransition();

  async function send(path: string, method: "POST" | "DELETE", body?: unknown) {
    setBusy(true);
    setError("");
    try {
      const res = await fetch(path, {
        method,
        ...(body ? { headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) } : {}),
      });
      if (!res.ok) {
        const payload = (await res.json().catch(() => ({}))) as { error?: { message?: string } };
        throw new Error(payload.error?.message ?? labels.failed);
      }
      startTransition(() => router.refresh());
      return true;
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : labels.failed);
      return false;
    } finally {
      setBusy(false);
    }
  }

  const field = (name: keyof typeof empty, label: string, required = true) => (
    <label className="space-y-1">
      <span className="text-muted-foreground text-xs">{label}</span>
      <input
        value={draft[name]}
        required={required}
        onChange={(event) => setDraft({ ...draft, [name]: event.target.value })}
        className="border-input h-9 w-full rounded-md border px-3"
      />
    </label>
  );

  return (
    <div className="space-y-6">
      {addresses.length === 0 ? (
        <p className="text-muted-foreground rounded-lg border border-dashed py-10 text-center text-sm">
          {labels.empty}
        </p>
      ) : (
        <ul className="grid gap-3 sm:grid-cols-2">
          {addresses.map((address) => (
            <li key={address.id} className="card-surface space-y-2 p-4 text-sm">
              <div className="flex items-center justify-between">
                <p className="font-medium">
                  {address.recipient} · {address.phone}
                </p>
                {address.default ? (
                  <span className="bg-muted rounded-full px-2 text-xs">{labels.isDefault}</span>
                ) : null}
              </div>
              <p className="text-muted-foreground">
                {address.province} {address.city} {address.district} {address.line1}
                {address.postalCode ? ` · ${address.postalCode}` : ""}
              </p>
              <div className="flex gap-3 text-xs">
                {!address.default ? (
                  <button
                    type="button"
                    disabled={busy || pending}
                    onClick={() => void send(`/api/addresses/${address.id}/default`, "POST")}
                    className="underline"
                  >
                    {labels.makeDefault}
                  </button>
                ) : null}
                <button
                  type="button"
                  disabled={busy || pending}
                  onClick={() => void send(`/api/addresses/${address.id}`, "DELETE")}
                  className="text-muted-foreground underline"
                >
                  {labels.remove}
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}

      <form
        onSubmit={async (event) => {
          event.preventDefault();
          if (await send("/api/addresses", "POST", draft)) setDraft(empty);
        }}
        className="card-surface space-y-3 p-4"
      >
        <p className="font-medium">{labels.add}</p>
        <div className="grid gap-3 sm:grid-cols-3">
          {field("recipient", labels.recipient)}
          {field("phone", labels.phone)}
          {field("province", labels.province)}
          {field("city", labels.city)}
          {field("district", labels.district, false)}
          {field("postalCode", labels.postalCode, false)}
          <div className="sm:col-span-3">{field("line1", labels.line1)}</div>
        </div>
        <button
          type="submit"
          disabled={busy || pending}
          className="bg-primary text-primary-foreground h-9 rounded-md px-4 disabled:opacity-50"
        >
          {busy || pending ? labels.saving : labels.save}
        </button>
        {error ? <p className="text-sm text-red-600">{error}</p> : null}
      </form>
    </div>
  );
}
