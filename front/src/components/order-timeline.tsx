import { CheckCircle2, Circle, CreditCard, Package, Truck } from "lucide-react";

import { useI18n } from "@lib/i18n";
import { formatDate } from "@lib/format";
import { cn } from "@lib/utils";
import type { Order } from "@lib/types";

/** Fulfilment timeline for an order (placed → paid → shipped → completed). */
export function OrderTimeline({ order }: { order: Order }) {
  const { t } = useI18n();

  if (order.status === "cancelled") {
    return <p className="text-muted-foreground text-sm">{t("track.cancelled")}</p>;
  }

  const steps = [
    { key: "placed", label: t("track.placed"), at: order.createdAt, done: true, icon: Package },
    { key: "paid", label: t("track.paid"), at: order.paidAt, done: Boolean(order.paidAt), icon: CreditCard },
    { key: "shipped", label: t("track.shipped"), at: order.shippedAt, done: Boolean(order.shippedAt), icon: Truck },
    {
      key: "completed",
      label: t("track.completed"),
      at: order.completedAt,
      done: Boolean(order.completedAt),
      icon: CheckCircle2,
    },
  ];

  return (
    <ol className="space-y-4">
      {steps.map((step) => {
        const Icon = step.icon;
        return (
          <li key={step.key} className="flex items-start gap-3">
            <span
              className={cn(
                "mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-full border",
                step.done
                  ? "bg-primary text-primary-foreground border-primary"
                  : "text-muted-foreground",
              )}
            >
              {step.done ? <Icon className="size-4" /> : <Circle className="size-3" />}
            </span>
            <div className="min-w-0">
              <p className={cn("text-sm font-medium", !step.done && "text-muted-foreground")}>
                {step.label}
              </p>
              {step.done && step.at ? (
                <p className="text-muted-foreground text-xs">{formatDate(step.at)}</p>
              ) : null}
              {step.key === "shipped" && order.trackingNo ? (
                <p className="text-muted-foreground text-xs">
                  {t("orders.tracking")}: <span className="font-mono">{order.trackingNo}</span>
                </p>
              ) : null}
            </div>
          </li>
        );
      })}
    </ol>
  );
}
