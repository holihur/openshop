import { useState, type FormEvent } from "react";
import { Plus } from "lucide-react";

import { Button } from "@lib/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@lib/components/ui/card";
import { Badge } from "@lib/components/ui/badge";
import { Input } from "@lib/components/ui/input";
import { Label } from "@lib/components/ui/label";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@lib/components/ui/table";
import { useCreateVariant, useUpdateVariant, useVariants } from "@lib/hooks/useAdmin";
import { useI18n } from "@lib/i18n";
import { formatMoney } from "@lib/format";

// VariantsEditor manages the SKUs of an existing product. It is only rendered
// once the product has been created, because variants reference a product id.
export function VariantsEditor({ productId }: { productId: string }) {
  const { t } = useI18n();
  const { data: variants, isLoading } = useVariants(productId);
  const create = useCreateVariant(productId);
  const update = useUpdateVariant(productId);

  const [name, setName] = useState("");
  const [sku, setSku] = useState("");
  const [price, setPrice] = useState("");
  const [cost, setCost] = useState("");
  const [stock, setStock] = useState("0");
  const [weight, setWeight] = useState("0");

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    create.mutate(
      {
        name,
        sku: sku || undefined,
        priceCents: Math.round(Number.parseFloat(price || "0") * 100),
        costCents: Math.round(Number.parseFloat(cost || "0") * 100),
        stock: Number.parseInt(stock || "0", 10),
        weightGrams: Number.parseInt(weight || "0", 10),
        active: true,
      },
      {
        onSuccess: () => {
          setName("");
          setSku("");
          setPrice("");
          setStock("0");
          setWeight("0");
        },
      },
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("ops.variants")}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-6">
        <form onSubmit={onSubmit} className="grid gap-3 sm:grid-cols-5">
          <div className="space-y-1 sm:col-span-2">
            <Label htmlFor="v-name">{t("ops.name")}</Label>
            <Input
              id="v-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Red / Large"
              required
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="v-sku">{t("ops.sku")}</Label>
            <Input id="v-sku" value={sku} onChange={(e) => setSku(e.target.value)} placeholder="auto" />
          </div>
          <div className="space-y-1">
            <Label htmlFor="v-price">{t("ops.price")}</Label>
            <Input
              id="v-price"
              type="number"
              step="0.01"
              min="0"
              value={price}
              onChange={(e) => setPrice(e.target.value)}
              placeholder={t("ops.inherit")}
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="v-cost">{t("ops.cost")}</Label>
            <Input
              id="v-cost"
              type="number"
              step="0.01"
              min="0"
              value={cost}
              onChange={(e) => setCost(e.target.value)}
              placeholder={t("ops.inherit")}
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="v-stock">{t("ops.stock")}</Label>
            <Input
              id="v-stock"
              type="number"
              min="0"
              value={stock}
              onChange={(e) => setStock(e.target.value)}
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="v-weight">{t("ops.weightG")}</Label>
            <Input
              id="v-weight"
              type="number"
              min="0"
              value={weight}
              onChange={(e) => setWeight(e.target.value)}
            />
          </div>
          <div className="sm:col-span-5">
            <Button type="submit" size="sm" disabled={create.isPending}>
              <Plus className="size-4" />
              {t("ops.addVariant")}
            </Button>
          </div>
        </form>

        {isLoading ? null : !variants || variants.length === 0 ? (
          <p className="text-muted-foreground text-sm">{t("ops.noVariants")}</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("ops.name")}</TableHead>
                <TableHead>{t("ops.sku")}</TableHead>
                <TableHead>{t("ops.price")}</TableHead>
                <TableHead>{t("ops.stock")}</TableHead>
                <TableHead>{t("common.status")}</TableHead>
                <TableHead className="text-right">{t("common.actions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {variants.map((v) => (
                <TableRow key={v.id}>
                  <TableCell className="font-medium">{v.name}</TableCell>
                  <TableCell className="font-mono text-xs">{v.sku}</TableCell>
                  <TableCell>
                    {v.priceCents > 0 ? formatMoney(v.priceCents) : <span className="text-muted-foreground">{t("ops.inherit")}</span>}
                  </TableCell>
                  <TableCell>{v.stock}</TableCell>
                  <TableCell>
                    <Badge variant={v.active ? "success" : "secondary"}>
                      {v.active ? t("common.active") : t("ops.archived")}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled={update.isPending}
                      onClick={() => update.mutate({ id: v.id, input: { active: !v.active } })}
                    >
                      {v.active ? t("ops.archive") : t("ops.restore")}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}
