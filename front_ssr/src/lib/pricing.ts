import { cookies } from "next/headers";

import { apiGet } from "@/lib/api";
import { formatMoney } from "@/lib/format";
import type { Locale } from "@/lib/i18n";

const MICRO = 1_000_000;
export const CURRENCY_COOKIE = "ssr_currency";

interface CurrenciesResponse {
  base: string;
  rates: { currency: string; rateMicro: number }[];
}

/**
 * Prices in the shopper's chosen display currency.
 *
 * The order is always settled in the store's base currency; this only changes
 * what is displayed, and the rate is applied with the same arithmetic the API
 * uses so a converted total matches what the shopper is charged.
 */
export interface Pricing {
  base: string;
  currency: string;
  available: string[];
  /** Converts a base-currency amount to the display currency. */
  convert: (baseCents: number) => number;
  /** Formats a base-currency amount in the display currency. */
  format: (baseCents: number) => string;
}

export async function getPricing(locale: Locale): Promise<Pricing> {
  const [rates, jar] = await Promise.all([
    // A changed rate must reach shoppers quickly; a minute is the same window
    // the site configuration uses.
    apiGet<CurrenciesResponse>("/currencies", { revalidate: 60, tags: ["currencies"] }).catch(
      () => ({ base: "CNY", rates: [] }) as CurrenciesResponse,
    ),
    cookies(),
  ]);

  const base = (rates.base || "CNY").toUpperCase();
  const available = [
    ...new Set(
      [base, ...rates.rates.map((r) => (r.currency || "").toUpperCase())].filter((c) =>
        /^[A-Z]{3}$/.test(c),
      ),
    ),
  ];

  const chosen = (jar.get(CURRENCY_COOKIE)?.value ?? "").toUpperCase();
  // An unknown or stale choice falls back to the base currency, so a price can
  // never be formatted with a rate that no longer exists.
  const currency = available.includes(chosen) ? chosen : base;
  const rateMicro =
    currency === base ? MICRO : (rates.rates.find((r) => r.currency === currency)?.rateMicro ?? MICRO);

  const convert = (baseCents: number) => Math.round((baseCents * rateMicro) / MICRO);
  return {
    base,
    currency,
    available,
    convert,
    format: (baseCents: number) => formatMoney(convert(baseCents), currency, locale),
  };
}
