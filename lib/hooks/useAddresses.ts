import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api } from "@lib/api";
import { t } from "@lib/i18n";
import { useAuth } from "@lib/auth";
import type { Address } from "@lib/types";

export interface AddressInput {
  recipient: string;
  phone?: string;
  province?: string;
  city?: string;
  district?: string;
  line1: string;
  postalCode?: string;
  default?: boolean;
}

const ADDRESSES_KEY = ["addresses"] as const;

export function useAddresses() {
  const { user } = useAuth();
  return useQuery({
    queryKey: ADDRESSES_KEY,
    queryFn: () => api.get<Address[]>("/addresses"),
    enabled: Boolean(user),
  });
}

function useAddressMutation<TArgs>(fn: (args: TArgs) => Promise<unknown>, message?: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: fn,
    onSuccess: () => {
      if (message) toast.success(message);
      void queryClient.invalidateQueries({ queryKey: ADDRESSES_KEY });
    },
    onError: (error: Error) => toast.error(error.message),
  });
}

export function useCreateAddress() {
  return useAddressMutation((input: AddressInput) => api.post<Address>("/addresses", input), t("toast.addressSaved"));
}

export function useUpdateAddress() {
  return useAddressMutation(
    ({ id, input }: { id: string; input: AddressInput }) => api.patch<Address>(`/addresses/${id}`, input),
    t("toast.addressUpdated"),
  );
}

export function useDeleteAddress() {
  return useAddressMutation((id: string) => api.del(`/addresses/${id}`), t("toast.addressRemoved"));
}

export function useSetDefaultAddress() {
  return useAddressMutation((id: string) => api.post<Address>(`/addresses/${id}/default`), t("toast.defaultUpdated"));
}
