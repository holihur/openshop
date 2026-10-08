import { Link, useNavigate } from "react-router-dom";
import { ArrowLeft } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { useI18n } from "@lib/i18n";
import { CouponForm } from "@/components/admin/coupon-form";

export function CouponNewPage() {
  const { t } = useI18n();
  const navigate = useNavigate();

  return (
    <div className="space-y-4">
      <Button variant="ghost" size="sm" asChild>
        <Link to="/coupons">
          <ArrowLeft className="size-4" />
          {t("ops.coupons")}
        </Link>
      </Button>
      <CouponForm onDone={() => navigate("/coupons")} />
    </div>
  );
}
