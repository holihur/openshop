import type { Metadata } from "next";
import { notFound } from "next/navigation";

import Link from "next/link";

import { apiGet, apiList, type Page } from "@/lib/api";
import { AddToCartButton } from "@/components/add-to-cart";
import { ReviewForm } from "@/components/review-form";
import { formatMoney, localizedName } from "@/lib/format";
import { translator } from "@/lib/i18n";
import { getPricing } from "@/lib/pricing";
import { resolveLocale } from "@/app/layout";
import { getShopper, isSignedIn } from "@/lib/session";
import { WishlistButton } from "@/components/wishlist-button";
import type { DeliveryEstimate, Product, Review } from "@/lib/types";

interface Props {
  params: Promise<{ id: string }>;
  searchParams: Promise<{ reviewPage?: string }>;
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

export default async function ProductPage({ params, searchParams }: Props) {
  const { id } = await params;
  const product = await loadProduct(id);
  if (!product) notFound();

  const locale = await resolveLocale();
  const t = translator(locale);
  const reviewPage = Math.max(1, Number.parseInt((await searchParams).reviewPage ?? "1", 10) || 1);
  const shopper = await getShopper();
  const [reviews, signedIn, saved] = await Promise.all([
    apiList<Review>(`/products/${id}/reviews?page=${reviewPage}&pageSize=5`, {
      revalidate: 30,
      tags: ["reviews", `reviews:${id}`],
    })
      .catch(() => ({ items: [] as Review[], total: 0, page: 1, pageSize: 5 }) as Page<Review>),
    isSignedIn(),
    shopper.token
      ? apiGet<Product[]>("/wishlist", { token: shopper.token })
          .then((items) => items.some((item) => item.id === id))
          .catch(() => false)
      : Promise.resolve(false),
  ]);
  const reviewPages = Math.max(1, Math.ceil(reviews.total / reviews.pageSize));
  // Prices are shown in the shopper's chosen currency; settlement stays in the
  // store's base currency.
  const pricing = await getPricing(locale);
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
          <p className="text-2xl font-semibold">{pricing.format(product.priceCents)}</p>
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
                      amount: pricing.format(estimate.freeRemainingCents),
                    })
                  : t("delivery.freeReached")}
              </p>
            </div>
          ) : null}

          <div className="flex items-center gap-2">
            <WishlistButton productId={product.id} saved={saved} locale={locale} />
          </div>

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

      <section className="space-y-4">
        <h2 className="text-xl font-semibold">
          {reviews.total === 1
            ? t("product.reviewOne")
            : t("product.reviews", { count: reviews.total })}
        </h2>

        {signedIn ? (
          <ReviewForm
            productId={product.id}
            labels={{
              title: t("review.title"),
              body: t("review.body"),
              rating: t("review.rating"),
              submit: t("review.submit"),
              sending: t("review.sending"),
              failed: t("error.generic"),
              thanks: t("review.thanks"),
            }}
          />
        ) : (
          <p className="text-muted-foreground text-sm">
            <Link href="/login" className="underline">
              {t("nav.signIn")}
            </Link>{" "}
            {t("review.signInHint")}
          </p>
        )}

        {reviews.items.length === 0 ? (
          <p className="text-muted-foreground text-sm">{t("review.empty")}</p>
        ) : (
          <ul className="space-y-3">
            {reviews.items.map((review) => (
              <li key={review.id} className="rounded-lg border p-4">
                <div className="flex items-center gap-2">
                  {/* role="img" makes the aria-label valid on an otherwise
                      generic element (axe: aria-prohibited-attr). */}
                  <span role="img" aria-label={`${review.rating} / 5`}>
                    {"★".repeat(review.rating)}
                    {"☆".repeat(Math.max(0, 5 - review.rating))}
                  </span>
                  <span className="font-medium">{review.title}</span>
                  {review.verifiedPurchase ? (
                    <span className="bg-muted rounded-full px-2 text-xs">
                      {t("review.verified")}
                    </span>
                  ) : null}
                </div>
                <p className="text-muted-foreground mt-1 text-sm whitespace-pre-line">
                  {review.body}
                </p>
              </li>
            ))}
          </ul>
        )}

        {reviewPages > 1 ? (
          <nav className="flex items-center justify-center gap-3 text-sm" aria-label="Reviews">
            {reviewPage > 1 ? (
              <Link href={`/products/${id}?reviewPage=${reviewPage - 1}`} className="rounded-md border px-3 py-1">
                ‹
              </Link>
            ) : null}
            <span>
              {reviewPage} / {reviewPages}
            </span>
            {reviewPage < reviewPages ? (
              <Link href={`/products/${id}?reviewPage=${reviewPage + 1}`} className="rounded-md border px-3 py-1">
                ›
              </Link>
            ) : null}
          </nav>
        ) : null}
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
