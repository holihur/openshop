import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { errorMessage } from "@lib/errors";
import { api } from "@lib/api";
import { t } from "@lib/i18n";
import { useAuth } from "@lib/auth";
import type { Cart } from "@lib/types";

const CART_KEY = ["cart"] as const;

// The cart is keyed by the signed-in user, or by the browser's guest id, so an
// anonymous shopper still has a cart (and can check out as a guest).
export function useCart() {
  const { user } = useAuth();
  return useQuery({
    queryKey: ["cart", user?.id ?? "guest"],
    queryFn: () => api.get<Cart>("/cart"),
    staleTime: 10_000,
  });
}

function useCartMutation<TArgs>(
  fn: (args: TArgs) => Promise<Cart>,
  successMessage?: string,
) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: fn,
    onSuccess: (cart) => {
      queryClient.setQueriesData({ queryKey: CART_KEY }, cart);
      if (successMessage) toast.success(successMessage);
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}

interface LineRef {
  productId: string;
  variantId?: string;
}

export function useAddToCart() {
  return useCartMutation(
    ({ productId, variantId, quantity }: LineRef & { quantity?: number }) =>
      api.post<Cart>("/cart/items", {
        productId,
        variantId: variantId ?? "",
        quantity: quantity ?? 1,
      }),
    t("toast.addedToCart"),
  );
}

export function useUpdateCartItem() {
  return useCartMutation(({ productId, variantId, quantity }: LineRef & { quantity: number }) =>
    api.patch<Cart>(`/cart/items/${productId}`, { variantId: variantId ?? "", quantity }),
  );
}

export function useRemoveCartItem() {
  return useCartMutation(({ productId, variantId }: LineRef) =>
    api.del<Cart>(
      `/cart/items/${productId}${variantId ? `?variantId=${encodeURIComponent(variantId)}` : ""}`,
    ),
  );
}

export function useClearCart() {
  return useCartMutation(() => api.del<Cart>("/cart"), t("toast.cartCleared"));
}
