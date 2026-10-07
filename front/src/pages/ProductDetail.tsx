import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { ImageOff, Minus, Plus, ShoppingCart } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { Badge } from "@lib/components/ui/badge";
import { Separator } from "@lib/components/ui/separator";
import { Skeleton } from "@lib/components/ui/skeleton";
import { ReviewsSection } from "@/components/reviews-section";
import { WishlistButton } from "@lib/components/wishlist-button";
import { api } from "@lib/api";
import { useAddToCart } from "@lib/hooks/useCart";
import { useI18n } from "@lib/i18n";
import { useSeo } from "@lib/hooks/useSeo";
import { usePrice } from "@lib/hooks/usePrice";
import { responsiveSrcSet } from "@lib/media";
import { cn } from "@lib/utils";
import type { Product, Variant } from "@lib/types";

export function ProductDetailPage() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const [quantity, setQuantity] = useState(1);
  const [activeImage, setActiveImage] = useState<string | null>(null);
  const [variantId, setVariantId] = useState<string>("");
  const { t } = useI18n();
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
      : { title: t("product.seoTitle") },
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
        <p className="text-muted-foreground">{t("product.notFound")}</p>
        <Button variant="link" asChild>
          <Link to="/products">{t("product.backToProducts")}</Link>
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
      <nav className="text-muted-foreground flex items-center gap-1.5 text-sm">
        <Link to="/" className="hover:text-foreground">
          {t("nav.home")}
        </Link>
        <span>/</span>
        <Link to="/products" className="hover:text-foreground">
          {t("nav.products")}
        </Link>
        <span>/</span>
        <span className="text-foreground truncate">{product.title}</span>
      </nav>

      <div className="grid gap-8 md:grid-cols-2">
        <div className="space-y-3">
          <div className="bg-muted aspect-square overflow-hidden rounded-xl border">
            {shown ? (
              <img
                src={shown}
                srcSet={responsiveSrcSet(shown)}
                sizes="(max-width: 768px) 100vw, 50vw"
                alt={product.title}
                decoding="async"
                className="size-full object-cover"
              />
            ) : (
              <div className="text-muted-foreground flex size-full items-center justify-center">
                <ImageOff className="size-10" />
              </div>
            )}
          </div>
          {gallery.length > 1 && (
            <div className="flex gap-2">
              {gallery.map((src, index) => (
                <button
                  key={src}
                  onClick={() => setActiveImage(src)}
                  aria-label={`${product.title} image ${index + 1}`}
                  aria-current={activeImage === src ? "true" : undefined}
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
                <Badge variant="secondary">{t("product.outOfStock")}</Badge>
              ) : (
                <Badge variant="success">{t("product.inStock", { count: effectiveStock })}</Badge>
              )}
            </div>
            {product.reviewCount ? (
              <p className="text-muted-foreground mt-2 text-sm">
                ★ {product.rating?.toFixed(1)} ·{" "}
                {product.reviewCount === 1
                  ? t("product.reviewOne")
                  : t("product.reviewsCount", { count: product.reviewCount })}
              </p>
            ) : null}
          </div>

          <Separator />

          <p className="text-muted-foreground whitespace-pre-line">{product.description}</p>

          {hasVariants && (
            <div className="space-y-2">
              <p className="text-sm font-medium">{t("product.options")}</p>
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

          <div className="flex flex-wrap items-center gap-3">
            <div className="flex items-center rounded-md border">
              <Button
                variant="ghost"
                size="icon"
                aria-label={t("cart.decrease")}
                disabled={quantity <= 1}
                onClick={() => setQuantity((q) => Math.max(1, q - 1))}
              >
                <Minus className="size-4" />
              </Button>
              <span className="w-10 text-center text-sm">{quantity}</span>
              <Button
                variant="ghost"
                size="icon"
                aria-label={t("cart.increase")}
                disabled={quantity >= effectiveStock}
                onClick={() => setQuantity((q) => q + 1)}
              >
                <Plus className="size-4" />
              </Button>
            </div>

            <div className="flex flex-1 flex-col gap-2 sm:flex-row">
              <Button
                className="flex-1"
                size="lg"
                disabled={outOfStock || addToCart.isPending}
                onClick={() =>
                  addToCart.mutate({
                    productId: product.id,
                    variantId: selected?.id,
                    quantity,
                  })
                }
              >
                <ShoppingCart className="size-4" />
                {t("product.addToCart")}
              </Button>
              <Button
                variant="outline"
                size="lg"
                disabled={outOfStock || addToCart.isPending}
                onClick={() =>
                  addToCart.mutate(
                    { productId: product.id, variantId: selected?.id, quantity },
                    { onSuccess: () => navigate("/cart") },
                  )
                }
              >
                {t("product.buyNow")}
              </Button>
            </div>
          </div>
        </div>
      </div>

      <ReviewsSection productId={product.id} />
    </div>
  );
}
