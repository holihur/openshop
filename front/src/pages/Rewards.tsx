import { useState } from "react";

import { Badge } from "@lib/components/ui/badge";
import { Button } from "@lib/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@lib/components/ui/card";
import { Skeleton } from "@lib/components/ui/skeleton";
import { useI18n } from "@lib/i18n";
import { formatDate, formatMoney } from "@lib/format";
import {
  useMyCommissions,
  usePoints,
  usePointsTransactions,
  useReferralSummary,
} from "@lib/hooks/useLoyalty";

const COMMISSION_VARIANT: Record<string, "warning" | "success" | "secondary"> = {
  pending: "warning",
  approved: "success",
  reversed: "secondary",
};

export function RewardsPage() {
  const { t } = useI18n();
  const { data: points, isLoading } = usePoints();
  const { data: summary } = useReferralSummary();
  const { data: commissions } = useMyCommissions();
  const { data: ledger } = usePointsTransactions();
  const [copied, setCopied] = useState(false);

  async function copyCode() {
    if (!summary?.code) return;
    try {
      await navigator.clipboard.writeText(summary.code);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      /* clipboard unavailable */
    }
  }

  return (
    <div className="mx-auto max-w-3xl space-y-6 py-8">
      <h1 className="text-2xl font-semibold">{t("rewards.title")}</h1>

      <div className="grid gap-4 sm:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">{t("rewards.points")}</CardTitle>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <Skeleton className="h-8 w-24" />
            ) : (
              <>
                <p className="text-3xl font-semibold">{points?.balance ?? 0}</p>
                <p className="text-muted-foreground text-xs">
                  {t("rewards.lifetime", { count: points?.lifetimeEarned ?? 0 })}
                </p>
              </>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">{t("rewards.referral")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            <div className="flex items-center gap-2">
              <code className="bg-muted rounded px-2 py-1 font-mono text-sm">
                {summary?.code ?? "—"}
              </code>
              <Button variant="outline" size="sm" onClick={copyCode}>
                {copied ? t("rewards.copied") : t("rewards.copy")}
              </Button>
            </div>
            <p className="text-muted-foreground text-xs">
              {t("rewards.referrals", { count: summary?.referrals ?? 0 })}
            </p>
            <div className="flex gap-4 text-sm">
              <span>
                {t("rewards.pending")}: {formatMoney(summary?.pendingCents ?? 0)}
              </span>
              <span>
                {t("rewards.earned")}: {formatMoney(summary?.approvedCents ?? 0)}
              </span>
            </div>
          </CardContent>
        </Card>
      </div>

      <div className="space-y-3">
        <h2 className="text-lg font-medium">{t("rewards.commissions")}</h2>
        {commissions?.items.length ? (
          <ul className="space-y-2">
            {commissions.items.map((c) => (
              <li key={c.id} className="flex items-center justify-between rounded-md border p-3 text-sm">
                <div>
                  <span className="font-medium">{formatMoney(c.amountCents)}</span>
                  <span className="text-muted-foreground ml-2">
                    {t("rewards.rate", { percent: (c.rateBps / 100).toFixed(2) })}
                  </span>
                </div>
                <div className="flex items-center gap-2">
                  <Badge variant={COMMISSION_VARIANT[c.status] ?? "secondary"}>
                    {t(`commission.status.${c.status}`)}
                  </Badge>
                  <span className="text-muted-foreground text-xs">
                    {c.status === "pending"
                      ? t("rewards.availableOn", { date: formatDate(c.holdUntil) })
                      : formatDate(c.approvedAt ?? c.createdAt)}
                  </span>
                </div>
              </li>
            ))}
          </ul>
        ) : (
          <p className="text-muted-foreground text-sm">{t("rewards.noCommissions")}</p>
        )}
      </div>

      {ledger?.items.length ? (
        <div className="space-y-3">
          <h2 className="text-lg font-medium">{t("rewards.pointsHistory")}</h2>
          <ul className="space-y-2">
            {ledger.items.map((tx) => (
              <li key={tx.id} className="flex items-center justify-between rounded-md border p-3 text-sm">
                <div>
                  <span className="font-medium">
                    {tx.points > 0 ? "+" : ""}
                    {tx.points}
                  </span>
                  <span className="text-muted-foreground ml-2">{tx.description}</span>
                </div>
                <span className="text-muted-foreground text-xs">{formatDate(tx.createdAt)}</span>
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </div>
  );
}
