import { Link, useNavigate, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { ArrowLeft } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { OrderStatusBadge } from "@/components/order-status-badge";
import { api } from "@/lib/api";
import { formatDate, formatMoney } from "@/lib/format";
import type { Order, Payment } from "@/lib/types";

export function OrderDetailPage() {
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
    onError: (error: Error) => toast.error(error.message),
  });

  const cancel = useMutation({
    mutationFn: () => api.post<Order>(`/orders/${id}/cancel`),
    onSuccess: () => {
      toast.success("Order cancelled");
      void queryClient.invalidateQueries({ queryKey: ["order", id] });
      void queryClient.invalidateQueries({ queryKey: ["orders"] });
    },
    onError: (error: Error) => toast.error(error.message),
  });

  const confirmReceipt = useMutation({
    mutationFn: () => api.post<Order>(`/orders/${id}/complete`),
    onSuccess: () => {
      toast.success("Thanks for confirming delivery");
      void queryClient.invalidateQueries({ queryKey: ["order", id] });
      void queryClient.invalidateQueries({ queryKey: ["orders"] });
    },
    onError: (error: Error) => toast.error(error.message),
  });

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
        <p className="text-muted-foreground">Order not found.</p>
        <Button variant="link" asChild>
          <Link to="/orders">Back to orders</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <Button variant="ghost" size="sm" asChild className="-ml-2">
        <Link to="/orders">
          <ArrowLeft className="size-4" />
          Back to orders
        </Link>
      </Button>

      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold">{order.orderNo}</h1>
            <OrderStatusBadge status={order.status} />
          </div>
          <p className="text-muted-foreground mt-1 text-sm">
            Placed {formatDate(order.createdAt)}
            {order.paidAt ? ` · Paid ${formatDate(order.paidAt)}` : ""}
          </p>
        </div>

        {order.status === "pending_payment" && (
          <div className="flex gap-2">
            <Button
              variant="outline"
              disabled={cancel.isPending}
              onClick={() => cancel.mutate()}
            >
              Cancel
            </Button>
            <Button disabled={payNow.isPending} onClick={() => payNow.mutate()}>
              {payNow.isPending ? "Redirecting…" : "Pay now"}
            </Button>
          </div>
        )}
        {order.status === "shipped" && (
          <Button disabled={confirmReceipt.isPending} onClick={() => confirmReceipt.mutate()}>
            {confirmReceipt.isPending ? "Confirming…" : "Confirm receipt"}
          </Button>
        )}
      </div>

      <div className="grid gap-6 lg:grid-cols-[1fr_320px]">
        <Card>
          <CardHeader>
            <CardTitle>Items</CardTitle>
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
            <CardTitle>Summary</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">Subtotal</span>
              <span>{formatMoney(order.subtotalCents, order.currency)}</span>
            </div>
            {order.discountCents > 0 && (
              <div className="flex justify-between text-sm text-emerald-600">
                <span>Discount{order.couponCode ? ` (${order.couponCode})` : ""}</span>
                <span>-{formatMoney(order.discountCents, order.currency)}</span>
              </div>
            )}
            {order.shippingCents > 0 && (
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">
                  Shipping{order.shippingMethod ? ` (${order.shippingMethod})` : ""}
                </span>
                <span>{formatMoney(order.shippingCents, order.currency)}</span>
              </div>
            )}
            {order.taxCents > 0 && (
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">Tax</span>
                <span>{formatMoney(order.taxCents, order.currency)}</span>
              </div>
            )}
            <Separator />
            <div className="flex justify-between font-semibold">
              <span>Total</span>
              <span>{formatMoney(order.totalCents, order.currency)}</span>
            </div>

            {order.shippingAddress && (
              <>
                <Separator />
                <div className="space-y-1 text-sm">
                  <p className="font-medium">Shipping to</p>
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
                <span className="text-muted-foreground">Tracking</span>
                <span className="font-mono text-xs">{order.trackingNo}</span>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
