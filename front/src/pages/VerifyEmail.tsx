import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { CheckCircle2, Loader2, XCircle } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { Card, CardContent } from "@lib/components/ui/card";
import { api } from "@lib/api";
import { useAuth } from "@lib/auth";

type State = "loading" | "success" | "error";

export function VerifyEmailPage() {
  const [params] = useSearchParams();
  const token = params.get("token") ?? "";
  const { reload } = useAuth();
  const [state, setState] = useState<State>("loading");
  const [message, setMessage] = useState("");

  useEffect(() => {
    if (!token) {
      setState("error");
      setMessage("This verification link is missing or invalid.");
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
          setMessage(err.message);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [token, reload]);

  return (
    <div className="mx-auto max-w-md py-16">
      <Card>
        <CardContent className="flex flex-col items-center gap-4 py-4 text-center">
          {state === "loading" && (
            <>
              <Loader2 className="text-muted-foreground size-12 animate-spin" />
              <h1 className="text-xl font-semibold">Verifying your email…</h1>
            </>
          )}
          {state === "success" && (
            <>
              <CheckCircle2 className="size-12 text-emerald-600" />
              <h1 className="text-xl font-semibold">Email verified</h1>
              <p className="text-muted-foreground">Your account is fully set up.</p>
              <Button asChild>
                <Link to="/">Start shopping</Link>
              </Button>
            </>
          )}
          {state === "error" && (
            <>
              <XCircle className="text-destructive size-12" />
              <h1 className="text-xl font-semibold">Verification failed</h1>
              <p className="text-muted-foreground">{message}</p>
              <Button variant="outline" asChild>
                <Link to="/">Back to home</Link>
              </Button>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
