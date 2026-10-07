import { useQuery } from "@tanstack/react-query";

import { api } from "@lib/api";

// The payment channels the shopper can choose, configured in the ops console.
export function usePaymentMethods() {
  return useQuery({
    queryKey: ["payment-methods"],
    queryFn: () => api.get<string[]>("/payment-methods"),
    staleTime: 10 * 60_000,
  });
}
