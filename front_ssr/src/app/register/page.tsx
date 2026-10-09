import { redirect } from "next/navigation";

import { AuthForm } from "@/components/auth-form";
import { translator } from "@/lib/i18n";
import { getSiteConfig, isSignedIn } from "@/lib/session";
import { resolveLocale } from "@/app/layout";

export default async function RegisterPage() {
  if (await isSignedIn()) redirect("/account/orders");
  const locale = await resolveLocale();
  const t = translator(locale);
  const site = await getSiteConfig().catch(() => null);

  // The operator can turn registration off; the API rejects it too, so this is
  // only about not showing a form that cannot work.
  if (site && !site.allowRegistration) {
    return <p className="text-muted-foreground text-center">{t("error.generic")}</p>;
  }

  return (
    <AuthForm
      mode="register"
      allowRegistration
      labels={{
        email: t("auth.email"),
        password: t("auth.password"),
        name: t("auth.name"),
        submit: t("auth.register"),
        switchTo: t("auth.haveAccount"),
        switchHref: "/login",
        failed: t("error.generic"),
      }}
    />
  );
}
