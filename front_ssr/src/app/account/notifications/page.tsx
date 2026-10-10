import Link from "next/link";
import { redirect } from "next/navigation";

import { apiList } from "@/lib/api";
import { AccountNav } from "@/components/account-nav";
import { MarkAllReadButton } from "@/components/mark-all-read";
import { formatDate } from "@/lib/format";
import { translator } from "@/lib/i18n";
import { getShopper } from "@/lib/session";
import { resolveLocale } from "@/app/layout";
import type { Notification } from "@/lib/types";

/** Everything the shop has told this shopper. */
export default async function NotificationsPage() {
  const shopper = await getShopper();
  if (!shopper.token) redirect("/login");

  const locale = await resolveLocale();
  const t = translator(locale);
  const notifications = await apiList<Notification>("/notifications?pageSize=30", {
    token: shopper.token,
  }).catch(() => ({ items: [] as Notification[], total: 0, page: 1, pageSize: 30 }));
  const unread = notifications.items.filter((n) => !n.read).length;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">{t("account.notifications")}</h1>
        {unread > 0 ? <MarkAllReadButton label={t("account.markAllRead")} /> : null}
      </div>
      <AccountNav locale={locale} current="/account/notifications" />

      {notifications.items.length === 0 ? (
        <p className="text-muted-foreground rounded-lg border border-dashed py-16 text-center">
          {t("account.noNotifications")}
        </p>
      ) : (
        <ul className="divide-y rounded-lg border">
          {notifications.items.map((item) => (
            <li key={item.id} className={item.read ? "p-4" : "bg-muted/40 p-4"}>
              <div className="flex items-start justify-between gap-3">
                <div>
                  <p className="font-medium">
                    {!item.read ? <span aria-hidden="true">● </span> : null}
                    {item.title}
                  </p>
                  {item.body ? (
                    <p className="text-muted-foreground mt-1 text-sm">{item.body}</p>
                  ) : null}
                </div>
                <span className="text-muted-foreground shrink-0 text-xs">
                  {formatDate(item.createdAt, locale)}
                </span>
              </div>
              {item.link ? (
                <Link href={item.link} className="mt-2 inline-block text-sm underline">
                  {t("account.open")}
                </Link>
              ) : null}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
