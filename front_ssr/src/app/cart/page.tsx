import Link from "next/link";

import { apiGet } from "@/lib/api";
import { CartLineActions } from "@/components/cart-line-actions";
import { formatMoney } from "@/lib/format";
import { translator } from "@/lib/i18n";
import { getPricing } from "@/lib/pricing";
import { getShopper } from "@/lib/session";
import { resolveLocale } from "@/app/layout";
import type { Cart } from "@/lib/types";

/**
 * The cart is rendered on the server from the API's own cart, so the items and
 * the total are authoritative (stock, pricing and promotions all live there).
 * Quantity changes are small client islands.
 */
export default async function CartPage() {
  const locale = await resolveLocale();
  const t = translator(locale);
  const shopper = await getShopper();
  const [cart, pricing] = await Promise.all([
    apiGet<Cart>("/cart", { ...shopper }).catch(
      () => ({ items: [], totalCents: 0, totalCount: 0 }) as Cart,
    ),
    getPricing(locale),
  ]);

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t("cart.title")}</h1>

      {cart.items.length === 0 ? (
        <div className="rounded-lg border border-dashed py-16 text-center">
          <p className="text-muted-foreground">{t("cart.empty")}</p>
          <Link href="/products" className="mt-3 inline-block underline">
            {t("nav.products")}
          </Link>
        </div>
      ) : (
        <div className="space-y-4">
          <ul className="divide-y rounded-lg border">
            {cart.items.map((item) => (
              <li
                key={`${item.productId}-${item.variantId ?? ""}`}
                className="flex items-center gap-4 p-3"
              >
                {item.coverImage ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={item.coverImage}
                    alt=""
                    className="size-16 rounded-md object-cover"
                  />
                ) : null}
                <div className="flex-1">
                  <Link href={`/products/${item.productId}`} className="font-medium">
                    {item.title}
                  </Link>
                  {item.variantName ? (
                    <p className="text-muted-foreground text-sm">{item.variantName}</p>
                  ) : null}
                  <p className="text-sm">
                    {pricing.format(item.priceCents)}
                  </p>
                </div>
                <CartLineActions
                  productId={item.productId}
                  variantId={item.variantId}
                  quantity={item.quantity}
                  labels={{
                    quantity: t("cart.quantity"),
                    remove: t("cart.remove"),
                    updating: t("cart.updating"),
                  }}
                />
                <p className="w-24 text-right font-medium">
                  {pricing.format(item.priceCents * item.quantity)}
                </p>
              </li>
            ))}
          </ul>

          <div className="ml-auto w-full max-w-sm space-y-2">
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">{t("cart.subtotal")}</span>
              <span>{pricing.format(cart.totalCents)}</span>
            </div>
            <div className="flex justify-between border-t pt-2 text-lg font-semibold">
              <span>{t("cart.total")}</span>
              <span>{pricing.format(cart.totalCents)}</span>
            </div>
            <Link
              href="/checkout"
              className="bg-primary text-primary-foreground block rounded-md px-4 py-2.5 text-center"
            >
              {t("cart.checkout")}
            </Link>
          </div>
        </div>
      )}
    </div>
  );
}
