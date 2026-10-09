import { apiGet } from "@/lib/api";
import { getShopper } from "@/lib/session";
import type { Cart } from "@/lib/types";

/**
 * The cart count is rendered on the server from the same cart the API owns, so
 * it is correct on the very first paint (and for guests, whose cart is keyed by
 * the guest id cookie).
 */
export async function CartBadge() {
  const shopper = await getShopper();
  if (!shopper.token && !shopper.guestId) return null;
  const cart = await apiGet<Cart>("/cart", { ...shopper }).catch(() => null);
  if (!cart?.totalCount) return null;
  return (
    <span className="bg-primary text-primary-foreground rounded-full px-2 text-xs">
      {cart.totalCount}
    </span>
  );
}
