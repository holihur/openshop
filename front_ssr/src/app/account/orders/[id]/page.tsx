import Link from "next/link";
import { notFound, redirect } from "next/navigation";

import { apiGet } from "@/lib/api";
import { formatDate, formatMoney } from "@/lib/format";
import { translator } from "@/lib/i18n";
import { getShopper } from "@/lib/session";
import { resolveLocale } from "@/app/layout";
import type { Order } from "@/lib/types";

/** A single order, rendered on the server for the signed-in shopper. */
export default async function OrderDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const shopper = await getShopper();
  if (!shopper.token) redirect("/login");

  const { id } = await params;
  const locale = await resolveLocale();
  const t = translator(locale);
  const order = await apiGet<Order>(`/orders/${id}`, { token: shopper.token }).catch(() => null);
  if (!order) notFound();

  return (
    <div className="space-y-6">
      <div>
        <Link href="/account/orders" className="text-muted-foreground text-sm underline">
          {t("orders.title")}
        </Link>
        <h1 className="text-2xl font-bold">{order.orderNo}</h1>
        <p className="text-muted-foreground text-sm">
          {formatDate(order.createdAt, locale)} · {order.status}
        </p>
      </div>

      <ul className="divide-y rounded-lg border">
        {order.items.map((item) => (
          <li key={item.id} className="flex items-center justify-between p-3 text-sm">
            <span>
              {item.title}
              {item.variantName ? ` · ${item.variantName}` : ""} × {item.quantity}
            </span>
            <span>{formatMoney(item.priceCents * item.quantity, order.currency, locale)}</span>
          </li>
        ))}
      </ul>

      <dl className="ml-auto w-full max-w-sm space-y-1 text-sm">
        <div className="flex justify-between">
          <dt className="text-muted-foreground">{t("cart.subtotal")}</dt>
          <dd>{formatMoney(order.subtotalCents, order.currency, locale)}</dd>
        </div>
        <div className="flex justify-between">
          <dt className="text-muted-foreground">{t("cart.shipping")}</dt>
          <dd>{formatMoney(order.shippingCents, order.currency, locale)}</dd>
        </div>
        <div className="flex justify-between border-t pt-1 text-base font-semibold">
          <dt>{t("cart.total")}</dt>
          <dd>{formatMoney(order.totalCents, order.currency, locale)}</dd>
        </div>
      </dl>
    </div>
  );
}
