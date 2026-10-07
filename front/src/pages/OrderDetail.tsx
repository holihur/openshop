import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { ArrowLeft, Download } from "lucide-react";
import { errorMessage } from "@lib/errors";
import { Button } from "@lib/components/ui/button";
import { Badge } from "@lib/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@lib/components/ui/card";
import { Separator } from "@lib/components/ui/separator";
import { Skeleton } from "@lib/components/ui/skeleton";
import { OrderStatusBadge } from "@lib/components/order-status-badge";
import { api } from "@lib/api";
import { formatDate, formatMoney } from "@lib/format";
import { useI18n } from "@lib/i18n";
import { useOrderReturns, useRequestReturn } from "@lib/hooks/useReturns";
import type { Order, Payment } from "@lib/types";

export function OrderDetailPage() {
  const { t } = useI18n();
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const { data: order, isLoading } = useQuery({
    queryKey: ["order", id],
    queryFn: () => api.get<Order>(`/orders/${id}`),
    enabled: Boolean(id),
  });

  const payNow = useMutation({
    mutationFn: () =>
      api.post<Payment>("/payments", {
        orderId: id,
        provider: "mock",
        returnUrl: `${window.location.origin}/payment/result`,
      }),
    onSuccess: (payment) => {
      if (payment.redirectUrl) window.location.href = payment.redirectUrl;
      else void navigate(`/orders/${id}`);
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });

  const cancel = useMutation({
    mutationFn: () => api.post<Order>(`/orders/${id}/cancel`),
    onSuccess: () => {
      toast.success(t("orders.cancelled"));
      void queryClient.invalidateQueries({ queryKey: ["order", id] });
      void queryClient.invalidateQueries({ queryKey: ["orders"] });
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });

  const confirmReceipt = useMutation({
    mutationFn: () => api.post<Order>(`/orders/${id}/complete`),
    onSuccess: () => {
      toast.success(t("orders.receiptConfirmed"));
      void queryClient.invalidateQueries({ queryKey: ["order", id] });
      void queryClient.invalidateQueries({ queryKey: ["orders"] });
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });

  const [downloading, setDownloading] = useState(false);
  const { data: returns } = useOrderReturns(id);
  const requestReturn = useRequestReturn(id);
  const openReturn = returns?.find((r) => r.status === "requested" || r.status === "approved");
  const latestReturn = returns?.[0];
  async function downloadInvoice() {
    if (!order) return;
    setDownloading(true);
    try {
      await api.download(`/orders/${order.id}/invoice`, `invoice-${order.orderNo}.pdf`);
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setDownloading(false);
    }
  }

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (!order) {
    return (
      <div className="py-16 text-center">
        <p className="text-muted-foreground">{t("orders.notFound")}</p>
        <Button variant="link" asChild>
          <Link to="/orders">{t("orders.backToOrders")}</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <Button variant="ghost" size="sm" asChild className="-ml-2">
        <Link to="/orders">
          <ArrowLeft className="size-4" />
          {t("orders.backToOrders")}
        </Link>
      </Button>

      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold">{order.orderNo}</h1>
            <OrderStatusBadge status={order.status} />
          </div>
          <p className="text-muted-foreground mt-1 text-sm">
            {t("orders.placedAt", { date: formatDate(order.createdAt) })}
            {order.paidAt ? ` · ${t("orders.paidAt", { date: formatDate(order.paidAt) })}` : ""}
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {order.status === "pending_payment" && (
            <>
              <Button
                variant="outline"
                disabled={cancel.isPending}
                onClick={() => cancel.mutate()}
              >
                {t("orders.cancel")}
              </Button>
              <Button disabled={payNow.isPending} onClick={() => payNow.mutate()}>
                {payNow.isPending ? t("orders.paying") : t("orders.payNow")}
              </Button>
            </>
          )}
          {order.status === "shipped" && (
            <Button disabled={confirmReceipt.isPending} onClick={() => confirmReceipt.mutate()}>
              {confirmReceipt.isPending ? t("orders.confirming") : t("orders.confirmReceipt")}
            </Button>
          )}
          {order.status === "completed" && !openReturn && (
            <Button
              variant="outline"
              disabled={requestReturn.isPending}
              onClick={() => {
                const reason = window.prompt(t("orders.returnReason"), "");
                if (reason !== null) requestReturn.mutate(reason);
              }}
            >
              {t("orders.requestReturn")}
            </Button>
          )}
          {latestReturn && (
            <Badge variant="secondary">{t(`return.${latestReturn.status}`)}</Badge>
          )}
          <Button variant="outline" disabled={downloading} onClick={downloadInvoice}>
            <Download className="size-4" />
            {t("orders.invoice")}
          </Button>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-[1fr_320px]">
        <Card>
          <CardHeader>
            <CardTitle>{t("orders.items")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {order.items.map((item) => (
              <div key={item.id} className="flex items-center justify-between gap-4">
                <div className="min-w-0">
                  <Link
                    to={`/products/${item.productId}`}
                    className="font-medium hover:underline"
                  >
                    {item.title}
                  </Link>
                  {item.variantName && (
                    <p className="text-muted-foreground text-xs">{item.variantName}</p>
                  )}
                  <p className="text-muted-foreground text-sm">
                    {formatMoney(item.priceCents, order.currency)} × {item.quantity}
                  </p>
                </div>
                <span className="font-medium">
                  {formatMoney(item.subtotal, order.currency)}
                </span>
              </div>
            ))}
          </CardContent>
        </Card>

        <Card className="h-fit">
          <CardHeader>
            <CardTitle>{t("orders.summary")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">{t("cart.subtotal")}</span>
              <span>{formatMoney(order.subtotalCents, order.currency)}</span>
            </div>
            {order.discountCents > 0 && (
              <div className="flex justify-between text-sm text-emerald-600">
                <span>
                  {order.couponCode ? t("orders.discountCode", { code: order.couponCode }) : t("cart.discount")}
                </span>
                <span>-{formatMoney(order.discountCents, order.currency)}</span>
              </div>
            )}
            {order.shippingCents > 0 && (
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">
                  {order.shippingMethod ? t("orders.shippingWithMethod", { method: order.shippingMethod }) : t("cart.shipping")}
                </span>
                <span>{formatMoney(order.shippingCents, order.currency)}</span>
              </div>
            )}
            {order.taxCents > 0 && (
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">{t("cart.tax")}</span>
                <span>{formatMoney(order.taxCents, order.currency)}</span>
              </div>
            )}
            <Separator />
            <div className="flex justify-between font-semibold">
              <span>{t("cart.total")}</span>
              <span>{formatMoney(order.totalCents, order.currency)}</span>
            </div>
            {order.refundedCents > 0 && (
              <div className="flex justify-between text-sm text-emerald-600">
                <span>{t("orders.refunded")}</span>
                <span>-{formatMoney(order.refundedCents, order.currency)}</span>
              </div>
            )}

            {order.shippingAddress && (
              <>
                <Separator />
                <div className="space-y-1 text-sm">
                  <p className="font-medium">{t("orders.shippingTo")}</p>
                  <p>
                    {order.shippingAddress.recipient}
                    {order.shippingAddress.phone ? ` · ${order.shippingAddress.phone}` : ""}
                  </p>
                  <p className="text-muted-foreground">
                    {[
                      order.shippingAddress.province,
                      order.shippingAddress.city,
                      order.shippingAddress.district,
                      order.shippingAddress.line1,
                      order.shippingAddress.postalCode,
                    ]
                      .filter(Boolean)
                      .join(" ")}
                  </p>
                </div>
              </>
            )}
            {order.trackingNo && (
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">{t("orders.tracking")}</span>
                <span className="font-mono text-xs">{order.trackingNo}</span>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
