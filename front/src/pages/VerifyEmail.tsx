import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { CheckCircle2, Loader2, XCircle } from "lucide-react";
import { errorMessage } from "@lib/errors";
import { Button } from "@lib/components/ui/button";
import { Card, CardContent } from "@lib/components/ui/card";
import { api } from "@lib/api";
import { useAuth } from "@lib/auth";
import { useI18n } from "@lib/i18n";

type State = "loading" | "success" | "error";

export function VerifyEmailPage() {
  const { t } = useI18n();
  const [params] = useSearchParams();
  const token = params.get("token") ?? "";
  const { reload } = useAuth();
  const [state, setState] = useState<State>("loading");
  const [message, setMessage] = useState("");

  useEffect(() => {
    if (!token) {
      setState("error");
      setMessage(t("auth.invalidLink"));
      return;
    }
    let cancelled = false;
    api
      .post("/auth/email/verify", { token })
      .then(() => {
        if (!cancelled) {
          setState("success");
          void reload();
        }
      })
      .catch((err: Error) => {
        if (!cancelled) {
          setState("error");
          setMessage(errorMessage(err));
        }
      });
    return () => {
      cancelled = true;
    };
  }, [token, reload, t]);

  return (
    <div className="mx-auto max-w-md py-16">
      <Card>
        <CardContent className="flex flex-col items-center gap-4 py-4 text-center">
          {state === "loading" && (
            <>
              <Loader2 className="text-muted-foreground size-12 animate-spin" />
              <h1 className="text-xl font-semibold">{t("auth.verifyingEmail")}</h1>
            </>
          )}
          {state === "success" && (
            <>
              <CheckCircle2 className="size-12 text-emerald-600" />
              <h1 className="text-xl font-semibold">{t("auth.verified")}</h1>
              <p className="text-muted-foreground">{t("auth.accountReady")}</p>
              <Button asChild>
                <Link to="/">{t("auth.startShopping")}</Link>
              </Button>
            </>
          )}
          {state === "error" && (
            <>
              <XCircle className="text-destructive size-12" />
              <h1 className="text-xl font-semibold">{t("auth.verifyFailed")}</h1>
              <p className="text-muted-foreground">{message}</p>
              <Button variant="outline" asChild>
                <Link to="/">{t("auth.backHome")}</Link>
              </Button>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
