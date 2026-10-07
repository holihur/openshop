import { useState, type FormEvent } from "react";

import { Badge } from "@lib/components/ui/badge";
import { Button } from "@lib/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@lib/components/ui/card";
import { Input } from "@lib/components/ui/input";
import { Label } from "@lib/components/ui/label";
import { Skeleton } from "@lib/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@lib/components/ui/table";
import { Pagination } from "@lib/components/pagination";
import { useTopUpWallet, useWallet, useWalletTransactions } from "@lib/hooks/useLoyalty";
import { useCancelWithdrawal, useRequestWithdrawal, useWithdrawals } from "@lib/hooks/useLoyalty";
import { useSite } from "@lib/hooks/useSite";
import { useI18n } from "@lib/i18n";
import { formatDate, formatMoney } from "@lib/format";
import type { WithdrawalStatus } from "@lib/types";

const WITHDRAWAL_VARIANT: Record<
  WithdrawalStatus,
  "warning" | "default" | "success" | "secondary" | "destructive"
> = {
  requested: "warning",
  approved: "default",
  paid: "success",
  rejected: "destructive",
  cancelled: "secondary",
};

export function WalletPage() {
  const { t } = useI18n();
  const { data: wallet, isLoading } = useWallet();
  const [page, setPage] = useState(1);
  const { data: ledger } = useWalletTransactions(page);
  const { data: site } = useSite();
  const { data: withdrawals } = useWithdrawals();
  const request = useRequestWithdrawal();
  const cancel = useCancelWithdrawal();
  const topUp = useTopUpWallet();
  const [amount, setAmount] = useState("");
  const [withdrawAmount, setWithdrawAmount] = useState("");
  const [method, setMethod] = useState("bank");
  const [accountName, setAccountName] = useState("");
  const [accountNo, setAccountNo] = useState("");
  const currency = wallet?.currency ?? "USD";

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    const cents = Math.round(Number.parseFloat(amount) * 100);
    if (!Number.isFinite(cents) || cents <= 0) return;
    topUp.mutate(cents, { onSuccess: () => setAmount("") });
  }

  return (
    <div className="mx-auto max-w-3xl space-y-6 py-8">
      <h1 className="text-2xl font-semibold">{t("wallet.title")}</h1>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("wallet.balance")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {isLoading ? (
            <Skeleton className="h-8 w-40" />
          ) : (
            <p className="text-3xl font-semibold">{formatMoney(wallet?.balanceCents ?? 0, currency)}</p>
          )}
          <form onSubmit={onSubmit} className="flex flex-wrap items-end gap-3">
            <div className="space-y-2">
              <Label htmlFor="topup">{t("wallet.topUpAmount")}</Label>
              <Input
                id="topup"
                type="number"
                min="1"
                step="0.01"
                value={amount}
                onChange={(e) => setAmount(e.target.value)}
                className="w-40"
              />
            </div>
            <Button type="submit" disabled={topUp.isPending}>
              {topUp.isPending ? t("common.saving") : t("wallet.topUp")}
            </Button>
          </form>
          <p className="text-muted-foreground text-xs">{t("wallet.topUpHint")}</p>
        </CardContent>
      </Card>

      {site?.withdrawalEnabled && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">{t("withdraw.title")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {site.withdrawalInstructions && (
              <p className="text-muted-foreground text-xs">{site.withdrawalInstructions}</p>
            )}
            <form
              className="grid gap-3 sm:grid-cols-2"
              onSubmit={(e) => {
                e.preventDefault();
                const cents = Math.round(Number.parseFloat(withdrawAmount) * 100);
                if (!Number.isFinite(cents) || cents <= 0) return;
                request.mutate(
                  { amountCents: cents, method, accountName, accountNo },
                  { onSuccess: () => setWithdrawAmount("") },
                );
              }}
            >
              <div className="space-y-2">
                <Label htmlFor="withdraw-amount">{t("withdraw.amount")}</Label>
                <Input
                  id="withdraw-amount"
                  type="number"
                  min={site.withdrawalMinCents / 100}
                  step="0.01"
                  required
                  value={withdrawAmount}
                  onChange={(e) => setWithdrawAmount(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="withdraw-method">{t("withdraw.method")}</Label>
                <select
                  id="withdraw-method"
                  value={method}
                  onChange={(e) => setMethod(e.target.value)}
                  className="border-input bg-background h-9 w-full rounded-md border px-3 text-sm"
                >
                  <option value="bank">{t("withdraw.method.bank")}</option>
                  <option value="alipay">{t("withdraw.method.alipay")}</option>
                  <option value="wechat">{t("withdraw.method.wechat")}</option>
                  <option value="other">{t("withdraw.method.other")}</option>
                </select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="withdraw-name">{t("withdraw.accountName")}</Label>
                <Input
                  id="withdraw-name"
                  value={accountName}
                  onChange={(e) => setAccountName(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="withdraw-no">{t("withdraw.accountNo")}</Label>
                <Input
                  id="withdraw-no"
                  required
                  value={accountNo}
                  onChange={(e) => setAccountNo(e.target.value)}
                />
              </div>
              <div className="sm:col-span-2">
                <Button type="submit" disabled={request.isPending}>
                  {t("withdraw.submit")}
                </Button>
              </div>
            </form>

            {withdrawals?.items.length ? (
              <ul className="space-y-2">
                {withdrawals.items.map((w) => (
                  <li
                    key={w.id}
                    className="flex items-center justify-between gap-3 rounded-md border p-3 text-sm"
                  >
                    <div>
                      <span className="font-medium">
                        {formatMoney(w.amountCents, w.currency || currency)}
                      </span>
                      <span className="text-muted-foreground ml-2">{w.accountNo}</span>
                      {w.rejectReason && (
                        <span className="text-destructive ml-2 text-xs">{w.rejectReason}</span>
                      )}
                    </div>
                    <div className="flex items-center gap-2">
                      <Badge variant={WITHDRAWAL_VARIANT[w.status] ?? "secondary"}>
                        {t(`withdraw.status.${w.status}`)}
                      </Badge>
                      {w.status === "requested" && (
                        <Button
                          variant="ghost"
                          size="sm"
                          disabled={cancel.isPending}
                          onClick={() => cancel.mutate(w.id)}
                        >
                          {t("common.cancel")}
                        </Button>
                      )}
                    </div>
                  </li>
                ))}
              </ul>
            ) : null}
          </CardContent>
        </Card>
      )}

      <div className="space-y-3">
        <h2 className="text-lg font-medium">{t("wallet.history")}</h2>
        <Card className="py-0">
          <CardContent className="px-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("wallet.type")}</TableHead>
                  <TableHead>{t("wallet.description")}</TableHead>
                  <TableHead className="text-right">{t("wallet.amount")}</TableHead>
                  <TableHead className="text-right">{t("wallet.balanceAfter")}</TableHead>
                  <TableHead>{t("ops.date")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {ledger?.items.map((tx) => (
                  <TableRow key={tx.id}>
                    <TableCell>{t(`wallet.txType.${tx.type}`)}</TableCell>
                    <TableCell className="text-sm">{tx.description}</TableCell>
                    <TableCell
                      className={`text-right font-medium ${tx.amountCents < 0 ? "text-destructive" : "text-emerald-600"}`}
                    >
                      {tx.amountCents > 0 ? "+" : ""}
                      {formatMoney(tx.amountCents, currency)}
                    </TableCell>
                    <TableCell className="text-right">
                      {formatMoney(tx.balanceAfter, currency)}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {formatDate(tx.createdAt)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
        <Pagination
          page={ledger?.page ?? page}
          pageSize={ledger?.pageSize ?? 20}
          total={ledger?.total ?? 0}
          onChange={setPage}
        />
      </div>
    </div>
  );
}
