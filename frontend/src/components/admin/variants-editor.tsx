import { useState, type FormEvent } from "react";
import { Plus } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useCreateVariant, useUpdateVariant, useVariants } from "@/hooks/useAdmin";
import { formatMoney } from "@/lib/format";

// VariantsEditor manages the SKUs of an existing product. It is only rendered
// once the product has been created, because variants reference a product id.
export function VariantsEditor({ productId }: { productId: string }) {
  const { data: variants, isLoading } = useVariants(productId);
  const create = useCreateVariant(productId);
  const update = useUpdateVariant(productId);

  const [name, setName] = useState("");
  const [sku, setSku] = useState("");
  const [price, setPrice] = useState("");
  const [stock, setStock] = useState("0");
  const [weight, setWeight] = useState("0");

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    create.mutate(
      {
        name,
        sku: sku || undefined,
        priceCents: Math.round(Number.parseFloat(price || "0") * 100),
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
        <CardTitle>Variants (SKUs)</CardTitle>
      </CardHeader>
      <CardContent className="space-y-6">
        <form onSubmit={onSubmit} className="grid gap-3 sm:grid-cols-5">
          <div className="space-y-1 sm:col-span-2">
            <Label htmlFor="v-name">Name</Label>
            <Input
              id="v-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Red / Large"
              required
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="v-sku">SKU</Label>
            <Input id="v-sku" value={sku} onChange={(e) => setSku(e.target.value)} placeholder="auto" />
          </div>
          <div className="space-y-1">
            <Label htmlFor="v-price">Price</Label>
            <Input
              id="v-price"
              type="number"
              step="0.01"
              min="0"
              value={price}
              onChange={(e) => setPrice(e.target.value)}
              placeholder="inherit"
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="v-stock">Stock</Label>
            <Input
              id="v-stock"
              type="number"
              min="0"
              value={stock}
              onChange={(e) => setStock(e.target.value)}
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="v-weight">Weight (g)</Label>
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
              Add variant
            </Button>
          </div>
        </form>

        {isLoading ? null : !variants || variants.length === 0 ? (
          <p className="text-muted-foreground text-sm">
            No variants. This product sells as a single SKU using its own price and stock.
          </p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>SKU</TableHead>
                <TableHead>Price</TableHead>
                <TableHead>Stock</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {variants.map((v) => (
                <TableRow key={v.id}>
                  <TableCell className="font-medium">{v.name}</TableCell>
                  <TableCell className="font-mono text-xs">{v.sku}</TableCell>
                  <TableCell>
                    {v.priceCents > 0 ? formatMoney(v.priceCents) : <span className="text-muted-foreground">inherit</span>}
                  </TableCell>
                  <TableCell>{v.stock}</TableCell>
                  <TableCell>
                    <Badge variant={v.active ? "success" : "secondary"}>
                      {v.active ? "active" : "archived"}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled={update.isPending}
                      onClick={() => update.mutate({ id: v.id, input: { active: !v.active } })}
                    >
                      {v.active ? "Archive" : "Restore"}
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
