import { TokenManager } from "@lib/components/token-manager";
import { useI18n } from "@lib/i18n";

export function AccountTokensPage() {
  const { t } = useI18n();
  return (
    <div className="mx-auto max-w-3xl space-y-6 py-8">
      <div>
        <h1 className="text-2xl font-semibold">{t("tokens.title")}</h1>
        <p className="text-muted-foreground text-sm">{t("tokens.subtitle")}</p>
      </div>
      <TokenManager realm="front" />
    </div>
  );
}
