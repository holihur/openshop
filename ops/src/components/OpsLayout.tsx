import { NavLink, Outlet, useNavigate } from "react-router-dom";
import {
  BadgePercent,
  BarChart3,
  Boxes,
  ExternalLink,
  LogOut,
  RotateCcw,
  ScrollText,
  ShoppingCart,
  Star,
  Store,
  Truck,
  Wallet,
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

const nav: { to: string; label: MessageKey; icon: LucideIcon; end?: boolean }[] = [
  { to: "/", label: "ops.dashboard", icon: BarChart3, end: true },
  { to: "/products", label: "ops.products", icon: Boxes },
  { to: "/orders", label: "ops.orders", icon: ShoppingCart },
  { to: "/coupons", label: "ops.coupons", icon: BadgePercent },
  { to: "/reviews", label: "ops.reviews", icon: Star },
  { to: "/returns", label: "ops.returns", icon: RotateCcw },
  { to: "/shipping", label: "ops.shipping", icon: Truck },
  { to: "/currency", label: "ops.currency", icon: Wallet },
  { to: "/audit", label: "ops.audit", icon: ScrollText },
];

/** Chrome for the operations console: sidebar navigation, storefront link and sign-out. */
export function OpsLayout() {
  const { user, logout } = useAuth();
  const { t } = useI18n();
  const navigate = useNavigate();

  async function onLogout() {
    await logout();
    navigate("/login", { replace: true });
  }

  return (
    <div className="bg-background min-h-screen">
      <header className="bg-background/80 sticky top-0 z-20 border-b backdrop-blur">
        <div className="flex h-14 items-center justify-between px-4">
          <NavLink to="/" className="flex items-center gap-2 font-semibold">
            <Store className="size-5" />
            OpenShop Ops
          </NavLink>
          <div className="flex items-center gap-3 text-sm">
            <LocaleSwitcher className="hidden sm:inline-flex" />
            <span className="text-muted-foreground hidden sm:inline">{user?.email}</span>
            <Button variant="outline" size="sm" asChild>
              <a href={storefrontURL} target="_blank" rel="noreferrer">
                <ExternalLink className="size-4" />
                {t("common.storefront")}
              </a>
            </Button>
            <Button variant="ghost" size="sm" onClick={onLogout}>
              <LogOut className="size-4" />
              {t("common.signOut")}
            </Button>
          </div>
        </div>
      </header>

      <div className="mx-auto flex max-w-7xl gap-6 px-4 py-6">
        <aside className="hidden w-52 shrink-0 md:block">
          <nav className="sticky top-20 space-y-1">
            {nav.map(({ to, label, icon: Icon, end }) => (
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
    </div>
  );
}
