import { Link } from "react-router-dom";
import { ImageOff, ShoppingCart } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { Card, CardContent, CardFooter } from "@lib/components/ui/card";
import { Badge } from "@lib/components/ui/badge";
import { WishlistButton } from "@lib/components/wishlist-button";
import { useAddToCart } from "@lib/hooks/useCart";
import { usePrice } from "@lib/hooks/usePrice";
import { useAuth } from "@lib/auth";
import { useI18n } from "@lib/i18n";
import { responsiveSrcSet } from "@lib/media";
import type { Product } from "@lib/types";

export function ProductCard({ product }: { product: Product }) {
  const addToCart = useAddToCart();
  const { user } = useAuth();
  const { t } = useI18n();
  const price = usePrice();
  const outOfStock = product.stock <= 0;

  return (
    <Card className="group overflow-hidden pt-0">
      <Link to={`/products/${product.id}`} className="block">
        <div className="bg-muted relative aspect-[4/3] w-full overflow-hidden">
          {product.coverImage ? (
            <img
              src={product.coverImage}
              srcSet={responsiveSrcSet(product.coverImage)}
              sizes="(max-width: 640px) 100vw, 33vw"
              alt={product.title}
              loading="lazy"
              decoding="async"
              className="size-full object-cover transition-transform duration-300 group-hover:scale-105"
            />
          ) : (
            <div className="text-muted-foreground flex size-full items-center justify-center">
              <ImageOff className="size-8" />
            </div>
          )}
          {outOfStock && (
            <Badge variant="secondary" className="absolute top-2 left-2">
              {t("product.outOfStock")}
            </Badge>
          )}
          <WishlistButton productId={product.id} className="absolute top-2 right-2" />
        </div>
      </Link>
      <CardContent className="flex-1">
        <Link to={`/products/${product.id}`}>
          <h3 className="line-clamp-2 font-medium hover:underline">{product.title}</h3>
        </Link>
        <p className="text-muted-foreground mt-1 line-clamp-2 text-sm">
          {product.description}
        </p>
      </CardContent>
      <CardFooter className="justify-between gap-2">
        <span className="text-lg font-semibold">{price(product.priceCents)}</span>
        <Button
          size="sm"
          disabled={outOfStock || !user || addToCart.isPending}
          onClick={() => addToCart.mutate({ productId: product.id })}
        >
          <ShoppingCart className="size-4" />
          {t("product.add")}
        </Button>
      </CardFooter>
    </Card>
  );
}
