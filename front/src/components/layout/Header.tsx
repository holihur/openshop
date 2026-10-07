import { Link, NavLink } from "react-router-dom";
import { Menu, ShoppingBag, ShoppingCart, X } from "lucide-react";
import { useState } from "react";

import { Button } from "@lib/components/ui/button";
import { Badge } from "@lib/components/ui/badge";
import { AccountMenu } from "@lib/components/account-menu";
import { NotificationBell } from "@/components/notification-bell";
import { useCart } from "@lib/hooks/useCart";
import { useI18n } from "@lib/i18n";
import type { MessageKey } from "@lib/i18n/messages";
import { cn } from "@lib/utils";

const navItems: { to: string; label: MessageKey; end?: boolean }[] = [
  { to: "/", label: "nav.home", end: true },
  { to: "/products", label: "nav.products" },
];

const linkClass = ({ isActive }: { isActive: boolean }) =>
  cn(
    "rounded-md px-3 py-2 text-sm font-medium transition-colors",
    isActive ? "bg-accent text-accent-foreground" : "text-muted-foreground hover:text-foreground",
  );

export function Header() {
  const { data: cart } = useCart();
  const { t } = useI18n();
  const [menuOpen, setMenuOpen] = useState(false);

  return (
    <header className="bg-background/95 supports-[backdrop-filter]:bg-background/60 sticky top-0 z-40 border-b backdrop-blur">
      <div className="mx-auto flex h-16 max-w-6xl items-center gap-3 px-4">
        <button
          type="button"
          className="text-muted-foreground hover:text-foreground -ml-1 inline-flex size-9 items-center justify-center rounded-md lg:hidden"
          aria-label={menuOpen ? t("nav.closeMenu") : t("nav.openMenu")}
          aria-expanded={menuOpen}
          onClick={() => setMenuOpen((o) => !o)}
        >
          {menuOpen ? <X className="size-5" /> : <Menu className="size-5" />}
        </button>

        <Link to="/" className="flex shrink-0 items-center gap-2 font-semibold">
          <ShoppingBag className="size-5" />
          <span>OpenShop</span>
        </Link>

        {/* Primary navigation on large screens. */}
        <nav className="ml-4 hidden items-center gap-1 lg:flex">
          {navItems.map((item) => (
            <NavLink key={item.to} to={item.to} end={item.end} className={linkClass}>
              {t(item.label)}
            </NavLink>
          ))}
        </nav>

        <div className="ml-auto flex items-center gap-1.5 sm:gap-2">
          <Button variant="ghost" size="icon" asChild aria-label={t("nav.cart")}>
            <Link to="/cart" className="relative">
              <ShoppingCart className="size-4" />
              {cart && cart.totalCount > 0 && (
                <Badge className="absolute -top-1 -right-1 h-5 min-w-5 justify-center rounded-full px-1 text-[10px]">
                  {cart.totalCount}
                </Badge>
              )}
            </Link>
          </Button>

          <NotificationBell />
          <AccountMenu />
        </div>
      </div>

      {/* Collapsible primary navigation on small/medium screens. */}
      {menuOpen && (
        <nav className="mx-auto flex max-w-6xl flex-col gap-1 border-t px-4 py-3 lg:hidden">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              className={linkClass}
              onClick={() => setMenuOpen(false)}
            >
              {t(item.label)}
            </NavLink>
          ))}
        </nav>
      )}
    </header>
  );
}
