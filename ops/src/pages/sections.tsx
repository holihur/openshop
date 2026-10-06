import { useState, type FormEvent } from "react";
import { Pencil, Plus, RotateCcw, Truck } from "lucide-react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { Button } from "@lib/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@lib/components/ui/card";
import { Badge } from "@lib/components/ui/badge";
import { Input } from "@lib/components/ui/input";
import { Label } from "@lib/components/ui/label";
import { Skeleton } from "@lib/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@lib/components/ui/table";
import { ProductForm } from "@/components/admin/product-form";
import { VariantsEditor } from "@/components/admin/variants-editor";
import { OrderStatusBadge } from "@lib/components/order-status-badge";
import { Pagination } from "@lib/components/pagination";
import { useAdminCoupons, useAdminOrders, useAdminProducts, useAdminReviews, useAuditLogs, useCreateCoupon, useCurrencies, useDashboard, useDeleteReviewAdmin, useSetCurrencyRate, useUpdateCoupon } from "@lib/hooks/useAdmin";
import {
  useAdminShippingMethods,
  useAdminShippingZones,
  useCreateShippingMethod,
  useCreateShippingZone,
  useSetShippingRate,
  useUpdateShippingMethod,
} from "@lib/hooks/useShipping";
import { api } from "@lib/api";
import { formatDate, formatMoney } from "@lib/format";
import type { AuditLog, Coupon, ExchangeRate, Order, Product, Review, ShippingMethod, ShippingZone } from "@lib/types";

