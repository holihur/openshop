import type { ReactNode } from "react";

import { formatMoney } from "@lib/format";
import { useI18n } from "@lib/i18n";
import { useOpsSummary } from "@lib/hooks/useOpsSummary";
import type { DailyPoint, PeriodMetric } from "@lib/types";

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

/** Signed change against the equivalent previous window. */
function Delta({ value, previous }: { value: number; previous: number }) {
  const { t } = useI18n();
  const diff = value - previous;
  if (diff === 0) {
    return <span className="text-muted-foreground text-[10px]">{t("stats.noChange")}</span>;
  }
  const pct = previous > 0 ? Math.round((diff / previous) * 100) : null;
  return (
    <span
      className={`text-[10px] ${diff > 0 ? "text-emerald-600" : "text-destructive"}`}
      title={t("stats.previousWas", { value: previous })}
    >
      {diff > 0 ? "▲" : "▼"} {pct !== null ? `${Math.abs(pct)}%` : Math.abs(diff)}
    </span>
  );
}

/**
 * One metric across day / yesterday / week / fortnight / month, each with its
 * change against the equivalent window before it.
 */
function Periods({
  label,
  metric,
  format,
}: {
  label: string;
  metric: PeriodMetric;
  format?: (n: number) => string;
}) {
  const { t } = useI18n();
  const fmt = format ?? ((n: number) => String(n));
  const cells: [string, number, number][] = [
    [t("stats.period.today"), metric.current.today, metric.previous.today],
    [t("stats.period.yesterday"), metric.current.yesterday, metric.previous.yesterday],
    [t("stats.period.week"), metric.current.week, metric.previous.week],
    [t("stats.period.fortnight"), metric.current.fortnight, metric.previous.fortnight],
    [t("stats.period.month"), metric.current.month, metric.previous.month],
  ];
  return (
    <div className="rounded-md border px-3 py-2">
      <p className="text-muted-foreground text-xs">{label}</p>
      <div className="mt-1 flex flex-wrap gap-x-6 gap-y-2">
        {cells.map(([key, value, previous]) => (
          <div key={key}>
            <p className="text-muted-foreground text-[10px] uppercase">{key}</p>
            <p className="flex items-baseline gap-1.5">
              <span className="text-sm font-semibold">{fmt(value)}</span>
              <Delta value={value} previous={previous} />
            </p>
          </div>
        ))}
      </div>
    </div>
  );
}

function Sparkline({ points, label }: { points: DailyPoint[]; label: string }) {
  if (points.length < 2) return null;
  const width = 220;
  const height = 40;
  const max = Math.max(1, ...points.map((p) => p.value));
  const step = (width - 4) / (points.length - 1);
  const xy = points.map((p, i) => [2 + i * step, height - 2 - (p.value / max) * (height - 4)] as const);
  const line = xy.map(([x, y], i) => `${i === 0 ? "M" : "L"}${x.toFixed(1)},${y.toFixed(1)}`).join(" ");
  const area = `${line} L${width - 2},${height - 2} L2,${height - 2} Z`;
  return (
    <svg
      viewBox={`0 0 ${width} ${height}`}
      className="h-10 w-full max-w-56"
      role="img"
      aria-label={label}
      preserveAspectRatio="none"
    >
      <path d={area} className="fill-primary/10" />
      <path d={line} className="stroke-primary" fill="none" strokeWidth="1.5" vectorEffect="non-scaling-stroke" />
    </svg>
  );
}

function Chips({ children }: { children: ReactNode }) {
  return <div className="grid grid-cols-2 gap-2 lg:grid-cols-4">{children}</div>;
}

function Trend({ points, label }: { points: DailyPoint[]; label: string }) {
  const { t } = useI18n();
  return (
    <div className="rounded-md border px-3 py-2">
      <p className="text-muted-foreground text-xs">
        {t("stats.trend30", { label })}
      </p>
      <Sparkline points={points} label={t("stats.trend30", { label })} />
    </div>
  );
}

