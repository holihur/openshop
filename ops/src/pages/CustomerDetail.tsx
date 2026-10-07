import { Link, useParams } from "react-router-dom";
import { ArrowLeft } from "lucide-react";

import { Badge } from "@lib/components/ui/badge";
import { Button } from "@lib/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@lib/components/ui/card";
import { Skeleton } from "@lib/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@lib/components/ui/table";
import { OrderStatusBadge } from "@lib/components/order-status-badge";
import { useI18n } from "@lib/i18n";
import { useAdminCustomer, useCustomerOrders, useUpdateCustomer } from "@lib/hooks/useAdmin";
import { formatDate, formatMoney } from "@lib/format";

/** Customer detail: profile, enable/disable and order history. */
export function CustomerDetailPage() {
  const { id = "" } = useParams();
  const { t } = useI18n();
  const { data: customer, isLoading } = useAdminCustomer(id);
  const { data: orders } = useCustomerOrders(id);
  const update = useUpdateCustomer();

  if (isLoading) {
    return <Skeleton className="h-64 w-full" />;
  }
  if (!customer) {
    return <p className="text-muted-foreground">{t("ops.customerNotFound")}</p>;
  }

  const disabled = customer.status === "disabled";

  return (
    <div className="space-y-4">
      <Button variant="ghost" size="sm" asChild>
        <Link to="/customers">
          <ArrowLeft className="size-4" />
          {t("ops.backToCustomers")}
        </Link>
      </Button>

      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>{customer.name || customer.email}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-1 text-sm">
            <Row label={t("ops.email")} value={customer.email} />
            <Row label={t("ops.phone")} value={customer.phone || "—"} />
            <Row label={t("ops.registeredAt")} value={formatDate(customer.createdAt)} />
            <Row
              label={t("ops.emailVerified")}
              value={customer.emailVerified ? t("common.yes") : t("common.no")}
            />
            <div className="pt-1">
              <Badge variant={disabled ? "secondary" : "success"}>
                {disabled ? t("common.inactive") : t("common.active")}
              </Badge>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{t("common.actions")}</CardTitle>
          </CardHeader>
          <CardContent>
            <Button
              variant={disabled ? "default" : "outline"}
              size="sm"
              disabled={update.isPending}
              onClick={() =>
                update.mutate({
                  id: customer.id,
                  input: { status: disabled ? "active" : "disabled" },
                })
              }
            >
              {disabled ? t("ops.enableCustomer") : t("ops.disableCustomer")}
            </Button>
          </CardContent>
        </Card>
      </div>

      <Card className="py-0">
        <CardContent className="px-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("orders.orderNo")}</TableHead>
                <TableHead>{t("common.status")}</TableHead>
                <TableHead>{t("orders.total")}</TableHead>
                <TableHead>{t("orders.placed")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {orders?.items.map((o) => (
                <TableRow key={o.id}>
                  <TableCell className="font-mono text-xs">
                    <Link to={`/orders/${o.id}`} className="hover:underline">
                      {o.orderNo}
                    </Link>
                  </TableCell>
                  <TableCell>
                    <OrderStatusBadge status={o.status} />
                  </TableCell>
                  <TableCell>{formatMoney(o.totalCents, o.currency)}</TableCell>
                  <TableCell className="text-muted-foreground text-sm">
                    {formatDate(o.createdAt)}
                  </TableCell>
                </TableRow>
              ))}
              {orders && orders.items.length === 0 && (
                <TableRow>
                  <TableCell colSpan={4} className="text-muted-foreground text-center text-sm">
                    {t("ops.noOrders")}
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-4">
      <span className="text-muted-foreground">{label}</span>
      <span className="truncate">{value}</span>
    </div>
  );
}
