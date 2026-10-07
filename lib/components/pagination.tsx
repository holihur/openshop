import { ChevronLeft, ChevronRight } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { useI18n } from "@lib/i18n";

export function Pagination({
  page,
  pageSize,
  total,
  onChange,
}: {
  page: number;
  pageSize: number;
  total: number;
  onChange: (page: number) => void;
}) {
  const { t } = useI18n();
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  if (totalPages <= 1) return null;

  return (
    <div className="mt-8 flex items-center justify-center gap-3">
      <Button
        variant="outline"
        size="sm"
        disabled={page <= 1}
        onClick={() => onChange(page - 1)}
      >
        <ChevronLeft className="size-4" />
        {t("common.previous")}
      </Button>
      <span className="text-muted-foreground text-sm">
        {t("common.pageOf", { page, total: totalPages })}
      </span>
      <Button
        variant="outline"
        size="sm"
        disabled={page >= totalPages}
        onClick={() => onChange(page + 1)}
      >
        {t("common.next")}
        <ChevronRight className="size-4" />
      </Button>
    </div>
  );
}
