import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api } from "@lib/api";
import { errorMessage } from "@lib/errors";
import { t } from "@lib/i18n";
import type { CreatedPersonalAccessToken, PersonalAccessToken, ScopeInfo } from "@lib/types";

export type TokenRealm = "front" | "ops";

const base = (realm: TokenRealm) => (realm === "ops" ? "/ops/tokens" : "/account/tokens");

export function useTokenScopes(realm: TokenRealm = "front") {
  const path = realm === "ops" ? "/ops/token-scopes" : "/account/tokens/scopes";
  return useQuery({
    queryKey: ["token-scopes", realm],
    queryFn: () => api.get<ScopeInfo[]>(path),
    staleTime: Infinity,
  });
}

export function useTokens(realm: TokenRealm = "front") {
  return useQuery({
    queryKey: ["tokens", realm],
    queryFn: () => api.get<PersonalAccessToken[]>(base(realm)),
  });
}

export interface CreateTokenInput {
  name: string;
  scopes: string[];
  cidrs: string[];
  expiresInDays?: number;
}

export function useCreateToken(realm: TokenRealm = "front") {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateTokenInput) =>
      api.post<CreatedPersonalAccessToken>(base(realm), input),
    onSuccess: () => {
      toast.success(t("toast.tokenCreated"));
      void queryClient.invalidateQueries({ queryKey: ["tokens", realm] });
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}

export function useRevokeToken(realm: TokenRealm = "front") {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.del<{ ok: boolean }>(`${base(realm)}/${id}`),
    onSuccess: () => {
      toast.success(t("toast.tokenRevoked"));
      void queryClient.invalidateQueries({ queryKey: ["tokens", realm] });
    },
    onError: (error: Error) => toast.error(errorMessage(error)),
  });
}
