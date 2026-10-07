import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api } from "@lib/api";
import { t } from "@lib/i18n";
import type { Review } from "@lib/types";

export function useReviews(productId: string) {
  return useQuery({
    queryKey: ["reviews", productId],
    queryFn: () => api.getPage<Review[]>(`/products/${productId}/reviews?pageSize=20`),
    enabled: Boolean(productId),
  });
}

export function useAddReview(productId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { rating: number; title?: string; body?: string }) =>
      api.post<Review>(`/products/${productId}/reviews`, input),
    onSuccess: () => {
      toast.success(t("toast.reviewThanks"));
      void queryClient.invalidateQueries({ queryKey: ["reviews", productId] });
      void queryClient.invalidateQueries({ queryKey: ["product", productId] });
    },
    onError: (error: Error) => toast.error(error.message),
  });
}

export function useDeleteReview(productId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (reviewId: string) => api.del(`/reviews/${reviewId}`),
    onSuccess: () => {
      toast.success(t("toast.reviewRemoved"));
      void queryClient.invalidateQueries({ queryKey: ["reviews", productId] });
      void queryClient.invalidateQueries({ queryKey: ["product", productId] });
    },
    onError: (error: Error) => toast.error(error.message),
  });
}
