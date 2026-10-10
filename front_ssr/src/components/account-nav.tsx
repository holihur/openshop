import Link from "next/link";

import { translator, type Locale } from "@/lib/i18n";

/**
 * The account sections, rendered on the server. Every entry is a real page, so
 * the account area works without client-side routing.
 */
export function AccountNav({ locale, current }: { locale: Locale; current: string }) {
  const t = translator(locale);
  const items = [
    { href: "/account", key: "account.overview" },
    { href: "/account/orders", key: "orders.title" },
    { href: "/account/wishlist", key: "account.wishlist" },
    { href: "/account/addresses", key: "account.addresses" },
    { href: "/account/wallet", key: "account.wallet" },
    { href: "/account/points", key: "account.points" },
    { href: "/account/notifications", key: "account.notifications" },
    { href: "/support", key: "account.support" },
  ];
  return (
    <nav className="flex flex-wrap gap-2 text-sm" aria-label={t("account.title")}>
      {items.map((item) => {
        const active = current === item.href;
        return (
          <Link
            key={item.href}
            href={item.href}
            aria-current={active ? "page" : undefined}
            className={
              active
                ? "bg-primary text-primary-foreground rounded-full px-3 py-1.5"
                : "hover:border-primary/40 rounded-full border px-3 py-1.5"
            }
          >
            {t(item.key)}
          </Link>
        );
      })}
    </nav>
  );
}
