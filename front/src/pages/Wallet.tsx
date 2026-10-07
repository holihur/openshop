import { useState, type FormEvent } from "react";

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
import { useI18n } from "@lib/i18n";
import { formatDate, formatMoney } from "@lib/format";

export function WalletPage() {
  const { t } = useI18n();
  const { data: wallet, isLoading } = useWallet();
  const [page, setPage] = useState(1);
  const { data: ledger } = useWalletTransactions(page);
  const topUp = useTopUpWallet();
  const [amount, setAmount] = useState("");
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
