import { redirect } from "next/navigation";

import { AuthForm } from "@/components/auth-form";
import { translator } from "@/lib/i18n";
import { getSiteConfig, isSignedIn } from "@/lib/session";
import { resolveLocale } from "@/app/layout";

export default async function LoginPage() {
  if (await isSignedIn()) redirect("/account/orders");
  const locale = await resolveLocale();
  const t = translator(locale);
  const site = await getSiteConfig().catch(() => null);

  return (
    <AuthForm
      mode="login"
      allowRegistration={site?.allowRegistration ?? true}
      labels={{
        email: t("auth.email"),
        password: t("auth.password"),
        name: t("auth.name"),
        submit: t("auth.signIn"),
        switchTo: t("auth.noAccount"),
        switchHref: "/register",
        failed: t("error.generic"),
      }}
    />
  );
}
