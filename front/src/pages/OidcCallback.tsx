import { useEffect } from "react";
import { useNavigate } from "react-router-dom";

import { Skeleton } from "@lib/components/ui/skeleton";
import { tokenStore } from "@lib/api";
import { useAuth } from "@lib/auth";
import { useI18n } from "@lib/i18n";

/** Completes an OIDC login: the tokens arrive in the URL fragment. */
export function OidcCallbackPage() {
  const navigate = useNavigate();
  const { reload } = useAuth();
  const { t } = useI18n();

  useEffect(() => {
    const params = new URLSearchParams(window.location.hash.replace(/^#/, ""));
    const access = params.get("access_token");
    const refresh = params.get("refresh_token");
    if (!access || !refresh) {
      navigate("/login", { replace: true });
      return;
    }
    tokenStore.set(access, refresh);
    void reload().then(() => navigate("/", { replace: true }));
  }, [navigate, reload]);

  return (
    <div className="mx-auto max-w-md space-y-4 py-16">
      <Skeleton className="h-32 w-full" />
      <p className="text-muted-foreground text-center text-sm">{t("auth.signingIn")}</p>
    </div>
  );
}
