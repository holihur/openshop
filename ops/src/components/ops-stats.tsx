import type { ReactNode } from "react";

import { formatMoney } from "@lib/format";
import { useI18n } from "@lib/i18n";
import { useOpsSummary } from "@lib/hooks/useOpsSummary";
import type { PeriodCounts } from "@lib/types";

export type OpsModule =
  | "orders"
  | "products"
  | "customers"
  | "coupons"
  | "tickets"
  | "returns"
  | "withdrawals"
  | "reviews"
  | "commissions";

type Tone = "default" | "warn" | "danger" | "success";

const TONE: Record<Tone, string> = {
  default: "",
  warn: "text-amber-600",
  danger: "text-destructive",
  success: "text-emerald-600",
};

function Chip({ label, value, tone = "default" }: { label: string; value: ReactNode; tone?: Tone }) {
  return (
    <div className="rounded-md border px-3 py-2">
      <p className="text-muted-foreground truncate text-xs">{label}</p>
      <p className={`text-lg font-semibold ${TONE[tone]}`}>{value}</p>
    </div>
  );
}

/** One metric, shown for each rolling window rather than a single instant. */
function Periods({
  label,
  counts,
  format,
}: {
  label: string;
  counts: PeriodCounts;
  format?: (n: number) => string;
}) {
  const { t } = useI18n();
  const cells: [string, number][] = [
    [t("stats.period.today"), counts.today],
    [t("stats.period.week"), counts.week],
    [t("stats.period.fortnight"), counts.fortnight],
    [t("stats.period.month"), counts.month],
  ];
  return (
    <div className="rounded-md border px-3 py-2">
      <p className="text-muted-foreground text-xs">{label}</p>
      <div className="mt-1 flex flex-wrap gap-x-6 gap-y-1">
        {cells.map(([key, value]) => (
          <div key={key}>
            <p className="text-muted-foreground text-[10px] uppercase">{key}</p>
            <p className="text-sm font-semibold">{format ? format(value) : value}</p>
          </div>
        ))}
      </div>
    </div>
  );
}

function Chips({ children }: { children: ReactNode }) {
  return <div className="grid grid-cols-2 gap-2 lg:grid-cols-4">{children}</div>;
}

function Rows({ children }: { children: ReactNode }) {
  return <div className="grid gap-2 lg:grid-cols-2">{children}</div>;
}

/**
 * The counters shown at the top of an operations page: short snapshot chips
 * plus day/week/fortnight/month buckets for the module's activity.
 */
