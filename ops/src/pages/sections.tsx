import { useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import { Pencil, Plus } from "lucide-react";
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
import { OrderStatusBadge } from "@lib/components/order-status-badge";
import { Pagination } from "@lib/components/pagination";
import { useI18n } from "@lib/i18n";
import { useAdminCoupons, useAdminOrders, useAdminProducts, useAdminReviews, useAuditLogs, useCategories, useCreateCoupon, useCurrencies, useDashboard, useDeleteReviewAdmin, useSetCurrencyRate, useUpdateCoupon } from "@lib/hooks/useAdmin";
import { useAdminReturns, useApproveReturn, useRejectReturn } from "@lib/hooks/useReturns";
import {
  useAdminShippingMethods,
  useAdminShippingZones,
  useCreateShippingMethod,
  useCreateShippingZone,
  useSetShippingRate,
  useUpdateShippingMethod,
} from "@lib/hooks/useShipping";
import { formatDate, formatMoney } from "@lib/format";
import type { AuditLog, Coupon, ExchangeRate, Review, ShippingMethod, ShippingZone } from "@lib/types";

export function AdminCurrency() {
  const { t } = useI18n();
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
          <CardTitle>{t("ops.setExchangeRate")}</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit} className="grid gap-3 sm:grid-cols-4">
            <div className="space-y-1">
              <Label htmlFor="c-code">{t("ops.currency")}</Label>
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
              <Label htmlFor="c-rate">{t("ops.unitsOfBase", { base: data?.base ?? "base" })}</Label>
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
                {t("ops.saveRate")}
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
                  <TableHead>{t("ops.currency")}</TableHead>
                  <TableHead>{t("ops.rateMicro")}</TableHead>
                  <TableHead>{t("ops.rate")}</TableHead>
                  <TableHead>{t("ops.updated")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow>
                  <TableCell className="font-medium">{data?.base} {t("ops.baseSuffix")}</TableCell>
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
  const { t } = useI18n();
  const [page, setPage] = useState(1);
  const [action, setAction] = useState("");
  const { data, isLoading } = useAuditLogs(page, 20, action);

  return (
    <div className="space-y-4">
      <Input
        value={action}
        onChange={(e) => {
          setAction(e.target.value);
          setPage(1);
        }}
        placeholder={t("ops.filterAction")}
        className="max-w-xs"
      />
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
                  <TableHead>{t("ops.time")}</TableHead>
                  <TableHead>{t("ops.actor")}</TableHead>
                  <TableHead>{t("ops.action")}</TableHead>
                  <TableHead>{t("ops.resource")}</TableHead>
                  <TableHead>{t("ops.ip")}</TableHead>
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
          )}
        </CardContent>
      </Card>
      {data && (
        <Pagination page={data.page} pageSize={data.pageSize} total={data.total} onChange={setPage} />
      )}
    </div>
  );
}

