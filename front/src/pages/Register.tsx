import { useState, type FormEvent } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { errorMessage } from "@lib/errors";
import { Button } from "@lib/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@lib/components/ui/card";
import { Input } from "@lib/components/ui/input";
import { Label } from "@lib/components/ui/label";
import { useAuth } from "@lib/auth";
import { useI18n } from "@lib/i18n";
import { useSite } from "@lib/hooks/useSite";

export function RegisterPage() {
  const { register } = useAuth();
  const { t } = useI18n();
  const { data: site } = useSite();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [referralCode, setReferralCode] = useState(() => searchParams.get("ref") ?? "");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  if (site && !site.allowRegistration) {
    return (
      <div className="mx-auto max-w-md py-16 text-center">
        <h1 className="text-xl font-semibold">{t("auth.registrationDisabled")}</h1>
        <p className="text-muted-foreground mt-2 text-sm">
          {t("auth.registrationDisabledHint")}
        </p>
        <Button className="mt-6" asChild>
          <Link to="/login">{t("auth.signIn")}</Link>
        </Button>
      </div>
    );
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setSubmitting(true);
    try {
      await register({ name, email, password, ...(referralCode ? { referralCode } : {}) });
      navigate("/", { replace: true });
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="mx-auto max-w-md py-8">
      <Card>
        <CardHeader>
          <CardTitle>{t("auth.registerTitle")}</CardTitle>
          <CardDescription>{t("auth.createSubtitle")}</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">{t("auth.name")}</Label>
              <Input id="name" value={name} onChange={(e) => setName(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="email">{t("auth.email")}</Label>
              <Input
                id="email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                autoComplete="email"
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">{t("auth.password")}</Label>
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
            <div className="space-y-2">
              <Label htmlFor="referral">{t("auth.referralCode")}</Label>
              <Input
                id="referral"
                value={referralCode}
                onChange={(e) => setReferralCode(e.target.value)}
                placeholder={t("auth.referralCodeHint")}
              />
            </div>
            {error && <p className="text-destructive text-sm">{error}</p>}
            <Button type="submit" className="w-full" disabled={submitting}>
              {submitting ? t("auth.registering") : t("auth.createAccount")}
            </Button>
          </form>
          <p className="text-muted-foreground mt-4 text-center text-sm">
            {t("auth.haveAccount")}{" "}
            <Link to="/login" className="text-foreground underline">
              {t("auth.signIn")}
            </Link>
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