export function OpsModuleStats({ module }: { module: OpsModule }) {
  const { t } = useI18n();
  const { data } = useOpsSummary();
  if (!data) return null;
  const money = (cents: number) => formatMoney(cents);

  switch (module) {
    case "orders":
      return (
        <div className="space-y-2">
          <Chips>
            <Chip label={t("ops.total")} value={data.orders.total} />
            <Chip
              label={t("status.pending_payment")}
              value={data.orders.byStatus.pending_payment ?? 0}
              tone="warn"
            />
            <Chip label={t("status.paid")} value={data.orders.byStatus.paid ?? 0} />
            <Chip
              label={t("ops.paidOrders")}
              value={
                (data.orders.byStatus.paid ?? 0) +
                (data.orders.byStatus.shipped ?? 0) +
                (data.orders.byStatus.completed ?? 0)
              }
              tone="success"
            />
          </Chips>
          <Rows>
            <Periods label={t("stats.ordersCreated")} counts={data.orders.created} />
            <Periods label={t("stats.revenue")} counts={data.orders.revenueCents} format={money} />
          </Rows>
        </div>
      );

    case "products":
      return (
        <Chips>
          <Chip label={t("ops.total")} value={data.products.total} />
          <Chip label={t("common.active")} value={data.products.published} tone="success" />
          <Chip label={t("ops.draft")} value={data.products.draft} />
          <Chip
            label={t("ops.lowStock")}
            value={data.products.lowStock}
            tone={data.products.lowStock > 0 ? "warn" : "default"}
          />
        </Chips>
      );

    case "customers":
      return (
        <div className="space-y-2">
          <Chips>
            <Chip label={t("ops.total")} value={data.customers.total} />
            <Chip
              label={t("common.inactive")}
              value={data.customers.disabled}
              tone={data.customers.disabled > 0 ? "warn" : "default"}
            />
            <Chip label={t("stats.newToday")} value={data.customers.new.today} tone="success" />
            <Chip label={t("stats.newMonth")} value={data.customers.new.month} />
          </Chips>
          <Rows>
            <Periods label={t("stats.newCustomers")} counts={data.customers.new} />
          </Rows>
        </div>
      );

    case "coupons":
      return (
        <div className="space-y-2">
          <Chips>
            <Chip label={t("ops.total")} value={data.coupons.total} />
            <Chip label={t("common.active")} value={data.coupons.active} tone="success" />
            <Chip label={t("stats.redemptionsToday")} value={data.coupons.redemptions.today} />
            <Chip label={t("stats.redemptionsMonth")} value={data.coupons.redemptions.month} />
          </Chips>
          <Rows>
            <Periods label={t("stats.redemptions")} counts={data.coupons.redemptions} />
          </Rows>
        </div>
      );

    case "tickets":
      return (
        <div className="space-y-2">
          <Chips>
            <Chip
              label={t("ticket.status.open")}
              value={data.tickets.open}
              tone={data.tickets.open > 0 ? "warn" : "default"}
            />
            <Chip label={t("ticket.status.pending")} value={data.tickets.pending} />
            <Chip
              label={t("ops.unassigned")}
              value={data.tickets.unassigned}
              tone={data.tickets.unassigned > 0 ? "danger" : "default"}
            />
            <Chip label={t("stats.newToday")} value={data.tickets.created.today} />
          </Chips>
          <Rows>
            <Periods label={t("stats.ticketsCreated")} counts={data.tickets.created} />
          </Rows>
        </div>
      );

    case "returns":
      return (
        <div className="space-y-2">
          <Chips>
            <Chip
              label={t("ops.awaitingDecision")}
              value={data.returns.requested}
              tone={data.returns.requested > 0 ? "warn" : "default"}
            />
            <Chip label={t("ops.approved")} value={data.returns.approved} tone="success" />
            <Chip label={t("stats.requestedToday")} value={data.returns.created.today} />
            <Chip label={t("stats.requestedMonth")} value={data.returns.created.month} />
          </Chips>
          <Rows>
            <Periods label={t("stats.returnsCreated")} counts={data.returns.created} />
          </Rows>
        </div>
      );

    case "withdrawals":
      return (
        <div className="space-y-2">
          <Chips>
            <Chip
              label={t("withdraw.status.requested")}
              value={data.withdrawals.requested}
              tone={data.withdrawals.requested > 0 ? "warn" : "default"}
            />
            <Chip label={t("withdraw.status.approved")} value={data.withdrawals.approved} />
            <Chip label={t("withdraw.status.paid")} value={data.withdrawals.paid} tone="success" />
            <Chip label={t("stats.requestedToday")} value={data.withdrawals.created.today} />
          </Chips>
          <Rows>
            <Periods label={t("stats.withdrawalsCreated")} counts={data.withdrawals.created} />
          </Rows>
        </div>
      );

    case "reviews":
      return (
        <div className="space-y-2">
          <Chips>
            <Chip label={t("ops.total")} value={data.reviews.total} />
            <Chip label={t("stats.newToday")} value={data.reviews.created.today} />
            <Chip label={t("stats.newWeek")} value={data.reviews.created.week} />
            <Chip label={t("stats.newMonth")} value={data.reviews.created.month} />
          </Chips>
          <Rows>
            <Periods label={t("stats.reviewsCreated")} counts={data.reviews.created} />
          </Rows>
        </div>
      );

    case "commissions":
      return (
        <div className="space-y-2">
          <Chips>
            <Chip
              label={t("commission.status.pending")}
              value={data.commissions.pending}
              tone={data.commissions.pending > 0 ? "warn" : "default"}
            />
            <Chip
              label={t("commission.status.approved")}
              value={data.commissions.approved}
              tone="success"
            />
            <Chip label={t("stats.newToday")} value={data.commissions.created.today} />
            <Chip label={t("stats.newMonth")} value={data.commissions.created.month} />
          </Chips>
          <Rows>
            <Periods label={t("stats.commissionsCreated")} counts={data.commissions.created} />
          </Rows>
        </div>
      );
  }
}
