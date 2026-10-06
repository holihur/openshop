import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api } from "@/lib/api";
import type { ShippingMethod } from "@/lib/types";

export function useShippingMethods() {
  return useQuery({
    queryKey: ["shipping-methods"],
    queryFn: () => api.get<ShippingMethod[]>("/shipping-methods"),
    staleTime: 5 * 60_000,
  });
}

export function useAdminShippingMethods() {
  return useQuery({
    queryKey: ["admin", "shipping-methods"],
    queryFn: () => api.get<ShippingMethod[]>("/admin/shipping-methods"),
  });
}

export interface ShippingMethodInput {
  code?: string;
  name: string;
  flatRateCents: number;
  freeThresholdCents: number;
  active?: boolean;
  sort?: number;
}

export function useCreateShippingMethod() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: ShippingMethodInput) =>
      api.post<ShippingMethod>("/admin/shipping-methods", input),
    onSuccess: () => {
      toast.success("Shipping method created");
      void queryClient.invalidateQueries({ queryKey: ["admin", "shipping-methods"] });
      void queryClient.invalidateQueries({ queryKey: ["shipping-methods"] });
    },
    onError: (error: Error) => toast.error(error.message),
  });
}

export function useUpdateShippingMethod() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: ShippingMethodInput }) =>
      api.patch<ShippingMethod>(`/admin/shipping-methods/${id}`, input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["admin", "shipping-methods"] });
      void queryClient.invalidateQueries({ queryKey: ["shipping-methods"] });
    },
    onError: (error: Error) => toast.error(error.message),
  });
}
