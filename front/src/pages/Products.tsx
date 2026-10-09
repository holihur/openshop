import { useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Search, SlidersHorizontal, X } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { Input } from "@lib/components/ui/input";
import { ProductGrid } from "@lib/components/product-grid";
import { Pagination } from "@lib/components/pagination";
import { api } from "@lib/api";
import { useI18n } from "@lib/i18n";
import { usePrice } from "@lib/hooks/usePrice";
import type { Category, Product, ProductFacets } from "@lib/types";
import { useSeo } from "@lib/hooks/useSeo";

export function ProductsPage() {
  const { t } = useI18n();
  useSeo({ title: t("products.title"), description: t("products.seoDesc") });
  const [params, setParams] = useSearchParams();
  const categoryId = params.get("categoryId") ?? "";
  const keyword = params.get("keyword") ?? "";
  const sort = params.get("sort") ?? (keyword ? "relevance" : "newest");
  const minPrice = params.get("minPrice") ?? "";
  const maxPrice = params.get("maxPrice") ?? "";
  const page = Number(params.get("page") ?? "1") || 1;
  const [search, setSearch] = useState(keyword);
  const [suggestOpen, setSuggestOpen] = useState(false);
  const [priceDraft, setPriceDraft] = useState({ min: minPrice, max: maxPrice });
  const [filtersOpen, setFiltersOpen] = useState(false);
  // Attribute filters are carried as attr.<name>=<value> query parameters.
  const activeAttrs = useMemo(() => {
    const out: Record<string, string> = {};
    for (const [key, value] of params.entries()) {
      if (key.startsWith("attr.") && value) out[key.slice(5)] = value;
    }
    return out;
  }, [params]);
  const navigate = useNavigate();
  const price = usePrice();

  useEffect(() => setSearch(keyword), [keyword]);
  useEffect(() => setPriceDraft({ min: minPrice, max: maxPrice }), [minPrice, maxPrice]);

  const sorts = [
    { value: "relevance", label: t("products.sortRelevance") },
    { value: "newest", label: t("products.sortNewest") },
    { value: "price_asc", label: t("products.sortPriceAsc") },
    { value: "price_desc", label: t("products.sortPriceDesc") },
  ];

  const { data: categories } = useQuery({
    queryKey: ["categories"],
    queryFn: () => api.get<Category[]>("/categories"),
  });

  // Which filters actually apply here: a price range and the variant
  // attributes that occur in this category.
  const { data: facets } = useQuery({
    queryKey: ["products", "facets", categoryId],
    queryFn: () => api.get<ProductFacets>(`/products/facets?categoryId=${encodeURIComponent(categoryId)}`),
    staleTime: 60_000,
  });

  // Typeahead suggestions for the search box.
  const { data: suggestions } = useQuery({
    queryKey: ["products", "suggest", search],
    queryFn: () => api.getPage<Product[]>(`/products?keyword=${encodeURIComponent(search)}&pageSize=6`),
    enabled: search.trim().length >= 2,
    staleTime: 30_000,
  });

  const queryString = useMemo(() => {
    const qs = new URLSearchParams();
    qs.set("page", String(page));
    qs.set("pageSize", "12");
    qs.set("sort", sort);
    if (categoryId) qs.set("categoryId", categoryId);
    if (keyword) qs.set("keyword", keyword);
    if (minPrice) qs.set("minPrice", minPrice);
    if (maxPrice) qs.set("maxPrice", maxPrice);
    for (const [name, value] of Object.entries(activeAttrs)) qs.set(`attr.${name}`, value);
    return qs.toString();
  }, [page, sort, categoryId, keyword, minPrice, maxPrice, activeAttrs]);

  const { data, isLoading } = useQuery({
    queryKey: ["products", queryString],
    queryFn: () => api.getPage<Product[]>(`/products?${queryString}`),
  });

  const update = (patch: Record<string, string | undefined>) => {
    const next = new URLSearchParams(params);
    for (const [key, value] of Object.entries(patch)) {
      if (value === undefined || value === "") next.delete(key);
      else next.set(key, value);
    }
    if (!("page" in patch)) next.delete("page");
    setParams(next);
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t("products.title")}</h1>
        <p className="text-muted-foreground">{t("products.subtitle")}</p>
      </div>

      <form
        className="flex flex-col gap-3 sm:flex-row"
        onSubmit={(e) => {
          e.preventDefault();
          update({ keyword: search });
        }}
      >
        <div className="relative flex-1">
          <Search className="text-muted-foreground absolute top-1/2 left-3 size-4 -translate-y-1/2" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            onFocus={() => setSuggestOpen(true)}
            onBlur={() => window.setTimeout(() => setSuggestOpen(false), 150)}
            placeholder={t("products.searchPlaceholder")}
            className="pl-9"
          />
          {suggestOpen && search.trim().length >= 2 && suggestions && suggestions.items.length > 0 && (
            <ul className="bg-popover absolute z-20 mt-1 w-full overflow-hidden rounded-md border shadow-md">
              {suggestions.items.map((p) => (
                <li key={p.id}>
                  <button
                    type="button"
                    onMouseDown={(e) => {
                      e.preventDefault();
                      navigate(`/products/${p.id}`);
                    }}
                    className="hover:bg-accent flex w-full items-center gap-2 px-3 py-2 text-left text-sm"
                  >
                    {p.coverImage ? (
                      <img src={p.coverImage} alt="" className="size-8 rounded object-cover" />
                    ) : null}
                    <span className="flex-1 truncate">{p.title}</span>
                    <span className="text-muted-foreground">{price(p.priceCents)}</span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
        <select
          value={categoryId}
          onChange={(e) => update({ categoryId: e.target.value })}
          className="border-input bg-background h-9 rounded-md border px-3 text-sm"
        >
          <option value="">{t("products.allCategories")}</option>
          {categories?.map((c) => (
            <option key={c.id} value={c.id}>
              {c.name}
            </option>
          ))}
        </select>
        <select
          value={sort}
          onChange={(e) => update({ sort: e.target.value })}
          className="border-input bg-background h-9 rounded-md border px-3 text-sm"
        >
          {sorts.map((s) => (
            <option key={s.value} value={s.value}>
              {s.label}
            </option>
          ))}
        </select>
        <Button type="submit">{t("common.search")}</Button>
        <Button
          type="button"
          variant="outline"
          onClick={() => setFiltersOpen((open) => !open)}
          aria-expanded={filtersOpen}
        >
          <SlidersHorizontal className="mr-2 size-4" />
          {t("products.filters")}
        </Button>
      </form>

      {filtersOpen && (
        <div className="bg-muted/40 grid gap-4 rounded-lg border p-4 sm:grid-cols-2 lg:grid-cols-3">
          <div className="space-y-2">
            <p className="text-sm font-medium">{t("products.priceRange")}</p>
            <div className="flex items-center gap-2">
              <Input
                type="number"
                min="0"
                step="0.01"
                value={priceDraft.min}
                onChange={(e) => setPriceDraft((d) => ({ ...d, min: e.target.value }))}
                placeholder={facets ? (facets.minPriceCents / 100).toFixed(2) : t("products.min")}
                aria-label={t("products.min")}
              />
              <span className="text-muted-foreground">–</span>
              <Input
                type="number"
                min="0"
                step="0.01"
                value={priceDraft.max}
                onChange={(e) => setPriceDraft((d) => ({ ...d, max: e.target.value }))}
                placeholder={facets ? (facets.maxPriceCents / 100).toFixed(2) : t("products.max")}
                aria-label={t("products.max")}
              />
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() =>
                  update({ minPrice: priceDraft.min, maxPrice: priceDraft.max })
                }
              >
                {t("common.apply")}
              </Button>
            </div>
          </div>
          {facets?.attributes.map((facet) => (
            <div key={facet.name} className="space-y-2">
              <p className="text-sm font-medium capitalize">{facet.name}</p>
              <div className="flex flex-wrap gap-2">
                {facet.values.map((value) => {
                  const selected = activeAttrs[facet.name] === value;
                  return (
                    <button
                      key={value}
                      type="button"
                      aria-pressed={selected}
                      onClick={() =>
                        update({
                          [`attr.${facet.name}`]: selected ? "" : value,
                        })
                      }
                      className={
                        selected
                          ? "bg-primary text-primary-foreground rounded-full px-3 py-1 text-sm"
                          : "bg-background rounded-full border px-3 py-1 text-sm"
                      }
                    >
                      {value}
                    </button>
                  );
                })}
              </div>
            </div>
          ))}
        </div>
      )}

      {(keyword || categoryId) && (
        <div className="flex flex-wrap items-center gap-2">
          {keyword && (
            <button
              type="button"
              onClick={() => update({ keyword: "" })}
              className="bg-accent text-accent-foreground inline-flex items-center gap-1 rounded-full px-3 py-1 text-sm"
            >
              {keyword}
              <X className="size-3" />
            </button>
          )}
          {categoryId && (
            <button
              type="button"
              onClick={() => update({ categoryId: "" })}
              className="bg-accent text-accent-foreground inline-flex items-center gap-1 rounded-full px-3 py-1 text-sm"
            >
              {categories?.find((c) => c.id === categoryId)?.name ?? t("products.allCategories")}
              <X className="size-3" />
            </button>
          )}
          {(minPrice || maxPrice) && (
            <button
              type="button"
              onClick={() => update({ minPrice: "", maxPrice: "" })}
              className="bg-accent text-accent-foreground inline-flex items-center gap-1 rounded-full px-3 py-1 text-sm"
            >
              {t("products.priceRangeChip", {
                min: minPrice || "0",
                max: maxPrice || t("products.max"),
              })}
              <X className="size-3" />
            </button>
          )}
          {Object.entries(activeAttrs).map(([name, value]) => (
            <button
              key={name}
              type="button"
              onClick={() => update({ [`attr.${name}`]: "" })}
              className="bg-accent text-accent-foreground inline-flex items-center gap-1 rounded-full px-3 py-1 text-sm"
            >
              {name}: {value}
              <X className="size-3" />
            </button>
          ))}
        </div>
      )}

      {!isLoading && data && (
        <div className="space-y-1">
          <p className="text-muted-foreground text-sm">
            {t("products.resultCount", { count: data.total })}
          </p>
          {data.fuzzy && (
            <p className="text-sm" role="status">
              {t("products.fuzzyMatch", { keyword })}
            </p>
          )}
        </div>
      )}

      {!isLoading && data && data.items.length === 0 && (keyword || categoryId) ? (
        <div className="rounded-lg border border-dashed py-16 text-center">
          <p className="text-muted-foreground">{t("products.empty")}</p>
          <Button variant="link" onClick={() => setParams(new URLSearchParams())}>
            {t("products.clearFilters")}
          </Button>
        </div>
      ) : (
        <ProductGrid products={data?.items} loading={isLoading} emptyMessage={t("products.empty")} />
      )}

      {data && (
        <Pagination
          page={data.page}
          pageSize={data.pageSize}
          total={data.total}
          onChange={(next) => update({ page: String(next) })}
        />
      )}
    </div>
  );
}
