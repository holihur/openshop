import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api } from "@lib/api";
import { errorMessage } from "@lib/errors";
import { t } from "@lib/i18n";

export interface OutboxStats {
  pending: number;
  processing: number;
  published: number;
  failed: number;
  oldestPending?: string;
  oldestPendingAgeSeconds?: number;
}

export interface OutboxEvent {
  id: string;
  subject: string;
  status: "pending" | "processing" | "published" | "failed";
  attempts: number;
  lastError?: string;
  createdAt: string;
  availableAt: string;
}

/** Queue depth, dead letters and backlog age. */
export function useOutboxStats() {
  return useQuery({
    queryKey: ["outbox", "stats"],
    queryFn: () => api.get<OutboxStats>("/ops/outbox/stats"),
    refetchInterval: 30_000,
  });
}

export function useOutboxEvents(params: { status?: string; subject?: string; page?: number; pageSize?: number } = {}) {
  const { status, subject, page = 1, pageSize = 20 } = params;
  const search = new URLSearchParams({ page: String(page), pageSize: String(pageSize) });
  if (status) search.set("status", status);
  if (subject) search.set("subject", subject);
  return useQuery({
    queryKey: ["outbox", "events", status, subject, page, pageSize],
    queryFn: () => api.getPage<OutboxEvent[]>(`/ops/outbox?${search.toString()}`),
  });
}

/** Return a dead-lettered event to the queue. */
export function useReplayOutboxEvent() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.post<{ ok: boolean }>(`/ops/outbox/${id}/replay`),
    onSuccess: () => {
      toast.success(t("toast.outboxReplayed"));
      void queryClient.invalidateQueries({ queryKey: ["outbox"] });
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}
