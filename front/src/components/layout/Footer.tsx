import { useI18n } from "@lib/i18n";

export function Footer() {
  const { t } = useI18n();
  return (
    <footer className="text-muted-foreground mt-16 border-t">
      <div className="mx-auto flex max-w-6xl flex-col items-center justify-between gap-2 px-4 py-8 text-sm sm:flex-row">
        <p>
          © {new Date().getFullYear()} OpenShop. {t("footer.rights")}
        </p>
        <p>{t("footer.tagline")}</p>
      </div>
    </footer>
  );
}
