import { Link, useNavigate } from "react-router-dom";
import { ArrowLeft } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { useI18n } from "@lib/i18n";
import { CategoryForm } from "@/components/admin/category-form";

export function CategoryNewPage() {
  const { t } = useI18n();
  const navigate = useNavigate();

  return (
    <div className="space-y-4">
      <Button variant="ghost" size="sm" asChild>
        <Link to="/categories">
          <ArrowLeft className="size-4" />
          {t("ops.categories")}
        </Link>
      </Button>
      <CategoryForm onDone={() => navigate("/categories")} />
    </div>
  );
}
