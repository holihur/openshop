import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Search, X } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { Input } from "@lib/components/ui/input";
import { ProductGrid } from "@lib/components/product-grid";
import { Pagination } from "@lib/components/pagination";
import { api } from "@lib/api";
import { useI18n } from "@lib/i18n";
import type { Category, Product } from "@lib/types";
import { useSeo } from "@lib/hooks/useSeo";

export function ProductsPage() {
  const { t } = useI18n();
  useSeo({ title: t("products.title"), description: t("products.seoDesc") });
  const [params, setParams] = useSearchParams();
  const categoryId = params.get("categoryId") ?? "";
  const keyword = params.get("keyword") ?? "";
  const sort = params.get("sort") ?? "newest";
  const page = Number(params.get("page") ?? "1") || 1;
  const [search, setSearch] = useState(keyword);

  useEffect(() => setSearch(keyword), [keyword]);

  const sorts = [
    { value: "newest", label: t("products.sortNewest") },
    { value: "price_asc", label: t("products.sortPriceAsc") },
    { value: "price_desc", label: t("products.sortPriceDesc") },
  ];

  const { data: categories } = useQuery({
    queryKey: ["categories"],
    queryFn: () => api.get<Category[]>("/categories"),
  });

  const queryString = useMemo(() => {
    const qs = new URLSearchParams();
    qs.set("page", String(page));
    qs.set("pageSize", "12");
    qs.set("sort", sort);
    if (categoryId) qs.set("categoryId", categoryId);
    if (keyword) qs.set("keyword", keyword);
    return qs.toString();
  }, [page, sort, categoryId, keyword]);

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
            placeholder={t("products.searchPlaceholder")}
            className="pl-9"
          />
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
      </form>

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
        </div>
      )}

      {!isLoading && data && (
        <p className="text-muted-foreground text-sm">
          {t("products.resultCount", { count: data.total })}
        </p>
      )}

      {!isLoading && data && data.items.length === 0 && (keyword || categoryId) ? (
        <div className="rounded-lg border border-dashed py-16 text-center">
          <p className="text-muted-foreground">{t("products.empty")}</p>
          <Button variant="link" onClick={() => update({ keyword: "", categoryId: "" })}>
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
