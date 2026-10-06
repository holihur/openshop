import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import type { Cart } from "@/lib/types";

const CART_KEY = ["cart"] as const;

export function useCart() {
  const { user } = useAuth();
  return useQuery({
    queryKey: CART_KEY,
    queryFn: () => api.get<Cart>("/cart"),
    enabled: Boolean(user),
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
      queryClient.setQueryData(CART_KEY, cart);
      if (successMessage) toast.success(successMessage);
    },
    onError: (error: Error) => toast.error(error.message),
  });
}

export function useAddToCart() {
  return useCartMutation(
    ({ productId, quantity }: { productId: string; quantity?: number }) =>
      api.post<Cart>("/cart/items", { productId, quantity: quantity ?? 1 }),
    "Added to cart",
  );
}

export function useUpdateCartItem() {
  return useCartMutation(({ productId, quantity }: { productId: string; quantity: number }) =>
    api.patch<Cart>(`/cart/items/${productId}`, { quantity }),
  );
}

export function useRemoveCartItem() {
  return useCartMutation(({ productId }: { productId: string }) =>
    api.del<Cart>(`/cart/items/${productId}`),
  );
}

export function useClearCart() {
  return useCartMutation(() => api.del<Cart>("/cart"), "Cart cleared");
}
