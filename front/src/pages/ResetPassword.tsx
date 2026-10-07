import { useState, type FormEvent } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { toast } from "sonner";

import { Button } from "@lib/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@lib/components/ui/card";
import { Input } from "@lib/components/ui/input";
import { Label } from "@lib/components/ui/label";
import { api } from "@lib/api";
import { useI18n } from "@lib/i18n";

export function ResetPasswordPage() {
  const { t } = useI18n();
  const [params] = useSearchParams();
  const navigate = useNavigate();
  const token = params.get("token") ?? "";

  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setSubmitting(true);
    try {
      await api.post("/auth/password/reset", { token, newPassword: password });
      toast.success(t("auth.resetTitle"));
      navigate("/login", { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : t("common.unexpectedError"));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="mx-auto max-w-md py-8">
      <Card>
        <CardHeader>
          <CardTitle>{t("auth.resetTitle")}</CardTitle>
          <CardDescription>{t("auth.resetSubtitle")}</CardDescription>
        </CardHeader>
        <CardContent>
          {token === "" ? (
            <div className="space-y-4 text-sm">
              <p className="text-destructive">{t("auth.invalidLink")}</p>
              <Button variant="outline" asChild>
                <Link to="/forgot-password">{t("auth.requestNewLink")}</Link>
              </Button>
            </div>
          ) : (
            <form onSubmit={onSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="password">{t("auth.newPassword")}</Label>
                <Input
                  id="password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete="new-password"
                  minLength={8}
                  required
                />
                <p className="text-muted-foreground text-xs">{t("auth.passwordHint")}</p>
              </div>
              {error && <p className="text-destructive text-sm">{error}</p>}
              <Button type="submit" className="w-full" disabled={submitting}>
                {submitting ? t("auth.updating") : t("auth.updatePassword")}
              </Button>
            </form>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
