import { LocaleSwitcher } from "@lib/components/locale-switcher";
import { ThemeToggle } from "@lib/components/theme-toggle";
import { useCurrency } from "@lib/currency";
import { useSite } from "@lib/hooks/useSite";
import { useI18n } from "@lib/i18n";

export function Footer() {
  const { t } = useI18n();
  const { currency, available, setCurrency } = useCurrency();
  const { data: site } = useSite();
  const tagline = site?.tagline?.trim();
  return (
    <footer className="text-muted-foreground mt-16 border-t">
      <div className="mx-auto flex max-w-6xl flex-col items-center gap-4 px-4 py-8 text-sm sm:flex-row sm:justify-between">
        <div className="flex flex-col items-center gap-1 sm:items-start">
          <p>
            © {new Date().getFullYear()} OpenShop. {t("footer.rights")}
          </p>
          {tagline ? <p>{tagline}</p> : null}
        </div>
        <div className="flex items-center gap-3">
          {available.length > 1 && (
            <select
              aria-label={t("common.currency")}
              value={currency}
              onChange={(e) => setCurrency(e.target.value)}
              className="border-input bg-background h-8 rounded-md border px-2 text-sm"
            >
              {available.map((c) => (
                <option key={c} value={c}>
                  {c}
                </option>
              ))}
            </select>
          )}
          <LocaleSwitcher />
          <ThemeToggle />
        </div>
      </div>
    </footer>
  );
}
