import Link from "next/link";

import { apiGet, apiList, type Page } from "@/lib/api";
import { ProductCard } from "@/components/product-card";
import { translateCategoryName } from "@/lib/format";
import { translator } from "@/lib/i18n";
import { getPricing } from "@/lib/pricing";
import { resolveLocale } from "@/app/layout";
import type { Category, Product, ProductFacets } from "@/lib/types";

interface SearchParams {
  keyword?: string;
  categoryId?: string;
  sort?: string;
  minPrice?: string;
  maxPrice?: string;
  page?: string;
  [key: string]: string | undefined;
}

/**
 * The catalog is server rendered. The filters live in the URL, so a filtered
 * page is linkable, shareable and cached by the browser without any client
 * state — the same URL always produces the same HTML.
 */
export default async function ProductsPage({
  searchParams,
}: {
  searchParams: Promise<SearchParams>;
}) {
  const params = await searchParams;
  const locale = await resolveLocale();
  const t = translator(locale);

  const page = Math.max(1, Number.parseInt(params.page ?? "1", 10) || 1);
  const keyword = params.keyword ?? "";
  const categoryId = params.categoryId ?? "";
  const sort = params.sort ?? (keyword ? "relevance" : "newest");

  // Attribute filters arrive as attr.<name>=<value>.
  const attributes = Object.entries(params).filter(([key]) => key.startsWith("attr."));

  const query = new URLSearchParams({ page: String(page), pageSize: "12", sort });
  if (keyword) query.set("keyword", keyword);
  if (categoryId) query.set("categoryId", categoryId);
  if (params.minPrice) query.set("minPrice", params.minPrice);
  if (params.maxPrice) query.set("maxPrice", params.maxPrice);
  for (const [key, value] of attributes) if (value) query.set(key, value);

  const [products, categories, facets, pricing] = await Promise.all([
    apiList<Product>(`/products?${query.toString()}`, { revalidate: 30, tags: ["catalog"] }).catch(
      () => ({ items: [] as Product[], total: 0, page: 1, pageSize: 12, fuzzy: undefined }) as Page<Product>,
    ),
    apiGet<Category[]>("/categories", { revalidate: 60, tags: ["catalog"] }).catch(() => []),
    apiGet<ProductFacets>(`/products/facets?categoryId=${encodeURIComponent(categoryId)}`, {
      revalidate: 60,
      tags: ["catalog"],
    }).catch(() => ({ minPriceCents: 0, maxPriceCents: 0, attributes: [] })),
    getPricing(locale),
  ]);

  const totalPages = Math.max(1, Math.ceil(products.total / products.pageSize));
  const pageHref = (next: number) => {
    const copy = new URLSearchParams(query);
    copy.set("page", String(next));
    return `/products?${copy.toString()}`;
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t("products.title")}</h1>

      <form method="get" action="/products" className="flex flex-col gap-3 sm:flex-row">
        <input
          type="search"
          name="keyword"
          defaultValue={keyword}
          placeholder={t("products.search")}
          aria-label={t("products.search")}
          className="border-input h-9 flex-1 rounded-md border px-3"
        />
        <select
          name="categoryId"
          defaultValue={categoryId}
          aria-label={t("products.allCategories")}
          className="border-input h-9 rounded-md border px-3"
        >
          <option value="">{t("products.allCategories")}</option>
          {categories.map((category) => (
            <option key={category.id} value={category.id}>
              {translateCategoryName(category, locale)}
            </option>
          ))}
        </select>
        <select
          name="sort"
          defaultValue={sort}
          aria-label={t("products.sort")}
          className="border-input h-9 rounded-md border px-3"
        >
          <option value="relevance">{t("products.sortRelevance")}</option>
          <option value="newest">{t("products.sortNewest")}</option>
          <option value="price_asc">{t("products.sortPriceAsc")}</option>
          <option value="price_desc">{t("products.sortPriceDesc")}</option>
        </select>
        <input
          type="number"
          name="minPrice"
          step="0.01"
          min="0"
          defaultValue={params.minPrice}
          placeholder={String(facets.minPriceCents / 100)}
          aria-label={t("products.priceRange")}
          className="border-input h-9 w-28 rounded-md border px-3"
        />
        <input
          type="number"
          name="maxPrice"
          step="0.01"
          min="0"
          defaultValue={params.maxPrice}
          placeholder={String(facets.maxPriceCents / 100)}
          aria-label={t("products.priceRange")}
          className="border-input h-9 w-28 rounded-md border px-3"
        />
        <button type="submit" className="bg-primary text-primary-foreground h-9 rounded-md px-4">
          {t("products.apply")}
        </button>
      </form>

      {facets.attributes.length > 0 ? (
        <div className="flex flex-wrap gap-2">
          {facets.attributes.flatMap((facet) =>
            facet.values.map((value) => {
              const name = `attr.${facet.name}`;
              const selected = params[name] === value;
              const next = new URLSearchParams(query);
              if (selected) next.delete(name);
              else next.set(name, value);
              next.delete("page");
              return (
                <Link
                  key={`${name}-${value}`}
                  href={`/products?${next.toString()}`}
                  aria-pressed={selected}
                  className={
                    selected
                      ? "bg-primary text-primary-foreground rounded-full px-3 py-1 text-sm"
                      : "rounded-full border px-3 py-1 text-sm"
                  }
                >
                  {facet.name}: {value}
                </Link>
              );
            }),
          )}
        </div>
      ) : null}

      <p className="text-muted-foreground text-sm">
        {t("products.resultCount", { count: products.total })}
      </p>
      {products.fuzzy ? <p className="text-sm">{t("products.fuzzy")}</p> : null}

      {products.items.length === 0 ? (
        <p className="text-muted-foreground rounded-lg border border-dashed py-16 text-center">
          {t("products.empty")}
        </p>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {products.items.map((product) => (
            <ProductCard key={product.id} product={product} locale={locale} format={pricing.format} />
          ))}
        </div>
      )}

      {totalPages > 1 ? (
        <nav className="flex items-center justify-center gap-2" aria-label="Pagination">
          {page > 1 ? (
            <Link href={pageHref(page - 1)} className="rounded-md border px-3 py-1.5 text-sm">
              ‹
            </Link>
          ) : null}
          <span className="text-sm">
            {page} / {totalPages}
          </span>
          {page < totalPages ? (
            <Link href={pageHref(page + 1)} className="rounded-md border px-3 py-1.5 text-sm">
              ›
            </Link>
          ) : null}
        </nav>
      ) : null}
    </div>
  );
}
