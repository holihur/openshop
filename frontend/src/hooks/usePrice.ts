import { useCallback } from "react";

import { useCurrency } from "@/lib/currency";
import { formatMoney } from "@/lib/format";

// usePrice formats a base-currency amount in the user's selected currency.
export function usePrice(): (baseCents: number) => string {
  const { convert, currency } = useCurrency();
  return useCallback((baseCents: number) => formatMoney(convert(baseCents), currency), [convert, currency]);
}
