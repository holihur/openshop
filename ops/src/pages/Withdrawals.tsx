import { useState } from "react";

import { Badge } from "@lib/components/ui/badge";
import { Button } from "@lib/components/ui/button";
import { Card, CardContent } from "@lib/components/ui/card";
import { Input } from "@lib/components/ui/input";
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
import {
  useAdminWithdrawals,
  useApproveWithdrawal,
  usePayWithdrawal,
  useRejectWithdrawal,
} from "@lib/hooks/useLoyalty";
import { useI18n } from "@lib/i18n";
import { formatDate, formatMoney } from "@lib/format";
import { OpsModuleStats } from "@/components/ops-stats";
import type { WithdrawalStatus } from "@lib/types";

const STATUS_VARIANT: Record<
  WithdrawalStatus,
  "warning" | "default" | "success" | "secondary" | "destructive"
> = {
  requested: "warning",
  approved: "default",
  paid: "success",
  rejected: "destructive",
  cancelled: "secondary",
};

export function WithdrawalsPage() {
  const { t } = useI18n();
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState("");
  const { data, isLoading } = useAdminWithdrawals({ page, pageSize: 20, status });
  const approve = useApproveWithdrawal();
  const reject = useRejectWithdrawal();
  const pay = usePayWithdrawal();
  const [action, setAction] = useState<{ id: string; kind: "reject" | "pay" } | null>(null);
  const [text, setText] = useState("");

  function submitAction(id: string) {
    if (!action) return;
    if (action.kind === "reject") {
      reject.mutate({ id, body: { reason: text } });
    } else {
      pay.mutate({ id, body: { reference: text } });
    }
    setAction(null);
    setText("");
  }

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold">{t("withdraw.opsTitle")}</h1>
        <p className="text-muted-foreground text-sm">{t("withdraw.opsSubtitle")}</p>
      </div>

      <OpsModuleStats module="withdrawals" />

      <select
        aria-label={t("common.status")}
        value={status}
        onChange={(e) => {
          setStatus(e.target.value);
          setPage(1);
        }}
        className="border-input bg-background h-9 rounded-md border px-3 text-sm"
      >
        <option value="">{t("ops.allStatuses")}</option>
        {(["requested", "approved", "paid", "rejected", "cancelled"] as const).map((s) => (
          <option key={s} value={s}>
            {t(`withdraw.status.${s}`)}
          </option>
        ))}
      </select>

      {isLoading ? (
        <Skeleton className="h-64 w-full" />
      ) : (
        <Card className="py-0">
          <CardContent className="px-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("ops.ticketCustomer")}</TableHead>
                  <TableHead className="text-right">{t("withdraw.amount")}</TableHead>
                  <TableHead>{t("withdraw.method")}</TableHead>
                  <TableHead>{t("withdraw.account")}</TableHead>
                  <TableHead>{t("common.status")}</TableHead>
                  <TableHead>{t("ops.date")}</TableHead>
                  <TableHead className="text-right">{t("common.actions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data?.items.map((w) => (
                  <TableRow key={w.id}>
                    <TableCell className="font-mono text-xs">{w.userId.slice(0, 8)}</TableCell>
                    <TableCell className="text-right font-medium">
                      {formatMoney(w.amountCents, w.currency || "CNY")}
                    </TableCell>
                    <TableCell className="text-sm">{t(`withdraw.method.${w.method}`)}</TableCell>
                    <TableCell className="text-sm">
                      {w.accountName}
                      <div className="text-muted-foreground font-mono text-xs">{w.accountNo}</div>
                    </TableCell>
                    <TableCell>
                      <Badge variant={STATUS_VARIANT[w.status] ?? "secondary"}>
                        {t(`withdraw.status.${w.status}`)}
                      </Badge>
                      {w.rejectReason && (
                        <div className="text-muted-foreground text-xs">{w.rejectReason}</div>
                      )}
                      {w.paidReference && (
                        <div className="text-muted-foreground text-xs">{w.paidReference}</div>
                      )}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {formatDate(w.createdAt)}
                    </TableCell>
                    <TableCell className="text-right">
                      {action?.id === w.id ? (
                        <div className="flex items-center justify-end gap-1">
                          <Input
                            value={text}
                            onChange={(e) => setText(e.target.value)}
                            placeholder={
                              action.kind === "reject"
                                ? t("withdraw.rejectReason")
                                : t("withdraw.paidReference")
                            }
                            className="h-8 w-40"
                          />
                          <Button size="sm" onClick={() => submitAction(w.id)}>
                            {t("ops.apply")}
                          </Button>
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={() => {
                              setAction(null);
                              setText("");
                            }}
                          >
                            {t("common.cancel")}
                          </Button>
                        </div>
                      ) : (
                        <div className="flex justify-end gap-1">
                          {w.status === "requested" && (
                            <>
                              <Button
                                size="sm"
                                disabled={approve.isPending}
                                onClick={() => approve.mutate({ id: w.id })}
                              >
                                {t("withdraw.approve")}
                              </Button>
                              <Button
                                size="sm"
                                variant="outline"
                                onClick={() => setAction({ id: w.id, kind: "reject" })}
                              >
                                {t("withdraw.reject")}
                              </Button>
                            </>
                          )}
                          {w.status === "approved" && (
                            <Button
                              size="sm"
                              onClick={() => setAction({ id: w.id, kind: "pay" })}
                            >
                              {t("withdraw.markPaid")}
                            </Button>
                          )}
                        </div>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
                {data && data.items.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={7} className="text-muted-foreground text-center text-sm">
                      {t("withdraw.empty")}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      <Pagination
        page={data?.page ?? page}
        pageSize={data?.pageSize ?? 20}
        total={data?.total ?? 0}
        onChange={setPage}
      />
    </div>
  );
}
