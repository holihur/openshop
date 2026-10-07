import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { useQuery } from "@tanstack/react-query";

import { api } from "@lib/api";
import type { CurrenciesResponse } from "@lib/types";

const CURRENCY_KEY = "openshop.currency";
const MICRO = 1_000_000;

interface CurrencyContextValue {
  base: string;
  currency: string;
  rateMicro: number;
  available: string[];
  setCurrency: (currency: string) => void;
  /** Convert an amount in the base currency to the selected currency. */
  convert: (baseCents: number) => number;
}

const CurrencyContext = createContext<CurrencyContextValue | null>(null);

export function CurrencyProvider({ children }: { children: ReactNode }) {
  const { data } = useQuery({
    queryKey: ["currencies"],
    queryFn: () => api.get<CurrenciesResponse>("/currencies"),
    staleTime: 10 * 60_000,
  });

  const base = data?.base ?? "CNY";
  const [currency, setCurrencyState] = useState<string>(
    () => localStorage.getItem(CURRENCY_KEY) ?? "",
  );

  useEffect(() => {
    if (!currency) setCurrencyState(base);
  }, [base, currency]);

  const setCurrency = useCallback((next: string) => {
    localStorage.setItem(CURRENCY_KEY, next);
    setCurrencyState(next);
  }, []);

  const rateMicro = useMemo(() => {
    if (currency === base) return MICRO;
    const rate = data?.rates.find((r) => r.currency === currency);
    return rate?.rateMicro ?? MICRO;
  }, [currency, base, data]);

  const convert = useCallback(
    (baseCents: number) => Math.round((baseCents * rateMicro) / MICRO),
    [rateMicro],
  );

  const available = useMemo(() => {
    // Only 3-letter ISO codes are valid; normalize case and drop junk.
    const codes = [base, ...(data?.rates ?? []).map((r) => r.currency)].map((c) =>
      (c || "").toUpperCase(),
    );
    return [...new Set(codes.filter((c) => /^[A-Z]{3}$/.test(c)))];
  }, [base, data]);

  // If the stored/selected currency is no longer offered (or was invalid),
  // fall back to the base currency so formatting never breaks.
  useEffect(() => {
    if (data && currency && !available.includes(currency)) {
      setCurrencyState(base);
      localStorage.setItem(CURRENCY_KEY, base);
    }
  }, [data, available, currency, base]);

  const value = useMemo<CurrencyContextValue>(
    () => ({ base, currency: currency || base, rateMicro, available, setCurrency, convert }),
    [base, currency, rateMicro, available, setCurrency, convert],
  );

  return <CurrencyContext.Provider value={value}>{children}</CurrencyContext.Provider>;
}

export function useCurrency(): CurrencyContextValue {
  const ctx = useContext(CurrencyContext);
  if (!ctx) throw new Error("useCurrency must be used within a CurrencyProvider");
  return ctx;
}
