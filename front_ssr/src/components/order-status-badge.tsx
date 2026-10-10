import type { Locale } from "@/lib/i18n";

const TONES: Record<string, string> = {
  pending_payment: "bg-amber-100 text-amber-900 dark:bg-amber-500/20 dark:text-amber-200",
  paid: "bg-emerald-100 text-emerald-900 dark:bg-emerald-500/20 dark:text-emerald-200",
  shipped: "bg-sky-100 text-sky-900 dark:bg-sky-500/20 dark:text-sky-200",
  completed: "bg-emerald-100 text-emerald-900 dark:bg-emerald-500/20 dark:text-emerald-200",
  cancelled: "bg-muted text-muted-foreground",
  refunded: "bg-rose-100 text-rose-900 dark:bg-rose-500/20 dark:text-rose-200",
};

const LABELS: Record<string, { en: string; zh: string }> = {
  pending_payment: { en: "Awaiting payment", zh: "待付款" },
  paid: { en: "Paid", zh: "已付款" },
  shipped: { en: "Shipped", zh: "已发货" },
  completed: { en: "Completed", zh: "已完成" },
  cancelled: { en: "Cancelled", zh: "已取消" },
  refunded: { en: "Refunded", zh: "已退款" },
};

/** A colour-coded order status, readable without knowing the enum values. */
export function OrderStatusBadge({ status, locale }: { status: string; locale: Locale }) {
  const label = LABELS[status]?.[locale] ?? status;
  return (
    <span className={`rounded-full px-2.5 py-0.5 text-xs font-medium ${TONES[status] ?? "bg-muted"}`}>
      {label}
    </span>
  );
}
