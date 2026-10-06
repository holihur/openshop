import { Link, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { Button } from "@lib/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@lib/components/ui/card";
import { Separator } from "@lib/components/ui/separator";
import { Skeleton } from "@lib/components/ui/skeleton";
import { OrderStatusBadge } from "@lib/components/order-status-badge";
import { api } from "@lib/api";
import { formatDate, formatMoney } from "@lib/format";
import type { Order, Payment } from "@lib/types";

// GuestOrderPage lets an anonymous buyer view, pay, cancel or confirm their
// order using the access token returned at checkout.
export function GuestOrderPage() {
  const { token = "" } = useParams();
  const queryClient = useQueryClient();

  const { data: order, isLoading } = useQuery({
    queryKey: ["guest-order", token],
    queryFn: () => api.get<Order>(`/guest/orders/${token}`),
    enabled: Boolean(token),
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["guest-order", token] });

  const pay = useMutation({
    mutationFn: () =>
      api.post<Payment>(`/guest/orders/${token}/pay`, {
        provider: "mock",
        returnUrl: `${window.location.origin}/payment/result`,
      }),
    onSuccess: (payment) => {
      if (payment.redirectUrl) window.location.href = payment.redirectUrl;
      else void invalidate();
    },
    onError: (error: Error) => toast.error(error.message),
  });

  const cancel = useMutation({
    mutationFn: () => api.post<Order>(`/guest/orders/${token}/cancel`),
    onSuccess: () => {
      toast.success("Order cancelled");
      void invalidate();
    },
    onError: (error: Error) => toast.error(error.message),
  });

  const confirmReceipt = useMutation({
    mutationFn: () => api.post<Order>(`/guest/orders/${token}/complete`),
    onSuccess: () => {
      toast.success("Thanks for confirming delivery");
      void invalidate();
    },
    onError: (error: Error) => toast.error(error.message),
  });

  if (isLoading) {
    return <Skeleton className="h-64 w-full" />;
  }
  if (!order) {
    return (
      <div className="py-16 text-center">
        <p className="text-muted-foreground">Order not found.</p>
        <Button variant="link" asChild>
          <Link to="/">Back to home</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold">{order.orderNo}</h1>
            <OrderStatusBadge status={order.status} />
          </div>
          <p className="text-muted-foreground mt-1 text-sm">Placed {formatDate(order.createdAt)}</p>
        </div>
        <div className="flex gap-2">
          {order.status === "pending_payment" && (
            <>
              <Button variant="outline" disabled={cancel.isPending} onClick={() => cancel.mutate()}>
                Cancel
              </Button>
              <Button disabled={pay.isPending} onClick={() => pay.mutate()}>
                {pay.isPending ? "Redirecting…" : "Pay now"}
              </Button>
            </>
          )}
          {order.status === "shipped" && (
            <Button disabled={confirmReceipt.isPending} onClick={() => confirmReceipt.mutate()}>
              Confirm receipt
            </Button>
          )}
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Items</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {order.items.map((item) => (
            <div key={item.id} className="flex items-center justify-between text-sm">
              <span>
                {item.title}
                {item.variantName ? ` · ${item.variantName}` : ""} × {item.quantity}
              </span>
              <span>{formatMoney(item.subtotal, order.currency)}</span>
            </div>
          ))}
          <Separator />
          <div className="flex justify-between text-sm">
            <span className="text-muted-foreground">Subtotal</span>
            <span>{formatMoney(order.subtotalCents, order.currency)}</span>
          </div>
          {order.shippingCents > 0 && (
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">Shipping</span>
              <span>{formatMoney(order.shippingCents, order.currency)}</span>
            </div>
          )}
          {order.taxCents > 0 && (
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">Tax</span>
              <span>{formatMoney(order.taxCents, order.currency)}</span>
            </div>
          )}
          <div className="flex justify-between font-semibold">
            <span>Total</span>
            <span>{formatMoney(order.totalCents, order.currency)}</span>
          </div>
          {order.trackingNo && (
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">Tracking</span>
              <span className="font-mono text-xs">{order.trackingNo}</span>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
