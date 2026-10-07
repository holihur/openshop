import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api } from "@lib/api";
import { errorMessage } from "@lib/errors";
import { t } from "@lib/i18n";
import type {
  Commission,
  PointsAccount,
  PointsTransaction,
  ReferralSummary,
  Wallet,
  WalletTransaction,
  Withdrawal,
} from "@lib/types";

// --- Customer ------------------------------------------------------------

export function useWallet(enabled = true) {
  return useQuery({
    queryKey: ["wallet"],
    queryFn: () => api.get<Wallet>("/wallet"),
    enabled,
  });
}

export function useWalletTransactions(page = 1, pageSize = 20) {
  return useQuery({
    queryKey: ["wallet", "transactions", page, pageSize],
    queryFn: () => api.getPage<WalletTransaction[]>(`/wallet/transactions?page=${page}&pageSize=${pageSize}`),
  });
}

export function useTopUpWallet() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (amountCents: number) => api.post<Wallet>("/wallet/topup", { amountCents }),
    onSuccess: () => {
      toast.success(t("toast.walletToppedUp"));
      void queryClient.invalidateQueries({ queryKey: ["wallet"] });
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}

export function usePoints(enabled = true) {
  return useQuery({
    queryKey: ["points"],
    queryFn: () => api.get<PointsAccount>("/points"),
    enabled,
  });
}

export function usePointsTransactions(page = 1, pageSize = 20) {
  return useQuery({
    queryKey: ["points", "transactions", page, pageSize],
    queryFn: () => api.getPage<PointsTransaction[]>(`/points/transactions?page=${page}&pageSize=${pageSize}`),
  });
}

export function useReferralSummary() {
  return useQuery({
    queryKey: ["referrals"],
    queryFn: () => api.get<ReferralSummary>("/referrals"),
  });
}

export function useMyCommissions(page = 1, pageSize = 20) {
  return useQuery({
    queryKey: ["referrals", "commissions", page, pageSize],
    queryFn: () => api.getPage<Commission[]>(`/referrals/commissions?page=${page}&pageSize=${pageSize}`),
  });
}

// --- Withdrawals ---------------------------------------------------------

export function useWithdrawals(page = 1, pageSize = 20) {
  return useQuery({
    queryKey: ["withdrawals", "mine", page, pageSize],
    queryFn: () => api.getPage<Withdrawal[]>(`/wallet/withdrawals?page=${page}&pageSize=${pageSize}`),
  });
}

export interface WithdrawalInput {
  amountCents: number;
  method: string;
  accountName: string;
  accountNo: string;
  note?: string;
}

export function useRequestWithdrawal() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: WithdrawalInput) => api.post<Withdrawal>("/wallet/withdrawals", input),
    onSuccess: () => {
      toast.success(t("toast.withdrawalRequested"));
      void queryClient.invalidateQueries({ queryKey: ["withdrawals"] });
      void queryClient.invalidateQueries({ queryKey: ["wallet"] });
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}

export function useCancelWithdrawal() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.post<Withdrawal>(`/wallet/withdrawals/${id}/cancel`),
    onSuccess: () => {
      toast.success(t("toast.withdrawalCancelled"));
      void queryClient.invalidateQueries({ queryKey: ["withdrawals"] });
      void queryClient.invalidateQueries({ queryKey: ["wallet"] });
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}

export function useAdminWithdrawals(params: { status?: string; page?: number; pageSize?: number } = {}) {
  const { status, page = 1, pageSize = 20 } = params;
  const search = new URLSearchParams({ page: String(page), pageSize: String(pageSize) });
  if (status) search.set("status", status);
  return useQuery({
    queryKey: ["withdrawals", "admin", status, page, pageSize],
    queryFn: () => api.getPage<Withdrawal[]>(`/ops/withdrawals?${search.toString()}`),
  });
}

function useWithdrawalAction(path: string, successKey: "toast.withdrawalApproved" | "toast.withdrawalRejected" | "toast.withdrawalPaid") {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { id: string; body?: unknown }) =>
      api.post<Withdrawal>(`/ops/withdrawals/${input.id}/${path}`, input.body),
    onSuccess: () => {
      toast.success(t(successKey));
      void queryClient.invalidateQueries({ queryKey: ["withdrawals"] });
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}

export function useApproveWithdrawal() {
  return useWithdrawalAction("approve", "toast.withdrawalApproved");
}

export function useRejectWithdrawal() {
  return useWithdrawalAction("reject", "toast.withdrawalRejected");
}

export function usePayWithdrawal() {
  return useWithdrawalAction("pay", "toast.withdrawalPaid");
}

// --- Ops console ---------------------------------------------------------

export function useWalletLedger(params: { userId?: string; type?: string; page?: number; pageSize?: number } = {}) {
  const { userId, type, page = 1, pageSize = 20 } = params;
  const search = new URLSearchParams({ page: String(page), pageSize: String(pageSize) });
  if (userId) search.set("userId", userId);
  if (type) search.set("type", type);
  return useQuery({
    queryKey: ["wallet", "ledger", userId, type, page, pageSize],
    queryFn: () => api.getPage<WalletTransaction[]>(`/ops/wallet/transactions?${search.toString()}`),
  });
}

export function useAdjustWallet(customerId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { amountCents: number; description?: string }) =>
      api.post<{ balanceCents: number }>(`/ops/customers/${customerId}/wallet/adjust`, input),
    onSuccess: () => {
      toast.success(t("toast.walletAdjusted"));
      void queryClient.invalidateQueries({ queryKey: ["wallet"] });
      void queryClient.invalidateQueries({ queryKey: ["customer", customerId] });
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}

export function useAdjustPoints(customerId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { points: number; description?: string }) =>
      api.post<{ ok: boolean }>(`/ops/customers/${customerId}/points/adjust`, input),
    onSuccess: () => {
      toast.success(t("toast.pointsAdjusted"));
      void queryClient.invalidateQueries({ queryKey: ["points"] });
      void queryClient.invalidateQueries({ queryKey: ["customer", customerId] });
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}

export function useAdminCommissions(params: { status?: string; page?: number; pageSize?: number } = {}) {
  const { status, page = 1, pageSize = 20 } = params;
  const search = new URLSearchParams({ page: String(page), pageSize: String(pageSize) });
  if (status) search.set("status", status);
  return useQuery({
    queryKey: ["commissions", "admin", status, page, pageSize],
    queryFn: () => api.getPage<Commission[]>(`/ops/commissions?${search.toString()}`),
  });
}
