"use client";

import { useRouter } from "next/navigation";
import { useTransition } from "react";

import type { Locale } from "@/lib/i18n";

/**
 * Writes the chosen locale to a cookie and re-renders the current page on the
 * server, so the language applies to the HTML that arrives from now on.
 */
export function LocaleSwitcher({ locale }: { locale: Locale }) {
  const router = useRouter();
  const [pending, startTransition] = useTransition();

  function choose(next: Locale) {
    document.cookie = `ssr_lang=${next}; path=/; max-age=${60 * 60 * 24 * 365}`;
    startTransition(() => router.refresh());
  }

  return (
    <span className="flex items-center gap-1">
      <button
        type="button"
        onClick={() => choose("en")}
        aria-pressed={locale === "en"}
        className={locale === "en" ? "font-semibold" : "text-muted-foreground"}
        disabled={pending}
      >
        EN
      </button>
      <span aria-hidden="true">/</span>
      <button
        type="button"
        onClick={() => choose("zh")}
        aria-pressed={locale === "zh"}
        className={locale === "zh" ? "font-semibold" : "text-muted-foreground"}
        disabled={pending}
      >
        中文
      </button>
    </span>
  );
}
