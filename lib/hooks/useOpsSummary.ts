import { useQuery } from "@tanstack/react-query";

import { api } from "@lib/api";
import type { OpsSummary } from "@lib/types";

/**
 * The per-module console counters, bucketed into day/week/fortnight/month
 * windows. Shared by every ops page so a whole console session costs one
 * request per refresh.
 */
export function useOpsSummary() {
  return useQuery({
    queryKey: ["ops", "summary"],
    queryFn: () => api.get<OpsSummary>("/ops/stats/summary"),
    staleTime: 15_000,
    refetchInterval: 60_000,
  });
}
