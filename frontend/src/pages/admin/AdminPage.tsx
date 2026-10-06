import { useState } from "react";
import { Pencil, Plus, RotateCcw } from "lucide-react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ProductForm } from "@/components/admin/product-form";
import { OrderStatusBadge } from "@/components/order-status-badge";
import { useAdminOrders, useAdminProducts } from "@/hooks/useAdmin";
import { api } from "@/lib/api";
import { formatDate, formatMoney } from "@/lib/format";
import type { Order, Product } from "@/lib/types";

type Tab = "products" | "orders";

export function AdminPage() {
  const [tab, setTab] = useState<Tab>("products");

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Admin</h1>
        <p className="text-muted-foreground">Manage the catalog and review orders.</p>
      </div>

      <div className="flex gap-2">
        {(["products", "orders"] as Tab[]).map((t) => (
          <Button
            key={t}
            variant={tab === t ? "default" : "outline"}
            size="sm"
            onClick={() => setTab(t)}
            className="capitalize"
          >
            {t}
          </Button>
        ))}
      </div>

      {tab === "products" ? <AdminProducts /> : <AdminOrders />}
    </div>
  );
}

function AdminProducts() {
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

      {showForm && <ProductForm product={editing ?? undefined} onDone={closeForm} />}

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

function AdminOrders() {
  const { data, isLoading } = useAdminOrders(1, 50);
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
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
