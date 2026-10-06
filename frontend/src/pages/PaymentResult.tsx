import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { CheckCircle2, Loader2, XCircle } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { api } from "@/lib/api";

type State = "loading" | "success" | "error";

/**
 * Landing page the payment provider redirects to. In production the provider
 * also calls the webhook asynchronously; the sandbox provider is confirmed here
 * so the demo flow completes end to end.
 */
export function PaymentResultPage() {
  const [params] = useSearchParams();
  const providerRef = params.get("payment_ref") ?? "";
  const provider = params.get("provider") ?? "mock";
  const [state, setState] = useState<State>("loading");
  const [message, setMessage] = useState("");

  useEffect(() => {
    if (!providerRef) {
      setState("error");
      setMessage("Missing payment reference.");
      return;
    }
    let cancelled = false;
    api
      .post("/payments/simulate", { providerRef, provider })
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
  }, [providerRef, provider]);

  return (
    <div className="mx-auto max-w-md py-16">
      <Card>
        <CardContent className="flex flex-col items-center gap-4 py-4 text-center">
          {state === "loading" && (
            <>
              <Loader2 className="text-muted-foreground size-12 animate-spin" />
              <h1 className="text-xl font-semibold">Confirming your payment…</h1>
            </>
          )}
          {state === "success" && (
            <>
              <CheckCircle2 className="size-12 text-emerald-600" />
              <h1 className="text-xl font-semibold">Payment successful</h1>
              <p className="text-muted-foreground">Your order has been paid.</p>
              <Button asChild>
                <Link to="/orders">View your orders</Link>
              </Button>
            </>
          )}
          {state === "error" && (
            <>
              <XCircle className="text-destructive size-12" />
              <h1 className="text-xl font-semibold">Payment not completed</h1>
              <p className="text-muted-foreground">{message}</p>
              <Button variant="outline" asChild>
                <Link to="/orders">Back to orders</Link>
              </Button>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
