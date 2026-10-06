import { Link } from "react-router-dom";
import { ImageOff, ShoppingCart } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { WishlistButton } from "@/components/wishlist-button";
import { formatMoney } from "@/lib/format";
import { useAddToCart } from "@/hooks/useCart";
import { useAuth } from "@/lib/auth";
import type { Product } from "@/lib/types";

export function ProductCard({ product }: { product: Product }) {
  const addToCart = useAddToCart();
  const { user } = useAuth();
  const outOfStock = product.stock <= 0;

  return (
    <Card className="group overflow-hidden pt-0">
      <Link to={`/products/${product.id}`} className="block">
        <div className="bg-muted relative aspect-[4/3] w-full overflow-hidden">
          {product.coverImage ? (
            <img
              src={product.coverImage}
              alt={product.title}
              loading="lazy"
              className="size-full object-cover transition-transform duration-300 group-hover:scale-105"
            />
          ) : (
            <div className="text-muted-foreground flex size-full items-center justify-center">
              <ImageOff className="size-8" />
            </div>
          )}
          {outOfStock && (
            <Badge variant="secondary" className="absolute top-2 left-2">
              Out of stock
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
        <span className="text-lg font-semibold">
          {formatMoney(product.priceCents, product.currency)}
        </span>
        <Button
          size="sm"
          disabled={outOfStock || !user || addToCart.isPending}
          onClick={() => addToCart.mutate({ productId: product.id })}
        >
          <ShoppingCart className="size-4" />
          Add
        </Button>
      </CardFooter>
    </Card>
  );
}
