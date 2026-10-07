import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { Bell } from "lucide-react";

import { useAuth } from "@lib/auth";
import { useI18n } from "@lib/i18n";
import {
  useMarkAllNotificationsRead,
  useNotifications,
  useUnreadCount,
} from "@lib/hooks/useNotifications";
import { notificationText } from "@lib/notifications";
import { formatDate } from "@lib/format";
import { cn } from "@lib/utils";

/** Header bell: unread badge plus the five most recent notifications. */
export function NotificationBell() {
  const { user } = useAuth();
  const { t } = useI18n();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const { data: unread } = useUnreadCount();
  const { data: list } = useNotifications(1, 5);
  const markAll = useMarkAllNotificationsRead();

  useEffect(() => {
    if (!open) return;
    const onDoc = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    document.addEventListener("mousedown", onDoc);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDoc);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  if (!user) return null;
  const count = unread?.count ?? 0;

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        aria-label={t("notifications.title")}
        aria-haspopup="menu"
        aria-expanded={open}
        onClick={() => setOpen((o) => !o)}
        className="hover:bg-accent relative rounded-md p-2"
      >
        <Bell className="size-4" />
        {count > 0 && (
          <span className="bg-destructive absolute -top-0.5 -right-0.5 flex size-4 items-center justify-center rounded-full text-[10px] font-medium text-white">
            {count > 9 ? "9+" : count}
          </span>
        )}
      </button>

      {open && (
        <div
          role="menu"
          className="bg-popover absolute right-0 z-50 mt-1 w-80 rounded-md border p-1 shadow-md"
        >
          <div className="flex items-center justify-between px-2 py-1.5">
            <span className="text-sm font-medium">{t("notifications.title")}</span>
            {count > 0 && (
              <button
                type="button"
                className="text-muted-foreground text-xs underline"
                disabled={markAll.isPending}
                onClick={() => markAll.mutate()}
              >
                {t("notifications.markAllRead")}
              </button>
            )}
          </div>
          <ul className="max-h-80 overflow-auto">
            {list?.items.map((n) => {
              const { title, body } = notificationText(n, t);
              return (
                <li key={n.id}>
                  <Link
                    to={n.link || "/account/notifications"}
                    onClick={() => setOpen(false)}
                    className={cn(
                      "hover:bg-accent block rounded-md px-2 py-2",
                      !n.read && "bg-accent/50",
                    )}
                  >
                    <p className="text-sm font-medium">{title}</p>
                    {body && <p className="text-muted-foreground text-xs">{body}</p>}
                    <p className="text-muted-foreground mt-0.5 text-[10px]">
                      {formatDate(n.createdAt)}
                    </p>
                  </Link>
                </li>
              );
            })}
            {list && list.items.length === 0 && (
              <li className="text-muted-foreground px-2 py-4 text-center text-sm">
                {t("notifications.empty")}
              </li>
            )}
          </ul>
          <Link
            to="/account/notifications"
            onClick={() => setOpen(false)}
            className="hover:bg-accent block rounded-md px-2 py-2 text-center text-sm"
          >
            {t("notifications.viewAll")}
          </Link>
        </div>
      )}
    </div>
  );
}
