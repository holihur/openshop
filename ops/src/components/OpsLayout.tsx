import { NavLink, Outlet, useNavigate } from "react-router-dom";
import {
  BadgePercent,
  BarChart3,
  Boxes,
  LogOut,
  ScrollText,
  ShoppingCart,
  Star,
  Store,
  Truck,
  Wallet,
} from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { cn } from "@lib/utils";
import { useAuth } from "@lib/auth";

const nav = [
  { to: "/", label: "Dashboard", icon: BarChart3, end: true },
  { to: "/products", label: "Products", icon: Boxes },
  { to: "/orders", label: "Orders", icon: ShoppingCart },
  { to: "/coupons", label: "Coupons", icon: BadgePercent },
  { to: "/reviews", label: "Reviews", icon: Star },
  { to: "/shipping", label: "Shipping", icon: Truck },
  { to: "/currency", label: "Currency", icon: Wallet },
  { to: "/audit", label: "Audit", icon: ScrollText },
];

/** Chrome for the operations console: sidebar navigation, storefront link and sign-out. */
export function OpsLayout() {
  const { user, logout } = useAuth();
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
            <span className="text-muted-foreground hidden sm:inline">{user?.email}</span>
            <Button variant="outline" size="sm" asChild>
              <a href="/" target="_blank" rel="noreferrer">
                Storefront
              </a>
            </Button>
            <Button variant="ghost" size="sm" onClick={onLogout}>
              <LogOut className="size-4" />
              Sign out
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
                {label}
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
