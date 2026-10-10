import Link from "next/link";
import { redirect } from "next/navigation";

import { apiGet, apiList } from "@/lib/api";
import { AccountNav } from "@/components/account-nav";
import { formatMoney } from "@/lib/format";
import { translator } from "@/lib/i18n";
import { getShopper } from "@/lib/session";
import { resolveLocale } from "@/app/layout";
import type { Order, PointsAccount, Wallet } from "@/lib/types";

/** The account hub: a summary of everything a shopper owns. */
export default async function AccountPage() {
  const shopper = await getShopper();
  if (!shopper.token) redirect("/login");

  const locale = await resolveLocale();
  const t = translator(locale);
  const auth = { token: shopper.token };
  const [wallet, points, orders] = await Promise.all([
    apiGet<Wallet>("/wallet", auth).catch(() => null),
    apiGet<PointsAccount>("/points", auth).catch(() => null),
    apiList<Order>("/orders?pageSize=3", auth).catch(
      () => ({ items: [] as Order[], total: 0, page: 1, pageSize: 3 }),
    ),
  ]);

  const cards = [
    {
      href: "/account/wallet",
      label: t("account.wallet"),
      value: wallet ? formatMoney(wallet.balanceCents, wallet.currency, locale) : "—",
    },
    {
      href: "/account/points",
      label: t("account.points"),
      value: points ? String(points.balance) : "—",
    },
    {
      href: "/account/orders",
      label: t("orders.title"),
      value: String(orders.total),
    },
  ];

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t("account.title")}</h1>
      <AccountNav locale={locale} current="/account" />

      <div className="grid gap-4 sm:grid-cols-3">
        {cards.map((card) => (
          <Link key={card.href} href={card.href} className="card-surface p-4">
            <p className="text-muted-foreground text-sm">{card.label}</p>
            <p className="text-xl font-semibold">{card.value}</p>
          </Link>
        ))}
      </div>

      {orders.items.length > 0 ? (
        <section className="space-y-2">
          <h2 className="font-semibold">{t("account.recentOrders")}</h2>
          <ul className="divide-y rounded-lg border">
            {orders.items.map((order) => (
              <li key={order.id} className="flex items-center justify-between p-3 text-sm">
                <Link href={`/account/orders/${order.id}`} className="underline">
                  {order.orderNo}
                </Link>
                <span>{order.status}</span>
                <span>{formatMoney(order.totalCents, order.currency, locale)}</span>
              </li>
            ))}
          </ul>
        </section>
      ) : null}
    </div>
  );
}
