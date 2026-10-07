import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api } from "@lib/api";
import { t } from "@lib/i18n";
import { useAuth } from "@lib/auth";
import type { Product } from "@lib/types";

const WISHLIST_KEY = ["wishlist"] as const;

export function useWishlist() {
  const { user } = useAuth();
  return useQuery({
    queryKey: WISHLIST_KEY,
    queryFn: () => api.get<Product[]>("/wishlist"),
    enabled: Boolean(user),
  });
}

export function useWishlistIds(): Set<string> {
  const { data } = useWishlist();
  return new Set((data ?? []).map((p) => p.id));
}

export function useToggleWishlist() {
  const queryClient = useQueryClient();
  const { user } = useAuth();
  const saved = useWishlistIds();

  return useMutation({
    mutationFn: async (productId: string) => {
      if (saved.has(productId)) {
        await api.del(`/wishlist/${productId}`);
        return { saved: false };
      }
      await api.post("/wishlist", { productId });
      return { saved: true };
    },
    onSuccess: (res) => {
      if (!user) toast.error(t("toast.signInToSave"));
      else toast.success(res.saved ? t("toast.savedToWishlist") : t("toast.removedFromWishlist"));
      void queryClient.invalidateQueries({ queryKey: WISHLIST_KEY });
    },
    onError: (error: Error) => toast.error(error.message),
  });
}
