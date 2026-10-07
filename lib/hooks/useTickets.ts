import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api } from "@lib/api";
import { errorMessage } from "@lib/errors";
import { t } from "@lib/i18n";
import type { Ticket, TicketMessage } from "@lib/types";

export interface TicketThread {
  ticket: Ticket;
  messages: TicketMessage[];
}

export interface CreateTicketInput {
  subject: string;
  body: string;
  kind?: string;
  orderId?: string;
  productId?: string;
  email?: string;
  name?: string;
}

/** A customer's own tickets. */
export function useMyTickets(page = 1, pageSize = 20) {
  return useQuery({
    queryKey: ["tickets", "mine", page, pageSize],
    queryFn: () => api.getPage<Ticket[]>(`/tickets?page=${page}&pageSize=${pageSize}`),
  });
}

/** A customer's own ticket thread. */
export function useMyTicket(id: string) {
  return useQuery({
    queryKey: ["ticket", "mine", id],
    queryFn: () => api.get<TicketThread>(`/tickets/${id}`),
    enabled: Boolean(id),
  });
}

export function useCreateTicket() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateTicketInput) => api.post<Ticket>("/tickets", input),
    onSuccess: () => {
      toast.success(t("toast.ticketCreated"));
      void queryClient.invalidateQueries({ queryKey: ["tickets"] });
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}

export function useReplyTicket(id: string, admin = false) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { body: string; internal?: boolean }) =>
      api.post<TicketMessage>(`${admin ? "/ops" : ""}/tickets/${id}/messages`, input),
    onSuccess: () => {
      toast.success(t("toast.ticketReplied"));
      void queryClient.invalidateQueries({ queryKey: ["ticket"] });
      void queryClient.invalidateQueries({ queryKey: ["tickets"] });
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}

// --- Ops console ---------------------------------------------------------

export interface AdminTicketParams {
  page?: number;
  pageSize?: number;
  status?: string;
  kind?: string;
  assignee?: string;
  q?: string;
}

export function useAdminTickets(params: AdminTicketParams = {}) {
  const { page = 1, pageSize = 20, status, kind, assignee, q } = params;
  const search = new URLSearchParams({ page: String(page), pageSize: String(pageSize) });
  if (status) search.set("status", status);
  if (kind) search.set("kind", kind);
  if (assignee) search.set("assignee", assignee);
  if (q) search.set("q", q);
  return useQuery({
    queryKey: ["tickets", "admin", page, pageSize, status, kind, assignee, q],
    queryFn: () => api.getPage<Ticket[]>(`/ops/tickets?${search.toString()}`),
  });
}

export function useAdminTicket(id: string) {
  return useQuery({
    queryKey: ["ticket", "admin", id],
    queryFn: () => api.get<TicketThread>(`/ops/tickets/${id}`),
    enabled: Boolean(id),
  });
}

export function useUpdateTicket(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (patch: Partial<Pick<Ticket, "status" | "priority" | "kind" | "assigneeId">>) =>
      api.patch<Ticket>(`/ops/tickets/${id}`, patch),
    onSuccess: () => {
      toast.success(t("toast.ticketUpdated"));
      void queryClient.invalidateQueries({ queryKey: ["ticket"] });
      void queryClient.invalidateQueries({ queryKey: ["tickets"] });
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}

export function useAssignTicketToMe(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => api.post<Ticket>(`/ops/tickets/${id}/assign`),
    onSuccess: () => {
      toast.success(t("toast.ticketAssigned"));
      void queryClient.invalidateQueries({ queryKey: ["ticket"] });
      void queryClient.invalidateQueries({ queryKey: ["tickets"] });
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}
