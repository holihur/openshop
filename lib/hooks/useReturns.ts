import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api } from "@lib/api";
import { errorMessage } from "@lib/errors";
import { t } from "@lib/i18n";
import type { ReturnRequest } from "@lib/types";

const RETURNS_KEY = ["returns"] as const;

/** A customer's own return requests for an order. */
export function useOrderReturns(orderId: string) {
  return useQuery({
    queryKey: ["order-returns", orderId],
    queryFn: () => api.get<ReturnRequest[]>(`/orders/${orderId}/returns`),
    enabled: Boolean(orderId),
  });
}

export function useRequestReturn(orderId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (reason: string) =>
      api.post<ReturnRequest>(`/orders/${orderId}/returns`, { reason }),
    onSuccess: () => {
      toast.success(t("toast.returnRequested"));
      void queryClient.invalidateQueries({ queryKey: ["order-returns", orderId] });
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}

/** Ops: all return requests. */
export function useAdminReturns() {
  return useQuery({
    queryKey: RETURNS_KEY,
    queryFn: () => api.getPage<ReturnRequest[]>("/ops/returns?pageSize=100"),
  });
}

function useReturnDecision(action: "approve" | "reject") {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.post<ReturnRequest>(`/ops/returns/${id}/${action}`),
    onSuccess: () => {
      toast.success(t(action === "approve" ? "toast.returnApproved" : "toast.returnRejected"));
      void queryClient.invalidateQueries({ queryKey: RETURNS_KEY });
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}

export function useApproveReturn() {
  return useReturnDecision("approve");
}

export function useRejectReturn() {
  return useReturnDecision("reject");
}
