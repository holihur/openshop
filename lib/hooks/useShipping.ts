import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api } from "@lib/api";
import type { ShippingMethod, ShippingZone } from "@lib/types";

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
    queryFn: () => api.get<ShippingMethod[]>("/ops/shipping-methods"),
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
      api.post<ShippingMethod>("/ops/shipping-methods", input),
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
      api.patch<ShippingMethod>(`/ops/shipping-methods/${id}`, input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["admin", "shipping-methods"] });
      void queryClient.invalidateQueries({ queryKey: ["shipping-methods"] });
    },
    onError: (error: Error) => toast.error(error.message),
  });
}

export function useAdminShippingZones() {
  return useQuery({
    queryKey: ["admin", "shipping-zones"],
    queryFn: () => api.get<ShippingZone[]>("/ops/shipping-zones"),
  });
}

export function useCreateShippingZone() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { name: string; provinces: string[]; active?: boolean; sort?: number }) =>
      api.post<ShippingZone>("/ops/shipping-zones", input),
    onSuccess: () => {
      toast.success("Zone created");
      void queryClient.invalidateQueries({ queryKey: ["admin", "shipping-zones"] });
    },
    onError: (error: Error) => toast.error(error.message),
  });
}

export function useSetShippingRate() {
  return useMutation({
    mutationFn: ({ zoneId, methodId, input }: {
      zoneId: string;
      methodId: string;
      input: { flatRateCents: number; freeThresholdCents: number; perKgCents: number };
    }) => api.put(`/ops/shipping-zones/${zoneId}/rates/${methodId}`, input),
    onSuccess: () => toast.success("Rate saved"),
    onError: (error: Error) => toast.error(error.message),
  });
}
