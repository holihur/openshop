import { redirect } from "next/navigation";

import { apiGet } from "@/lib/api";
import { AccountNav } from "@/components/account-nav";
import { ProductCard } from "@/components/product-card";
import { translator } from "@/lib/i18n";
import { getPricing } from "@/lib/pricing";
import { getShopper } from "@/lib/session";
import { resolveLocale } from "@/app/layout";
import type { Product } from "@/lib/types";

/** Products the shopper saved for later. */
export default async function WishlistPage() {
  const shopper = await getShopper();
  if (!shopper.token) redirect("/login");

  const locale = await resolveLocale();
  const t = translator(locale);
  const [items, pricing] = await Promise.all([
    apiGet<Product[]>("/wishlist", { token: shopper.token }).catch(() => [] as Product[]),
    getPricing(locale),
  ]);

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t("account.wishlist")}</h1>
      <AccountNav locale={locale} current="/account/wishlist" />

      {items.length === 0 ? (
        <p className="text-muted-foreground rounded-lg border border-dashed py-16 text-center">
          {t("account.wishlistEmpty")}
        </p>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {items.map((product) => (
            <ProductCard key={product.id} product={product} locale={locale} format={pricing.format} />
          ))}
        </div>
      )}
    </div>
  );
}
