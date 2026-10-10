import Link from "next/link";
import { redirect } from "next/navigation";

import { apiList } from "@/lib/api";
import { SignOutButton } from "@/components/sign-out-button";
import { formatDate, formatMoney } from "@/lib/format";
import { translator } from "@/lib/i18n";
import { getPricing } from "@/lib/pricing";
import { getShopper } from "@/lib/session";
import { resolveLocale } from "@/app/layout";
import type { Order } from "@/lib/types";

/** Order history, rendered on the server from the signed-in shopper's token. */
export default async function OrdersPage() {
  const shopper = await getShopper();
  if (!shopper.token) redirect("/login");

  const locale = await resolveLocale();
  const t = translator(locale);
  const [orders, pricing] = await Promise.all([
    apiList<Order>("/orders?pageSize=20", { token: shopper.token }).catch(
      () => ({ items: [] as Order[], total: 0, page: 1, pageSize: 20 }),
    ),
    getPricing(locale),
  ]);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">{t("orders.title")}</h1>
        <SignOutButton label={t("nav.signOut")} />
      </div>

      {orders.items.length === 0 ? (
        <p className="text-muted-foreground rounded-lg border border-dashed py-16 text-center">
          {t("orders.empty")}
        </p>
      ) : (
        <table className="w-full text-sm">
          <thead className="text-muted-foreground text-left">
            <tr>
              <th className="py-2">{t("orders.order")}</th>
              <th className="py-2">{t("orders.date")}</th>
              <th className="py-2">{t("orders.status")}</th>
              <th className="py-2 text-right">{t("orders.total")}</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {orders.items.map((order) => (
              <tr key={order.id}>
                <td className="py-3">
                  <Link href={`/account/orders/${order.id}`} className="font-medium underline">
                    {order.orderNo}
                  </Link>
                </td>
                <td className="text-muted-foreground py-3">{formatDate(order.createdAt, locale)}</td>
                <td className="py-3">{order.status}</td>
                <td className="py-3 text-right font-medium">
                  {pricing.format(order.totalCents)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
