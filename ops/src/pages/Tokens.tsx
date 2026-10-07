import { TokenManager } from "@lib/components/token-manager";
import { useI18n } from "@lib/i18n";

export function TokensPage() {
  const { t } = useI18n();
  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold">{t("ops.tokens")}</h1>
        <p className="text-muted-foreground text-sm">{t("tokens.opsSubtitle")}</p>
      </div>
      <TokenManager realm="ops" />
    </div>
  );
}
