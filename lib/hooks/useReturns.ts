import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api } from "@lib/api";
import { errorMessage } from "@lib/errors";
import { t } from "@lib/i18n";
import type { ReturnRequest } from "@lib/types";

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

/** Ops: all return requests, paged and optionally filtered by status. */
export function useAdminReturns(page = 1, pageSize = 20, status?: string) {
  const params = new URLSearchParams({ page: String(page), pageSize: String(pageSize) });
  if (status) params.set("status", status);
  return useQuery({
    queryKey: ["returns", "admin", page, pageSize, status],
    queryFn: () => api.getPage<ReturnRequest[]>(`/ops/returns?${params.toString()}`),
  });
}

function useReturnDecision(action: "approve" | "reject") {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.post<ReturnRequest>(`/ops/returns/${id}/${action}`),
    onSuccess: () => {
      toast.success(t(action === "approve" ? "toast.returnApproved" : "toast.returnRejected"));
      void queryClient.invalidateQueries({ queryKey: ["returns"] });
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
