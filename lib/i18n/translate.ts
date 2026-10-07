import { catalogs, type Locale, type MessageKey } from "./messages";

type Vars = Record<string, string | number>;

// Module-level locale so non-component code (hooks, toast helpers) can
// translate without a React context. The provider keeps it in sync.
let currentLocale: Locale = "en";

export function setCurrentLocale(locale: Locale): void {
  currentLocale = locale;
}

export function getCurrentLocale(): Locale {
  return currentLocale;
}

function interpolate(template: string, vars?: Vars): string {
  if (!vars) return template;
  return template.replace(/\{(\w+)\}/g, (_, key: string) =>
    key in vars ? String(vars[key]) : `{${key}}`,
  );
}

/** Translate a key using the active locale. Usable outside React components. */
export function t(key: MessageKey, vars?: Vars): string {
  const catalog = catalogs[currentLocale] ?? catalogs.en;
  return interpolate(catalog[key] ?? catalogs.en[key] ?? key, vars);
}
