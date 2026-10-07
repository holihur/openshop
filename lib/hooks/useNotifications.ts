import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@lib/api";
import type { Notification } from "@lib/types";

/** The signed-in customer's inbox, newest first. */
export function useNotifications(page = 1, pageSize = 20, unreadOnly = false) {
  const params = new URLSearchParams({ page: String(page), pageSize: String(pageSize) });
  if (unreadOnly) params.set("unread", "true");
  return useQuery({
    queryKey: ["notifications", page, pageSize, unreadOnly],
    queryFn: () => api.getPage<Notification[]>(`/notifications?${params.toString()}`),
  });
}

/** The unread badge count, refreshed periodically. */
export function useUnreadCount() {
  return useQuery({
    queryKey: ["notifications", "unread"],
    queryFn: () => api.get<{ count: number }>("/notifications/unread-count"),
    refetchInterval: 60_000,
  });
}

export function useMarkNotificationRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.post<{ ok: boolean }>(`/notifications/${id}/read`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["notifications"] });
    },
  });
}

export function useMarkAllNotificationsRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => api.post<{ ok: boolean }>("/notifications/read-all"),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["notifications"] });
    },
  });
}

/** Ops: send a notification to every customer. */
export function useBroadcastNotification() {
  return useMutation({
    mutationFn: (input: { title: string; body?: string; link?: string }) =>
      api.post<{ recipients: number }>("/ops/notifications/broadcast", input),
  });
}
