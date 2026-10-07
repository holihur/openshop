/** Format a minor-unit price (e.g. cents) as a localized currency string. */
export function formatMoney(cents: number, currency = "CNY"): string {
  const code = (currency || "CNY").toUpperCase();
  try {
    return new Intl.NumberFormat(undefined, {
      style: "currency",
      currency: code,
    }).format(cents / 100);
  } catch {
    // Unknown/invalid ISO code (e.g. a typo): never crash the page.
    return `${(cents / 100).toFixed(2)} ${code}`;
  }
}

export function formatDate(value?: string): string {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}
