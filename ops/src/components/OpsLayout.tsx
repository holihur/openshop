import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { useState } from "react";
import {
  BadgePercent,
  BarChart3,
  Boxes,
  ExternalLink,
  LogOut,
  Menu,
  RotateCcw,
  LifeBuoy,
  HandCoins,
  Banknote,
  KeyRound,
  Inbox,
  ScrollText,
  ShoppingCart,
  SlidersHorizontal,
  Star,
  Store,
  Tags,
  Truck,
  Users,
  Wallet,
  X,
  type LucideIcon,
} from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { LocaleSwitcher } from "@lib/components/locale-switcher";
import { useI18n } from "@lib/i18n";
import type { MessageKey } from "@lib/i18n/messages";
import { cn } from "@lib/utils";
import { useAuth } from "@lib/auth";

// The public storefront lives on a different host/port (this console is on the
// intranet), so the link is configurable at build time.
const storefrontURL = import.meta.env.VITE_STOREFRONT_URL ?? "http://localhost:8080";

const nav: { to: string; label: MessageKey; icon: LucideIcon; end?: boolean; perm?: string }[] = [
  { to: "/", label: "ops.dashboard", icon: BarChart3, end: true, perm: "analytics:read" },
  { to: "/products", label: "ops.products", icon: Boxes, perm: "products:read" },
  { to: "/categories", label: "ops.categories", icon: Tags, perm: "categories:write" },
  { to: "/orders", label: "ops.orders", icon: ShoppingCart, perm: "orders:read" },
  { to: "/customers", label: "ops.customers", icon: Users, perm: "customers:read" },
  { to: "/coupons", label: "ops.coupons", icon: BadgePercent, perm: "coupons:read" },
  { to: "/reviews", label: "ops.reviews", icon: Star, perm: "reviews:read" },
  { to: "/returns", label: "ops.returns", icon: RotateCcw, perm: "returns:read" },
  { to: "/tickets", label: "ops.tickets", icon: LifeBuoy, perm: "tickets:read" },
  { to: "/commissions", label: "ops.commissions", icon: HandCoins, perm: "loyalty:read" },
  { to: "/withdrawals", label: "ops.withdrawals", icon: Banknote, perm: "withdrawals:read" },
  { to: "/tokens", label: "ops.tokens", icon: KeyRound },
  { to: "/outbox", label: "ops.outbox", icon: Inbox, perm: "audit:read" },
  { to: "/shipping", label: "ops.shipping", icon: Truck, perm: "shipping:read" },
  { to: "/currency", label: "ops.currency", icon: Wallet, perm: "currency:write" },
  { to: "/audit", label: "ops.audit", icon: ScrollText, perm: "audit:read" },
  { to: "/settings", label: "ops.settings", icon: SlidersHorizontal, perm: "settings:read" },
];

/** Chrome for the operations console: sidebar navigation, storefront link and sign-out. */
export function OpsLayout() {
  const { user, logout } = useAuth();
  const { t } = useI18n();
  const navigate = useNavigate();
  const [navOpen, setNavOpen] = useState(false);

  // Hide sections the caller cannot access. A missing permissions list (older
  // token) is treated permissively; the API enforces access regardless.
  const can = (perm?: string) => !perm || !user?.permissions || user.permissions.includes(perm);
  const items = nav.filter((item) => can(item.perm));

  async function onLogout() {
    await logout();
    navigate("/login", { replace: true });
  }

  return (
    <div className="bg-background flex min-h-screen flex-col">
      <header className="bg-background/80 sticky top-0 z-20 border-b backdrop-blur">
        <div className="flex h-14 items-center justify-between gap-2 px-4">
          <div className="flex items-center gap-2">
            <button
              type="button"
              className="text-muted-foreground hover:text-foreground -ml-1 inline-flex size-9 items-center justify-center rounded-md md:hidden"
              aria-label={navOpen ? t("nav.closeMenu") : t("nav.openMenu")}
              aria-expanded={navOpen}
              onClick={() => setNavOpen((o) => !o)}
            >
              {navOpen ? <X className="size-5" /> : <Menu className="size-5" />}
            </button>
            <NavLink to="/" className="flex items-center gap-2 font-semibold">
              <Store className="size-5" />
              OpenShop Ops
            </NavLink>
          </div>
          <div className="flex items-center gap-2 text-sm">
            <span className="text-muted-foreground hidden lg:inline">{user?.email}</span>
            <Button variant="outline" size="sm" asChild>
              <a href={storefrontURL} target="_blank" rel="noreferrer" aria-label={t("common.storefront")}>
                <ExternalLink className="size-4" />
                <span className="hidden sm:inline">{t("common.storefront")}</span>
              </a>
            </Button>
            <Button variant="ghost" size="sm" onClick={onLogout} aria-label={t("common.signOut")}>
              <LogOut className="size-4" />
              <span className="hidden sm:inline">{t("common.signOut")}</span>
            </Button>
          </div>
        </div>

        {navOpen && (
          <nav className="flex flex-col gap-1 border-t px-4 py-3 md:hidden">
            {items.map(({ to, label, icon: Icon, end }) => (
              <NavLink
                key={to}
                to={to}
                end={end}
                onClick={() => setNavOpen(false)}
                className={({ isActive }) =>
                  cn(
                    "flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                    isActive
                      ? "bg-accent text-accent-foreground"
                      : "text-muted-foreground hover:bg-accent/50 hover:text-foreground",
                  )
                }
              >
                <Icon className="size-4" />
                {t(label)}
              </NavLink>
            ))}
          </nav>
        )}
      </header>

      <div className="mx-auto flex w-full max-w-7xl flex-1 gap-6 px-4 py-6">
        <aside className="hidden w-52 shrink-0 md:block">
          <nav className="sticky top-20 space-y-1">
            {items.map(({ to, label, icon: Icon, end }) => (
              <NavLink
                key={to}
                to={to}
                end={end}
                className={({ isActive }) =>
                  cn(
                    "flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                    isActive
                      ? "bg-accent text-accent-foreground"
                      : "text-muted-foreground hover:bg-accent/50 hover:text-foreground",
                  )
                }
              >
                <Icon className="size-4" />
                {t(label)}
              </NavLink>
            ))}
          </nav>
        </aside>

        <main className="min-w-0 flex-1">
          <Outlet />
        </main>
      </div>

      <footer className="text-muted-foreground border-t">
        <div className="mx-auto flex w-full max-w-7xl items-center justify-between gap-3 px-4 py-4 text-sm">
          <span>© {new Date().getFullYear()} OpenShop Ops</span>
          <LocaleSwitcher />
        </div>
      </footer>
    </div>
  );
}