export function AdminCurrency() {
  const { data, isLoading } = useCurrencies();
  const saveRate = useSetCurrencyRate();
  const [code, setCode] = useState("");
  const [rate, setRate] = useState("1");

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    saveRate.mutate(
      {
        code: code.trim().toUpperCase(),
        rateMicro: Math.round(Number.parseFloat(rate || "1") * 1_000_000),
      },
      { onSuccess: () => setCode("") },
    );
  }

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>Set exchange rate</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit} className="grid gap-3 sm:grid-cols-4">
            <div className="space-y-1">
              <Label htmlFor="c-code">Currency</Label>
              <Input
                id="c-code"
                value={code}
                onChange={(e) => setCode(e.target.value)}
                placeholder="USD"
                maxLength={8}
                required
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="c-rate">Units of {data?.base ?? "base"} per 1</Label>
              <Input
                id="c-rate"
                type="number"
                step="0.000001"
                min="0"
                value={rate}
                onChange={(e) => setRate(e.target.value)}
                required
              />
            </div>
            <div className="flex items-end">
              <Button type="submit" size="sm" disabled={saveRate.isPending}>
                Save rate
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      <Card className="py-0">
        <CardContent className="px-0">
          {isLoading ? (
            <div className="p-4">
              <Skeleton className="h-10 w-full" />
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Currency</TableHead>
                  <TableHead>Rate (micro)</TableHead>
                  <TableHead>Rate</TableHead>
                  <TableHead>Updated</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow>
                  <TableCell className="font-medium">{data?.base} (base)</TableCell>
                  <TableCell>1,000,000</TableCell>
                  <TableCell>1</TableCell>
                  <TableCell className="text-muted-foreground text-sm">—</TableCell>
                </TableRow>
                {data?.rates.map((r: ExchangeRate) => (
                  <TableRow key={r.currency}>
                    <TableCell className="font-medium">{r.currency}</TableCell>
                    <TableCell>{r.rateMicro.toLocaleString()}</TableCell>
                    <TableCell>{(r.rateMicro / 1_000_000).toFixed(4)}</TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {formatDate(r.updatedAt)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

export function AdminAudit() {
  const { data, isLoading } = useAuditLogs();

  if (isLoading) {
    return <Skeleton className="h-64 w-full" />;
  }

  return (
    <Card className="py-0">
      <CardContent className="px-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Time</TableHead>
              <TableHead>Actor</TableHead>
              <TableHead>Action</TableHead>
              <TableHead>Resource</TableHead>
              <TableHead>IP</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data?.items.map((l: AuditLog) => (
              <TableRow key={l.id}>
                <TableCell className="text-muted-foreground whitespace-nowrap text-xs">
                  {formatDate(l.createdAt)}
                </TableCell>
                <TableCell className="text-sm">{l.actorRole || "—"}</TableCell>
                <TableCell className="font-mono text-xs">{l.action}</TableCell>
                <TableCell className="text-muted-foreground text-xs">
                  {l.resourceType}
                  {l.resourceId ? ` · ${l.resourceId.slice(0, 8)}` : ""}
                </TableCell>
                <TableCell className="text-muted-foreground text-xs">{l.ip}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}

export function AdminZones() {
  const { data: zones } = useAdminShippingZones();
  const { data: methods } = useAdminShippingMethods();
  const createZone = useCreateShippingZone();
  const setRate = useSetShippingRate();

  const [name, setName] = useState("");
  const [provinces, setProvinces] = useState("");
  const [zoneId, setZoneId] = useState("");
  const [methodId, setMethodId] = useState("");
  const [flat, setFlat] = useState("0");
  const [perKg, setPerKg] = useState("0");

  function onCreate(e: FormEvent) {
    e.preventDefault();
    createZone.mutate(
      {
        name,
        provinces: provinces.split(",").map((p) => p.trim()).filter(Boolean),
        active: true,
      },
      { onSuccess: () => { setName(""); setProvinces(""); } },
    );
  }

  function onSetRate(e: FormEvent) {
    e.preventDefault();
    if (!zoneId || !methodId) return;
    setRate.mutate({
      zoneId,
      methodId,
      input: {
        flatRateCents: Math.round(Number.parseFloat(flat || "0") * 100),
        freeThresholdCents: 0,
        perKgCents: Math.round(Number.parseFloat(perKg || "0") * 100),
      },
    });
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Shipping zones & rates</CardTitle>
      </CardHeader>
      <CardContent className="space-y-6">
        <form onSubmit={onCreate} className="grid gap-3 sm:grid-cols-4">
          <div className="space-y-1 sm:col-span-2">
            <Label htmlFor="z-name">Zone name</Label>
            <Input id="z-name" value={name} onChange={(e) => setName(e.target.value)} required />
          </div>
          <div className="space-y-1 sm:col-span-2">
            <Label htmlFor="z-prov">Provinces (comma-separated)</Label>
            <Input id="z-prov" value={provinces} onChange={(e) => setProvinces(e.target.value)} placeholder="Beijing, Tianjin" />
          </div>
          <div className="sm:col-span-4">
            <Button type="submit" size="sm" disabled={createZone.isPending}>
              <Plus className="size-4" />
              Add zone
            </Button>
          </div>
        </form>

        {zones && zones.length > 0 && (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Zone</TableHead>
                <TableHead>Provinces</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {zones.map((z: ShippingZone) => (
                <TableRow key={z.id}>
                  <TableCell className="font-medium">{z.name}</TableCell>
                  <TableCell className="text-muted-foreground text-sm">
                    {z.provinces.length > 0 ? z.provinces.join(", ") : "All regions"}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}

        <form onSubmit={onSetRate} className="grid gap-3 sm:grid-cols-4">
          <div className="space-y-1">
            <Label>Zone</Label>
            <select
              value={zoneId}
              onChange={(e) => setZoneId(e.target.value)}
              className="border-input bg-background h-9 w-full rounded-md border px-3 text-sm"
              required
            >
              <option value="">Select…</option>
              {zones?.map((z: ShippingZone) => (
                <option key={z.id} value={z.id}>{z.name}</option>
              ))}
            </select>
          </div>
          <div className="space-y-1">
            <Label>Method</Label>
            <select
              value={methodId}
              onChange={(e) => setMethodId(e.target.value)}
              className="border-input bg-background h-9 w-full rounded-md border px-3 text-sm"
              required
            >
              <option value="">Select…</option>
              {methods?.map((m: ShippingMethod) => (
                <option key={m.id} value={m.id}>{m.name}</option>
              ))}
            </select>
          </div>
          <div className="space-y-1">
            <Label htmlFor="z-flat">Flat rate (CNY)</Label>
            <Input id="z-flat" type="number" step="0.01" min="0" value={flat} onChange={(e) => setFlat(e.target.value)} />
          </div>
          <div className="space-y-1">
            <Label htmlFor="z-perkg">Per kg (CNY)</Label>
            <Input id="z-perkg" type="number" step="0.01" min="0" value={perKg} onChange={(e) => setPerKg(e.target.value)} />
          </div>
          <div className="sm:col-span-4">
            <Button type="submit" size="sm" disabled={setRate.isPending}>
              Save rate
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}

export function AdminReviews() {
  const { data, isLoading } = useAdminReviews();
  const remove = useDeleteReviewAdmin();

  if (isLoading) {
    return <Skeleton className="h-64 w-full" />;
  }

  return (
    <Card className="py-0">
      <CardContent className="px-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Rating</TableHead>
              <TableHead>Review</TableHead>
              <TableHead>Product</TableHead>
              <TableHead>Date</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data?.items.map((r: Review) => (
              <TableRow key={r.id}>
                <TableCell>{"★".repeat(r.rating)}</TableCell>
                <TableCell className="max-w-md">
                  {r.title && <span className="font-medium">{r.title}</span>}
                  {r.body && <p className="text-muted-foreground text-sm">{r.body}</p>}
                </TableCell>
                <TableCell className="text-muted-foreground font-mono text-xs">
                  {r.productId.slice(0, 8)}
                </TableCell>
                <TableCell className="text-muted-foreground text-sm">
                  {formatDate(r.createdAt)}
                </TableCell>
                <TableCell className="text-right">
                  <Button
                    variant="ghost"
                    size="sm"
                    disabled={remove.isPending}
                    onClick={() => remove.mutate(r.id)}
                  >
                    Delete
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}

export function AdminShipping() {
  const { data, isLoading } = useAdminShippingMethods();
  const create = useCreateShippingMethod();
  const update = useUpdateShippingMethod();
  const [name, setName] = useState("");
  const [rate, setRate] = useState("0");
  const [threshold, setThreshold] = useState("0");

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    create.mutate(
      {
        name,
        flatRateCents: Math.round(Number.parseFloat(rate || "0") * 100),
        freeThresholdCents: Math.round(Number.parseFloat(threshold || "0") * 100),
        active: true,
      },
      {
        onSuccess: () => {
          setName("");
          setRate("0");
          setThreshold("0");
        },
      },
    );
  }

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>New shipping method</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit} className="grid gap-3 sm:grid-cols-4">
            <div className="space-y-1 sm:col-span-2">
              <Label htmlFor="s-name">Name</Label>
              <Input id="s-name" value={name} onChange={(e) => setName(e.target.value)} required />
            </div>
            <div className="space-y-1">
              <Label htmlFor="s-rate">Flat rate (CNY)</Label>
              <Input
                id="s-rate"
                type="number"
                step="0.01"
                min="0"
                value={rate}
                onChange={(e) => setRate(e.target.value)}
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="s-threshold">Free over (CNY)</Label>
              <Input
                id="s-threshold"
                type="number"
                step="0.01"
                min="0"
                value={threshold}
                onChange={(e) => setThreshold(e.target.value)}
              />
            </div>
            <div className="sm:col-span-4">
              <Button type="submit" size="sm" disabled={create.isPending}>
                <Plus className="size-4" />
                Add method
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      <Card className="py-0">
        <CardContent className="px-0">
          {isLoading ? (
            <div className="p-4">
              <Skeleton className="h-10 w-full" />
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Flat rate</TableHead>
                  <TableHead>Free over</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data?.map((m: ShippingMethod) => (
                  <TableRow key={m.id}>
                    <TableCell className="font-medium">{m.name}</TableCell>
                    <TableCell>{formatMoney(m.flatRateCents)}</TableCell>
                    <TableCell>
                      {m.freeThresholdCents > 0 ? formatMoney(m.freeThresholdCents) : "—"}
                    </TableCell>
                    <TableCell>
                      <Badge variant={m.active ? "success" : "secondary"}>
                        {m.active ? "active" : "inactive"}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        disabled={update.isPending}
                        onClick={() =>
                          update.mutate({
                            id: m.id,
                            input: {
                              name: m.name,
                              flatRateCents: m.flatRateCents,
                              freeThresholdCents: m.freeThresholdCents,
                              active: !m.active,
                              sort: m.sort,
                            },
                          })
                        }
                      >
                        {m.active ? "Disable" : "Enable"}
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

export function AdminDashboard() {
  const { data, isLoading } = useDashboard();

  if (isLoading || !data) {
    return (
      <div className="grid gap-4 sm:grid-cols-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <Skeleton key={i} className="h-24 w-full" />
        ))}
      </div>
    );
  }

  const cards = [
    { label: "Revenue", value: formatMoney(data.revenueCents) },
    { label: "Paid orders", value: data.paidOrders },
    { label: "Pending orders", value: data.pendingOrders },
    { label: "Total orders", value: data.totalOrders },
    { label: "Products", value: data.totalProducts },
    { label: "Customers", value: data.totalUsers },
  ];

  return (
    <div className="space-y-6">
      <div className="grid gap-4 sm:grid-cols-3">
        {cards.map((c) => (
          <Card key={c.label}>
            <CardContent>
              <p className="text-muted-foreground text-sm">{c.label}</p>
              <p className="mt-1 text-2xl font-semibold">{c.value}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      {data.lowStock.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Low stock ({data.lowStock.length})</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            {data.lowStock.slice(0, 10).map((item) => (
              <div key={`${item.type}-${item.id}`} className="flex items-center justify-between text-sm">
                <span>
                  {item.title}
                  {item.variantName ? ` · ${item.variantName}` : ""}
                </span>
                <Badge variant={item.stock === 0 ? "destructive" : "warning"}>
                  {item.stock} left
                </Badge>
              </div>
            ))}
          </CardContent>
        </Card>
      )}

      <Card className="py-0">
        <CardHeader>
          <CardTitle className="py-4">Recent orders</CardTitle>
        </CardHeader>
        <CardContent className="px-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Order</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Total</TableHead>
                <TableHead>Placed</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.recentOrders.map((order) => (
                <TableRow key={order.id}>
                  <TableCell className="font-mono text-xs">{order.orderNo}</TableCell>
                  <TableCell>
                    <OrderStatusBadge status={order.status} />
                  </TableCell>
                  <TableCell>{formatMoney(order.totalCents, order.currency)}</TableCell>
                  <TableCell className="text-muted-foreground text-sm">
                    {formatDate(order.createdAt)}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}

export function AdminCoupons() {
  const { data, isLoading } = useAdminCoupons();
  const create = useCreateCoupon();
  const update = useUpdateCoupon();
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
      {
        onSuccess: () => {
          setCode("");
          setValue("10");
          setUsageLimit("0");
        },
      },
    );
  }

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>New coupon</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit} className="grid gap-3 sm:grid-cols-5">
            <div className="space-y-1 sm:col-span-2">
              <Label htmlFor="c-code">Code</Label>
              <Input id="c-code" value={code} onChange={(e) => setCode(e.target.value)} required />
            </div>
            <div className="space-y-1">
              <Label htmlFor="c-type">Type</Label>
              <select
                id="c-type"
                value={discountType}
                onChange={(e) => setDiscountType(e.target.value as "percent" | "fixed")}
                className="border-input bg-background h-9 w-full rounded-md border px-3 text-sm"
              >
                <option value="percent">Percent (%)</option>
                <option value="fixed">Fixed (CNY)</option>
              </select>
            </div>
            <div className="space-y-1">
              <Label htmlFor="c-value">Value</Label>
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
              <Label htmlFor="c-limit">Usage limit</Label>
              <Input
                id="c-limit"
                type="number"
                min="0"
                value={usageLimit}
                onChange={(e) => setUsageLimit(e.target.value)}
              />
            </div>
            <div className="sm:col-span-5">
              <Button type="submit" size="sm" disabled={create.isPending}>
                <Plus className="size-4" />
                Create coupon
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      <Card className="py-0">
        <CardContent className="px-0">
          {isLoading ? (
            <div className="p-4">
              <Skeleton className="h-10 w-full" />
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Code</TableHead>
                  <TableHead>Discount</TableHead>
                  <TableHead>Used / limit</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data?.map((c: Coupon) => (
                  <TableRow key={c.id}>
                    <TableCell className="font-mono text-xs">{c.code}</TableCell>
                    <TableCell>
                      {c.discountType === "percent"
                        ? `${c.discountValue}%`
                        : formatMoney(c.discountValue)}
                    </TableCell>
                    <TableCell>
                      {c.usedCount} / {c.usageLimit === 0 ? "∞" : c.usageLimit}
                    </TableCell>
                    <TableCell>
                      <Badge variant={c.active ? "success" : "secondary"}>
                        {c.active ? "active" : "inactive"}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          disabled={update.isPending}
                          onClick={() => update.mutate({ id: c.id, input: { active: !c.active } })}
                        >
                          {c.active ? "Deactivate" : "Activate"}
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          disabled={update.isPending}
                          onClick={() => {
                            const v = window.prompt("Usage limit (0 = unlimited)", String(c.usageLimit));
                            if (v !== null) {
                              update.mutate({ id: c.id, input: { usageLimit: Number.parseInt(v, 10) || 0 } });
                            }
                          }}
                        >
                          Edit limit
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

export function AdminProducts() {
  const [page, setPage] = useState(1);
  const { data, isLoading } = useAdminProducts(page, 50);
  const [editing, setEditing] = useState<Product | null>(null);
  const [creating, setCreating] = useState(false);

  const showForm = creating || editing !== null;
  const closeForm = () => {
    setCreating(false);
    setEditing(null);
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-muted-foreground text-sm">{data?.total ?? 0} products</p>
        {!showForm && (
          <Button size="sm" onClick={() => setCreating(true)}>
            <Plus className="size-4" />
            New product
          </Button>
        )}
      </div>

      {showForm && (
        <div className="space-y-4">
          <ProductForm product={editing ?? undefined} onDone={closeForm} />
          {editing && <VariantsEditor productId={editing.id} />}
        </div>
      )}

      <Card className="py-0">
        <CardContent className="px-0">
          {isLoading ? (
            <div className="space-y-2 p-4">
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-10 w-full" />
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Title</TableHead>
                  <TableHead>Price</TableHead>
                  <TableHead>Stock</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data?.items.map((product) => (
                  <TableRow key={product.id}>
                    <TableCell className="max-w-xs truncate font-medium">{product.title}</TableCell>
                    <TableCell>{formatMoney(product.priceCents, product.currency)}</TableCell>
                    <TableCell>{product.stock}</TableCell>
                    <TableCell>
                      <Badge variant={product.status === "published" ? "success" : "secondary"}>
                        {product.status}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                          setCreating(false);
                          setEditing(product);
                        }}
                      >
                        <Pencil className="size-4" />
                        Edit
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {data && data.total > data.pageSize && (
        <div className="flex justify-center gap-2">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
            Previous
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={page * data.pageSize >= data.total}
            onClick={() => setPage((p) => p + 1)}
          >
            Next
          </Button>
        </div>
      )}
    </div>
  );
}

export function AdminOrders() {
  const [page, setPage] = useState(1);
  const { data, isLoading } = useAdminOrders(page, 50);
  const queryClient = useQueryClient();

  const refund = useMutation({
    mutationFn: (orderId: string) =>
      api.post<Order>(`/admin/orders/${orderId}/refund`, { reason: "admin refund" }),
    onSuccess: () => {
      toast.success("Order refunded");
      void queryClient.invalidateQueries({ queryKey: ["admin", "orders"] });
    },
    onError: (error: Error) => toast.error(error.message),
  });

  const ship = useMutation({
    mutationFn: ({ id, trackingNo }: { id: string; trackingNo: string }) =>
      api.post<Order>(`/admin/orders/${id}/ship`, { trackingNo }),
    onSuccess: () => {
      toast.success("Order shipped");
      void queryClient.invalidateQueries({ queryKey: ["admin", "orders"] });
    },
    onError: (error: Error) => toast.error(error.message),
  });

  const complete = useMutation({
    mutationFn: (id: string) => api.post<Order>(`/admin/orders/${id}/complete`),
    onSuccess: () => {
      toast.success("Order completed");
      void queryClient.invalidateQueries({ queryKey: ["admin", "orders"] });
    },
    onError: (error: Error) => toast.error(error.message),
  });

  if (isLoading) {
    return (
      <div className="space-y-2">
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
      </div>
    );
  }

  return (
    <Card className="py-0">
      <CardContent className="px-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Order</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Items</TableHead>
              <TableHead>Total</TableHead>
              <TableHead>Placed</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data?.items.map((order) => (
              <TableRow key={order.id}>
                <TableCell className="font-mono text-xs">{order.orderNo}</TableCell>
                <TableCell>
                  <OrderStatusBadge status={order.status} />
                </TableCell>
                <TableCell>{order.items.length}</TableCell>
                <TableCell>{formatMoney(order.totalCents, order.currency)}</TableCell>
                <TableCell className="text-muted-foreground text-sm">
                  {formatDate(order.createdAt)}
                </TableCell>
                <TableCell className="text-right">
                  <div className="flex justify-end gap-1">
                    {order.status === "paid" && (
                      <Button
                        variant="ghost"
                        size="sm"
                        disabled={ship.isPending}
                        onClick={() => {
                          const trackingNo = window.prompt("Tracking number", "");
                          if (trackingNo !== null) ship.mutate({ id: order.id, trackingNo });
                        }}
                      >
                        <Truck className="size-4" />
                        Ship
                      </Button>
                    )}
                    {order.status === "shipped" && (
                      <Button
                        variant="ghost"
                        size="sm"
                        disabled={complete.isPending}
                        onClick={() => complete.mutate(order.id)}
                      >
                        Complete
                      </Button>
                    )}
                    {["paid", "shipped", "completed"].includes(order.status) && (
                      <Button
                        variant="ghost"
                        size="sm"
                        disabled={refund.isPending}
                        onClick={() => refund.mutate(order.id)}
                      >
                        <RotateCcw className="size-4" />
                        Refund
                      </Button>
                    )}
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
      {data && data.total > data.pageSize && (
        <Pagination
          page={data.page}
          pageSize={data.pageSize}
          total={data.total}
          onChange={setPage}
        />
      )}
    </Card>
  );
}
