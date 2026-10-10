import { apiGet } from "@/lib/api";
import { OrderStatusBadge } from "@/components/order-status-badge";
import { formatDate, formatMoney } from "@/lib/format";
import { translator } from "@/lib/i18n";
import { resolveLocale } from "@/app/layout";
import type { Order } from "@/lib/types";

/**
 * Order lookup for a shopper who checked out as a guest. The order's own access
 * token is the credential, so the link in the confirmation email keeps working
 * without an account.
 */
export default async function GuestOrderPage({
  searchParams,
}: {
  searchParams: Promise<{ token?: string }>;
}) {
  const params = await searchParams;
  const locale = await resolveLocale();
  const t = translator(locale);
  const token = (params.token ?? "").trim();

  const order = token
    ? await apiGet<Order>(`/guest/orders/${encodeURIComponent(token)}`).catch(() => null)
    : null;

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <h1 className="text-2xl font-bold">{t("orders.guestTitle")}</h1>

      <form method="get" action="/orders/guest" className="flex gap-2">
        <input
          name="token"
          defaultValue={token}
          placeholder={t("orders.guestToken")}
          aria-label={t("orders.guestToken")}
          required
          className="border-input h-9 flex-1 rounded-md border px-3"
        />
        <button
          type="submit"
          className="bg-primary text-primary-foreground h-9 rounded-md px-4"
        >
          {t("orders.lookup")}
        </button>
      </form>

      {token && !order ? (
        <p className="text-muted-foreground rounded-lg border border-dashed py-10 text-center text-sm">
          {t("orders.guestNotFound")}
        </p>
      ) : null}

      {order ? (
        <div className="space-y-4">
          <div className="card-surface flex items-center justify-between p-4">
            <div>
              <p className="font-medium">{order.orderNo}</p>
              <p className="text-muted-foreground text-sm">{formatDate(order.createdAt, locale)}</p>
            </div>
            <OrderStatusBadge status={order.status} locale={locale} />
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
      ) : null}
    </div>
  );
}
