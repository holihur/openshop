import Link from "next/link";

import { localizedName } from "@/lib/format";
import type { Product } from "@/lib/types";
import type { Locale } from "@/lib/i18n";

/**
 * A server-rendered product tile. Everything a shopper needs to decide is in
 * the HTML: image, name, price and availability.
 */
export function ProductCard({
  product,
  locale,
  format,
}: {
  product: Product;
  locale: Locale;
  /** Formats a base-currency amount in the shopper's display currency. */
  format: (baseCents: number) => string;
}) {
  const outOfStock = product.stock <= 0;
  return (
    <Link href={`/products/${product.id}`} className="card-surface group block overflow-hidden">
      <div className="bg-muted relative aspect-square overflow-hidden">
        {product.coverImage ? (
          // Images are served by the API's storage adapter at their original
          // size; the tile crops them so the grid stays even.
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={product.coverImage}
            alt={localizedName(product, locale)}
            className="size-full object-cover transition-transform duration-300 group-hover:scale-[1.03]"
            loading="lazy"
          />
        ) : null}
        {outOfStock ? (
          <span className="absolute top-2 left-2 rounded-full bg-black/70 px-2 py-0.5 text-xs font-medium text-white">
            {locale === "zh" ? "已售罄" : "Sold out"}
          </span>
        ) : null}
      </div>
      <div className="space-y-1 p-3">
        <h3 className="line-clamp-2 min-h-10 text-sm font-medium">
          {localizedName(product, locale)}
        </h3>
        <div className="flex items-baseline justify-between gap-2">
          <p className="font-semibold">{format(product.priceCents)}</p>
          {product.reviewCount ? (
            <p className="text-muted-foreground text-xs" aria-label={`${product.rating ?? 0} / 5`}>
              ★ {product.rating?.toFixed(1)}
            </p>
          ) : null}
        </div>
      </div>
    </Link>
  );
}
