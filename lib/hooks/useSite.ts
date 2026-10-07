import { useQuery } from "@tanstack/react-query";

import { api } from "@lib/api";
import type { Site } from "@lib/types";

// Public site configuration (banner, public URL). Cached for a few minutes;
// operators edit the values from the ops console.
export function useSite() {
  return useQuery({
    queryKey: ["site"],
    queryFn: () => api.get<Site>("/site"),
    staleTime: 5 * 60_000,
  });
}
