import { Link, NavLink } from "react-router-dom";
import { Moon, ShoppingBag, ShoppingCart, Sun, User } from "lucide-react";
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
  { to: "/orders", label: "nav.orders" },
];

export function Header() {
  const { user, logout } = useAuth();
  const { data: cart } = useCart();
  const { dark, toggle } = useTheme();
  const { currency, available, setCurrency } = useCurrency();
  const { t } = useI18n();

  return (
    <header className="bg-background/95 supports-[backdrop-filter]:bg-background/60 sticky top-0 z-40 border-b backdrop-blur">
      <div className="mx-auto flex h-16 max-w-6xl items-center gap-4 px-4">
        <Link to="/" className="flex items-center gap-2 font-semibold">
          <ShoppingBag className="size-5" />
          <span>OpenShop</span>
        </Link>

        <nav className="ml-4 hidden items-center gap-1 md:flex">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              className={({ isActive }) =>
                cn(
                  "rounded-md px-3 py-2 text-sm font-medium transition-colors",
                  isActive
                    ? "bg-accent text-accent-foreground"
                    : "text-muted-foreground hover:text-foreground",
                )
              }
            >
              {t(item.label)}
            </NavLink>
          ))}
          {user && (
            <NavLink
              to="/account/addresses"
              className={({ isActive }) =>
                cn(
                  "rounded-md px-3 py-2 text-sm font-medium transition-colors",
                  isActive
                    ? "bg-accent text-accent-foreground"
                    : "text-muted-foreground hover:text-foreground",
                )
              }
            >
              {t("nav.addresses")}
            </NavLink>
          )}
          {user && (
            <NavLink
              to="/account/wishlist"
              className={({ isActive }) =>
                cn(
                  "rounded-md px-3 py-2 text-sm font-medium transition-colors",
                  isActive
                    ? "bg-accent text-accent-foreground"
                    : "text-muted-foreground hover:text-foreground",
                )
              }
            >
              {t("nav.wishlist")}
            </NavLink>
          )}
          {user && (
            <NavLink
              to="/account/settings"
              className={({ isActive }) =>
                cn(
                  "rounded-md px-3 py-2 text-sm font-medium transition-colors",
                  isActive
                    ? "bg-accent text-accent-foreground"
                    : "text-muted-foreground hover:text-foreground",
                )
              }
            >
              {t("nav.settings")}
            </NavLink>
          )}
        </nav>

        <div className="ml-auto flex items-center gap-2">
          {available.length > 1 && (
            <select
              aria-label="Currency"
              value={currency}
              onChange={(e) => setCurrency(e.target.value)}
              className="border-input bg-background h-8 rounded-md border px-2 text-sm"
            >
              {available.map((c) => (
                <option key={c} value={c}>
                  {c}
                </option>
              ))}
            </select>
          )}
          <LocaleSwitcher className="hidden sm:inline-flex" />
          <Button variant="ghost" size="icon" onClick={toggle} aria-label="Toggle theme">
            {dark ? <Sun className="size-4" /> : <Moon className="size-4" />}
          </Button>

          <Button variant="ghost" size="icon" asChild aria-label="Cart">
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
              <span className="hidden text-sm text-muted-foreground sm:inline">
                {user.name || user.email}
              </span>
              <Button variant="outline" size="sm" onClick={() => void logout()}>
                {t("common.signOut")}
              </Button>
            </div>
          ) : (
            <Button size="sm" asChild>
              <Link to="/login">
                <User className="size-4" />
                {t("nav.signIn")}
              </Link>
            </Button>
          )}
        </div>
      </div>
    </header>
  );
}
