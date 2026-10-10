/**
 * Formats an amount in the storefront's settlement currency.
 *
 * narrowSymbol is used so a price reads "¥329.00" rather than "CN¥329.00": the
 * short form is what shoppers expect and it keeps cards aligned.
 */
export function formatMoney(cents: number, currency = "CNY", locale = "en"): string {
  const locales: Record<string, string> = { en: "en-US", zh: "zh-CN" };
  const options: Intl.NumberFormatOptions = {
    style: "currency",
    currency,
    currencyDisplay: "narrowSymbol",
  };
  try {
    return new Intl.NumberFormat(locales[locale] ?? "en-US", options).format(cents / 100);
  } catch {
    // An unknown currency code should not blank the price out.
    return new Intl.NumberFormat(locales[locale] ?? "en-US", {
      style: "decimal",
      minimumFractionDigits: 2,
    }).format(cents / 100);
  }
}

/** Formats an ISO date in the current locale. */
export function formatDate(iso: string, locale = "en"): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "";
  return new Intl.DateTimeFormat(locale === "zh" ? "zh-CN" : "en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
  }).format(date);
}

/** Picks the localised title, falling back to the default one. */
export function localizedName(
  entity: { title?: string; name?: string; names?: Record<string, string> },
  locale: string,
): string {
  return entity.names?.[locale] ?? entity.title ?? entity.name ?? "";
}

/** Picks the localised category name. */
export function translateCategoryName(
  category: { name: string; names?: Record<string, string> },
  locale: string,
): string {
  return category.names?.[locale] ?? category.name;
}
