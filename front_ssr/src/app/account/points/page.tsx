import { redirect } from "next/navigation";

import { apiGet, apiList } from "@/lib/api";
import { AccountNav } from "@/components/account-nav";
import { formatDate } from "@/lib/format";
import { translator } from "@/lib/i18n";
import { getShopper } from "@/lib/session";
import { resolveLocale } from "@/app/layout";
import type { PointsAccount, PointsTransaction } from "@/lib/types";

/** Loyalty points: the balance, how it was earned and what it was spent on. */
export default async function PointsPage() {
  const shopper = await getShopper();
  if (!shopper.token) redirect("/login");

  const locale = await resolveLocale();
  const t = translator(locale);
  const auth = { token: shopper.token };
  const [account, ledger] = await Promise.all([
    apiGet<PointsAccount>("/points", auth).catch(() => null),
    apiList<PointsTransaction>("/points/transactions?pageSize=20", auth).catch(
      () => ({ items: [] as PointsTransaction[], total: 0, page: 1, pageSize: 20 }),
    ),
  ]);

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t("account.points")}</h1>
      <AccountNav locale={locale} current="/account/points" />

      <div className="grid gap-4 sm:grid-cols-2">
        <div className="card-surface p-6">
          <p className="text-muted-foreground text-sm">{t("account.pointsBalance")}</p>
          <p className="text-3xl font-bold">{account?.balance ?? 0}</p>
        </div>
        <div className="card-surface p-6">
          <p className="text-muted-foreground text-sm">{t("account.pointsLifetime")}</p>
          <p className="text-3xl font-bold">{account?.lifetimeEarned ?? 0}</p>
        </div>
      </div>

      <section className="space-y-2">
        <h2 className="font-semibold">{t("account.ledger")}</h2>
        {ledger.items.length === 0 ? (
          <p className="text-muted-foreground rounded-lg border border-dashed py-10 text-center text-sm">
            {t("account.noTransactions")}
          </p>
        ) : (
          <table className="w-full text-sm">
            <thead className="text-muted-foreground text-left">
              <tr>
                <th className="py-2">{t("account.date")}</th>
                <th className="py-2">{t("account.description")}</th>
                <th className="py-2 text-right">{t("account.pointsChange")}</th>
                <th className="py-2 text-right">{t("account.balanceAfter")}</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {ledger.items.map((entry) => (
                <tr key={entry.id}>
                  <td className="text-muted-foreground py-2">{formatDate(entry.createdAt, locale)}</td>
                  <td className="py-2">{entry.description}</td>
                  <td className={entry.points < 0 ? "py-2 text-right text-red-600" : "py-2 text-right"}>
                    {entry.points > 0 ? "+" : ""}
                    {entry.points}
                  </td>
                  <td className="text-muted-foreground py-2 text-right">{entry.balanceAfter}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
    </div>
  );
}
