import { Truck } from "lucide-react";

import { useI18n } from "@lib/i18n";
import type { DeliveryEstimate } from "@lib/types";

function formatDay(iso: string, locale: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "";
  return new Intl.DateTimeFormat(locale, { month: "short", day: "numeric" }).format(date);
}

/**
 * DeliveryEstimateLine states when an order would arrive. It is shown on the
 * product page and in the cart so the delivery time is never a surprise at
 * checkout.
 */
export function DeliveryEstimateLine({
  estimate,
  className,
}: {
  estimate?: DeliveryEstimate;
  className?: string;
}) {
  const { t, locale } = useI18n();
  if (!estimate) return null;
  const earliest = formatDay(estimate.earliest, locale);
  const latest = formatDay(estimate.latest, locale);
  return (
    <p className={className ?? "text-muted-foreground flex items-center gap-1.5 text-sm"}>
      <Truck className="size-4 shrink-0" aria-hidden="true" />
      {earliest && latest
        ? t("delivery.window", { earliest, latest })
        : t("delivery.businessDays", { min: estimate.minDays, max: estimate.maxDays })}
    </p>
  );
}

/**
 * FreeShippingProgress nudges the shopper towards the free-shipping threshold
 * with a progress bar, which is the single cheapest way to raise basket size.
 */
export function FreeShippingProgress({
  remainingCents,
  thresholdCents,
  subtotalCents,
  format,
}: {
  remainingCents: number;
  thresholdCents: number;
  subtotalCents: number;
  format: (cents: number) => string;
}) {
  const { t } = useI18n();
  if (thresholdCents <= 0) return null;
  if (remainingCents <= 0) {
    // green-600 on white is 3.1:1, below the 4.5:1 minimum for body text, so
    // the accessible tones are used instead (axe: color-contrast).
    return (
      <p className="text-sm font-medium text-green-700 dark:text-green-400" role="status">
        {t("delivery.freeShippingReached")}
      </p>
    );
  }
  const progress = Math.min(100, Math.round((subtotalCents / thresholdCents) * 100));
  return (
    <div className="space-y-1.5" role="status">
      <p className="text-sm">{t("delivery.freeShippingRemaining", { amount: format(remainingCents) })}</p>
      <div
        className="bg-muted h-2 w-full overflow-hidden rounded-full"
        role="progressbar"
        aria-valuenow={progress}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-label={t("delivery.freeShippingProgress")}
      >
        <div
          className="bg-primary h-full rounded-full transition-all"
          style={{ width: `${progress}%` }}
        />
      </div>
    </div>
  );
}
