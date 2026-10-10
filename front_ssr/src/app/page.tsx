import Link from "next/link";
import { headers } from "next/headers";

import { apiList } from "@/lib/api";
import { ProductCard } from "@/components/product-card";
import { getSiteConfig } from "@/lib/session";
import { brandColor, contrastForeground, withAlpha } from "@/lib/theme";
import { localeFromHeader, translator } from "@/lib/i18n";
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
      () => ({ items: [] as Product[], total: 0, page: 1, pageSize: 8 }),
    ),
  ]);

  const brand = brandColor(site?.themeColor);
  const onBrand = contrastForeground(brand);
  const heroTitle = site?.hero?.title?.trim() || t("home.title");
  const heroSubtitle = site?.hero?.subtitle?.trim() || t("home.subtitle");
  // Only cards that have content are rendered: an empty one used to appear as a
  // blank box on a half-configured store.
  const features = (site?.features ?? []).filter((f) => f.title?.trim() || f.text?.trim());

  return (
    <div className="space-y-12">
      <section
        className="relative overflow-hidden rounded-2xl px-6 py-12 sm:px-12 sm:py-16"
        style={{
          background: `linear-gradient(135deg, ${withAlpha(brand, 0.14)}, ${withAlpha(brand, 0.04)})`,
          border: `1px solid ${withAlpha(brand, 0.18)}`,
        }}
      >
        {site?.hero?.image ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={site.hero.image}
            alt=""
            className="pointer-events-none absolute inset-0 size-full object-cover opacity-25"
          />
        ) : null}
        <div className="relative max-w-2xl">
          <h1 className="text-3xl font-bold tracking-tight sm:text-4xl">{heroTitle}</h1>
          {heroSubtitle ? (
            <p className="text-muted-foreground mt-3 text-lg">{heroSubtitle}</p>
          ) : null}
          <Link
            href={site?.hero?.ctaUrl || "/products"}
            className="mt-6 inline-block rounded-lg px-5 py-2.5 font-medium shadow-sm"
            style={{ background: brand, color: onBrand }}
          >
            {t("home.browse")}
          </Link>
        </div>
      </section>

      {features.length > 0 ? (
        <section className="grid gap-4 sm:grid-cols-3">
          {features.map((feature) => (
            <div key={feature.title || feature.text} className="card-surface p-4">
              <h2 className="font-semibold">{feature.title}</h2>
              {feature.text ? (
                <p className="text-muted-foreground mt-1 text-sm">{feature.text}</p>
              ) : null}
            </div>
          ))}
        </section>
      ) : null}

      {featured.items.length > 0 ? (
        <section className="space-y-4">
          <div className="flex items-baseline justify-between">
            <h2 className="text-xl font-semibold">{t("home.featured")}</h2>
            <Link href="/products" className="text-sm hover:opacity-70" style={{ color: brand }}>
              {t("home.allProducts")} →
            </Link>
          </div>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {featured.items.map((product) => (
              <ProductCard key={product.id} product={product} locale={locale} />
            ))}
          </div>
        </section>
      ) : null}
    </div>
  );
}