/**
 * The counters shown at the top of an operations page: snapshot chips, the
 * day/week/fortnight/month windows with their change, and a 30-day trend.
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
          <div className="grid gap-2 lg:grid-cols-2">
            <Periods label={t("stats.ordersCreated")} metric={data.orders.created} />
            <Periods
              label={t("stats.revenue")}
              metric={data.orders.revenueCents}
              format={money}
            />
          </div>
          <Trend points={data.orders.series} label={t("stats.ordersCreated")} />
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
            <Chip label={t("stats.newToday")} value={data.customers.new.current.today} />
            <Chip label={t("stats.newMonth")} value={data.customers.new.current.month} />
          </Chips>
          <div className="grid gap-2 lg:grid-cols-2">
            <Periods label={t("stats.newCustomers")} metric={data.customers.new} />
            <Trend points={data.customers.series} label={t("stats.newCustomers")} />
          </div>
        </div>
      );

    case "coupons":
      return (
        <div className="space-y-2">
          <Chips>
            <Chip label={t("ops.total")} value={data.coupons.total} />
            <Chip label={t("common.active")} value={data.coupons.active} tone="success" />
            <Chip label={t("stats.redemptionsToday")} value={data.coupons.redemptions.current.today} />
            <Chip label={t("stats.redemptionsMonth")} value={data.coupons.redemptions.current.month} />
          </Chips>
          <div className="grid gap-2 lg:grid-cols-2">
            <Periods label={t("stats.redemptions")} metric={data.coupons.redemptions} />
            <Trend points={data.coupons.series} label={t("stats.redemptions")} />
          </div>
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
            <Chip label={t("stats.newToday")} value={data.tickets.created.current.today} />
          </Chips>
          <div className="grid gap-2 lg:grid-cols-2">
            <Periods label={t("stats.ticketsCreated")} metric={data.tickets.created} />
            <Trend points={data.tickets.series} label={t("stats.ticketsCreated")} />
          </div>
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
            <Chip label={t("stats.requestedToday")} value={data.returns.created.current.today} />
            <Chip label={t("stats.requestedMonth")} value={data.returns.created.current.month} />
          </Chips>
          <div className="grid gap-2 lg:grid-cols-2">
            <Periods label={t("stats.returnsCreated")} metric={data.returns.created} />
            <Trend points={data.returns.series} label={t("stats.returnsCreated")} />
          </div>
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
            <Chip label={t("stats.requestedToday")} value={data.withdrawals.created.current.today} />
          </Chips>
          <div className="grid gap-2 lg:grid-cols-2">
            <Periods label={t("stats.withdrawalsCreated")} metric={data.withdrawals.created} />
            <Trend points={data.withdrawals.series} label={t("stats.withdrawalsCreated")} />
          </div>
        </div>
      );

    case "reviews":
      return (
        <div className="space-y-2">
          <Chips>
            <Chip label={t("ops.total")} value={data.reviews.total} />
            <Chip label={t("stats.newToday")} value={data.reviews.created.current.today} />
            <Chip label={t("stats.newWeek")} value={data.reviews.created.current.week} />
            <Chip label={t("stats.newMonth")} value={data.reviews.created.current.month} />
          </Chips>
          <div className="grid gap-2 lg:grid-cols-2">
            <Periods label={t("stats.reviewsCreated")} metric={data.reviews.created} />
            <Trend points={data.reviews.series} label={t("stats.reviewsCreated")} />
          </div>
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
            <Chip label={t("stats.newToday")} value={data.commissions.created.current.today} />
            <Chip label={t("stats.newMonth")} value={data.commissions.created.current.month} />
          </Chips>
          <div className="grid gap-2 lg:grid-cols-2">
            <Periods label={t("stats.commissionsCreated")} metric={data.commissions.created} />
            <Trend points={data.commissions.series} label={t("stats.commissionsCreated")} />
          </div>
        </div>
      );
  }
}
