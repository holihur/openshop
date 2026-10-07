import { Languages } from "lucide-react";

import { cn } from "@lib/utils";
import { useI18n } from "@lib/i18n";
import { localeNames, supportedLocales } from "@lib/i18n/messages";

/** Compact language selector used in both the storefront and the ops console. */
export function LocaleSwitcher({ className }: { className?: string }) {
  const { locale, setLocale, t } = useI18n();
  return (
    <label className={cn("text-muted-foreground inline-flex items-center gap-1", className)}>
      <Languages className="size-4" aria-hidden />
      <span className="sr-only">{t("common.language")}</span>
      <select
        aria-label={t("common.language")}
        data-testid="locale-switcher"
        value={locale}
        onChange={(e) => setLocale(e.target.value as (typeof supportedLocales)[number])}
        className="bg-transparent text-sm outline-none"
      >
        {supportedLocales.map((code) => (
          <option key={code} value={code}>
            {localeNames[code]}
          </option>
        ))}
      </select>
    </label>
  );
}
