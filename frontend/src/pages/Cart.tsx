import { Link, useNavigate } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Minus, Plus, ShoppingBag, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { api } from "@/lib/api";
import { formatMoney } from "@/lib/format";
import { useCart, useClearCart, useRemoveCartItem, useUpdateCartItem } from "@/hooks/useCart";
import type { Order, Payment } from "@/lib/types";

export function CartPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { data: cart, isLoading } = useCart();
  const updateItem = useUpdateCartItem();
  const removeItem = useRemoveCartItem();
  const clearCart = useClearCart();

  const checkout = useMutation({
    mutationFn: async () => {
      // Creating the order reserves stock atomically on the server.
      const order = await api.post<Order>("/orders");
      // Then open a payment session with the (sandbox) provider.
      const payment = await api.post<Payment>("/payments", {
        orderId: order.id,
        provider: "mock",
        returnUrl: `${window.location.origin}/payment/result`,
      });
      return { order, payment };
    },
    onSuccess: ({ order, payment }) => {
      void queryClient.invalidateQueries({ queryKey: ["cart"] });
      void queryClient.invalidateQueries({ queryKey: ["orders"] });
      if (payment.redirectUrl) {
        window.location.href = payment.redirectUrl;
      } else {
        navigate(`/orders/${order.id}`);
      }
    },
    onError: (error: Error) => toast.error(error.message),
  });

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-24 w-full" />
      </div>
    );
  }

  if (!cart || cart.items.length === 0) {
    return (
      <div className="py-20 text-center">
        <ShoppingBag className="text-muted-foreground mx-auto size-10" />
        <p className="mt-4 text-lg font-medium">Your cart is empty</p>
        <p className="text-muted-foreground">Add some products to get started.</p>
        <Button className="mt-6" asChild>
          <Link to="/products">Browse products</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Your cart</h1>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => clearCart.mutate()}
          disabled={clearCart.isPending}
        >
          Clear cart
        </Button>
      </div>

      <div className="grid gap-6 lg:grid-cols-[1fr_320px]">
        <div className="space-y-3">
          {cart.items.map((item) => (
            <Card key={item.productId} className="flex-row items-center gap-4 p-4">
              <Link to={`/products/${item.productId}`} className="shrink-0">
                <div className="bg-muted size-20 overflow-hidden rounded-md">
                  {item.coverImage ? (
                    <img src={item.coverImage} alt={item.title} className="size-full object-cover" />
                  ) : null}
                </div>
              </Link>
              <div className="min-w-0 flex-1">
                <Link to={`/products/${item.productId}`} className="font-medium hover:underline">
                  {item.title}
                </Link>
                <p className="text-muted-foreground text-sm">
                  {formatMoney(item.priceCents, item.currency)}
                </p>
              </div>
              <div className="flex items-center rounded-md border">
                <Button
                  variant="ghost"
                  size="icon"
                  className="size-8"
                  disabled={item.quantity <= 1 || updateItem.isPending}
                  onClick={() =>
                    updateItem.mutate({ productId: item.productId, quantity: item.quantity - 1 })
                  }
                >
                  <Minus className="size-3" />
                </Button>
                <span className="w-8 text-center text-sm">{item.quantity}</span>
                <Button
                  variant="ghost"
                  size="icon"
                  className="size-8"
                  disabled={updateItem.isPending}
                  onClick={() =>
                    updateItem.mutate({ productId: item.productId, quantity: item.quantity + 1 })
                  }
                >
                  <Plus className="size-3" />
                </Button>
              </div>
              <Button
                variant="ghost"
                size="icon"
                disabled={removeItem.isPending}
                onClick={() => removeItem.mutate({ productId: item.productId })}
              >
                <Trash2 className="size-4" />
              </Button>
            </Card>
          ))}
        </div>

        <Card className="h-fit">
          <CardHeader>
            <CardTitle>Order summary</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">Items</span>
              <span>{cart.totalCount}</span>
            </div>
            <Separator />
            <div className="flex justify-between text-base font-semibold">
              <span>Total</span>
              <span>{formatMoney(cart.totalCents)}</span>
            </div>
            <Button
              className="w-full"
              size="lg"
              disabled={checkout.isPending}
              onClick={() => checkout.mutate()}
            >
              {checkout.isPending ? "Processing…" : "Checkout"}
            </Button>
            <p className="text-muted-foreground text-center text-xs">
              Stock is reserved atomically when you check out.
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
