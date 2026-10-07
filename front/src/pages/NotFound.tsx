import { Link } from "react-router-dom";

import { Button } from "@lib/components/ui/button";
import { useI18n } from "@lib/i18n";

export function NotFoundPage() {
  const { t } = useI18n();
  return (
    <div className="py-24 text-center">
      <p className="text-muted-foreground text-sm font-medium">404</p>
      <h1 className="mt-2 text-3xl font-bold">{t("notfound.title")}</h1>
      <p className="text-muted-foreground mt-2">{t("notfound.hint")}</p>
      <Button className="mt-6" asChild>
        <Link to="/">{t("notfound.backHome")}</Link>
      </Button>
    </div>
  );
}
