import { useState, type FormEvent } from "react";
import { Plus } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@lib/components/ui/card";
import { Input } from "@lib/components/ui/input";
import { Label } from "@lib/components/ui/label";
import { useI18n } from "@lib/i18n";
import { useCreateCoupon } from "@lib/hooks/useAdmin";

/** Coupon creation form, used by its own page. */
export function CouponForm({ onDone }: { onDone?: () => void }) {
  const { t } = useI18n();
  const create = useCreateCoupon();
  const [code, setCode] = useState("");
  const [discountType, setDiscountType] = useState<"percent" | "fixed">("percent");
  const [value, setValue] = useState("10");
  const [usageLimit, setUsageLimit] = useState("0");

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    create.mutate(
      {
        code,
        discountType,
        discountValue:
          discountType === "percent"
            ? Number.parseInt(value || "0", 10)
            : Math.round(Number.parseFloat(value || "0") * 100),
        usageLimit: Number.parseInt(usageLimit || "0", 10),
        perUserLimit: 1,
        active: true,
      },
      { onSuccess: () => onDone?.() },
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("ops.newCoupon")}</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={onSubmit} className="grid gap-3 sm:grid-cols-2">
          <div className="space-y-1">
            <Label htmlFor="c-code">{t("ops.code")}</Label>
            <Input id="c-code" value={code} onChange={(e) => setCode(e.target.value)} required />
          </div>
          <div className="space-y-1">
            <Label htmlFor="c-type">{t("ops.type")}</Label>
            <select
              id="c-type"
              value={discountType}
              onChange={(e) => setDiscountType(e.target.value as "percent" | "fixed")}
              className="border-input bg-background h-9 w-full rounded-md border px-3 text-sm"
            >
              <option value="percent">{t("ops.percent")}</option>
              <option value="fixed">{t("ops.fixed")}</option>
            </select>
          </div>
          <div className="space-y-1">
            <Label htmlFor="c-value">{t("ops.value")}</Label>
            <Input
              id="c-value"
              type="number"
              min="0"
              step={discountType === "fixed" ? "0.01" : "1"}
              value={value}
              onChange={(e) => setValue(e.target.value)}
              required
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="c-limit">{t("ops.usageLimit")}</Label>
            <Input
              id="c-limit"
              type="number"
              min="0"
              value={usageLimit}
              onChange={(e) => setUsageLimit(e.target.value)}
            />
          </div>
          <div className="sm:col-span-2">
            <Button type="submit" size="sm" disabled={create.isPending}>
              <Plus className="size-4" />
              {t("ops.createCoupon")}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
