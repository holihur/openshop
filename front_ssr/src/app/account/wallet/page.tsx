import { redirect } from "next/navigation";

import { apiGet, apiList } from "@/lib/api";
import { AccountNav } from "@/components/account-nav";
import { WalletTopUp } from "@/components/wallet-topup";
import { formatDate, formatMoney } from "@/lib/format";
import { translator } from "@/lib/i18n";
import { getShopper } from "@/lib/session";
import { resolveLocale } from "@/app/layout";
import type { SiteConfig, Wallet, WalletTransaction } from "@/lib/types";

/** Wallet balance and its ledger. The ledger is the truth, so it is shown. */
export default async function WalletPage() {
  const shopper = await getShopper();
  if (!shopper.token) redirect("/login");

  const locale = await resolveLocale();
  const t = translator(locale);
  const auth = { token: shopper.token };
  const [wallet, ledger, site] = await Promise.all([
    apiGet<Wallet>("/wallet", auth).catch(() => null),
    apiList<WalletTransaction>("/wallet/transactions?pageSize=20", auth).catch(
      () => ({ items: [] as WalletTransaction[], total: 0, page: 1, pageSize: 20 }),
    ),
    apiGet<SiteConfig>("/site", { revalidate: 60 }).catch(() => null),
  ]);

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t("account.wallet")}</h1>
      <AccountNav locale={locale} current="/account/wallet" />

      <div className="card-surface p-6">
        <p className="text-muted-foreground text-sm">{t("account.balance")}</p>
        <p className="text-3xl font-bold">
          {wallet ? formatMoney(wallet.balanceCents, wallet.currency, locale) : "—"}
        </p>
        <p className="text-muted-foreground mt-2 text-xs">{t("account.walletNote")}</p>
      </div>

      {/* Top-ups are only offered when the operator has switched the wallet on;
          the API would reject the request otherwise. */}
      {site?.walletTopUpEnabled ? (
        <WalletTopUp
          minCents={site.walletMinTopUpCents || 1000}
          maxCents={site.walletMaxTopUpCents || 100000}
          labels={{
            title: t("account.topUp"),
            amount: t("account.amount"),
            submit: t("account.topUpButton"),
            sending: t("cart.updating"),
            failed: t("error.generic"),
          }}
        />
      ) : null}

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
                <th className="py-2 text-right">{t("account.amount")}</th>
                <th className="py-2 text-right">{t("account.balanceAfter")}</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {ledger.items.map((entry) => (
                <tr key={entry.id}>
                  <td className="text-muted-foreground py-2">{formatDate(entry.createdAt, locale)}</td>
                  <td className="py-2">{entry.description}</td>
                  <td
                    className={
                      entry.amountCents < 0 ? "py-2 text-right text-red-600" : "py-2 text-right"
                    }
                  >
                    {entry.amountCents > 0 ? "+" : ""}
                    {formatMoney(entry.amountCents, wallet?.currency, locale)}
                  </td>
                  <td className="text-muted-foreground py-2 text-right">
                    {formatMoney(entry.balanceAfter, wallet?.currency, locale)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
    </div>
  );
}
