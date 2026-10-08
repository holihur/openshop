import { useState } from "react";
import { Link } from "react-router-dom";

import { Badge } from "@lib/components/ui/badge";
import { Card, CardContent } from "@lib/components/ui/card";
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
import { useAdminCommissions } from "@lib/hooks/useLoyalty";
import { useI18n } from "@lib/i18n";
import { formatDate, formatMoney } from "@lib/format";
import { OpsModuleStats } from "@/components/ops-stats";
import type { CommissionStatus } from "@lib/types";

const STATUS_VARIANT: Record<CommissionStatus, "warning" | "success" | "secondary"> = {
  pending: "warning",
  approved: "success",
  reversed: "secondary",
};

export function CommissionsPage() {
  const { t } = useI18n();
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState("");
  const { data, isLoading } = useAdminCommissions({ page, pageSize: 20, status });

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold">{t("ops.commissions")}</h1>
      </div>

      <OpsModuleStats module="commissions" />

      <select
        aria-label={t("ops.commissionStatus")}
        value={status}
        onChange={(e) => {
          setStatus(e.target.value);
          setPage(1);
        }}
        className="border-input bg-background h-9 rounded-md border px-3 text-sm"
      >
        <option value="">{t("ops.allStatuses")}</option>
        {(["pending", "approved", "reversed"] as const).map((s) => (
          <option key={s} value={s}>
            {t(`commission.status.${s}`)}
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
                  <TableHead>{t("orders.orderNo")}</TableHead>
                  <TableHead>{t("ops.ticketCustomer")}</TableHead>
                  <TableHead className="text-right">{t("orders.total")}</TableHead>
                  <TableHead className="text-right">{t("rewards.commissions")}</TableHead>
                  <TableHead>{t("ops.commissionStatus")}</TableHead>
                  <TableHead>{t("ops.date")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data?.items.map((c) => (
                  <TableRow key={c.id}>
                    <TableCell className="font-mono text-xs">
                      <Link to={`/orders/${c.orderId}`} className="hover:underline">
                        {c.orderId.slice(0, 8)}
                      </Link>
                    </TableCell>
                    <TableCell className="font-mono text-xs">
                      {c.referrerId.slice(0, 8)}
                    </TableCell>
                    <TableCell className="text-right">{formatMoney(c.baseCents)}</TableCell>
                    <TableCell className="text-right font-medium">
                      {formatMoney(c.amountCents)}
                      <span className="text-muted-foreground ml-1 text-xs">
                        ({(c.rateBps / 100).toFixed(2)}%)
                      </span>
                    </TableCell>
                    <TableCell>
                      <Badge variant={STATUS_VARIANT[c.status] ?? "secondary"}>
                        {t(`commission.status.${c.status}`)}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {c.status === "pending"
                        ? t("rewards.availableOn", { date: formatDate(c.holdUntil) })
                        : formatDate(c.approvedAt ?? c.createdAt)}
                    </TableCell>
                  </TableRow>
                ))}
                {data && data.items.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={6} className="text-muted-foreground text-center text-sm">
                      {t("rewards.noCommissions")}
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
