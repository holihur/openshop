import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

import { en, supportedLocales, type Locale, type MessageKey } from "./messages";
import { registerCatalogue, setCurrentLocale } from "./translate";

const STORAGE_KEY = "openshop.locale";

type Vars = Record<string, string | number>;

interface I18nContextValue {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  t: (key: MessageKey, vars?: Vars) => string;
}

const I18nContext = createContext<I18nContextValue | null>(null);

type Catalogue = Record<MessageKey, string>;

/**
 * Catalogues are held in a ref and filled on demand, so switching language
 * fetches that language's messages instead of shipping every language to every
 * shopper. Until it arrives (one frame) English is shown.
 */
const loadedCatalogues: Partial<Record<Locale, Catalogue>> = { en };

const catalogueLoads: Partial<Record<Locale, Promise<Catalogue>>> = {};

function loadCatalogue(locale: Locale): Promise<Catalogue> {
  if (locale === "en") return Promise.resolve(en);
  const pending = catalogueLoads[locale];
  if (pending) return pending;
  const load = import("./messages.zh").then((module) => module.zh as Catalogue);
  catalogueLoads[locale] = load;
  return load;
}

function isLocale(value: string | null): value is Locale {
  return value !== null && (supportedLocales as string[]).includes(value);
}

function detectLocale(): Locale {
  if (typeof localStorage !== "undefined") {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (isLocale(stored)) return stored;
  }
  if (typeof navigator !== "undefined" && navigator.language?.toLowerCase().startsWith("zh")) {
    return "zh";
  }
  return "en";
}

function interpolate(template: string, vars?: Vars): string {
  if (!vars) return template;
  return template.replace(/\{(\w+)\}/g, (_, key: string) =>
    key in vars ? String(vars[key]) : `{${key}}`,
  );
}

/** Provides translations and the active locale. Persists the choice and keeps
 * <html lang> in sync so screen readers and the browser agree. */
export function I18nProvider({
  children,
  defaultLocale,
}: {
  children: ReactNode;
  defaultLocale?: Locale;
}) {
  const [locale, setLocaleState] = useState<Locale>(() => defaultLocale ?? detectLocale());

  useEffect(() => {
    document.documentElement.lang = locale;
    setCurrentLocale(locale);
    try {
      localStorage.setItem(STORAGE_KEY, locale);
    } catch {
      // Storage can be unavailable (private mode); ignore.
    }
  }, [locale]);

  const setLocale = useCallback((next: Locale) => setLocaleState(next), []);

  // Fetch the messages for the active locale, then re-render with them.
  const [catalogueVersion, setCatalogueVersion] = useState(0);
  useEffect(() => {
    let cancelled = false;
    void loadCatalogue(locale).then((catalogue) => {
      if (cancelled) return;
      loadedCatalogues[locale] = catalogue;
      // Non-React callers read the same registry.
      registerCatalogue(locale, catalogue);
      setCatalogueVersion((v) => v + 1);
    });
    return () => {
      cancelled = true;
    };
  }, [locale]);

  const t = useCallback(
    (key: MessageKey, vars?: Vars) => {
      // English is always present, so a translation can never be missing.
      const catalog = loadedCatalogues[locale] ?? en;
      return interpolate(catalog[key] ?? en[key] ?? key, vars);
    },
    // catalogueVersion is a dependency so the translator is rebuilt once the
    // active catalogue has arrived.
    [locale, catalogueVersion],
  );

  const value = useMemo(() => ({ locale, setLocale, t }), [locale, setLocale, t]);
  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n(): I18nContextValue {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error("useI18n must be used within an I18nProvider");
  return ctx;
}
