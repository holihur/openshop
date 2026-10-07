import { Link, useParams } from "react-router-dom";
import { ArrowLeft } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { Skeleton } from "@lib/components/ui/skeleton";
import { useI18n } from "@lib/i18n";
import { useAdminProduct } from "@lib/hooks/useAdmin";
import { ProductForm } from "@/components/admin/product-form";
import { VariantsEditor } from "@/components/admin/variants-editor";

/** A deep-linkable ops product page: edit the product and its variants. */
export function ProductDetailPage() {
  const { id = "" } = useParams();
  const { t } = useI18n();
  const { data: product, isLoading } = useAdminProduct(id);

  if (isLoading) {
    return <Skeleton className="h-64 w-full" />;
  }
  if (!product) {
    return <p className="text-muted-foreground">{t("ops.productNotFound")}</p>;
  }

  return (
    <div className="space-y-4">
      <Button variant="ghost" size="sm" asChild>
        <Link to="/products">
          <ArrowLeft className="size-4" />
          {t("ops.backToProducts")}
        </Link>
      </Button>
      <ProductForm product={product} onDone={() => {}} />
      <VariantsEditor productId={product.id} />
    </div>
  );
}
