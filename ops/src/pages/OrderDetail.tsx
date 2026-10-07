import { Link, useParams } from "react-router-dom";
import { ArrowLeft, Download, RotateCcw, Truck } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@lib/components/ui/card";
import { Skeleton } from "@lib/components/ui/skeleton";
import { OrderStatusBadge } from "@lib/components/order-status-badge";
import { useI18n } from "@lib/i18n";
import { api } from "@lib/api";
import { formatDate, formatMoney } from "@lib/format";
import {
  useAdminOrder,
  useCompleteOrder,
  useRefundOrder,
  useShipOrder,
} from "@lib/hooks/useAdmin";

/** A deep-linkable ops order page: items, money, fulfilment and actions. */
export function OrderDetailPage() {
  const { id = "" } = useParams();
  const { t } = useI18n();
  const { data: order, isLoading } = useAdminOrder(id);
  const ship = useShipOrder();
  const complete = useCompleteOrder();
  const refund = useRefundOrder();

  if (isLoading) {
    return <Skeleton className="h-64 w-full" />;
  }
  if (!order) {
    return <p className="text-muted-foreground">{t("orders.notFound")}</p>;
  }

  const refundable = order.totalCents - order.refundedCents;

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="sm" asChild>
            <Link to="/orders">
              <ArrowLeft className="size-4" />
              {t("orders.backToOrders")}
            </Link>
          </Button>
          <h1 className="font-mono text-lg font-semibold">{order.orderNo}</h1>
          <OrderStatusBadge status={order.status} />
        </div>
        <div className="flex flex-wrap gap-2">
          {order.status === "paid" && (
            <Button
              size="sm"
              disabled={ship.isPending}
              onClick={() => {
                const trackingNo = window.prompt(t("ops.trackingPrompt"), "");
                if (trackingNo !== null) ship.mutate({ id: order.id, trackingNo });
              }}
            >
              <Truck className="size-4" />
              {t("ops.ship")}
            </Button>
          )}
          {order.status === "shipped" && (
            <Button size="sm" disabled={complete.isPending} onClick={() => complete.mutate(order.id)}>
              {t("ops.complete")}
            </Button>
          )}
          {["paid", "shipped", "completed"].includes(order.status) && refundable > 0 && (
            <Button
              variant="outline"
              size="sm"
              disabled={refund.isPending}
              onClick={() => {
                const v = window.prompt(t("ops.refundPrompt"), (refundable / 100).toFixed(2));
                if (v === null) return;
                const amountCents = Math.round(Number.parseFloat(v || "0") * 100);
                const restock = window.confirm(t("ops.refundRestock"));
                refund.mutate({ id: order.id, amountCents, restock });
              }}
            >
              <RotateCcw className="size-4" />
              {t("ops.refund")}
            </Button>
          )}
          {["paid", "shipped", "completed", "refunded"].includes(order.status) && (
            <Button
              variant="outline"
              size="sm"
              onClick={() =>
                void api.download(`/ops/orders/${order.id}/invoice`, `invoice-${order.orderNo}.pdf`)
              }
            >
              <Download className="size-4" />
              {t("orders.invoice")}
            </Button>
          )}
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <Card className="md:col-span-2">
          <CardHeader>
            <CardTitle>{t("orders.items")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            {order.items.map((item) => (
              <div key={item.id} className="flex items-center justify-between text-sm">
                <span>
                  {item.title}
                  {item.variantName ? ` · ${item.variantName}` : ""} × {item.quantity}
                </span>
                <span>{formatMoney(item.subtotal, order.currency)}</span>
              </div>
            ))}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{t("cart.total")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-1 text-sm">
            <Row label={t("cart.subtotal")} value={formatMoney(order.subtotalCents, order.currency)} />
            {order.discountCents > 0 && (
              <Row
                label={t("cart.discount")}
                value={`-${formatMoney(order.discountCents, order.currency)}`}
              />
            )}
            <Row label={t("cart.shipping")} value={formatMoney(order.shippingCents, order.currency)} />
            <Row label={t("cart.tax")} value={formatMoney(order.taxCents, order.currency)} />
            <div className="flex justify-between border-t pt-1 font-medium">
              <span>{t("cart.total")}</span>
              <span>{formatMoney(order.totalCents, order.currency)}</span>
            </div>
            {order.refundedCents > 0 && (
              <div className="text-destructive flex justify-between">
                <span>{t("orders.refunded")}</span>
                <span>-{formatMoney(order.refundedCents, order.currency)}</span>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>{t("orders.shippingTo")}</CardTitle>
          </CardHeader>
          <CardContent className="text-sm">
            {order.shippingAddress ? (
              <>
                <p>
                  {order.shippingAddress.recipient} · {order.shippingAddress.phone}
                </p>
                <p className="text-muted-foreground">
                  {order.shippingAddress.province} {order.shippingAddress.city}{" "}
                  {order.shippingAddress.district} {order.shippingAddress.line1}
                </p>
              </>
            ) : (
              "—"
            )}
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>{t("ops.fulfilment")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-1 text-sm">
            <Row label={t("orders.placedAt")} value={formatDate(order.createdAt)} />
            {order.paidAt && <Row label={t("orders.paidAt")} value={formatDate(order.paidAt)} />}
            {order.shippedAt && <Row label={t("status.shipped")} value={formatDate(order.shippedAt)} />}
            {order.trackingNo && <Row label={t("orders.tracking")} value={order.trackingNo} />}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between">
      <span className="text-muted-foreground">{label}</span>
      <span>{value}</span>
    </div>
  );
}
