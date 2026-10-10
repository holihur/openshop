"use client";

import { useState } from "react";

import type { Address, DeliveryEstimate, PaymentMethod, ShippingMethod } from "@/lib/types";

/**
 * The checkout form. Everything that determines the price is fetched on the
 * server and passed in; this component only collects the shopper's choices and
 * posts them to the checkout route handler.
 */
export function PlaceOrder({
  signedIn,
  addresses,
  methods,
  estimates,
  providers,
  wallet,
  points,
  labels,
  cartSubtotalCents,
}: {
  signedIn: boolean;
  addresses: Address[];
  methods: ShippingMethod[];
  estimates: DeliveryEstimate[];
  providers: PaymentMethod[];
  /** Wallet balance and points available, when the shopper is signed in. */
  wallet: { balanceCents: number; currency: string } | null;
  points: { balance: number } | null;
  cartSubtotalCents: number;
  labels: {
    address: string;
    shipping: string;
    payment: string;
    email: string;
    recipient: string;
    phone: string;
    province: string;
    city: string;
    line1: string;
    postalCode: string;
    place: string;
    placing: string;
    failed: string;
    businessDays: string;
    scan: string;
    scanWith: string;
    coupon: string;
    apply: string;
    discount: string;
    wallet: string;
    points: string;
    invalidCoupon: string;
  };
}) {
  const [addressId, setAddressId] = useState(addresses.find((a) => a.default)?.id ?? addresses[0]?.id ?? "");
  const [shippingMethodId, setShippingMethodId] = useState(methods[0]?.id ?? "");
  const [provider, setProvider] = useState(providers[0] ?? "mock");
  const [email, setEmail] = useState("");
  const [guest, setGuest] = useState({
    recipient: "",
    phone: "",
    province: "",
    city: "",
    line1: "",
    postalCode: "",
  });
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  // WeChat Pay returns a code to scan instead of a URL to open.
  const [qr, setQr] = useState<{ svg: string; provider: string } | null>(null);
  // Optional money levers, all priced by the API.
  const [coupon, setCoupon] = useState("");
  const [discountCents, setDiscountCents] = useState(0);
  const [useWallet, setUseWallet] = useState(false);
  const [pointsToUse, setPointsToUse] = useState("");

  const estimateFor = (methodId: string) => estimates.find((e) => e.methodId === methodId);

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    setSaving(true);
    setError("");
    try {
      const res = await fetch("/api/checkout", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          addressId: signedIn ? addressId : undefined,
          address: signedIn ? undefined : guest,
          email: signedIn ? undefined : email,
          shippingMethodId,
          provider,
          couponCode: coupon || undefined,
          useWallet: useWallet && Boolean(wallet?.balanceCents),
          points: Number.parseInt(pointsToUse || "0", 10) || undefined,
          origin: window.location.origin,
        }),
      });
      const payload = (await res.json().catch(() => ({}))) as {
        data?: { redirectUrl?: string; qrSvg?: string };
        error?: { message?: string };
      };
      if (!res.ok) throw new Error(payload.error?.message ?? labels.failed);
      if (payload.data?.qrSvg) {
        // Nothing to navigate to: the shopper scans the code and the provider's
        // notification settles the order.
        setQr({ svg: payload.data.qrSvg, provider });
        setSaving(false);
        return;
      }
      // The provider takes over from here; a synchronous provider returns a
      // result URL on this origin.
      window.location.assign(payload.data?.redirectUrl ?? "/checkout/result");
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : labels.failed);
      setSaving(false);
    }
  }

  const field = (name: keyof typeof guest, label: string, required = true) => (
    <input
      name={name}
      aria-label={label}
      placeholder={label}
      required={required}
      value={guest[name]}
      onChange={(event) => setGuest((current) => ({ ...current, [name]: event.target.value }))}
      className="border-input h-9 w-full rounded-md border px-3"
    />
  );

  return (
    <form onSubmit={submit} className="space-y-6">
      <fieldset className="space-y-2">
        <legend className="font-medium">{labels.address}</legend>
        {signedIn && addresses.length > 0 ? (
          <select
            value={addressId}
            onChange={(event) => setAddressId(event.target.value)}
            aria-label={labels.address}
            className="border-input h-9 w-full rounded-md border px-3"
          >
            {addresses.map((address) => (
              <option key={address.id} value={address.id}>
                {address.recipient} · {address.province} {address.city} {address.line1}
              </option>
            ))}
          </select>
        ) : (
          <div className="grid gap-2 sm:grid-cols-2">
            {field("recipient", labels.recipient)}
            {field("phone", labels.phone)}
            {field("province", labels.province)}
            {field("city", labels.city)}
            {field("line1", labels.line1)}
            {field("postalCode", labels.postalCode, false)}
            {!signedIn ? (
              <input
                type="email"
                name="email"
                aria-label={labels.email}
                placeholder={labels.email}
                required
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                className="border-input h-9 w-full rounded-md border px-3 sm:col-span-2"
              />
            ) : null}
          </div>
        )}
      </fieldset>

      <fieldset className="space-y-2">
        <legend className="font-medium">{labels.shipping}</legend>
        {methods.map((method) => {
          const estimate = estimateFor(method.id);
          return (
            <label key={method.id} className="flex items-center gap-2 rounded-md border p-2 text-sm">
              <input
                type="radio"
                name="shipping"
                checked={shippingMethodId === method.id}
                onChange={() => setShippingMethodId(method.id)}
              />
              <span className="flex-1">
                {method.name}
                {estimate ? (
                  <span className="text-muted-foreground block text-xs">
                    {labels.businessDays} {estimate.minDays}–{estimate.maxDays}
                  </span>
                ) : null}
              </span>
              <span>{estimate ? (estimate.priceCents / 100).toFixed(2) : (method.flatRateCents / 100).toFixed(2)}</span>
            </label>
          );
        })}
      </fieldset>

      {/* Coupons, wallet and points all change the amount the API will charge,
          so the discount is priced by the API rather than guessed here. */}
      <fieldset className="space-y-3">
        <legend className="font-medium">{labels.coupon}</legend>
        <div className="flex gap-2">
          <input
            value={coupon}
            onChange={(event) => setCoupon(event.target.value.toUpperCase())}
            placeholder={labels.coupon}
            aria-label={labels.coupon}
            className="border-input h-9 flex-1 rounded-md border px-3"
          />
          <button
            type="button"
            onClick={async () => {
              setError("");
              try {
                const res = await fetch("/api/coupons/preview", {
                  method: "POST",
                  headers: { "Content-Type": "application/json" },
                  body: JSON.stringify({ code: coupon, subtotalCents: cartSubtotalCents }),
                });
                const payload = (await res.json().catch(() => ({}))) as {
                  data?: { discountCents?: number };
                  error?: { message?: string };
                };
                if (!res.ok) throw new Error(payload.error?.message ?? labels.invalidCoupon);
                setDiscountCents(payload.data?.discountCents ?? 0);
              } catch (cause) {
                setDiscountCents(0);
                setError(cause instanceof Error ? cause.message : labels.invalidCoupon);
              }
            }}
            className="rounded-md border px-4 text-sm"
          >
            {labels.apply}
          </button>
        </div>
        {discountCents > 0 ? (
          <p className="text-sm text-green-700 dark:text-green-400">
            {labels.discount}: −{(discountCents / 100).toFixed(2)}
          </p>
        ) : null}
      </fieldset>

      {(wallet?.balanceCents || points?.balance) ? (
        <fieldset className="space-y-2">
          <legend className="font-medium">{labels.payment}</legend>
          {wallet?.balanceCents ? (
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={useWallet}
                onChange={(event) => setUseWallet(event.target.checked)}
              />
              {labels.wallet} ({(wallet.balanceCents / 100).toFixed(2)})
            </label>
          ) : null}
          {points?.balance ? (
            <label className="flex items-center gap-2 text-sm">
              {labels.points}
              <input
                type="number"
                min={0}
                max={points.balance}
                value={pointsToUse}
                onChange={(event) => setPointsToUse(event.target.value)}
                aria-label={labels.points}
                className="border-input h-9 w-28 rounded-md border px-2"
              />
              <span className="text-muted-foreground">/ {points.balance}</span>
            </label>
          ) : null}
        </fieldset>
      ) : null}

      <fieldset className="space-y-2">
        <legend className="font-medium">{labels.payment}</legend>
        <div className="flex flex-wrap gap-2">
          {providers.map((option) => (
            <label key={option} className="flex items-center gap-2 rounded-md border p-2 text-sm">
              <input
                type="radio"
                name="provider"
                checked={provider === option}
                onChange={() => setProvider(option)}
              />
              {option}
            </label>
          ))}
        </div>
      </fieldset>

      {qr ? (
        <div className="flex flex-col items-center gap-2" role="group" aria-label={labels.scan}>
          {/* Generated by our own API from the provider payload. */}
          <div className="size-48 [&>svg]:size-full" dangerouslySetInnerHTML={{ __html: qr.svg }} />
          <p className="text-muted-foreground text-sm">
            {labels.scanWith} {qr.provider}
          </p>
        </div>
      ) : (
        <button
          type="submit"
          disabled={saving}
          className="bg-primary text-primary-foreground h-10 w-full rounded-md disabled:opacity-50"
        >
          {saving ? labels.placing : labels.place}
        </button>
      )}
      {error ? <p className="text-sm text-red-600">{error}</p> : null}
    </form>
  );
}
