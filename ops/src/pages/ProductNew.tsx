import { Link, useNavigate } from "react-router-dom";
import { ArrowLeft } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { useI18n } from "@lib/i18n";
import { ProductForm } from "@/components/admin/product-form";

/** Dedicated page for creating a product, rather than a form crammed onto the list. */
export function ProductNewPage() {
  const { t } = useI18n();
  const navigate = useNavigate();

  return (
    <div className="space-y-4">
      <Button variant="ghost" size="sm" asChild>
        <Link to="/products">
          <ArrowLeft className="size-4" />
          {t("ops.products")}
        </Link>
      </Button>
      <ProductForm onDone={() => navigate("/products")} />
    </div>
  );
}
