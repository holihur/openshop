import type { Metadata } from "next";
import Link from "next/link";
import { cookies, headers } from "next/headers";

import { CartBadge } from "@/components/cart-badge";
import { LocaleSwitcher } from "@/components/locale-switcher";
import { getSiteConfig } from "@/lib/session";
import { localeFromHeader, translator, type Locale } from "@/lib/i18n";

import "./globals.css";

export const metadata: Metadata = {
  title: "openshop",
  description: "A production-grade storefront, rendered on the server.",
};

/** Resolves the locale: an explicit choice wins over the request header. */
export async function resolveLocale(): Promise<Locale> {
  const jar = await cookies();
  const chosen = jar.get("ssr_lang")?.value;
  if (chosen === "en" || chosen === "zh") return chosen;
  return localeFromHeader((await headers()).get("accept-language"));
}

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const locale = await resolveLocale();
  const t = translator(locale);
  // The storefront configuration lives in the ops console, so the header and
  // theme reflect whatever the operator last saved.
  const site = await getSiteConfig().catch(() => null);
  const themeColor = site?.themeColor || "#111827";

  return (
    <html lang={locale}>
      <body className="min-h-screen antialiased">
        <style>{`:root { --brand: ${themeColor}; }`}</style>
        <header className="border-b">
          <div className="mx-auto flex max-w-6xl items-center gap-4 px-4 py-3">
            <Link href="/" className="text-lg font-bold">
              {site?.tagline || "openshop"}
            </Link>
            <nav className="flex items-center gap-4 text-sm">
              <Link href="/">{t("nav.home")}</Link>
              <Link href="/products">{t("nav.products")}</Link>
            </nav>
            <div className="ml-auto flex items-center gap-4 text-sm">
              <LocaleSwitcher locale={locale} />
              <Link href="/cart" className="flex items-center gap-1">
                {t("nav.cart")}
                <CartBadge />
              </Link>
              <Link href="/account/orders">{t("nav.account")}</Link>
            </div>
          </div>
          {site?.announcement?.text ? (
            <p className="bg-muted px-4 py-2 text-center text-sm">{site.announcement.text}</p>
          ) : null}
        </header>
        <main className="mx-auto max-w-6xl px-4 py-8">{children}</main>
        <footer className="text-muted-foreground border-t px-4 py-8 text-center text-sm">
          {site?.publicUrl ? new URL(site.publicUrl).host : "openshop"}
        </footer>
      </body>
    </html>
  );
}
