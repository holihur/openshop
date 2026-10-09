import Link from "next/link";

import { apiList } from "@/lib/api";
import { ProductCard } from "@/components/product-card";
import { getSiteConfig } from "@/lib/session";
import { localeFromHeader, translator } from "@/lib/i18n";
import { headers } from "next/headers";
import type { Product } from "@/lib/types";

/**
 * The home page is rendered on the server: the hero copy comes from the ops
 * configuration and the featured products are fetched once per minute. The
 * shopper receives finished HTML, so the first paint needs no JavaScript.
 */
export default async function HomePage() {
  const locale = localeFromHeader((await headers()).get("accept-language"));
  const t = translator(locale);
  const [site, featured] = await Promise.all([
    getSiteConfig().catch(() => null),
    apiList<Product>("/products?pageSize=8&sort=newest", { revalidate: 60, tags: ["catalog"] }).catch(
      () => ({ items: [], total: 0, page: 1, pageSize: 8 }),
    ),
  ]);

  return (
    <div className="space-y-10">
      <section className="bg-muted rounded-lg p-8 sm:p-12">
        <h1 className="text-3xl font-bold sm:text-4xl">
          {site?.hero?.title || site?.tagline || "openshop"}
        </h1>
        <p className="text-muted-foreground mt-3 max-w-2xl">{site?.hero?.subtitle}</p>
        <Link
          href={site?.hero?.ctaUrl || "/products"}
          className="bg-primary text-primary-foreground mt-6 inline-block rounded-md px-5 py-2.5 font-medium"
        >
          {t("home.browse")}
        </Link>
      </section>

      {site?.features?.length ? (
        <section className="grid gap-4 sm:grid-cols-3">
          {site.features.map((feature) => (
            <div key={feature.title} className="rounded-lg border p-4">
              <h2 className="font-semibold">{feature.title}</h2>
              <p className="text-muted-foreground mt-1 text-sm">{feature.text}</p>
            </div>
          ))}
        </section>
      ) : null}

      <section className="space-y-4">
        <h2 className="text-xl font-semibold">{t("home.featured")}</h2>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {featured.items.map((product) => (
            <ProductCard key={product.id} product={product} locale={locale} />
          ))}
        </div>
      </section>
    </div>
  );
}
