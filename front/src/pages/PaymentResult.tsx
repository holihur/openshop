import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { CheckCircle2, Loader2, XCircle } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { Card, CardContent } from "@lib/components/ui/card";
import { api } from "@lib/api";
import { useAuth } from "@lib/auth";
import { useI18n } from "@lib/i18n";

type State = "loading" | "success" | "error";

/**
 * Landing page the payment provider redirects to. In production the provider
 * also calls the webhook asynchronously; the sandbox provider is confirmed here
 * so the demo flow completes end to end.
 */
export function PaymentResultPage() {
  const { t } = useI18n();
  const [params] = useSearchParams();
  const { user } = useAuth();
  const providerRef = params.get("payment_ref") ?? "";
  const provider = params.get("provider") ?? "mock";
  const [state, setState] = useState<State>("loading");
  const [message, setMessage] = useState("");

  useEffect(() => {
    if (!providerRef) {
      setState("error");
      setMessage(t("payment.missingRef"));
      return;
    }
    let cancelled = false;
    const endpoint = user ? "/payments/simulate" : "/guest/payments/simulate";
    api
      .post(endpoint, { providerRef, provider })
      .then(() => {
        if (!cancelled) setState("success");
      })
      .catch((err: Error) => {
        if (!cancelled) {
          setState("error");
          setMessage(err.message);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [providerRef, provider, user, t]);

  return (
    <div className="mx-auto max-w-md py-16">
      <Card>
        <CardContent className="flex flex-col items-center gap-4 py-4 text-center">
          {state === "loading" && (
            <>
              <Loader2 className="text-muted-foreground size-12 animate-spin" />
              <h1 className="text-xl font-semibold">{t("payment.confirming")}</h1>
            </>
          )}
          {state === "success" && (
            <>
              <CheckCircle2 className="size-12 text-emerald-600" />
              <h1 className="text-xl font-semibold">{t("payment.success")}</h1>
              <p className="text-muted-foreground">{t("payment.paid")}</p>
              <Button asChild>
                <Link to="/orders">{t("payment.viewOrders")}</Link>
              </Button>
            </>
          )}
          {state === "error" && (
            <>
              <XCircle className="text-destructive size-12" />
              <h1 className="text-xl font-semibold">{t("payment.notCompleted")}</h1>
              <p className="text-muted-foreground">{message}</p>
              <Button variant="outline" asChild>
                <Link to="/orders">{t("payment.backToOrders")}</Link>
              </Button>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
