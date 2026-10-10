import type { Metadata } from "next";
import Link from "next/link";
import { cookies, headers } from "next/headers";

import { CartBadge } from "@/components/cart-badge";
import { NotificationBadge } from "@/components/notification-badge";
import { LocaleSwitcher } from "@/components/locale-switcher";
import { getSiteConfig, isSignedIn } from "@/lib/session";
import { brandColor, contrastForeground, withAlpha } from "@/lib/theme";
import { localeFromHeader, translator, type Locale } from "@/lib/i18n";

import "./globals.css";

/** Resolves the locale: an explicit choice wins over the request header. */
export async function resolveLocale(): Promise<Locale> {
  const jar = await cookies();
  const chosen = jar.get("ssr_lang")?.value;
  if (chosen === "en" || chosen === "zh") return chosen;
  return localeFromHeader((await headers()).get("accept-language"));
}

export async function generateMetadata(): Promise<Metadata> {
  const site = await getSiteConfig().catch(() => null);
  const name = site?.tagline?.trim() || "openshop";
  return {
    title: { default: name, template: `%s · ${name}` },
    description: site?.hero?.subtitle || undefined,
    applicationName: name,
  };
}

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const locale = await resolveLocale();
  const t = translator(locale);
  // The storefront configuration lives in the ops console, so the header, the
  // theme and the announcement reflect whatever the operator last saved.
  const [site, signedIn] = await Promise.all([getSiteConfig().catch(() => null), isSignedIn()]);
  const brand = brandColor(site?.themeColor);
  const onBrand = contrastForeground(brand);
  const storeName = site?.tagline?.trim() || "openshop";

  return (
    <html lang={locale}>
      <body className="flex min-h-screen flex-col antialiased">
        {/* The brand colour is the single theme input; everything else derives
            from it so a pale colour still gets readable text. */}
        <style>{`:root{
          --brand:${brand};
          --brand-foreground:${onBrand};
          --brand-soft:${withAlpha(brand, 0.08)};
          --brand-ring:${withAlpha(brand, 0.35)};
        }`}</style>

        <header
          className="border-b backdrop-blur"
          style={{ borderColor: withAlpha(brand, 0.15) }}
        >
          <div className="mx-auto flex max-w-6xl flex-wrap items-center gap-x-6 gap-y-2 px-4 py-3">
            <Link href="/" className="flex items-center gap-2 text-lg font-bold tracking-tight">
              <span
                className="grid size-7 place-items-center rounded-lg text-sm font-black"
                style={{ background: brand, color: onBrand }}
              >
                {storeName.slice(0, 1).toUpperCase()}
              </span>
              {storeName}
            </Link>
            <nav className="flex items-center gap-5 text-sm">
              <Link href="/" className="hover:opacity-70">
                {t("nav.home")}
              </Link>
              <Link href="/products" className="hover:opacity-70">
                {t("nav.products")}
              </Link>
            </nav>
            <div className="ml-auto flex items-center gap-4 text-sm">
              <LocaleSwitcher locale={locale} />
              <Link href="/cart" className="flex items-center gap-1.5 hover:opacity-70">
                <span
                  className="grid size-7 place-items-center rounded-full"
                  style={{ background: withAlpha(brand, 0.12), color: brand }}
                  aria-hidden="true"
                >
                  <svg viewBox="0 0 24 24" className="size-4" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M6 6h15l-1.5 9h-12z" />
                    <circle cx="9" cy="20" r="1.4" />
                    <circle cx="18" cy="20" r="1.4" />
                    <path d="M6 6 5 2H2" />
                  </svg>
                </span>
                {t("nav.cart")}
                <CartBadge />
              </Link>
              <Link href="/support" className="hover:opacity-70">
                {t("nav.support")}
              </Link>
              {signedIn ? (
                <>
                  <NotificationBadge label={t("account.notifications")} />
                  <Link href="/account" className="hover:opacity-70">
                    {t("nav.account")}
                  </Link>
                </>
              ) : (
                <Link
                  href="/login"
                  className="rounded-md px-3 py-1.5 font-medium"
                  style={{ background: brand, color: onBrand }}
                >
                  {t("nav.signIn")}
                </Link>
              )}
            </div>
          </div>
          {site?.announcement?.message ? (
            <p
              className="px-4 py-2 text-center text-sm"
              style={{ background: withAlpha(brand, 0.08), color: brand }}
            >
              {site.announcement.url ? (
                <Link href={site.announcement.url} className="underline">
                  {site.announcement.message}
                </Link>
              ) : (
                site.announcement.message
              )}
            </p>
          ) : null}
        </header>

        <main className="mx-auto w-full max-w-6xl flex-1 px-4 py-8">{children}</main>

        <footer className="text-muted-foreground border-t px-4 py-8 text-sm">
          <div className="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-2">
            <span>
              © {new Date().getFullYear()} {storeName}
            </span>
            <span className="flex items-center gap-4">
              <Link href="/products" className="hover:opacity-70">
                {t("nav.products")}
              </Link>
              <Link href="/support" className="hover:opacity-70">
                {t("nav.support")}
              </Link>
              <Link href="/account" className="hover:opacity-70">
                {t("nav.account")}
              </Link>
            </span>
          </div>
        </footer>
      </body>
    </html>
  );
}
