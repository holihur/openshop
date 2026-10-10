import { AlertTriangle, CheckCircle2 } from "lucide-react";

import { Badge } from "@lib/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@lib/components/ui/card";
import { Skeleton } from "@lib/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@lib/components/ui/table";
import { formatMoney } from "@lib/format";
import { useI18n } from "@lib/i18n";
import { useReconciliation } from "@lib/hooks/useAdmin";

/**
 * Money reconciliation. The checks compare each stored balance with the ledger
 * that produced it, so a discrepancy is visible here rather than discovered when
 * a shopper complains. Nothing is corrected automatically: the audit entry and
 * the number are the product.
 */
export function ReconciliationPage() {
  const { t } = useI18n();
  const { data, isLoading } = useReconciliation();

  if (isLoading) return <Skeleton className="h-64 w-full" />;

  const clean = data?.clean ?? true;
  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold">{t("ops.reconciliation")}</h1>
          <p className="text-muted-foreground text-sm">{t("ops.reconciliationSubtitle")}</p>
        </div>
        <Badge variant={clean ? "success" : "destructive"}>
          {clean ? (
            <>
              <CheckCircle2 className="mr-1 size-3.5" />
              {t("ops.reconciled")}
            </>
          ) : (
            <>
              <AlertTriangle className="mr-1 size-3.5" />
              {t("ops.mismatches", { count: data?.mismatches ?? 0 })}
            </>
          )}
        </Badge>
      </div>

      {(data?.walletDrift?.length ?? 0) > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>{t("ops.walletDrift")}</CardTitle>
          </CardHeader>
          <CardContent className="px-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("wallet.title")}</TableHead>
                  <TableHead>{t("ops.stored")}</TableHead>
                  <TableHead>{t("ops.ledger")}</TableHead>
                  <TableHead>{t("ops.difference")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data?.walletDrift?.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell className="font-mono text-xs">{row.ownerId}</TableCell>
                    <TableCell>{formatMoney(row.stored)}</TableCell>
                    <TableCell>{formatMoney(row.expected)}</TableCell>
                    <TableCell className="text-destructive">
                      {formatMoney(row.stored - row.expected)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {(data?.pointsDrift?.length ?? 0) > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>{t("ops.pointsDrift")}</CardTitle>
          </CardHeader>
          <CardContent className="px-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("ops.customer")}</TableHead>
                  <TableHead>{t("ops.stored")}</TableHead>
                  <TableHead>{t("ops.ledger")}</TableHead>
                  <TableHead>{t("ops.difference")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data?.pointsDrift?.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell className="font-mono text-xs">{row.ownerId}</TableCell>
                    <TableCell>{row.stored}</TableCell>
                    <TableCell>{row.expected}</TableCell>
                    <TableCell className="text-destructive">{row.stored - row.expected}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {(data?.orderDrift?.length ?? 0) > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>{t("ops.orderDrift")}</CardTitle>
          </CardHeader>
          <CardContent className="px-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("orders.orderNo")}</TableHead>
                  <TableHead>{t("ops.expected")}</TableHead>
                  <TableHead>{t("ops.collected")}</TableHead>
                  <TableHead>{t("ops.difference")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data?.orderDrift?.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell className="font-mono text-xs">{row.orderNo}</TableCell>
                    <TableCell>{formatMoney(row.expectedCents)}</TableCell>
                    <TableCell>{formatMoney(row.actualCents)}</TableCell>
                    <TableCell className="text-destructive">
                      {formatMoney(row.actualCents - row.expectedCents)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {clean && (
        <Card>
          <CardContent className="text-muted-foreground flex items-center gap-2 py-6 text-sm">
            <CheckCircle2 className="size-4" />
            {t("ops.reconciliationClean")}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
