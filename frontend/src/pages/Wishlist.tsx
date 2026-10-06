import { Link } from "react-router-dom";
import { Heart } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { ProductGrid } from "@/components/product-grid";
import { useWishlist } from "@/hooks/useWishlist";

export function WishlistPage() {
  const { data, isLoading } = useWishlist();

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Wishlist</h1>

      {isLoading ? (
        <div className="grid gap-4 sm:grid-cols-3">
          <Skeleton className="h-64 w-full" />
          <Skeleton className="h-64 w-full" />
          <Skeleton className="h-64 w-full" />
        </div>
      ) : !data || data.length === 0 ? (
        <div className="text-muted-foreground rounded-lg border border-dashed py-20 text-center">
          <Heart className="mx-auto size-8" />
          <p className="mt-2">Your wishlist is empty.</p>
          <Button className="mt-6" asChild>
            <Link to="/products">Browse products</Link>
          </Button>
        </div>
      ) : (
        <ProductGrid products={data} />
      )}
    </div>
  );
}
