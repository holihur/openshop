import { useState } from "react";
import { Link } from "react-router-dom";

import { Button } from "@lib/components/ui/button";
import { Skeleton } from "@lib/components/ui/skeleton";
import { Pagination } from "@lib/components/pagination";
import { useI18n } from "@lib/i18n";
import {
  useMarkAllNotificationsRead,
  useMarkNotificationRead,
  useNotifications,
} from "@lib/hooks/useNotifications";
import { notificationText } from "@lib/notifications";
import { formatDate } from "@lib/format";
import { cn } from "@lib/utils";

export function NotificationsPage() {
  const { t } = useI18n();
  const [page, setPage] = useState(1);
  const [unreadOnly, setUnreadOnly] = useState(false);
  const { data, isLoading } = useNotifications(page, 20, unreadOnly);
  const markRead = useMarkNotificationRead();
  const markAll = useMarkAllNotificationsRead();

  return (
    <div className="mx-auto max-w-3xl space-y-4 py-8">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h1 className="text-2xl font-semibold">{t("notifications.title")}</h1>
        <div className="flex items-center gap-2">
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={unreadOnly}
              onChange={(e) => {
                setUnreadOnly(e.target.checked);
                setPage(1);
              }}
            />
            {t("notifications.unreadOnly")}
          </label>
          <Button variant="outline" size="sm" disabled={markAll.isPending} onClick={() => markAll.mutate()}>
            {t("notifications.markAllRead")}
          </Button>
        </div>
      </div>

      {isLoading ? (
        <Skeleton className="h-64 w-full" />
      ) : data && data.items.length > 0 ? (
        <ul className="space-y-2">
          {data.items.map((n) => {
            const { title, body } = notificationText(n, t);
            const content = (
              <div
                className={cn(
                  "hover:bg-accent flex items-start justify-between gap-3 rounded-md border p-3",
                  !n.read && "bg-accent/50",
                )}
              >
                <div>
                  <p className="text-sm font-medium">{title}</p>
                  {body && <p className="text-muted-foreground text-sm">{body}</p>}
                  <p className="text-muted-foreground mt-1 text-xs">{formatDate(n.createdAt)}</p>
                </div>
                {!n.read && <span className="bg-primary mt-1 size-2 shrink-0 rounded-full" />}
              </div>
            );
            return (
              <li
                key={n.id}
                onClick={() => {
                  if (!n.read) markRead.mutate(n.id);
                }}
              >
                {n.link ? (
                  <Link to={n.link} className="block">
                    {content}
                  </Link>
                ) : (
                  content
                )}
              </li>
            );
          })}
        </ul>
      ) : (
        <p className="text-muted-foreground text-sm">{t("notifications.empty")}</p>
      )}

      <Pagination
        page={data?.page ?? page}
        pageSize={data?.pageSize ?? 20}
        total={data?.total ?? 0}
        onChange={setPage}
      />
    </div>
  );
}
