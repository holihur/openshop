import { Badge } from "@lib/components/ui/badge";
import { useI18n } from "@lib/i18n";
import type { OrderStatus } from "@lib/types";

const variants: Record<OrderStatus, "default" | "secondary" | "success" | "warning" | "destructive"> = {
  pending_payment: "warning",
  paid: "success",
  cancelled: "secondary",
  shipped: "default",
  completed: "success",
  refunded: "destructive",
};

export function OrderStatusBadge({ status }: { status: OrderStatus }) {
  const { t } = useI18n();
  return <Badge variant={variants[status]}>{t(`status.${status}`)}</Badge>;
}
