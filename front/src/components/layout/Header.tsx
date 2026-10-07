import { Link, NavLink } from "react-router-dom";
import { LogOut, Menu, Moon, ShoppingBag, ShoppingCart, Sun, User, X } from "lucide-react";
import { useEffect, useState } from "react";

import { Button } from "@lib/components/ui/button";
import { Badge } from "@lib/components/ui/badge";
import { useAuth } from "@lib/auth";
import { useCurrency } from "@lib/currency";
import { useCart } from "@lib/hooks/useCart";
import { useI18n } from "@lib/i18n";
import type { MessageKey } from "@lib/i18n/messages";
import { LocaleSwitcher } from "@lib/components/locale-switcher";
import { cn } from "@lib/utils";

function useTheme() {
  const [dark, setDark] = useState(
    () => localStorage.getItem("openshop.theme") === "dark",
  );
  useEffect(() => {
    document.documentElement.classList.toggle("dark", dark);
    localStorage.setItem("openshop.theme", dark ? "dark" : "light");
  }, [dark]);
  return { dark, toggle: () => setDark((d) => !d) };
}

const navItems: { to: string; label: MessageKey; end?: boolean }[] = [
  { to: "/", label: "nav.home", end: true },
  { to: "/products", label: "nav.products" },
];

const ordersItem: { to: string; label: MessageKey; end?: boolean } = { to: "/orders", label: "nav.orders" };

const accountItems: { to: string; label: MessageKey }[] = [
  { to: "/account/addresses", label: "nav.addresses" },
  { to: "/account/wishlist", label: "nav.wishlist" },
  { to: "/account/settings", label: "nav.settings" },
];

const linkClass = ({ isActive }: { isActive: boolean }) =>
  cn(
    "rounded-md px-3 py-2 text-sm font-medium transition-colors",
    isActive ? "bg-accent text-accent-foreground" : "text-muted-foreground hover:text-foreground",
  );

export function Header() {
  const { user, logout } = useAuth();
  const { data: cart } = useCart();
  const { dark, toggle } = useTheme();
  const { currency, available, setCurrency } = useCurrency();
  const { t } = useI18n();
  const [menuOpen, setMenuOpen] = useState(false);
  // Orders only makes sense for a signed-in shopper.
  const links = user ? [...navItems, ordersItem] : navItems;

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

        {/* Full navigation on large screens. */}
        <nav className="ml-4 hidden items-center gap-1 lg:flex">
          {links.map((item) => (
            <NavLink key={item.to} to={item.to} end={item.end} className={linkClass}>
              {t(item.label)}
            </NavLink>
          ))}
          {user &&
            accountItems.map((item) => (
              <NavLink key={item.to} to={item.to} className={linkClass}>
                {t(item.label)}
              </NavLink>
            ))}
        </nav>

        <div className="ml-auto flex items-center gap-1.5 sm:gap-2">
          {available.length > 1 && (
            <select
              aria-label={t("common.currency")}
              value={currency}
              onChange={(e) => setCurrency(e.target.value)}
              className="border-input bg-background hidden h-8 rounded-md border px-2 text-sm sm:block"
            >
              {available.map((c) => (
                <option key={c} value={c}>
                  {c}
                </option>
              ))}
            </select>
          )}
          <LocaleSwitcher className="hidden sm:inline-flex" />
          <Button variant="ghost" size="icon" onClick={toggle} aria-label={t("common.toggleTheme")}>
            {dark ? <Sun className="size-4" /> : <Moon className="size-4" />}
          </Button>

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

          {user ? (
            <div className="flex items-center gap-2">
              <span className="text-muted-foreground hidden max-w-32 truncate text-sm xl:inline">
                {user.name || user.email}
              </span>
              <Button
                variant="outline"
                size="sm"
                onClick={() => void logout()}
                aria-label={t("common.signOut")}
              >
                <LogOut className="size-4" />
                <span className="hidden sm:inline">{t("common.signOut")}</span>
              </Button>
            </div>
          ) : (
            <Button size="sm" asChild>
              <Link to="/login" aria-label={t("nav.signIn")}>
                <User className="size-4" />
                <span className="hidden sm:inline">{t("nav.signIn")}</span>
              </Link>
            </Button>
          )}
        </div>
      </div>

      {/* Collapsible navigation on small/medium screens. */}
      {menuOpen && (
        <nav className="mx-auto flex max-w-6xl flex-col gap-1 border-t px-4 py-3 lg:hidden">
          {links.map((item) => (
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
          {user &&
            accountItems.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
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
