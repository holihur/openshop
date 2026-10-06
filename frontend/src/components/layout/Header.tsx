import { Link, NavLink } from "react-router-dom";
import { Moon, ShoppingBag, ShoppingCart, Sun, User } from "lucide-react";
import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { useAuth } from "@/lib/auth";
import { useCart } from "@/hooks/useCart";
import { cn } from "@/lib/utils";

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

const navItems = [
  { to: "/", label: "Home", end: true },
  { to: "/products", label: "Products" },
  { to: "/orders", label: "Orders" },
];

export function Header() {
  const { user, logout } = useAuth();
  const { data: cart } = useCart();
  const { dark, toggle } = useTheme();

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
              {item.label}
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
              Addresses
            </NavLink>
          )}
          {user?.role === "admin" && (
            <NavLink
              to="/admin"
              className={({ isActive }) =>
                cn(
                  "rounded-md px-3 py-2 text-sm font-medium transition-colors",
                  isActive
                    ? "bg-accent text-accent-foreground"
                    : "text-muted-foreground hover:text-foreground",
                )
              }
            >
              Admin
            </NavLink>
          )}
        </nav>

        <div className="ml-auto flex items-center gap-2">
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
                Sign out
              </Button>
            </div>
          ) : (
            <Button size="sm" asChild>
              <Link to="/login">
                <User className="size-4" />
                Sign in
              </Link>
            </Button>
          )}
        </div>
      </div>
    </header>
  );
}
