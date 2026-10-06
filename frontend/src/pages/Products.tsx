import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Search } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ProductGrid } from "@/components/product-grid";
import { Pagination } from "@/components/pagination";
import { api } from "@/lib/api";
import type { Category, Product } from "@/lib/types";

const SORTS = [
  { value: "newest", label: "Newest" },
  { value: "price_asc", label: "Price: low to high" },
  { value: "price_desc", label: "Price: high to low" },
];

export function ProductsPage() {
  const [params, setParams] = useSearchParams();
  const categoryId = params.get("categoryId") ?? "";
  const keyword = params.get("keyword") ?? "";
  const sort = params.get("sort") ?? "newest";
  const page = Number(params.get("page") ?? "1") || 1;
  const [search, setSearch] = useState(keyword);

  useEffect(() => setSearch(keyword), [keyword]);

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
        <h1 className="text-2xl font-bold">Products</h1>
        <p className="text-muted-foreground">Browse our catalog.</p>
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
            placeholder="Search products…"
            className="pl-9"
          />
        </div>
        <select
          value={categoryId}
          onChange={(e) => update({ categoryId: e.target.value })}
          className="border-input bg-background h-9 rounded-md border px-3 text-sm"
        >
          <option value="">All categories</option>
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
          {SORTS.map((s) => (
            <option key={s.value} value={s.value}>
              {s.label}
            </option>
          ))}
        </select>
        <Button type="submit">Search</Button>
      </form>

      <ProductGrid
        products={data?.items}
        loading={isLoading}
        emptyMessage="No products match your filters."
      />

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
