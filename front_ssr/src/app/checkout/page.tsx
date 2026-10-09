import { redirect } from "next/navigation";

import { apiGet } from "@/lib/api";
import { PlaceOrder } from "@/components/place-order";
import { formatMoney } from "@/lib/format";
import { translator } from "@/lib/i18n";
import { getShopper } from "@/lib/session";
import { resolveLocale } from "@/app/layout";
import type {
  Address,
  Cart,
  DeliveryEstimate,
  PaymentMethod,
  ShippingMethod,
} from "@/lib/types";

/**
 * Checkout is rendered on the server: the cart, the shipping cost and the
 * delivery window are all resolved by the API before the page is sent, so the
 * shopper sees the final amount without waiting for anything to load.
 */
export default async function CheckoutPage() {
  const locale = await resolveLocale();
  const t = translator(locale);
  const shopper = await getShopper();
  const headers = { token: shopper.token, guestId: shopper.guestId };

  const cart = await apiGet<Cart>("/cart", headers).catch(
    () => ({ items: [], totalCents: 0, totalCount: 0 }) as Cart,
  );
  if (cart.items.length === 0) redirect("/cart");

  const [methods, estimates, providers, addresses] = await Promise.all([
    apiGet<ShippingMethod[]>("/shipping-methods", { revalidate: 60, tags: ["shipping"] }).catch(
      () => [] as ShippingMethod[],
    ),
    apiGet<DeliveryEstimate[]>(`/delivery-estimates?subtotalCents=${cart.totalCents}`, {
      revalidate: 60,
      tags: ["shipping"],
    }).catch(() => [] as DeliveryEstimate[]),
    apiGet<PaymentMethod[]>("/payment-methods", { revalidate: 60 }).catch(() => [] as PaymentMethod[]),
    shopper.token
      ? apiGet<Address[]>("/addresses", { token: shopper.token }).catch(() => [] as Address[])
      : Promise.resolve([] as Address[]),
  ]);

  return (
    <div className="grid gap-8 lg:grid-cols-[2fr_1fr]">
      <PlaceOrder
        signedIn={Boolean(shopper.token)}
        addresses={addresses}
        methods={methods.filter((method) => method.active)}
        estimates={estimates}
        providers={providers}
        labels={{
          address: t("checkout.address"),
          shipping: t("cart.shipping"),
          payment: t("checkout.payment"),
          email: t("auth.email"),
          recipient: t("checkout.recipient"),
          phone: t("checkout.phone"),
          province: t("checkout.province"),
          city: t("checkout.city"),
          line1: t("checkout.line1"),
          postalCode: t("checkout.postalCode"),
          place: t("cart.checkout"),
          placing: t("cart.updating"),
          failed: t("error.generic"),
          businessDays: t("checkout.deliveryIn"),
        }}
      />

      <aside className="space-y-2 rounded-lg border p-4">
        <h2 className="font-medium">{t("cart.title")}</h2>
        <ul className="space-y-1 text-sm">
          {cart.items.map((item) => (
            <li key={`${item.productId}-${item.variantId ?? ""}`} className="flex justify-between">
              <span>
                {item.title} × {item.quantity}
              </span>
              <span>{formatMoney(item.priceCents * item.quantity, item.currency, locale)}</span>
            </li>
          ))}
        </ul>
        <div className="flex justify-between border-t pt-2 text-lg font-semibold">
          <span>{t("cart.total")}</span>
          <span>{formatMoney(cart.totalCents, undefined, locale)}</span>
        </div>
      </aside>
    </div>
  );
}
