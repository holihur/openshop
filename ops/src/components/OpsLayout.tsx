import { Link, Outlet, useNavigate } from "react-router-dom";
import { LogOut, Store } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { useAuth } from "@lib/auth";

/** Chrome for the operations console: brand, storefront link and sign-out. */
export function OpsLayout() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();

  async function onLogout() {
    await logout();
    navigate("/login", { replace: true });
  }

  return (
    <div className="bg-background min-h-screen">
      <header className="bg-background/80 sticky top-0 z-10 border-b backdrop-blur">
        <div className="mx-auto flex h-14 max-w-7xl items-center justify-between px-4">
          <Link to="/" className="flex items-center gap-2 font-semibold">
            <Store className="size-5" />
            OpenShop Ops
          </Link>
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
      <main className="mx-auto max-w-7xl px-4 py-6">
        <Outlet />
      </main>
    </div>
  );
}
