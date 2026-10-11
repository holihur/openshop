import { useI18n } from "@lib/i18n";

/**
 * Placeholder while a route's chunk arrives. It matches the page frame so the
 * layout does not jump when the real screen replaces it.
 */
export function RouteFallback() {
  const { t } = useI18n();
  return (
    <div className="text-muted-foreground flex min-h-40 items-center justify-center text-sm" role="status">
      {t("common.loading")}
    </div>
  );
}
