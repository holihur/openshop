import { Badge } from "@lib/components/ui/badge";
import type { OrderStatus } from "@lib/types";

const labels: Record<OrderStatus, string> = {
  pending_payment: "Pending payment",
  paid: "Paid",
  cancelled: "Cancelled",
  shipped: "Shipped",
  completed: "Completed",
  refunded: "Refunded",
};

const variants: Record<OrderStatus, "default" | "secondary" | "success" | "warning" | "destructive"> = {
  pending_payment: "warning",
  paid: "success",
  cancelled: "secondary",
  shipped: "default",
  completed: "success",
  refunded: "destructive",
};

export function OrderStatusBadge({ status }: { status: OrderStatus }) {
  return <Badge variant={variants[status]}>{labels[status] ?? status}</Badge>;
}
