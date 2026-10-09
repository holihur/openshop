import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { apiGet } from "@/lib/api";
import { AddToCartButton } from "@/components/add-to-cart";
import { formatMoney, localizedName } from "@/lib/format";
import { translator } from "@/lib/i18n";
import { resolveLocale } from "@/app/layout";
import type { DeliveryEstimate, Product } from "@/lib/types";

interface Props {
  params: Promise<{ id: string }>;
}

async function loadProduct(id: string): Promise<Product | null> {
  try {
    return await apiGet<Product>(`/products/${id}`, { revalidate: 30, tags: ["catalog"] });
  } catch {
    return null;
  }
}

/**
 * Product metadata is generated on the server, so crawlers and social previews
 * see the real title, description and price without executing JavaScript.
 */
export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { id } = await params;
  const product = await loadProduct(id);
  if (!product) return { title: "Not found" };
  return {
    title: product.title,
    description: product.description.slice(0, 160),
    openGraph: {
      title: product.title,
      description: product.description.slice(0, 160),
      images: product.coverImage ? [product.coverImage] : [],
      type: "website",
    },
  };
}

export default async function ProductPage({ params }: Props) {
  const { id } = await params;
  const product = await loadProduct(id);
  if (!product) notFound();

  const locale = await resolveLocale();
  const t = translator(locale);
  const title = localizedName(product, locale);
  const outOfStock = product.stock <= 0;

  // Delivery expectation for a single unit, resolved by the API so it matches
  // exactly what checkout will charge.
  const estimate = await apiGet<DeliveryEstimate[]>(
    `/delivery-estimates?subtotalCents=${product.priceCents}&weightGrams=${product.weightGrams || 0}`,
    { revalidate: 60, tags: ["shipping"] },
  )
    .then((list) => list[0])
    .catch(() => undefined);

  const jsonLd = {
    "@context": "https://schema.org",
    "@type": "Product",
    name: title,
    description: product.description,
    image: product.coverImage ? [product.coverImage] : undefined,
    sku: product.variants?.[0]?.sku,
    offers: {
      "@type": "Offer",
      price: (product.priceCents / 100).toFixed(2),
      priceCurrency: product.currency,
      availability: outOfStock
        ? "https://schema.org/OutOfStock"
        : "https://schema.org/InStock",
    },
    aggregateRating:
      product.reviewCount && product.rating
        ? {
            "@type": "AggregateRating",
            ratingValue: product.rating.toFixed(1),
            reviewCount: product.reviewCount,
          }
        : undefined,
  };

  return (
    <div className="space-y-8">
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
      />
      <div className="grid gap-8 lg:grid-cols-2">
        <div className="space-y-2">
          <div className="bg-muted aspect-square overflow-hidden rounded-lg">
            {product.coverImage ? (
              // eslint-disable-next-line @next/next/no-img-element
              <img src={product.coverImage} alt={title} className="size-full object-cover" />
            ) : null}
          </div>
          {product.images.length > 1 ? (
            <div className="flex gap-2 overflow-x-auto">
              {product.images.map((image) => (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  key={image}
                  src={image}
                  alt=""
                  className="size-16 rounded-md object-cover"
                  loading="lazy"
                />
              ))}
            </div>
          ) : null}
        </div>

        <div className="space-y-4">
          <h1 className="text-2xl font-bold">{title}</h1>
          {product.reviewCount ? (
            <p className="text-muted-foreground text-sm">
              ★ {product.rating?.toFixed(1)} · {t("product.reviews", { count: product.reviewCount })}
            </p>
          ) : null}
          <p className="text-2xl font-semibold">
            {formatMoney(product.priceCents, product.currency, locale)}
          </p>
          <p className={outOfStock ? "text-sm text-red-600" : "text-muted-foreground text-sm"}>
            {outOfStock ? t("product.outOfStock") : t("product.inStock", { count: product.stock })}
          </p>

          {estimate ? (
            <div className="space-y-2 rounded-lg border border-dashed p-3 text-sm">
              <p>
                {t("delivery.arrives", {
                  earliest: new Date(estimate.earliest).toISOString().slice(0, 10),
                  latest: new Date(estimate.latest).toISOString().slice(0, 10),
                })}
              </p>
              <p className="text-muted-foreground">
                {estimate.freeRemainingCents > 0
                  ? t("delivery.freeRemaining", {
                      amount: formatMoney(estimate.freeRemainingCents, product.currency, locale),
                    })
                  : t("delivery.freeReached")}
              </p>
            </div>
          ) : null}

          <AddToCartButton
            productId={product.id}
            variants={(product.variants ?? []).map((variant) => ({
              id: variant.id,
              name: variant.name,
              priceCents: variant.priceCents || product.priceCents,
              stock: variant.stock,
            }))}
            disabled={outOfStock}
            labels={{
              add: t("product.addToCart"),
              adding: t("product.adding"),
              quantity: t("cart.quantity"),
              variant: "Variant",
              failed: t("error.generic"),
            }}
          />
        </div>
      </div>

      <section className="space-y-2">
        <h2 className="text-xl font-semibold">{t("product.description")}</h2>
        <p className="text-muted-foreground whitespace-pre-line">{product.description}</p>
      </section>

      {product.faqs?.length ? (
        <section className="space-y-2">
          <h2 className="text-xl font-semibold">{t("product.faq")}</h2>
          <div className="divide-y rounded-lg border">
            {product.faqs.map((faq) => (
              <details key={faq.id} className="px-4 py-3">
                <summary className="cursor-pointer font-medium">{faq.question}</summary>
                <p className="text-muted-foreground mt-2 text-sm whitespace-pre-line">
                  {faq.answer}
                </p>
              </details>
            ))}
          </div>
        </section>
      ) : null}
    </div>
  );
}
