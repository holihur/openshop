import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Package } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { Card, CardContent } from "@lib/components/ui/card";
import { Skeleton } from "@lib/components/ui/skeleton";
import { OrderStatusBadge } from "@lib/components/order-status-badge";
import { api } from "@lib/api";
import { formatDate, formatMoney } from "@lib/format";
import { useI18n } from "@lib/i18n";
import type { Order } from "@lib/types";

export function OrdersPage() {
  const { t } = useI18n();
  const { data, isLoading } = useQuery({
    queryKey: ["orders"],
    queryFn: () => api.getPage<Order[]>("/orders?pageSize=20"),
  });

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-40" />
        <div className="space-y-3">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="flex flex-wrap items-center justify-between gap-4 rounded-xl border p-6">
              <div className="space-y-2">
                <Skeleton className="h-4 w-40" />
                <Skeleton className="h-3 w-28" />
              </div>
              <Skeleton className="h-8 w-24" />
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (!data || data.items.length === 0) {
    return (
      <div className="py-20 text-center">
        <Package className="text-muted-foreground mx-auto size-10" />
        <p className="mt-4 text-lg font-medium">{t("orders.noOrders")}</p>
        <Button className="mt-6" asChild>
          <Link to="/products">{t("orders.startShopping")}</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t("orders.yourOrders")}</h1>
      <div className="space-y-3">
        {data.items.map((order) => (
          <Card key={order.id}>
            <CardContent className="flex flex-wrap items-center justify-between gap-4">
              <div className="min-w-0">
                <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
                  <Link to={`/orders/${order.id}`} className="font-medium hover:underline">
                    {order.orderNo}
                  </Link>
                  <OrderStatusBadge status={order.status} />
                </div>
                <p className="text-muted-foreground mt-1 text-sm">
                  {formatDate(order.createdAt)} · {t("orders.itemCount", { count: order.items.length })}
                </p>
              </div>
              <div className="flex items-center gap-4">
                <span className="font-semibold">
                  {formatMoney(order.totalCents, order.currency)}
                </span>
                <Button variant="outline" size="sm" asChild>
                  <Link to={`/orders/${order.id}`}>{t("orders.view")}</Link>
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
