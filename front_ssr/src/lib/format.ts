/** Formats an amount in the storefront's settlement currency. */
export function formatMoney(cents: number, currency = "CNY", locale = "en"): string {
  const locales: Record<string, string> = { en: "en-US", zh: "zh-CN" };
  return new Intl.NumberFormat(locales[locale] ?? "en-US", {
    style: "currency",
    currency,
  }).format(cents / 100);
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
