import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { ArrowRight, PackageSearch, ShieldCheck, Zap } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ProductGrid } from "@/components/product-grid";
import { api } from "@/lib/api";
import type { Category, Product } from "@/lib/types";

export function HomePage() {
  const { data: categories } = useQuery({
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
        <Badge variant="secondary" className="mb-4">
          Horizontally scalable commerce
        </Badge>
        <h1 className="max-w-2xl text-4xl font-bold tracking-tight sm:text-5xl">
          Everything you need, delivered.
        </h1>
        <p className="text-muted-foreground mt-4 max-w-xl">
          A production-grade storefront built on Go, PostgreSQL, Redis and NATS — designed to
          scale across many instances with shared state, distributed locks and event-driven
          workflows.
        </p>
        <div className="mt-8 flex flex-wrap gap-3">
          <Button size="lg" asChild>
            <Link to="/products">
              Browse products
              <ArrowRight className="size-4" />
            </Link>
          </Button>
        </div>

        <div className="mt-12 grid gap-4 sm:grid-cols-3">
          {[
            { icon: Zap, title: "Fast checkout", text: "Atomic stock reservation with distributed locks." },
            { icon: ShieldCheck, title: "Secure by design", text: "JWT auth, bcrypt and provider-verified payments." },
            { icon: PackageSearch, title: "Event driven", text: "NATS queue groups keep workers exactly-once." },
          ].map(({ icon: Icon, title, text }) => (
            <div key={title} className="bg-background/60 rounded-lg border p-4 backdrop-blur">
              <Icon className="size-5" />
              <p className="mt-2 font-medium">{title}</p>
              <p className="text-muted-foreground text-sm">{text}</p>
            </div>
          ))}
        </div>
      </section>

      {categories && categories.length > 0 && (
        <section>
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-xl font-semibold">Shop by category</h2>
          </div>
          <div className="flex flex-wrap gap-2">
            {categories.map((category) => (
              <Button key={category.id} variant="outline" size="sm" asChild>
                <Link to={`/products?categoryId=${category.id}`}>{category.name}</Link>
              </Button>
            ))}
          </div>
        </section>
      )}

      <section>
        <div className="mb-6 flex items-center justify-between">
          <h2 className="text-xl font-semibold">Featured products</h2>
          <Button variant="link" asChild>
            <Link to="/products">
              View all
              <ArrowRight className="size-4" />
            </Link>
          </Button>
        </div>
        <ProductGrid products={data?.items} loading={isLoading} />
      </section>
    </div>
  );
}
