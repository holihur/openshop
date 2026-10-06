import { Heart } from "lucide-react";
import { useNavigate } from "react-router-dom";

import { useAuth } from "@/lib/auth";
import { useToggleWishlist, useWishlistIds } from "@/hooks/useWishlist";
import { cn } from "@/lib/utils";

// WishlistButton toggles a product in the signed-in user's wishlist. It shares
// the cached wishlist so it stays in sync across the page.
export function WishlistButton({
  productId,
  className,
}: {
  productId: string;
  className?: string;
}) {
  const { user } = useAuth();
  const navigate = useNavigate();
  const saved = useWishlistIds();
  const toggle = useToggleWishlist();
  const isSaved = saved.has(productId);

  return (
    <button
      type="button"
      aria-label={isSaved ? "Remove from wishlist" : "Save to wishlist"}
      className={cn(
        "bg-background/80 hover:bg-background inline-flex size-8 items-center justify-center rounded-full border backdrop-blur transition-colors",
        className,
      )}
      onClick={(e) => {
        e.preventDefault();
        if (!user) {
          navigate("/login", { state: { from: `/products/${productId}` } });
          return;
        }
        toggle.mutate(productId);
      }}
    >
      <Heart className={cn("size-4", isSaved ? "fill-rose-500 text-rose-500" : "")} />
    </button>
  );
}
