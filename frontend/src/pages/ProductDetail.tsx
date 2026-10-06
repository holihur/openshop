import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft, ImageOff, Minus, Plus, ShoppingCart } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { ReviewsSection } from "@/components/reviews-section";
import { api } from "@/lib/api";
import { formatMoney } from "@/lib/format";
import { useAddToCart } from "@/hooks/useCart";
import { useAuth } from "@/lib/auth";
import type { Product } from "@/lib/types";

export function ProductDetailPage() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const [quantity, setQuantity] = useState(1);
  const [activeImage, setActiveImage] = useState<string | null>(null);
  const { user } = useAuth();
  const addToCart = useAddToCart();

  const { data: product, isLoading, isError } = useQuery({
    queryKey: ["product", id],
    queryFn: () => api.get<Product>(`/products/${id}`),
    enabled: Boolean(id),
  });

  if (isLoading) {
    return (
      <div className="grid gap-8 md:grid-cols-2">
        <Skeleton className="aspect-square w-full" />
        <div className="space-y-4">
          <Skeleton className="h-8 w-2/3" />
          <Skeleton className="h-6 w-1/3" />
          <Skeleton className="h-24 w-full" />
        </div>
      </div>
    );
  }

  if (isError || !product) {
    return (
      <div className="py-16 text-center">
        <p className="text-muted-foreground">Product not found.</p>
        <Button variant="link" asChild>
          <Link to="/products">Back to products</Link>
        </Button>
      </div>
    );
  }

  const gallery = [product.coverImage, ...(product.images ?? [])].filter(Boolean);
  const shown = activeImage ?? gallery[0];
  const outOfStock = product.stock <= 0;

  return (
    <div className="space-y-8">
      <Button variant="ghost" size="sm" asChild className="-ml-2">
        <Link to="/products">
          <ArrowLeft className="size-4" />
          Back
        </Link>
      </Button>

      <div className="grid gap-8 md:grid-cols-2">
        <div className="space-y-3">
          <div className="bg-muted aspect-square overflow-hidden rounded-xl border">
            {shown ? (
              <img src={shown} alt={product.title} className="size-full object-cover" />
            ) : (
              <div className="text-muted-foreground flex size-full items-center justify-center">
                <ImageOff className="size-10" />
              </div>
            )}
          </div>
          {gallery.length > 1 && (
            <div className="flex gap-2">
              {gallery.map((src) => (
                <button
                  key={src}
                  onClick={() => setActiveImage(src)}
                  className="bg-muted size-16 overflow-hidden rounded-md border"
                >
                  <img src={src} alt="" className="size-full object-cover" />
                </button>
              ))}
            </div>
          )}
        </div>

        <div className="space-y-5">
          <div>
            <h1 className="text-3xl font-bold">{product.title}</h1>
            <div className="mt-2 flex items-center gap-3">
              <span className="text-2xl font-semibold">
                {formatMoney(product.priceCents, product.currency)}
              </span>
              {outOfStock ? (
                <Badge variant="secondary">Out of stock</Badge>
              ) : (
                <Badge variant="success">{product.stock} in stock</Badge>
              )}
            </div>
            {product.reviewCount ? (
              <p className="text-muted-foreground mt-2 text-sm">
                ★ {product.rating?.toFixed(1)} · {product.reviewCount} review
                {product.reviewCount === 1 ? "" : "s"}
              </p>
            ) : null}
          </div>

          <Separator />

          <p className="text-muted-foreground whitespace-pre-line">{product.description}</p>

          <div className="flex items-center gap-4">
            <div className="flex items-center rounded-md border">
              <Button
                variant="ghost"
                size="icon"
                disabled={quantity <= 1}
                onClick={() => setQuantity((q) => Math.max(1, q - 1))}
              >
                <Minus className="size-4" />
              </Button>
              <span className="w-10 text-center text-sm">{quantity}</span>
              <Button
                variant="ghost"
                size="icon"
                disabled={quantity >= product.stock}
                onClick={() => setQuantity((q) => q + 1)}
              >
                <Plus className="size-4" />
              </Button>
            </div>

            <Button
              className="flex-1"
              size="lg"
              disabled={outOfStock || addToCart.isPending}
              onClick={() => {
                if (!user) {
                  navigate("/login", { state: { from: `/products/${product.id}` } });
                  return;
                }
                addToCart.mutate({ productId: product.id, quantity });
              }}
            >
              <ShoppingCart className="size-4" />
              {user ? "Add to cart" : "Sign in to buy"}
            </Button>
          </div>
        </div>
      </div>

      <ReviewsSection productId={product.id} />
    </div>
  );
}
