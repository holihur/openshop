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

  const providers = site?.socialProviders ?? [];
  // Single sign-on providers are configured in the ops console; an unusable one
  // is absent from the payload rather than rendered as a broken button.
  const sso = site?.oidcEnabled ? (site.oidcProviders ?? []) : [];

  return (
    <div className="space-y-4">
      {sso.length > 0 && (
        <div className="mx-auto max-w-sm space-y-2">
          {sso.map((provider) => (
            <a
              key={provider.id}
              href={`/api/auth/oidc/start${
                provider.id ? `?provider=${encodeURIComponent(provider.id)}` : ""
              }`}
              className="block w-full rounded-md border px-4 py-2 text-center text-sm"
            >
              {t("auth.signInWith", { provider: provider.name })}
            </a>
          ))}
        </div>
      )}
      {providers.length > 0 && (
        <div className="mx-auto max-w-sm space-y-2">
          {providers.map((provider) => (
            <a
              key={provider.id}
              href={`/api/auth/social/${encodeURIComponent(provider.id)}/start`}
              className="block w-full rounded-md border px-4 py-2 text-center text-sm"
            >
              {t("auth.signInWith", { provider: provider.name })}
            </a>
          ))}
        </div>
      )}
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
    </div>
  );
}
