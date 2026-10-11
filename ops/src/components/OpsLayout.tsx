import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import { useEffect, useState } from "react";
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
  Scale,
  ScrollText,
  ShoppingCart,
  SlidersHorizontal,
  Star,
  Store,
  Tags,
  Truck,
  ChevronDown,
  PanelLeftClose,
  PanelLeftOpen,
  Pin,
  Search,
  ShieldCheck,
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

type NavItem = { to: string; label: MessageKey; icon: LucideIcon; end?: boolean; perm?: string };

/**
 * The console has outgrown a flat list, so entries are grouped by the job they
 * belong to and each group collapses. A group whose entries are all hidden by
 * permissions disappears entirely, and a group opens by itself when the current
 * page is inside it.
 */
const navGroups: { id: string; label: MessageKey; icon: LucideIcon; items: NavItem[] }[] = [
  {
    id: "overview",
    label: "nav.group.overview",
    icon: BarChart3,
    items: [
      { to: "/", label: "ops.dashboard", icon: BarChart3, end: true, perm: "analytics:read" },
      { to: "/reconciliation", label: "ops.reconciliation", icon: Scale, perm: "audit:read" },
      { to: "/outbox", label: "ops.outbox", icon: Inbox, perm: "audit:read" },
    ],
  },
  {
    id: "catalog",
    label: "nav.group.catalog",
    icon: Boxes,
    items: [
      { to: "/products", label: "ops.products", icon: Boxes, perm: "products:read" },
      { to: "/categories", label: "ops.categories", icon: Tags, perm: "categories:write" },
      { to: "/reviews", label: "ops.reviews", icon: Star, perm: "reviews:read" },
      { to: "/coupons", label: "ops.coupons", icon: BadgePercent, perm: "coupons:read" },
    ],
  },
  {
    id: "sales",
    label: "nav.group.sales",
    icon: ShoppingCart,
    items: [
      { to: "/orders", label: "ops.orders", icon: ShoppingCart, perm: "orders:read" },
      { to: "/returns", label: "ops.returns", icon: RotateCcw, perm: "returns:read" },
      { to: "/customers", label: "ops.customers", icon: Users, perm: "customers:read" },
    ],
  },
  {
    id: "support",
    label: "nav.group.support",
    icon: LifeBuoy,
    items: [
      { to: "/tickets", label: "ops.tickets", icon: LifeBuoy, perm: "tickets:read" },
      { to: "/commissions", label: "ops.commissions", icon: HandCoins, perm: "loyalty:read" },
      { to: "/withdrawals", label: "ops.withdrawals", icon: Banknote, perm: "withdrawals:read" },
    ],
  },
  {
    id: "settings",
    label: "nav.group.settings",
    icon: SlidersHorizontal,
    items: [
      { to: "/shipping", label: "ops.shipping", icon: Truck, perm: "shipping:read" },
      { to: "/currency", label: "ops.currency", icon: Wallet, perm: "currency:write" },
      { to: "/settings", label: "ops.settings", icon: SlidersHorizontal, perm: "settings:read" },
    ],
  },
  {
    id: "access",
    label: "nav.group.access",
    icon: ShieldCheck,
    items: [
      { to: "/staff", label: "staff.nav", icon: Users, perm: "staff:read" },
      { to: "/roles", label: "roles.title", icon: ShieldCheck, perm: "staff:read" },
      { to: "/tokens", label: "ops.tokens", icon: KeyRound },
      { to: "/audit", label: "ops.audit", icon: ScrollText, perm: "audit:read" },
    ],
  },
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
  const location = useLocation();
  // Visibility is decided by permission; a group with nothing to show is hidden.
  const groups = navGroups
    .map((group) => ({ ...group, items: group.items.filter((item) => can(item.perm)) }))
    .filter((group) => group.items.length > 0);

  // The sidebar itself can be folded to an icon rail, and which groups are
  // collapsed is a browser preference: both survive reloads and navigation, so
  // the console opens the way the operator left it.
  const [rail, setRail] = useState<boolean>(() => localStorage.getItem("ops.nav.rail") === "1");
  const toggleRail = () =>
    setRail((current) => {
      localStorage.setItem("ops.nav.rail", current ? "0" : "1");
      return !current;
    });

  const [collapsed, setCollapsed] = useState<Record<string, boolean>>(() => {
    try {
      const raw = localStorage.getItem("ops.nav.collapsed");
      return raw ? (JSON.parse(raw) as Record<string, boolean>) : {};
    } catch {
      return {};
    }
  });
  const setGroupOpen = (id: string, open: boolean) =>
    setCollapsed((current) => {
      const next = { ...current, [id]: !open };
      localStorage.setItem("ops.nav.collapsed", JSON.stringify(next));
      return next;
    });
  const [query, setQuery] = useState("");
  // Pinned entries are a browser preference, not part of the account, so they
  // live in local storage and never need a round trip.
  const [pinned, setPinned] = useState<string[]>(() => {
    try {
      const raw = localStorage.getItem("ops.nav.pinned");
      return raw ? (JSON.parse(raw) as string[]) : [];
    } catch {
      return [];
    }
  });
  const togglePin = (to: string) => {
    setPinned((current) => {
      const next = current.includes(to) ? current.filter((item) => item !== to) : [...current, to];
      localStorage.setItem("ops.nav.pinned", JSON.stringify(next));
      return next;
    });
  };
  // The group holding the current page is always open.
  const activeGroup = groups.find((group) =>
    group.items.some(
      (item) => location.pathname === item.to ||
        (item.to !== "/" && location.pathname.startsWith(item.to)),
    ),
  )?.id;
  useEffect(() => {
    // The group holding the current page opens by itself, but only if the
    // operator has not explicitly collapsed it: an explicit choice wins.
    if (!activeGroup) return;
    setCollapsed((current) => {
      if (current[activeGroup] !== true) return current;
      const next = { ...current, [activeGroup]: false };
      localStorage.setItem("ops.nav.collapsed", JSON.stringify(next));
      return next;
    });
  }, [activeGroup]);

  // A search narrows every group at once and opens them, so a match is never
  // hidden behind a collapsed heading.
  const needle = query.trim().toLowerCase();
  const matches = (item: NavItem) => !needle || t(item.label).toLowerCase().includes(needle);
  const visibleGroups = groups
    .map((group) => ({ ...group, items: group.items.filter(matches) }))
    .filter((group) => group.items.length > 0);

  const pinnedItems = groups
    .flatMap((group) => group.items)
    .filter((item) => pinned.includes(item.to));

  const renderItems = (variant: "desktop" | "mobile", list: NavItem[], indent: boolean) => {
    // In rail mode an entry is an icon with a tooltip: the label lives in the
    // title attribute so the width never changes.
    const compact = variant === "desktop" && rail;
    return list.map(({ to, label, icon: Icon, end }) => (
      <div key={`${variant}-${to}`} className="group/item relative">
        <NavLink
          to={to}
          end={end}
          title={compact ? t(label) : undefined}
          aria-label={compact ? t(label) : undefined}
          onClick={() => setNavOpen(false)}
          className={({ isActive }) =>
            cn(
              "flex items-center gap-2 rounded-md py-1.5 text-sm font-medium transition-colors",
              compact ? "justify-center px-2" : indent ? "pr-7 pl-6" : "pr-7 pl-3",
              isActive
                ? "bg-accent text-accent-foreground"
                : "text-muted-foreground hover:bg-accent/50 hover:text-foreground",
            )
          }
        >
          <Icon className="size-4 shrink-0" />
          {!compact && t(label)}
        </NavLink>
        {!compact && (
        <button
          type="button"
          onClick={() => togglePin(to)}
          aria-pressed={pinned.includes(to)}
          aria-label={pinned.includes(to) ? t("nav.unpin") : t("nav.pin")}
          title={pinned.includes(to) ? t("nav.unpin") : t("nav.pin")}
          className={cn(
            "text-muted-foreground hover:text-foreground absolute top-1.5 right-2",
            pinned.includes(to) ? "opacity-100" : "opacity-0 group-hover/item:opacity-100",
          )}
        >
          <Pin className={cn("size-3.5", pinned.includes(to) ? "fill-current" : "")} />
        </button>
        )}
      </div>
    ));
  };

  const renderNav = (variant: "desktop" | "mobile") => (
    <>
      {!(variant === "desktop" && rail) && (
      <div className="relative px-3 pb-2">
        {/* Explicit size and position: the icon must look right even if a
            cached stylesheet is out of date. */}
        <Search
          width={14}
          height={14}
          className="text-muted-foreground absolute"
          style={{ left: "1.25rem", top: "50%", transform: "translateY(-50%)" }}
        />
        <input
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Escape") setQuery("");
          }}
          placeholder={t("nav.search")}
          aria-label={t("nav.search")}
          className="border-input bg-background h-8 w-full rounded-md border pr-2 pl-7 text-sm"
        />
      </div>
      )}

      {pinnedItems.length > 0 && !needle && (
        <div className="space-y-1">
          <p className="text-muted-foreground px-3 py-1.5 text-xs font-semibold tracking-wide uppercase">
            {t("nav.pinned")}
          </p>
          {renderItems(variant, pinnedItems, true)}
        </div>
      )}

      {needle &&
        visibleGroups.map((group) => (
          <div key={`${variant}-search-${group.id}`} className="space-y-1">
            <p className="text-muted-foreground px-3 py-1.5 text-xs font-semibold tracking-wide uppercase">
              {t(group.label)}
            </p>
            {renderItems(variant, group.items, true)}
          </div>
        ))}

      {!needle &&
        visibleGroups.map((group) => {
      const open = !collapsed[group.id];
      const GroupIcon = group.icon;
      return (
        <div key={group.id} className="space-y-1">
          {variant === "desktop" && rail ? (
            <div className="border-t px-2 py-1.5" aria-hidden="true" />
          ) : (
          <button
            type="button"
            onClick={() => setGroupOpen(group.id, open)}
            aria-expanded={open}
            className="text-muted-foreground hover:text-foreground flex w-full items-center gap-2 px-3 py-1.5 text-xs font-semibold tracking-wide uppercase"
          >
            <GroupIcon className="size-3.5" />
            <span className="flex-1 text-left">{t(group.label)}</span>
            <ChevronDown className={cn("size-3.5 transition-transform", open ? "" : "-rotate-90")} />
          </button>
          )}
          {(rail || open) && renderItems(variant, group.items, true)}
        </div>
      );
    })}

      {needle && visibleGroups.length === 0 && (
        <p className="text-muted-foreground px-3 py-2 text-sm">{t("nav.noMatches")}</p>
      )}
    </>
  );


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
            {/* Fold the sidebar to an icon rail; the choice is remembered. */}
            <Button
              variant="ghost"
              size="sm"
              onClick={toggleRail}
              aria-pressed={rail}
              aria-label={rail ? t("nav.expandSidebar") : t("nav.collapseSidebar")}
              title={rail ? t("nav.expandSidebar") : t("nav.collapseSidebar")}
              className="hidden md:inline-flex"
            >
              {rail ? <PanelLeftOpen className="size-4" /> : <PanelLeftClose className="size-4" />}
            </Button>
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
          <nav className="max-h-[70vh] space-y-2 overflow-y-auto border-t px-4 py-3 md:hidden">
            {renderNav("mobile")}
          </nav>
        )}
      </header>

      <div className="mx-auto flex w-full max-w-7xl flex-1 gap-6 px-4 py-6">
        <aside className={cn("hidden shrink-0 md:block", rail ? "w-14" : "w-60")}>
          <nav className="sticky top-20 space-y-2">{renderNav("desktop")}</nav>
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
