import { useEffect, useRef, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { ChevronDown, Gift, Heart, LifeBuoy, LogOut, MapPin, Package, Settings, User as UserIcon, Wallet } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { useAuth } from "@lib/auth";
import { useI18n } from "@lib/i18n";

/** Account dropdown: username, orders, addresses, wishlist, settings, sign out. */
export function AccountMenu() {
  const { user, logout } = useAuth();
  const { t } = useI18n();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const onDoc = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    document.addEventListener("mousedown", onDoc);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDoc);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  if (!user) {
    return (
      <Button size="sm" asChild>
        <Link to="/login" aria-label={t("nav.signIn")}>
          <UserIcon className="size-4" />
          <span className="hidden sm:inline">{t("nav.signIn")}</span>
        </Link>
      </Button>
    );
  }

  const items = [
    { to: "/orders", label: t("nav.orders"), icon: Package },
    { to: "/support", label: t("nav.support"), icon: LifeBuoy },
    { to: "/account/wallet", label: t("nav.wallet"), icon: Wallet },
    { to: "/account/rewards", label: t("nav.rewards"), icon: Gift },
    { to: "/account/addresses", label: t("nav.addresses"), icon: MapPin },
    { to: "/account/wishlist", label: t("nav.wishlist"), icon: Heart },
    { to: "/account/settings", label: t("nav.settings"), icon: Settings },
  ];

  async function onLogout() {
    setOpen(false);
    await logout();
    navigate("/");
  }

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        data-testid="account-menu"
        aria-haspopup="menu"
        aria-expanded={open}
        onClick={() => setOpen((o) => !o)}
        className="hover:bg-accent flex items-center gap-1.5 rounded-md px-1.5 py-1.5 text-sm"
      >
        <span className="bg-primary text-primary-foreground flex size-6 items-center justify-center rounded-full text-xs font-medium">
          {(user.name || user.email).charAt(0).toUpperCase()}
        </span>
        <span className="hidden max-w-24 truncate sm:inline">{user.name || user.email}</span>
        <ChevronDown className="size-3.5" />
      </button>

      {open && (
        <div
          role="menu"
          className="bg-popover absolute right-0 z-50 mt-1 w-52 rounded-md border p-1 shadow-md"
        >
          <div className="text-muted-foreground truncate px-2 py-1.5 text-xs">{user.email}</div>
          {items.map(({ to, label, icon: Icon }) => (
            <Link
              key={to}
              to={to}
              role="menuitem"
              onClick={() => setOpen(false)}
              className="hover:bg-accent flex items-center gap-2 rounded-sm px-2 py-1.5 text-sm"
            >
              <Icon className="size-4" />
              {label}
            </Link>
          ))}
          <div className="bg-border my-1 h-px" />
          <button
            type="button"
            role="menuitem"
            onClick={() => void onLogout()}
            className="hover:bg-accent text-destructive flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-left text-sm"
          >
            <LogOut className="size-4" />
            {t("common.signOut")}
          </button>
        </div>
      )}
    </div>
  );
}