export function AdminZones() {
  const { t } = useI18n();
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
        <CardTitle>{t("ops.zonesRates")}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-6">
        <form onSubmit={onCreate} className="grid gap-3 sm:grid-cols-4">
          <div className="space-y-1 sm:col-span-2">
            <Label htmlFor="z-name">{t("ops.zoneName")}</Label>
            <Input id="z-name" value={name} onChange={(e) => setName(e.target.value)} required />
          </div>
          <div className="space-y-1 sm:col-span-2">
            <Label htmlFor="z-prov">{t("ops.provinces")}</Label>
            <Input id="z-prov" value={provinces} onChange={(e) => setProvinces(e.target.value)} placeholder="Beijing, Tianjin" />
          </div>
          <div className="sm:col-span-4">
            <Button type="submit" size="sm" disabled={createZone.isPending}>
              <Plus className="size-4" />
              {t("ops.addZone")}
            </Button>
          </div>
        </form>

        {zones && zones.length > 0 && (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("ops.zone")}</TableHead>
                <TableHead>{t("ops.provincesLabel")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {zones.map((z: ShippingZone) => (
                <TableRow key={z.id}>
                  <TableCell className="font-medium">{z.name}</TableCell>
                  <TableCell className="text-muted-foreground text-sm">
                    {z.provinces.length > 0 ? z.provinces.join(", ") : t("ops.allRegions")}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}

        <form onSubmit={onSetRate} className="grid gap-3 sm:grid-cols-4">
          <div className="space-y-1">
            <Label>{t("ops.zone")}</Label>
            <select
              value={zoneId}
              onChange={(e) => setZoneId(e.target.value)}
              className="border-input bg-background h-9 w-full rounded-md border px-3 text-sm"
              required
            >
              <option value="">{t("ops.select")}</option>
              {zones?.map((z: ShippingZone) => (
                <option key={z.id} value={z.id}>{z.name}</option>
              ))}
            </select>
          </div>
          <div className="space-y-1">
            <Label>{t("ops.method")}</Label>
            <select
              value={methodId}
              onChange={(e) => setMethodId(e.target.value)}
              className="border-input bg-background h-9 w-full rounded-md border px-3 text-sm"
              required
            >
              <option value="">{t("ops.select")}</option>
              {methods?.map((m: ShippingMethod) => (
                <option key={m.id} value={m.id}>{m.name}</option>
              ))}
            </select>
          </div>
          <div className="space-y-1">
            <Label htmlFor="z-flat">{t("ops.flatRate")}</Label>
            <Input id="z-flat" type="number" step="0.01" min="0" value={flat} onChange={(e) => setFlat(e.target.value)} />
          </div>
          <div className="space-y-1">
            <Label htmlFor="z-perkg">{t("ops.perKg")}</Label>
            <Input id="z-perkg" type="number" step="0.01" min="0" value={perKg} onChange={(e) => setPerKg(e.target.value)} />
          </div>
          <div className="sm:col-span-4">
            <Button type="submit" size="sm" disabled={setRate.isPending}>
              {t("ops.saveRate")}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}

export function AdminReviews() {
  const { t } = useI18n();
  const [page, setPage] = useState(1);
  const { data, isLoading } = useAdminReviews(page, 20);
  const remove = useDeleteReviewAdmin();

  if (isLoading) {
    return <Skeleton className="h-64 w-full" />;
  }

  return (
    <div className="space-y-4">
      <Card className="py-0">
        <CardContent className="px-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("ops.rating")}</TableHead>
                <TableHead>{t("ops.review")}</TableHead>
                <TableHead>{t("ops.product")}</TableHead>
                <TableHead>{t("ops.date")}</TableHead>
                <TableHead className="text-right">{t("common.actions")}</TableHead>
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
                    <Link to={`/products/${r.productId}`} className="hover:underline">
                      {r.productId.slice(0, 8)}
                    </Link>
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
                      {t("common.delete")}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
      {data && (
        <Pagination page={data.page} pageSize={data.pageSize} total={data.total} onChange={setPage} />
      )}
    </div>
  );
}

export function AdminShipping() {
  const { t } = useI18n();
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
          <CardTitle>{t("ops.newShippingMethod")}</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit} className="grid gap-3 sm:grid-cols-4">
            <div className="space-y-1 sm:col-span-2">
              <Label htmlFor="s-name">{t("ops.name")}</Label>
              <Input id="s-name" value={name} onChange={(e) => setName(e.target.value)} required />
            </div>
            <div className="space-y-1">
              <Label htmlFor="s-rate">{t("ops.flatRate")}</Label>
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
              <Label htmlFor="s-threshold">{t("ops.freeOver")}</Label>
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
                {t("ops.addMethod")}
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
                  <TableHead>{t("ops.name")}</TableHead>
                  <TableHead>{t("ops.flatRateHeader")}</TableHead>
                  <TableHead>{t("ops.freeOverHeader")}</TableHead>
                  <TableHead>{t("common.status")}</TableHead>
                  <TableHead className="text-right">{t("common.actions")}</TableHead>
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
                        {m.active ? t("common.active") : t("common.inactive")}
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
                        {m.active ? t("ops.disable") : t("ops.enable")}
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
  const { t } = useI18n();

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
    { label: t("ops.revenue"), value: formatMoney(data.revenueCents) },
    { label: t("ops.paidOrders"), value: data.paidOrders },
    { label: t("ops.pendingOrders"), value: data.pendingOrders },
    { label: t("ops.totalOrders"), value: data.totalOrders },
    { label: t("ops.totalProducts"), value: data.totalProducts },
    { label: t("ops.totalUsers"), value: data.totalUsers },
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
            <CardTitle>{t("ops.lowStock", { count: data.lowStock.length })}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            {data.lowStock.slice(0, 10).map((item) => (
              <div key={`${item.type}-${item.id}`} className="flex items-center justify-between text-sm">
                <span>
                  {item.title}
                  {item.variantName ? ` · ${item.variantName}` : ""}
                </span>
                <Badge variant={item.stock === 0 ? "destructive" : "warning"}>
                  {t("ops.left", { count: item.stock })}
                </Badge>
              </div>
            ))}
          </CardContent>
        </Card>
      )}

      <Card className="py-0">
        <CardHeader>
          <CardTitle className="py-4">{t("ops.recentOrders")}</CardTitle>
        </CardHeader>
        <CardContent className="px-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("orders.orderNo")}</TableHead>
                <TableHead>{t("common.status")}</TableHead>
                <TableHead>{t("orders.total")}</TableHead>
                <TableHead>{t("orders.placed")}</TableHead>
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
  const { t } = useI18n();
  const [page, setPage] = useState(1);
  const { data, isLoading } = useAdminCoupons(page, 20);
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
          <CardTitle>{t("ops.newCoupon")}</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit} className="grid gap-3 sm:grid-cols-5">
            <div className="space-y-1 sm:col-span-2">
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
            <div className="sm:col-span-5">
              <Button type="submit" size="sm" disabled={create.isPending}>
                <Plus className="size-4" />
                {t("ops.createCoupon")}
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
                  <TableHead>{t("ops.code")}</TableHead>
                  <TableHead>{t("ops.discount")}</TableHead>
                  <TableHead>{t("ops.usedLimit")}</TableHead>
                  <TableHead>{t("common.status")}</TableHead>
                  <TableHead className="text-right">{t("common.actions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data?.items.map((c: Coupon) => (
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
                        {c.active ? t("common.active") : t("common.inactive")}
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
                          {c.active ? t("ops.deactivate") : t("ops.activate")}
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          disabled={update.isPending}
                          onClick={() => {
                            const v = window.prompt(t("ops.usageLimitPrompt"), String(c.usageLimit));
                            if (v !== null) {
                              update.mutate({ id: c.id, input: { usageLimit: Number.parseInt(v, 10) || 0 } });
                            }
                          }}
                        >
                          {t("ops.editLimit")}
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
      {data && (
        <Pagination page={data.page} pageSize={data.pageSize} total={data.total} onChange={setPage} />
      )}
    </div>
  );
}

export function AdminProducts() {
  const { t } = useI18n();
  const navigate = useNavigate();
  const [page, setPage] = useState(1);
  const [keyword, setKeyword] = useState("");
  const [categoryId, setCategoryId] = useState("");
  const { data: categories } = useCategories();
  const { data, isLoading } = useAdminProducts(page, 20, { keyword, categoryId });
  const [creating, setCreating] = useState(false);

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-muted-foreground text-sm">
          {t("ops.productCount", { count: data?.total ?? 0 })}
        </p>
        {!creating && (
          <Button size="sm" onClick={() => setCreating(true)}>
            <Plus className="size-4" />
            {t("ops.newProduct")}
          </Button>
        )}
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <Input
          value={keyword}
          onChange={(e) => {
            setKeyword(e.target.value);
            setPage(1);
          }}
          placeholder={t("ops.searchProducts")}
          className="max-w-xs"
        />
        <select
          value={categoryId}
          onChange={(e) => {
            setCategoryId(e.target.value);
            setPage(1);
          }}
          className="border-input bg-background h-9 rounded-md border px-3 text-sm"
        >
          <option value="">{t("ops.allCategories")}</option>
          {categories?.map((c) => (
            <option key={c.id} value={c.id}>
              {c.name}
            </option>
          ))}
        </select>
      </div>

      {creating && <ProductForm onDone={() => setCreating(false)} />}

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
                  <TableHead>{t("ops.title")}</TableHead>
                  <TableHead>{t("ops.price")}</TableHead>
                  <TableHead>{t("ops.stock")}</TableHead>
                  <TableHead>{t("common.status")}</TableHead>
                  <TableHead className="text-right">{t("common.actions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data?.items.map((product) => (
                  <TableRow key={product.id}>
                    <TableCell className="max-w-xs truncate font-medium">
                      <Link to={`/products/${product.id}`} className="hover:underline">
                        {product.title}
                      </Link>
                    </TableCell>
                    <TableCell>{formatMoney(product.priceCents, product.currency)}</TableCell>
                    <TableCell>{product.stock}</TableCell>
                    <TableCell>
                      <Badge variant={product.status === "published" ? "success" : "secondary"}>
                        {product.status}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right">
                      <Button variant="ghost" size="sm" onClick={() => navigate(`/products/${product.id}`)}>
                        <Pencil className="size-4" />
                        {t("common.edit")}
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {data && (
        <Pagination page={data.page} pageSize={data.pageSize} total={data.total} onChange={setPage} />
      )}
    </div>
  );
}

export function AdminOrders() {
  const { t } = useI18n();
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState("");
  const { data, isLoading } = useAdminOrders(page, 20, status);

  if (isLoading) {
    return (
      <div className="space-y-2">
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <select
        value={status}
        onChange={(e) => {
          setStatus(e.target.value);
          setPage(1);
        }}
        className="border-input bg-background h-9 rounded-md border px-3 text-sm"
      >
        <option value="">{t("ops.allStatuses")}</option>
        {(["pending_payment", "paid", "shipped", "completed", "cancelled", "refunded"] as const).map((s) => (
          <option key={s} value={s}>
            {t(`status.${s}`)}
          </option>
        ))}
      </select>
      <Card className="py-0">
        <CardContent className="px-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("orders.orderNo")}</TableHead>
                <TableHead>{t("common.status")}</TableHead>
                <TableHead>{t("ops.items")}</TableHead>
                <TableHead>{t("orders.total")}</TableHead>
                <TableHead>{t("orders.placed")}</TableHead>
                <TableHead className="text-right">{t("common.actions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data?.items.map((order) => (
                <TableRow key={order.id}>
                  <TableCell className="font-mono text-xs">
                    <Link to={`/orders/${order.id}`} className="hover:underline">
                      {order.orderNo}
                    </Link>
                  </TableCell>
                  <TableCell>
                    <OrderStatusBadge status={order.status} />
                  </TableCell>
                  <TableCell>{order.items.length}</TableCell>
                  <TableCell>{formatMoney(order.totalCents, order.currency)}</TableCell>
                  <TableCell className="text-muted-foreground text-sm">
                    {formatDate(order.createdAt)}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button variant="ghost" size="sm" asChild>
                      <Link to={`/orders/${order.id}`}>{t("ops.view")}</Link>
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
      {data && (
        <Pagination page={data.page} pageSize={data.pageSize} total={data.total} onChange={setPage} />
      )}
    </div>
  );
}

export function AdminReturns() {
  const { t } = useI18n();
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState("");
  const { data, isLoading } = useAdminReturns(page, 20, status);
  const approve = useApproveReturn();
  const reject = useRejectReturn();

  if (isLoading) {
    return <Skeleton className="h-64 w-full" />;
  }

  const variant: Record<string, "warning" | "success" | "destructive"> = {
    requested: "warning",
    approved: "success",
    rejected: "destructive",
  };

  return (
    <div className="space-y-4">
      <select
        value={status}
        onChange={(e) => {
          setStatus(e.target.value);
          setPage(1);
        }}
        className="border-input bg-background h-9 rounded-md border px-3 text-sm"
      >
        <option value="">{t("ops.allStatuses")}</option>
        {(["requested", "approved", "rejected"] as const).map((s) => (
          <option key={s} value={s}>
            {t(`return.${s}`)}
          </option>
        ))}
      </select>
    <Card className="py-0">
      <CardContent className="px-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("orders.orderNo")}</TableHead>
              <TableHead>{t("ops.refundReason")}</TableHead>
              <TableHead>{t("common.status")}</TableHead>
              <TableHead>{t("ops.date")}</TableHead>
              <TableHead className="text-right">{t("common.actions")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data?.items.map((r) => (
              <TableRow key={r.id}>
                <TableCell className="font-mono text-xs">
                  <Link to={`/orders/${r.orderId}`} className="hover:underline">
                    {r.orderId.slice(0, 8)}
                  </Link>
                </TableCell>
                <TableCell className="max-w-md">{r.reason}</TableCell>
                <TableCell>
                  <Badge variant={variant[r.status] ?? "secondary"}>{t(`return.${r.status}`)}</Badge>
                </TableCell>
                <TableCell className="text-muted-foreground text-sm">
                  {formatDate(r.createdAt)}
                </TableCell>
                <TableCell className="text-right">
                  {r.status === "requested" ? (
                    <div className="flex justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        disabled={approve.isPending}
                        onClick={() => approve.mutate(r.id)}
                      >
                        {t("ops.approve")}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        disabled={reject.isPending}
                        onClick={() => reject.mutate(r.id)}
                      >
                        {t("ops.reject")}
                      </Button>
                    </div>
                  ) : null}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
      {data && (
        <Pagination page={data.page} pageSize={data.pageSize} total={data.total} onChange={setPage} />
      )}
    </div>
  );
}
