import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft, ImageOff, Minus, Plus, ShoppingCart } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { ReviewsSection } from "@/components/reviews-section";
import { WishlistButton } from "@/components/wishlist-button";
import { api } from "@/lib/api";
import { useAddToCart } from "@/hooks/useCart";
import { useAuth } from "@/lib/auth";
import { useSeo } from "@/hooks/useSeo";
import { usePrice } from "@/hooks/usePrice";
import { cn } from "@/lib/utils";
import type { Product, Variant } from "@/lib/types";

export function ProductDetailPage() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const [quantity, setQuantity] = useState(1);
  const [activeImage, setActiveImage] = useState<string | null>(null);
  const [variantId, setVariantId] = useState<string>("");
  const { user } = useAuth();
  const addToCart = useAddToCart();
  const price = usePrice();

  const { data: product, isLoading, isError } = useQuery({
    queryKey: ["product", id],
    queryFn: () => api.get<Product>(`/products/${id}`),
    enabled: Boolean(id),
  });

  useSeo(
    product
      ? {
          title: product.title,
          description: product.description,
          image: product.coverImage,
          type: "product",
          jsonLd: {
            "@context": "https://schema.org",
            "@type": "Product",
            name: product.title,
            description: product.description,
            image: product.coverImage ? [product.coverImage] : undefined,
            offers: {
              "@type": "Offer",
              priceCurrency: product.currency,
              price: (product.priceCents / 100).toFixed(2),
              availability:
                product.stock > 0
                  ? "https://schema.org/InStock"
                  : "https://schema.org/OutOfStock",
            },
          },
        }
      : { title: "Product" },
  );

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

  const variants = (product.variants ?? []).filter((v) => v.active);
  const hasVariants = variants.length > 0;
  const selected: Variant | undefined = hasVariants
    ? variants.find((v) => v.id === variantId) ?? variants.find((v) => v.stock > 0) ?? variants[0]
    : undefined;

  const effectivePrice = selected
    ? selected.priceCents > 0
      ? selected.priceCents
      : product.priceCents
    : product.priceCents;
  const effectiveStock = selected ? selected.stock : product.stock;
  const outOfStock = effectiveStock <= 0;

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
            <div className="flex items-start justify-between gap-3">
              <h1 className="text-3xl font-bold">{product.title}</h1>
              <WishlistButton productId={product.id} />
            </div>
            <div className="mt-2 flex items-center gap-3">
              <span className="text-2xl font-semibold">
                {price(effectivePrice)}
              </span>
              {outOfStock ? (
                <Badge variant="secondary">Out of stock</Badge>
              ) : (
                <Badge variant="success">{effectiveStock} in stock</Badge>
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

          {hasVariants && (
            <div className="space-y-2">
              <p className="text-sm font-medium">Options</p>
              <div className="flex flex-wrap gap-2">
                {variants.map((v) => {
                  const isSelected = selected?.id === v.id;
                  const disabled = v.stock <= 0;
                  return (
                    <button
                      key={v.id}
                      disabled={disabled}
                      onClick={() => {
                        setVariantId(v.id);
                        setQuantity(1);
                      }}
                      className={cn(
                        "rounded-md border px-3 py-2 text-sm transition-colors",
                        isSelected
                          ? "border-primary bg-primary text-primary-foreground"
                          : "hover:bg-accent",
                        disabled && "cursor-not-allowed opacity-40 line-through",
                      )}
                    >
                      {v.name}
                      {v.priceCents > 0 && v.priceCents !== product.priceCents && (
                        <span className="ml-2 text-xs opacity-80">{price(v.priceCents)}</span>
                      )}
                    </button>
                  );
                })}
              </div>
            </div>
          )}

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
                disabled={quantity >= effectiveStock}
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
                addToCart.mutate({
                  productId: product.id,
                  variantId: selected?.id,
                  quantity,
                });
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
