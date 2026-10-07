import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { ArrowRight, PackageSearch, ShieldCheck, Zap } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { Badge } from "@lib/components/ui/badge";
import { Skeleton } from "@lib/components/ui/skeleton";
import { ProductGrid } from "@lib/components/product-grid";
import { api } from "@lib/api";
import { useI18n } from "@lib/i18n";
import type { Category, Product } from "@lib/types";
import { useSeo } from "@lib/hooks/useSeo";
import { useSite } from "@lib/hooks/useSite";

export function HomePage() {
  const { t } = useI18n();
  useSeo({ description: t("home.seoDesc") });
  const { data: site } = useSite();
  const heroTitle = site?.hero.title?.trim() || t("home.title");
  const heroSubtitle = site?.hero.subtitle?.trim() || t("home.subtitle");
  const heroCta = site?.hero.ctaUrl?.trim() || "/products";
  const { data: categories, isLoading: categoriesLoading } = useQuery({
    queryKey: ["categories"],
    queryFn: () => api.get<Category[]>("/categories"),
  });

  const { data, isLoading } = useQuery({
    queryKey: ["products", "featured"],
    queryFn: () => api.getPage<Product[]>("/products?pageSize=8"),
  });

  return (
    <div className="space-y-14">
      <section className="from-primary/10 via-background to-background relative overflow-hidden rounded-2xl border bg-gradient-to-br px-6 py-16 sm:px-12">
        {site?.hero.image && (
          <img
            src={site.hero.image}
            alt=""
            aria-hidden
            className="absolute inset-0 size-full object-cover opacity-20"
          />
        )}
        <div className="relative">
          <Badge variant="secondary" className="mb-4">
            {t("home.badge")}
          </Badge>
          <h1 className="max-w-2xl text-4xl font-bold tracking-tight sm:text-5xl">
            {heroTitle}
          </h1>
          <p className="text-muted-foreground mt-4 max-w-xl">{heroSubtitle}</p>
          <div className="mt-8 flex flex-wrap gap-3">
            <Button size="lg" asChild>
              <Link to={heroCta}>
                {t("home.browse")}
                <ArrowRight className="size-4" />
              </Link>
            </Button>
          </div>
        </div>

        <div className="relative mt-12 grid gap-4 sm:grid-cols-3">
          {[
            {
              icon: Zap,
              title: site?.features?.[0]?.title?.trim() || t("home.fastCheckout"),
              text: site?.features?.[0]?.text?.trim() || t("home.fastCheckoutText"),
            },
            {
              icon: ShieldCheck,
              title: site?.features?.[1]?.title?.trim() || t("home.secure"),
              text: site?.features?.[1]?.text?.trim() || t("home.secureText"),
            },
            {
              icon: PackageSearch,
              title: site?.features?.[2]?.title?.trim() || t("home.eventDriven"),
              text: site?.features?.[2]?.text?.trim() || t("home.eventDrivenText"),
            },
          ].map(({ icon: Icon, title, text }) => (
            <div key={title} className="bg-background/60 rounded-lg border p-4 backdrop-blur">
              <Icon className="size-5" />
              <p className="mt-2 font-medium">{title}</p>
              <p className="text-muted-foreground text-sm">{text}</p>
            </div>
          ))}
        </div>
      </section>

      {categoriesLoading ? (
        <section>
          <Skeleton className="mb-4 h-6 w-40" />
          <div className="flex flex-wrap gap-2">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-9 w-24" />
            ))}
          </div>
        </section>
      ) : categories && categories.length > 0 ? (
        <section>
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-xl font-semibold">{t("home.shopByCategory")}</h2>
          </div>
          <div className="flex flex-wrap gap-2">
            {categories.map((category) => (
              <Button key={category.id} variant="outline" size="sm" asChild>
                <Link to={`/products?categoryId=${category.id}`}>{category.name}</Link>
              </Button>
            ))}
          </div>
        </section>
      ) : null}

      <section>
        <div className="mb-6 flex items-center justify-between">
          <h2 className="text-xl font-semibold">{t("home.featured")}</h2>
          <Button variant="link" asChild>
            <Link to="/products">
              {t("home.viewAll")}
              <ArrowRight className="size-4" />
            </Link>
          </Button>
        </div>
        <ProductGrid products={data?.items} loading={isLoading} />
      </section>
    </div>
  );
}
