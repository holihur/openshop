import Link from "next/link";

import { formatMoney, localizedName } from "@/lib/format";
import type { Product } from "@/lib/types";
import type { Locale } from "@/lib/i18n";

/** A server-rendered product tile. */
export function ProductCard({ product, locale }: { product: Product; locale: Locale }) {
  return (
    <Link href={`/products/${product.id}`} className="group block rounded-lg border p-3">
      <div className="bg-muted aspect-square overflow-hidden rounded-md">
        {product.coverImage ? (
          // Images are served by the API's storage adapter.
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={product.coverImage}
            alt={localizedName(product, locale)}
            className="size-full object-cover transition-transform group-hover:scale-105"
            loading="lazy"
          />
        ) : null}
      </div>
      <h3 className="mt-3 line-clamp-2 text-sm font-medium">{localizedName(product, locale)}</h3>
      <p className="mt-1 font-semibold">{formatMoney(product.priceCents, product.currency, locale)}</p>
    </Link>
  );
}
