import Link from "next/link";

import { apiGet } from "@/lib/api";
import { getShopper } from "@/lib/session";

/**
 * Unread notification count, rendered on the server with the same request as
 * the page, so the badge is correct on the first paint.
 */
export async function NotificationBadge({ label }: { label: string }) {
  const shopper = await getShopper();
  if (!shopper.token) return null;
  const result = await apiGet<{ count: number }>("/notifications/unread-count", {
    token: shopper.token,
  }).catch(() => null);
  if (!result?.count) {
    return (
      <Link href="/account/notifications" className="hover:opacity-70">
        {label}
      </Link>
    );
  }
  return (
    <Link
      href="/account/notifications"
      className="flex items-center gap-1 hover:opacity-70"
      aria-label={`${label} (${result.count})`}
    >
      {label}
      <span className="bg-primary text-primary-foreground rounded-full px-2 text-xs">
        {result.count}
      </span>
    </Link>
  );
}
